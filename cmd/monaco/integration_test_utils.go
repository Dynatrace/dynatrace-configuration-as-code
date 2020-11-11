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
	"gotest.tools/assert"
)

func AssertConfigAvailable(projects []project.Project, t *testing.T, environments map[string]environment.Environment, available bool) {
	for _, project := range projects {
		for _, config := range project.GetConfigs() {
			AssertConfig(t, environments, available, config)
		}
	}
}

func AssertConfig(t *testing.T, environments map[string]environment.Environment, available bool, config config.Config) {
	configType := config.GetType()
	api := config.GetApi()
	name := config.GetProperties()[config.GetId()]["name"]
	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		_, existingId, _ := rest.GetObjectIdIfAlreadyExists(configType, api.GetUrl(environment), name, token)

		if config.IsSkipDeployment(environment) {
			assert.Equal(t, existingId, "", "Object should NOT be available, but was. environment.Environment: '"+environment.GetId()+"', failed for '"+name+"' ("+configType+")")
			continue
		}

		description := fmt.Sprintf("%s %s on environment %s", configType, name, environment.GetId())

		// Wait at most 60 * 2 seconds = 2 Minutes:
		err = rest.Wait(description, 60, func() bool {
			_, existingId, _ = rest.GetObjectIdIfAlreadyExists(configType, api.GetUrl(environment), name, token)
			return (available && len(existingId) > 0) || (!available && len(existingId) == 0)
		})
		assert.NilError(t, err)

		if available {
			assert.Check(t, len(existingId) > 0, "Object should be available, but wasn't. environment.Environment: '"+environment.GetId()+"', failed for '"+name+"' ("+configType+")")
		} else {
			assert.Equal(t, existingId, "", "Object should NOT be available, but was. environment.Environment: '"+environment.GetId()+"', failed for '"+name+"' ("+configType+")")
		}
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
func cleanupIntegrationTest(t *testing.T, suffix string, transformers []func(string) string) {
	var integrationTestReader, err = util.NewInMemoryFileReader(folder, transformers)
	assert.NilError(t, err)

	environments, errs := environment.LoadEnvironmentList("", environmentsFile, integrationTestReader)
	FailOnAnyError(errs, "loading of environments failed")

	apis := api.NewApis()
	suffix = "_" + suffix

	for _, environment := range environments {
		token, err := environment.GetToken()
		assert.NilError(t, err)
		for _, api := range apis {

			_, values, err := rest.GetExistingValuesFromEndpoint(api.GetId(), api.GetUrl(environment), token)
			assert.NilError(t, err)

			for _, value := range values {
				// For the calculated-metrics-log API, the suffix is part of the ID, not name
				if strings.HasSuffix(value.Name, suffix) || strings.HasSuffix(value.Id, suffix) {
					util.Log.Info("Deleting %s (%s)", value.Name, api.GetId())
					rest.Delete(api.GetUrl(environment), token, value.Id)
				}
			}
		}
	}
}

func RunIntegrationWithCleanup(t *testing.T, suffixTest string, testFunc func(transformers []func(string) string)) {
	suffix := getTimestamp() + suffixTest
	transformers := []func(string) string{getTransformerFunc(suffix)}

	testFunc(transformers)
	cleanupIntegrationTest(t, suffix, transformers)
}
