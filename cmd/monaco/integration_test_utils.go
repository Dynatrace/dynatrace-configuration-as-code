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

package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// checks all configurations of a given project with given availability
func AssertAllConfigsAvailability(projects []project.Project, t *testing.T, environments map[string]environment.Environment, available bool) {
	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), token)
		assert.NilError(t, err)

		for _, project := range projects {
			for _, config := range project.GetConfigs() {
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

func AssertConfig(t *testing.T, client rest.DynatraceClient, environment environment.Environment, shouldBeAvailable bool, config config.Config) {
	configType := config.GetType()
	api := config.GetApi()
	name := config.GetProperties()[config.GetId()]["name"]

	_, existingId, _ := client.ExistsByName(api, name)

	if config.IsSkipDeployment(environment) {
		assert.Equal(t, existingId, "", "Object should NOT be available, but was. environment.Environment: '"+environment.GetId()+"', failed for '"+name+"' ("+configType+")")
		return
	}

	description := fmt.Sprintf("%s %s on environment %s", configType, name, environment.GetId())

	// 120 polling cycles -> Wait at most 120 * 2 seconds = 4 Minutes:
	err := rest.Wait(description, 120, func() bool {
		_, existingId, _ := client.ExistsByName(api, name)
		return (shouldBeAvailable && len(existingId) > 0) || (!shouldBeAvailable && len(existingId) == 0)
	})
	assert.NilError(t, err)

	if shouldBeAvailable {
		assert.Check(t, len(existingId) > 0, "Object should be available, but wasn't. environment.Environment: '"+environment.GetId()+"', failed for '"+name+"' ("+configType+")")
	} else {
		assert.Equal(t, existingId, "", "Object should NOT be available, but was. environment.Environment: '"+environment.GetId()+"', failed for '"+name+"' ("+configType+")")
	}
}

func FailOnAnyError(errors []error, errorMessage string) {

	for _, err := range errors {
		util.FailOnError(err, errorMessage)
	}
}

func getTimestamp() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
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
	FailOnAnyError(errs, "loading of environments failed")

	apis := api.NewApis()
	suffix = "_" + suffix

	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), token)
		assert.NilError(t, err)

		for _, api := range apis {

			values, err := client.List(api)
			assert.NilError(t, err)

			for _, value := range values {
				// For the calculated-metrics-log API, the suffix is part of the ID, not name
				if strings.HasSuffix(value.Name, suffix) || strings.HasSuffix(value.Id, suffix) {
					util.Log.Info("Deleting %s (%s)", value.Name, api.GetId())
					client.DeleteByName(api, value.Name)
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

func RunIntegrationWithCleanup(t *testing.T, configFolder, envFile, suffixTest string, testFunc func(fs afero.Fs)) {

	suffix := getTimestamp() + suffixTest
	transformers := []func(string) string{getTransformerFunc(suffix)}
	var fs = util.CreateTestFileSystem()
	err := util.RewriteConfigNames(configFolder, fs, transformers)
	if err != nil {
		log.Fatal("Error rewriting configs names")
		return
	}
	testFunc(fs)
	cleanupIntegrationTest(t, fs, envFile, suffix)
}
