// +build unit

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
	"os"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"gotest.tools/assert"
)

func TestFailsOnMissingFileName(t *testing.T) {
	_, err := environment.LoadEnvironmentList("dev", "", util.NewFileReader())
	assert.Assert(t, len(err) == 1, "Expected error return")
}

func TestLoadsEnvironmentListCorrectly(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("", "test-resources/test-environments.yaml", util.NewFileReader())
	assert.Assert(t, len(err) == 0, "Expected no error")
	assert.Assert(t, len(environments) == 3, "Expected to load test environments 1-3!")
}

func TestLoadSpecificEnvironmentCorrectly(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("test2", "test-resources/test-environments.yaml", util.NewFileReader())
	assert.Assert(t, len(err) == 0, "Expected no error")
	assert.Assert(t, len(environments) == 1, "Expected to load test environment 2 only!")
	assert.Assert(t, environments["test2"] != nil, "test2 environment not found in returned list!")
}

func TestMissingSpecificEnvironmentResultsInError(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("test42", "test-resources/test-environments.yaml", util.NewFileReader())
	assert.Assert(t, len(err) == 1, "Expected error from referencing unknown environment")
	assert.Assert(t, len(environments) == 0, "Expected to get empty environment map even on error")
}

func TestReadPath(t *testing.T) {

	path := readPath([]string{"monaco", "--environments", "my-file.yaml", "test-resources"}, util.NewFileReader())
	assert.Equal(t, path, "test-resources"+string(os.PathSeparator))
}

func TestReadLongPath(t *testing.T) {

	location := util.ReplacePathSeparators("test-resources/transitional-dependency-test")
	path := readPath([]string{"monaco", "--environments", "my-file.yaml", location}, util.NewFileReader())
	assert.Equal(t, path, location+string(os.PathSeparator))
}

func TestReadPathNoDirectory(t *testing.T) {

	path := readPath([]string{"monaco", "--environments", "my-file.yaml", "main.go"}, util.NewFileReader())
	assert.Equal(t, path, "")
}

func TestReadPathNoPath(t *testing.T) {

	path := readPath([]string{"monaco", "--environments", "my-file.yaml"}, util.NewFileReader())
	assert.Equal(t, path, "")
}

func testGetExecuteApis() map[string]api.Api {
	apis := make(map[string]api.Api)
	apis["calculated-metrics-log"] = api.NewApi("calculated-metrics-log", "/api")
	apis["alerting-profile"] = api.NewApi("alerting-profile", "/api")
	return apis
}
func TestExecuteFailOnDuplicateNamesWithinSameConfig(t *testing.T) {
	environment := environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")

	apis := testGetExecuteApis()

	path := util.ReplacePathSeparators("test-resources/duplicate-name-test")
	projects, err := project.LoadProjectsToDeploy("project1", apis, path, util.NewFileReader())
	assert.NilError(t, err)

	err = execute(environment, projects, true, "")
	assert.ErrorContains(t, err, "duplicate UID 'calculated-metrics-log/metric' found in")
}

func TestExecutePassOnDifferentApis(t *testing.T) {
	environment := environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")

	apis := testGetExecuteApis()

	path := util.ReplacePathSeparators("test-resources/duplicate-name-test")
	projects, err := project.LoadProjectsToDeploy("project2", apis, path, util.NewFileReader())
	assert.NilError(t, err)

	err = execute(environment, projects, true, "")
	assert.NilError(t, err)
}

func TestExecuteFailOnDuplicateNamesInDifferentProjects(t *testing.T) {
	environment := environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")

	apis := testGetExecuteApis()

	path := util.ReplacePathSeparators("test-resources/duplicate-name-test")
	projects, err := project.LoadProjectsToDeploy("project1, project2", apis, path, util.NewFileReader())
	assert.NilError(t, err)

	err = execute(environment, projects, true, "")
	assert.ErrorContains(t, err, "duplicate UID 'calculated-metrics-log/metric' found in")
}

func TestExecutePassOnDuplicateNamesInDifferentEnvironments(t *testing.T) {
	environmentDev := environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")
	environmentProd := environment.NewEnvironment("prod", "Prod", "", "https://url/to/prod/environment", "PROD")

	apis := testGetExecuteApis()

	path := util.ReplacePathSeparators("test-resources/duplicate-name-test")
	projects, err := project.LoadProjectsToDeploy("project5", apis, path, util.NewFileReader())
	assert.NilError(t, err)

	err = execute(environmentDev, projects, true, "")
	assert.NilError(t, err)
	err = execute(environmentProd, projects, true, "")
	assert.NilError(t, err)
}

// TODO (CDF-6511) Currently here UnmarshallYaml logs fatal, only ever returns nil errors!
// func TestInvalidEnvironmentFileResultsInError(t *testing.T) {
// 	_, err := environment.LoadEnvironmentList("", "test-resources/invalid-environmentsfile.yaml")
// 	assert.Assert(t, err != nil, "Expected error return")
// }

// TODO (CDF-6511) add tests when execute failures of single environments don't crash program anymore
