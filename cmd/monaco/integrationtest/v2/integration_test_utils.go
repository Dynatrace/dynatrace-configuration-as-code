//go:build integration || download_restore || nightly

/**
 * @license
 * Copyright 2020 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v2

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2/topologysort"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// AssertAllConfigsAvailability checks all configurations of a given project with given availability
func AssertAllConfigsAvailability(t *testing.T, fs afero.Fs, manifestPath string, specificProjects []string, specificEnvironment string, available bool) {
	loadedManifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
	})
	testutils.FailTestOnAnyError(t, errs, "loading of environments failed")

	var specificEnvs []string
	if specificEnvironment != "" {
		specificEnvs = append(specificEnvs, specificEnvironment)
	}
	environments, err := loadedManifest.Environments.FilterByNames(specificEnvs)
	if err != nil {
		t.Fatalf("Failed to filter environments: %v", err)
	}

	projects := loadProjects(t, fs, manifestPath, loadedManifest)

	envNames := make([]string, 0, len(environments))

	for _, env := range environments {
		envNames = append(envNames, env.Name)
	}

	sortedConfigs, errs := topologysort.GetSortedConfigsForEnvironments(projects, envNames)
	testutils.FailTestOnAnyError(t, errs, "sorting configurations failed")

	checkString := "exist"
	if !available {
		checkString = "do NOT exist"
	}

	projectsToValidate := map[string]struct{}{}
	if len(specificProjects) > 0 {
		log.Info("Asserting configurations from projects: %s %s", specificProjects, checkString)
		for _, p := range specificProjects {
			projectsToValidate[p] = struct{}{}
		}
	} else {
		log.Info("Asserting configurations from all projects %s", checkString)
		for _, p := range projects {
			projectsToValidate[p.Id] = struct{}{}
		}
	}

	for envName, configs := range sortedConfigs {

		env := loadedManifest.Environments[envName]

		token, err := env.GetToken()
		assert.NilError(t, err)

		url, err := env.GetUrl()
		assert.NilError(t, err)

		client, err := client.NewDynatraceClient(url, token)
		assert.NilError(t, err)

		entities := make(map[coordinate.Coordinate]parameter.ResolvedEntity)
		var parameters []topologysort.ParameterWithName

		for _, theConfig := range configs {
			coord := theConfig.Coordinate

			if theConfig.Skip {
				entities[coord] = parameter.ResolvedEntity{
					EntityName: coord.ConfigId,
					Coordinate: coord,
					Properties: parameter.Properties{},
					Skip:       true,
				}
				continue
			}

			configParameters, errs := topologysort.SortParameters(theConfig.Group, theConfig.Environment, theConfig.Coordinate, theConfig.Parameters)
			testutils.FailTestOnAnyError(t, errs, "sorting of parameter values failed")

			parameters = append(parameters, configParameters...)

			properties, errs := deploy.ResolveParameterValues(&theConfig, entities, parameters)
			testutils.FailTestOnAnyError(t, errs, "resolving of parameter values failed")

			properties[config.IdParameter] = "NO REAL ID NEEDED FOR CHECKING AVAILABILITY"

			configName, err := extractConfigName(&theConfig, properties)
			assert.NilError(t, err)

			entities[coord] = parameter.ResolvedEntity{
				EntityName: configName,
				Coordinate: coord,
				Properties: properties,
				Skip:       false,
			}

			apis := api.NewApis()
			if _, found := projectsToValidate[coord.Project]; found {
				if theConfig.Type.IsSettings() {
					assertSetting(t, client, env, available, theConfig)
				} else if apis.Contains(theConfig.Type.Api) {
					AssertConfig(t, client, apis[theConfig.Type.Api], env, available, theConfig, configName)
				} else {
					t.Errorf("Can not assert config of unknown type %q", theConfig.Coordinate.Type)
				}
			}
		}
	}
}

func loadProjects(t *testing.T, fs afero.Fs, manifestPath string, loadedManifest manifest.Manifest) []project.Project {
	cwd, err := filepath.Abs(filepath.Dir(manifestPath))
	assert.NilError(t, err)

	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
		KnownApis:       api.GetApiNameLookup(api.NewApis()),
		WorkingDir:      cwd,
		Manifest:        loadedManifest,
		ParametersSerde: config.DefaultParameterParsers,
	})
	testutils.FailTestOnAnyError(t, errs, "loading of projects failed")
	return projects
}

func AssertConfig(t *testing.T, client client.ConfigClient, theApi api.Api, environment manifest.EnvironmentDefinition, shouldBeAvailable bool, config config.Config, name string) {

	configType := config.Coordinate.Type

	var exists bool

	if config.Skip {
		exists, _, _ = client.ConfigExistsByName(theApi, name)
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
		return
	}

	description := fmt.Sprintf("%s %s on environment %s", configType, name, environment.Name)

	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err := wait(description, 120, func() bool {
		exists, _, _ = client.ConfigExistsByName(theApi, name)
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NilError(t, err)

	if shouldBeAvailable {
		assert.Check(t, exists, "Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	} else {
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	}
}

func assertSetting(t *testing.T, c client.SettingsClient, environment manifest.EnvironmentDefinition, shouldBeAvailable bool, config config.Config) {
	expectedExtId := idutils.GenerateExternalID(config.Type.SchemaId, config.Coordinate.ConfigId)
	objects, err := c.ListSettings(config.Type.SchemaId, client.ListSettingsOptions{DiscardValue: true, Filter: func(o client.DownloadSettingsObject) bool { return o.ExternalId == expectedExtId }})
	assert.NilError(t, err)

	if len(objects) > 1 {
		t.Errorf("Expected a specific Settings Object with externalId %q, but %d are present instead.", expectedExtId, len(objects))
		return
	}

	exists := len(objects) == 1

	if config.Skip {
		assert.Check(t, !exists, "Skipped Settings Object should NOT be available but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, config.Type.SchemaId)
		return
	}

	if shouldBeAvailable {
		assert.Check(t, exists, "Settings Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, config.Type.SchemaId)
	} else {
		assert.Check(t, !exists, "Settings Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, config.Type.SchemaId)
	}
}

func getTimestamp() string {
	return time.Now().Format("20060102150405")
}

func addSuffix(suffix string) func(line string) string {
	var f = func(name string) string {
		return name + "_" + suffix
	}
	return f
}

// Deletes all configs that end with "_suffix", where suffix == suffixTest+suffixTimestamp
func cleanupIntegrationTest(t *testing.T, fs afero.Fs, manifestPath string, loadedManifest manifest.Manifest, specificEnvironment, suffix string) {

	log.Info("### Cleaning up after integration test ###")

	var specificEnvs []string
	if specificEnvironment != "" {
		specificEnvs = append(specificEnvs, specificEnvironment)
	}
	environments, err := loadedManifest.Environments.FilterByNames(specificEnvs)
	if err != nil {
		log.Fatal("Failed to filter environments: %v", err)
	}

	apis := api.NewApis()
	suffix = "_" + suffix

	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		url, err := environment.GetUrl()
		assert.NilError(t, err)

		client, err := client.NewDynatraceClient(url, token)
		assert.NilError(t, err)

		cleanupSettings(t, fs, manifestPath, loadedManifest, environment.Name, client)
		cleanupConfigs(t, apis, client, suffix)
	}
}

func cleanupConfigs(t *testing.T, apis api.ApiMap, c client.ConfigClient, suffix string) {
	for _, api := range apis {
		if api.GetId() == "calculated-metrics-log" {
			t.Logf("Skipping cleanup of legacy log monitoring API")
			continue
		}

		values, err := c.ListConfigs(api)
		if err != nil {
			t.Logf("Failed to cleanup any test configs of type %q: %v", api.GetId(), err)
		}

		for _, value := range values {
			// For the calculated-metrics-log API, the suffix is part of the ID, not name
			if strings.HasSuffix(value.Name, suffix) || strings.HasSuffix(value.Id, suffix) {
				err := c.DeleteConfigById(api, value.Id)
				if err != nil {
					t.Logf("Failed to cleanup test config: %s (%s): %v", value.Name, api.GetId(), err)
				} else {
					log.Info("Cleaned up test config %s (%s)", value.Name, value.Id)
				}
			}
		}
	}
}

func cleanupSettings(t *testing.T, fs afero.Fs, manifestPath string, loadedManifest manifest.Manifest, environment string, c client.SettingsClient) {
	projects := loadProjects(t, fs, manifestPath, loadedManifest)
	for _, p := range projects {
		cfgsForEnv, ok := p.Configs[environment]
		if !ok {
			t.Logf("Failed to cleanup Settings for env %s - no configs found", environment)
		}
		for _, configs := range cfgsForEnv {
			for _, cfg := range configs {
				if cfg.Type.IsSettings() {
					extID := idutils.GenerateExternalID(cfg.Type.SchemaId, cfg.Coordinate.ConfigId)
					deleteSettingsObjects(t, cfg.Type.SchemaId, extID, c)
				}
			}
		}
	}
}

func deleteSettingsObjects(t *testing.T, schema, externalID string, c client.SettingsClient) {
	objects, err := c.ListSettings(schema, client.ListSettingsOptions{DiscardValue: true, Filter: func(o client.DownloadSettingsObject) bool { return o.ExternalId == externalID }})
	if err != nil {
		t.Logf("Failed to cleanup test config: could not fetch settings 2.0 objects with schema ID %s: %v", schema, err)
		return
	}

	if len(objects) == 0 {
		t.Logf("No %s settings object found to cleanup: %s", schema, externalID)
		return
	}

	for _, obj := range objects {
		err := c.DeleteSettings(obj.ObjectId)
		if err != nil {
			t.Logf("Failed to cleanup test config: could not delete settings 2.0 object with object ID %s: %v", obj.ObjectId, err)
		} else {
			log.Info("Cleaned up test Setting %s (%s)", externalID, schema)
		}
	}
}

// RunIntegrationWithCleanup runs an integration test and cleans up the created configs afterwards
// This is done by using InMemoryFileReader, which rewrites the names of the read configs internally. It ready all the
// configs once and holds them in memory. Any subsequent modification of a config (applying them to an environment)
// is done based on the data in memory. The re-writing of config names ensures, that they have an unique name and don't
// conflict with other configs created by other integration tests.
//
// After the test run, the unique name also helps with finding the applied configs in all the environments and calling
// the respective DELETE api.
//
// The new naming scheme of created configs is defined in a transformer function. By default, this is:
//
// <original name>_<current timestamp><defined suffix>
// e.g. my-config_1605258980000_Suffix

func RunIntegrationWithCleanup(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc func(fs afero.Fs)) {

	fs := testutils.CreateTestFileSystem()
	runIntegrationWithCleanup(t, fs, configFolder, manifestPath, specificEnvironment, suffixTest, nil, testFunc)
}

func RunIntegrationWithCleanupOnGivenFs(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc func(fs afero.Fs)) {
	runIntegrationWithCleanup(t, testFs, configFolder, manifestPath, specificEnvironment, suffixTest, nil, testFunc)
}

func RunIntegrationWithCleanupGivenEnvs(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, envVars map[string]string, testFunc func(fs afero.Fs)) {
	fs := testutils.CreateTestFileSystem()

	runIntegrationWithCleanup(t, fs, configFolder, manifestPath, specificEnvironment, suffixTest, envVars, testFunc)
}

func runIntegrationWithCleanup(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, envVars map[string]string, testFunc func(fs afero.Fs)) {
	loadedManifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           testFs,
		ManifestPath: manifestPath,
	})
	testutils.FailTestOnAnyError(t, errs, "loading of manifest failed")

	configFolder, _ = filepath.Abs(configFolder)

	suffix := appendUniqueSuffixToIntegrationTestConfigs(t, testFs, configFolder, suffixTest)

	t.Cleanup(func() {
		cleanupIntegrationTest(t, testFs, manifestPath, loadedManifest, specificEnvironment, suffix)
	})

	for k, v := range envVars {
		t.Setenv(k, v) // register both just in case
		t.Setenv(fmt.Sprintf("%s_%s", k, suffix), v)
	}

	testFunc(testFs)
}

func appendUniqueSuffixToIntegrationTestConfigs(t *testing.T, fs afero.Fs, configFolder string, generalSuffix string) string {
	suffix := generateTestSuffix(generalSuffix)
	transformers := []func(line string) string{
		func(name string) string {
			return integrationtest.ReplaceName(name, addSuffix(suffix))
		},
		func(id string) string {
			return integrationtest.ReplaceId(id, addSuffix(suffix))
		},
	}

	err := integrationtest.RewriteConfigNames(configFolder, fs, transformers)
	if err != nil {
		t.Fatalf("Error rewriting configs names: %s", err)
		return suffix
	}

	return suffix
}

func generateTestSuffix(generalSuffix string) string {
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Intn(10000)

	return fmt.Sprintf("%s_%d_%s", getTimestamp(), randomNumber, generalSuffix)
}

func wait(description string, maxPollCount int, condition func() bool) error {

	for i := 0; i <= maxPollCount; i++ {

		if condition() {
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	log.Error("Error: Waiting for '%s' timed out!", description)

	return errors.New("Waiting for '" + description + "' timed out!")
}

func extractConfigName(conf *config.Config, properties parameter.Properties) (string, error) {
	val, found := properties[config.NameParameter]

	if !found {
		return "", fmt.Errorf("missing `name` for config")
	}

	name, success := val.(string)

	if !success {
		return "", fmt.Errorf("`name` in config is not of type string")
	}

	return name, nil
}
