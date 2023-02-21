//go:build integration_v1
// +build integration_v1

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

package v1

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/testutils"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	projectsV2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"math/rand"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// AssertAllConfigsAvailableInManifest checks all configurations of all environments/projects in the manifest
func AssertAllConfigsAvailableInManifest(t *testing.T, fs afero.Fs, manifestFile string) {
	mani := loadManifest(t, fs, manifestFile)
	AssertAllConfigsAvailable(t, fs, manifestFile, mani, mani.Projects, mani.Environments)
}

// AssertAllConfigsAvailable checks all configurations of a given project with given availability
func AssertAllConfigsAvailable(t *testing.T, fs afero.Fs, manifestFile string, mani manifest.Manifest, projectDefinitions map[string]manifest.ProjectDefinition, environments map[string]manifest.EnvironmentDefinition) {

	projects := loadProjects(t, fs, manifestFile, mani)

	for _, e := range environments {
		client := clientFromEnvDef(t, e)

		for _, projectDefinition := range projectDefinitions {
			log.Info("Asserting Configs from project are available: %s", projectDefinition.Name)

			p := findProjectByName(t, projects, projectDefinition.Name)

			configsPerApi := getConfigsForEnv(t, p, e)

			for _, configs := range configsPerApi {
				for _, c := range configs {
					log.Info("Asserting Config is available: %s", c.Coordinate)
					assertConfigAvailable(t, client, e, true, c)
				}
			}
		}
	}
}

func loadManifest(t *testing.T, fs afero.Fs, manifestFile string) manifest.Manifest {

	mani, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestFile,
	})
	testutils.FailTestOnAnyError(t, errs, "failed to load manifest")

	return mani
}

func loadProjects(t *testing.T, fs afero.Fs, manifestFile string, mani manifest.Manifest) []projectsV2.Project {
	dir := path.Dir(manifestFile)
	projects, errs := projectsV2.LoadProjects(fs, projectsV2.ProjectLoaderContext{
		KnownApis:       api.GetApiNameLookup(api.NewApis()),
		WorkingDir:      dir,
		Manifest:        mani,
		ParametersSerde: v2.DefaultParameterParsers,
	})
	testutils.FailTestOnAnyError(t, errs, "failed to load configs")
	assert.Assert(t, len(projects) != 0, "no projects loaded")

	return projects
}

func clientFromEnvDef(t *testing.T, envDefiniton manifest.EnvironmentDefinition) client.ConfigClient {

	u, err := envDefiniton.GetUrl()
	assert.NilError(t, err)

	token, err := envDefiniton.GetToken()
	assert.NilError(t, err)

	client, err := client.NewDynatraceClient(u, token)
	assert.NilError(t, err)

	return client
}

// AssertConfigAvailability checks specific configuration for availability
func AssertConfigAvailability(t *testing.T, fs afero.Fs, manifestFile string, coord coordinate.Coordinate, env, projName string, available bool) {

	mani := loadManifest(t, fs, manifestFile)

	envDefinition, found := mani.Environments[env]
	assert.Assert(t, found, "environment %s not found", env)

	client := clientFromEnvDef(t, envDefinition)

	projects := loadProjects(t, fs, manifestFile, mani)
	project := findProjectByName(t, projects, projName)

	configsPerApi := getConfigsForEnv(t, project, envDefinition)

	var conf *v2.Config = nil
	for _, configs := range configsPerApi {
		for _, c := range configs {
			if c.Coordinate == coord {
				conf = &c
			}
		}
	}

	assert.Assert(t, conf != nil, "config %s not found", coord)

	assertConfigAvailable(t, client, envDefinition, available, *conf)
}

func getConfigsForEnv(t *testing.T, project projectsV2.Project, env manifest.EnvironmentDefinition) projectsV2.ConfigsPerType {
	confsMap, found := project.Configs[env.Name]
	assert.Assert(t, found != false, "env %s not found", env)

	return confsMap
}

func findProjectByName(t *testing.T, projects []projectsV2.Project, projName string) projectsV2.Project {
	var project *projectsV2.Project

	for i := range projects {
		if projects[i].Id == projName {
			project = &projects[i]
			break
		}
	}

	assert.Assert(t, project != nil, "project %s not found", projName)

	return *project
}

func assertConfigAvailable(t *testing.T, client client.ConfigClient, env manifest.EnvironmentDefinition, shouldBeAvailable bool, config v2.Config) {

	nameParam, found := config.Parameters["name"]
	assert.Assert(t, found, "Config %s should have a name parameter", config.Coordinate)

	name, err := nameParam.ResolveValue(parameter.ResolveContext{})
	assert.NilError(t, err, "Config %s should have a trivial name to resolve", config.Coordinate)

	a, found := api.NewApis()[config.Type.Api]
	assert.Assert(t, found, "Config %s should have a known api, but does not. Api %s does not exist", config.Coordinate, config.Type.Api)

	if config.Skip {
		exists, _, err := client.ExistsByName(a, fmt.Sprint(name))
		assert.NilError(t, err)
		assert.Check(t, !exists, "Config '%s' should NOT be available on env '%s', but was. environment.", env.Name, config.Coordinate)

		return
	}

	description := fmt.Sprintf("%s on environment %s", config.Coordinate, env.Name)

	exists := false
	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err = wait(description, 120, func() bool {
		exists, _, err = client.ExistsByName(a, fmt.Sprint(name))
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NilError(t, err)

	if shouldBeAvailable {
		assert.Check(t, exists, "Object %s on environment %s should be available, but wasn't. environment.", config.Coordinate, env.Name)
	} else {
		assert.Check(t, !exists, "Object %s on environment %s should NOT be available, but was. environment.", config.Coordinate, env.Name)
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
		return integrationtest.ReplaceName(name, addSuffix(suffix))
	}
	return f
}

// Deletes all configs that end with "_suffix", where suffix == suffixTest+suffixTimestamp
func cleanupIntegrationTest(t *testing.T, fs afero.Fs, manifestFile, suffix string) {
	manifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestFile,
	})
	testutils.FailTestOnAnyError(t, errs, "loading manifest failed")

	environments := manifest.Environments

	apis := api.NewV1Apis()
	suffix = "_" + suffix

	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		url, err := environment.GetUrl()
		assert.NilError(t, err)

		client, err := client.NewDynatraceClient(url, token)
		assert.NilError(t, err)

		for _, api := range apis {
			if api.GetId() == "calculated-metrics-log" {
				t.Logf("Skipping cleanup of legacy log monitoring API")
				continue
			}

			values, err := client.List(api)
			if err != nil {
				t.Logf("Failed to cleanup any test configs of type %q: %v", api.GetId(), err)
			}

			for _, value := range values {
				if strings.HasSuffix(value.Name, suffix) {
					log.Info("Deleting %s (%s)", value.Name, api.GetId())
					err := client.DeleteById(api, value.Id)
					if err != nil {
						t.Logf("Failed to cleanup test config: %s (%s): %v", value.Name, api.GetId(), err)
					} else {
						log.Info("Cleaned up test config %s (%s)", value.Name, value.Id)
					}
				}
			}
		}
	}
}

// RunLegacyIntegrationWithCleanup runs an integration test and cleans up the created configs afterwards
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
func RunLegacyIntegrationWithCleanup(t *testing.T, configFolder, envFile, suffixTest string, testFunc func(fs afero.Fs, manifest string)) {
	runLegacyIntegration(t, configFolder, envFile, suffixTest, testFunc, true)
}

// RunLegacyIntegrationWithoutCleanup runs an integration test and cleans up the created configs afterwards
// This is done by using InMemoryFileReader, which rewrites the names of the read configs internally. It ready all the
// configs once and holds them in memory. Any subsequent modification of a config (applying them to an environment)
// is done based on the data in memory. The re-writing of config names ensures, that they have an unique name and don't
// conflict with other configs created by other integration tests
//
// The new naming scheme of created configs is defined in a transformer function. By default, this is:
//
// <original name>_<current timestamp><defined suffix>
// e.g. my-config_1605258980000_Suffix
func RunLegacyIntegrationWithoutCleanup(t *testing.T, configFolder, envFile, suffixTest string, testFunc func(fs afero.Fs, manifest string)) {
	runLegacyIntegration(t, configFolder, envFile, suffixTest, testFunc, false)
}

func runLegacyIntegration(t *testing.T, configFolder, envFile, suffixTest string, testFunc func(fs afero.Fs, manifest string), doCleanup bool) {
	configFolder, _ = filepath.Abs(configFolder)
	envFile, _ = filepath.Abs(envFile)

	var fs = testutils.CreateTestFileSystem()
	suffix := appendUniqueSuffixToIntegrationTestConfigs(t, fs, configFolder, suffixTest)

	targetDir, err := filepath.Abs("out")
	assert.NilError(t, err)

	manifestPath := path.Join(targetDir, "manifest.yaml")

	if doCleanup {
		t.Cleanup(func() {
			t.Log("Cleaning up environment")
			cleanupIntegrationTest(t, fs, manifestPath, suffix)
		})
	}

	t.Log("Converting monaco-v1 to monaco-v2")
	cmd := runner.BuildCli(fs)
	cmd.SetArgs([]string{
		"convert",
		"--verbose",
		envFile,
		configFolder,
		"-o",
		targetDir,
	})
	err = cmd.Execute()
	assert.NilError(t, err, "Conversion should had happened without errors")

	exists, err := afero.Exists(fs, manifestPath)
	assert.NilError(t, err)
	assert.Assert(t, exists, "manifest should exist on path '%s' but does not", manifestPath)

	t.Log("Running actual test...")

	testFunc(fs, manifestPath)
}

func appendUniqueSuffixToIntegrationTestConfigs(t *testing.T, fs afero.Fs, configFolder string, generalSuffix string) string {
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Intn(10000)

	suffix := fmt.Sprintf("%s_%d_%s", getTimestamp(), randomNumber, generalSuffix)
	transformers := []func(string) string{getTransformerFunc(suffix)}

	err := integrationtest.RewriteConfigNames(configFolder, fs, transformers)
	if err != nil {
		t.Fatalf("Error rewriting configs names: %s", err)
		return suffix
	}

	return suffix
}

func AbsOrPanicFromSlash(path string) string {
	result, err := filepath.Abs(filepath.FromSlash(path))

	if err != nil {
		panic(err)
	}

	return result
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
