//go:build integration_v1

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
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

// AssertConfigAvailability checks specific configuration for availability
func AssertConfigAvailability(t *testing.T, fs afero.Fs, manifestFile string, coord coordinate.Coordinate, env, projName string, available bool) {

	mani := integrationtest.LoadManifest(t, fs, manifestFile, "")

	envDefinition, found := mani.Environments[env]
	assert.True(t, found, "environment %s not found", env)

	clients := integrationtest.CreateDynatraceClients(t, envDefinition)

	projects := integrationtest.LoadProjects(t, fs, manifestFile, mani)
	project := findProjectByName(t, projects, projName)

	configsPerApi := getConfigsForEnv(t, project, envDefinition)

	var conf *config.Config = nil
	for _, configs := range configsPerApi {
		for _, c := range configs {
			if c.Coordinate == coord {
				conf = &c
			}
		}
	}

	assert.True(t, conf != nil, "config %s not found", coord)

	ctx := context.WithValue(context.TODO(), log.CtxKeyCoord{}, coord)
	ctx = context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: conf.Environment, Group: conf.Group})

	assertConfigAvailable(t, ctx, clients.ConfigClient, envDefinition, available, *conf)
}

func getConfigsForEnv(t *testing.T, project project.Project, env manifest.EnvironmentDefinition) project.ConfigsPerType {
	confsMap, found := project.Configs[env.Name]
	assert.True(t, found != false, "env %s not found", env)

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

	assert.True(t, project != nil, "project %s not found", projName)

	return *project
}

func assertConfigAvailable(t *testing.T, ctx context.Context, client client.ConfigClient, env manifest.EnvironmentDefinition, shouldBeAvailable bool, c config.Config) {

	nameParam, found := c.Parameters["name"]
	assert.True(t, found, "Config %s should have a name parameter", c.Coordinate)

	name, err := nameParam.ResolveValue(parameter.ResolveContext{})
	assert.NoError(t, err, "Config %s should have a trivial name to resolve", c.Coordinate)

	typ, ok := c.Type.(config.ClassicApiType)
	assert.True(t, ok, "Config %s should be a ClassicApiType, but is a %q", c.Coordinate, c.Type.ID())

	a, found := api.NewAPIs()[typ.Api]
	assert.True(t, found, "Config %s should have a known api, but does not. Api %s does not exist", c.Coordinate, typ.Api)

	if c.Skip {
		exists, _, err := client.ExistsWithName(ctx, a, fmt.Sprint(name))
		assert.NoError(t, err)
		assert.False(t, !exists, "Config '%s' should NOT be available on env '%s', but was. environment.", env.Name, c.Coordinate)

		return
	}

	description := fmt.Sprintf("%s on environment %s", c.Coordinate, env.Name)

	exists := false
	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err = wait(description, 120, func() bool {
		exists, _, err = client.ExistsWithName(ctx, a, fmt.Sprint(name))
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NoError(t, err)

	if shouldBeAvailable {
		assert.True(t, exists, "Object %s on environment %s should be available, but wasn't. environment.", c.Coordinate, env.Name)
	} else {
		assert.False(t, exists, "Object %s on environment %s should NOT be available, but was. environment.", c.Coordinate, env.Name)
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
	assert.NoError(t, err)

	manifestPath := path.Join(targetDir, "manifest.yaml")

	t.Log("Converting monaco-v1 to monaco-v2")
	err = monaco.RunWithFSf(fs, "monaco convert %s %s --output-folder=%s --verbose", envFile, configFolder, targetDir)
	assert.NoError(t, err, "Conversion should had happened without errors")

	exists, err := afero.Exists(fs, manifestPath)
	assert.NoError(t, err)
	assert.True(t, exists, "manifest should exist on path '%s' but does not", manifestPath)

	if doCleanup {
		t.Cleanup(func() {
			t.Log("Cleaning up environment")
			integrationtest.CleanupIntegrationTest(t, fs, manifestPath, "", suffix)
		})
	}

	t.Log("Running actual test...")

	testFunc(fs, manifestPath)
}

func appendUniqueSuffixToIntegrationTestConfigs(t *testing.T, fs afero.Fs, configFolder string, generalSuffix string) string {
	suffix := integrationtest.GenerateTestSuffix(t, fmt.Sprintf("%s_%s", generalSuffix, "v1"))
	transformers := []func(line string) string{
		func(name string) string {
			return integrationtest.ReplaceName(name, integrationtest.GetAddSuffixFunction(suffix))
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
