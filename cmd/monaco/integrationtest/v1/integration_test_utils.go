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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"path"
	"path/filepath"
	"testing"
	"time"
)

// AssertConfigAvailability checks specific configuration for availability
func AssertConfigAvailability(t *testing.T, fs afero.Fs, manifestFile string, coord coordinate.Coordinate, env, projName string, available bool) {

	mani := integrationtest.LoadManifest(t, fs, manifestFile, "")

	envDefinition, found := mani.Environments[env]
	assert.Assert(t, found, "environment %s not found", env)

	c := integrationtest.CreateDynatraceClient(t, envDefinition)

	projects := integrationtest.LoadProjects(t, fs, manifestFile, mani)
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

	assertConfigAvailable(t, c, envDefinition, available, *conf)
}

func getConfigsForEnv(t *testing.T, project project.Project, env manifest.EnvironmentDefinition) project.ConfigsPerType {
	confsMap, found := project.Configs[env.Name]
	assert.Assert(t, found != false, "env %s not found", env)

	return confsMap
}

func findProjectByName(t *testing.T, projects []project.Project, projName string) project.Project {
	var project *project.Project

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

	a, found := api.NewAPIs()[config.Type.Api]
	assert.Assert(t, found, "Config %s should have a known api, but does not. Api %s does not exist", config.Coordinate, config.Type.Api)

	if config.Skip {
		exists, _, err := client.ConfigExistsByName(a, fmt.Sprint(name))
		assert.NilError(t, err)
		assert.Check(t, !exists, "Config '%s' should NOT be available on env '%s', but was. environment.", env.Name, config.Coordinate)

		return
	}

	description := fmt.Sprintf("%s on environment %s", config.Coordinate, env.Name)

	exists := false
	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err = wait(description, 120, func() bool {
		exists, _, err = client.ConfigExistsByName(a, fmt.Sprint(name))
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NilError(t, err)

	if shouldBeAvailable {
		assert.Check(t, exists, "Object %s on environment %s should be available, but wasn't. environment.", config.Coordinate, env.Name)
	} else {
		assert.Check(t, !exists, "Object %s on environment %s should NOT be available, but was. environment.", config.Coordinate, env.Name)
	}
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

	loadedManifest, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
	})
	testutils.FailTestOnAnyError(t, errs, "loading of environments failed")

	if doCleanup {
		t.Cleanup(func() {
			t.Log("Cleaning up environment")
			integrationtest.CleanupIntegrationTest(t, fs, manifestPath, loadedManifest, suffix)
		})
	}

	t.Log("Running actual test...")

	testFunc(fs, manifestPath)
}

func appendUniqueSuffixToIntegrationTestConfigs(t *testing.T, fs afero.Fs, configFolder string, generalSuffix string) string {
	suffix := integrationtest.GenerateTestSuffix(fmt.Sprintf("%s_%s", generalSuffix, "v1"))
	transformers := []func(line string) string{
		func(name string) string {
			return integrationtest.ReplaceName(name, integrationtest.AddSuffix(suffix))
		},
	}

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
