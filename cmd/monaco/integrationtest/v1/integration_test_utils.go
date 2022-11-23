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
	"fmt"
	projectV1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v1"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/test"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// checks all configurations of a given project with given availability
func AssertAllConfigsAvailability(projects []projectV1.Project, t *testing.T, environments map[string]environment.Environment, available bool) {
	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), token)
		assert.NilError(t, err)

		for _, project := range projects {
			log.Info("Asserting Configs from project are available: %s", project.GetId())
			for _, config := range project.GetConfigs() {
				log.Info("Asserting Config is available: %s (%s)", config.GetProperties()[config.GetId()]["name"], config.GetType())
				AssertConfig(t, client, environment, available, config)
			}
		}
	}
}

// checks specific configuration for availability
func AssertConfigAvailability(t *testing.T, config config.Config, environment environment.Environment, available bool) {

	token, err := environment.GetToken()
	assert.NilError(t, err)

	client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), token)
	assert.NilError(t, err)

	AssertConfig(t, client, environment, available, config)
}

func AssertConfig(t *testing.T, client rest.ConfigClient, environment environment.Environment, shouldBeAvailable bool, config config.Config) {
	configType := config.GetType()
	api := config.GetApi()
	name := config.GetProperties()[config.GetId()]["name"]

	var exists bool

	if config.IsSkipDeployment(environment) {
		exists, _, _ = client.ExistsByName(api, name)
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.GetId(), name, configType)
		return
	}

	description := fmt.Sprintf("%s %s on environment %s", configType, name, environment.GetId())

	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err := rest.Wait(description, 120, func() bool {
		exists, _, _ = client.ExistsByName(api, name)
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NilError(t, err)

	if shouldBeAvailable {
		assert.Check(t, exists, "Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.GetId(), name, configType)
	} else {
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.GetId(), name, configType)
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
func cleanupIntegrationTest(t *testing.T, fs afero.Fs, envFile, suffix string) {

	environments, errs := environment.LoadEnvironmentList("", envFile, fs)
	test.FailTestOnAnyError(t, errs, "loading of environments failed")

	apis := api.NewV1Apis()
	suffix = "_" + suffix

	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), token)
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
				// For the calculated-metrics-log API, the suffix is part of the ID, not name
				if strings.HasSuffix(value.Name, suffix) || strings.HasSuffix(value.Id, suffix) {
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

func RunLegacyIntegrationWithCleanup(t *testing.T, configFolder, envFile, suffixTest string, testFunc func(fs afero.Fs)) {
	configFolder, _ = filepath.Abs(configFolder)
	envFile, _ = filepath.Abs(envFile)

	t.Setenv("CONFIG_V1", "1")

	var fs = util.CreateTestFileSystem()
	suffix := appendUniqueSuffixToIntegrationTestConfigs(t, fs, configFolder, suffixTest)

	t.Cleanup(func() {
		cleanupIntegrationTest(t, fs, envFile, suffix)
	})

	testFunc(fs)
}

func appendUniqueSuffixToIntegrationTestConfigs(t *testing.T, fs afero.Fs, configFolder string, generalSuffix string) string {
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Intn(10000)

	suffix := fmt.Sprintf("%s_%d_%s", getTimestamp(), randomNumber, generalSuffix)
	transformers := []func(string) string{getTransformerFunc(suffix)}

	err := util.RewriteConfigNames(configFolder, fs, transformers)
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
