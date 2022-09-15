//go:build integration
// +build integration

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
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	deploy "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/deploy/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// AssertAllConfigsAvailability checks all configurations of a given project with given availability
func AssertAllConfigsAvailability(t *testing.T, fs afero.Fs, manifestPath string, specificEnvironment string, available bool) {

	loadedManifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
	})
	FailOnAnyError(errs, "loading of environments failed")

	specificEnvs := []string{}
	if specificEnvironment != "" {
		specificEnvs = append(specificEnvs, specificEnvironment)
	}
	environments, err := loadedManifest.FilterEnvironmentsByNames([]string{specificEnvironment})
	if err != nil {
		log.Fatal("Failed to filter environments: %v", err)
	}

	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
		Apis:            api.GetApiNames(api.NewApis()),
		WorkingDir:      manifestPath,
		Manifest:        loadedManifest,
		ParametersSerde: config.DefaultParameterParsers,
	})
	FailOnAnyError(errs, "loading of projects failed")

	entities := make(map[coordinate.Coordinate]parameter.ResolvedEntity)

	for _, env := range environments {

		token, err := env.GetToken()
		assert.NilError(t, err)

		url, err := env.GetUrl()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(url, token)
		assert.NilError(t, err)

		for _, theProject := range projects {
			for _, apis := range theProject.Configs {
				for theApi, configs := range apis {
					for _, theConfig := range configs {

						if theConfig.Skip {
							continue
						}

						parameters, err := topologysort.SortParameters(theConfig.Group, theConfig.Environment, theConfig.Coordinate, theConfig.Parameters)
						FailOnAnyError(errs, "resolving of parameter values failed")

						properties, errs := deploy.ResolveParameterValues(client, &theConfig, entities, parameters, false)
						FailOnAnyError(errs, "resolving of parameter values failed")

						configName, err := deploy.ExtractConfigName(&theConfig, properties)
						assert.NilError(t, err)

						AssertConfig(t, client, env, available, theConfig, theApi, configName)
					}
				}
			}
		}
	}
}

func AssertConfig(t *testing.T, client rest.DynatraceClient, environment manifest.EnvironmentDefinition, shouldBeAvailable bool, config config.Config, apiId string, name string) {

	theApi := api.NewApis()[apiId]

	var exists bool

	if config.Skip {
		exists, _, _ = client.ExistsByName(theApi, name)
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, apiId)
		return
	}

	description := fmt.Sprintf("%s %s on environment %s", apiId, name, environment.Name)

	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err := rest.Wait(description, 120, func() bool {
		exists, _, _ = client.ExistsByName(theApi, name)
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NilError(t, err)

	if shouldBeAvailable {
		assert.Check(t, exists, "Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, apiId)
	} else {
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, apiId)
	}
}

func FailOnAnyError(errors []error, errorMessage string) {

	for _, err := range errors {
		util.FailOnError(err, errorMessage)
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

func getTransformerFunc(suffix string) func(line string) string {
	var f = func(name string) string {
		return util.ReplaceName(name, addSuffix(suffix))
	}
	return f
}

// Deletes all configs that end with "_suffix", where suffix == suffixTest+suffixTimestamp
func cleanupIntegrationTest(t *testing.T, loadedManifest manifest.Manifest, specificEnvironment, suffix string) {

	log.Info("### Cleaning up after integration test ###")

	environments := loadedManifest.Environments
	if specificEnvironment != "" {
		environments = make(map[string]manifest.EnvironmentDefinition)
		if val, ok := loadedManifest.Environments[specificEnvironment]; ok {
			environments[specificEnvironment] = val
		} else {
			log.Fatal("Environment %s not found in manifest", specificEnvironment)
			os.Exit(1)
		}
	}

	apis := api.NewApis()
	suffix = "_" + suffix

	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		url, err := environment.GetUrl()
		if err != nil {
			util.FailOnError(err, "failed to resolve URL")
		}

		client, err := rest.NewDynatraceClient(url, token)
		assert.NilError(t, err)

		for _, api := range apis {

			values, err := client.List(api)
			assert.NilError(t, err)

			for _, value := range values {
				// For the calculated-metrics-log API, the suffix is part of the ID, not name
				if strings.HasSuffix(value.Name, suffix) || strings.HasSuffix(value.Id, suffix) {
					err := client.DeleteByName(api, value.Name)
					if err != nil {
						log.Error("Failed to delete %s", value.Name)
					}
				}
			}
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

	fs := util.CreateTestFileSystem()
	RunIntegrationWithCleanupOnGivenFs(t, fs, configFolder, manifestPath, specificEnvironment, suffixTest, testFunc)
}

func RunIntegrationWithCleanupOnGivenFs(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc func(fs afero.Fs)) {
	loadedManifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           testFs,
		ManifestPath: manifestPath,
	})
	FailOnAnyError(errs, "loading of environments failed")

	configFolder, _ = filepath.Abs(configFolder)
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Intn(10000)

	suffix := fmt.Sprintf("%s_%d_%s", getTimestamp(), randomNumber, suffixTest)
	transformers := []func(string) string{getTransformerFunc(suffix)}
	err := util.RewriteConfigNames(configFolder, testFs, transformers)
	if err != nil {
		t.Fatalf("Error rewriting configs names: %s", err)
		return
	}

	defer cleanupIntegrationTest(t, loadedManifest, specificEnvironment, suffix)

	testFunc(testFs)
}
