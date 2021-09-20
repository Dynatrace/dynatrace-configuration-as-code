//go:build unit
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

package deploy

import (
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"

	"gotest.tools/assert"
)

func TestFailsOnMissingFileName(t *testing.T) {
	_, err := environment.LoadEnvironmentList("dev", "", util.CreateTestFileSystem())
	assert.Assert(t, len(err) == 1, "Expected error return")
}

func TestLoadsEnvironmentListCorrectly(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("", "../../cmd/monaco/test-resources/test-environments.yaml", util.CreateTestFileSystem())
	assert.Assert(t, len(err) == 0, "Expected no error")
	assert.Assert(t, len(environments) == 3, "Expected to load test environments 1-3!")
}

func TestLoadSpecificEnvironmentCorrectly(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("test2", "../../cmd/monaco/test-resources/test-environments.yaml", util.CreateTestFileSystem())
	assert.Assert(t, len(err) == 0, "Expected no error")
	assert.Assert(t, len(environments) == 1, "Expected to load test environment 2 only!")
	assert.Assert(t, environments["test2"] != nil, "test2 environment not found in returned list!")
}

func TestMissingSpecificEnvironmentResultsInError(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("test42", "../../cmd/monaco/test-resources/test-environments.yaml", util.CreateTestFileSystem())
	assert.Assert(t, len(err) == 1, "Expected error from referencing unknown environment")
	assert.Assert(t, len(environments) == 0, "Expected to get empty environment map even on error")
}

func testGetExecuteApis() map[string]api.Api {
	apis := make(map[string]api.Api)
	apis["calculated-metrics-log"] = api.NewStandardApi("calculated-metrics-log", "/api")
	apis["alerting-profile"] = api.NewStandardApi("alerting-profile", "/api")
	return apis
}
func TestExecuteFailOnDuplicateNamesWithinSameConfig(t *testing.T) {
	apis := testGetExecuteApis()
	fs := util.CreateTestFileSystem()
	environment := environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")
	//always use files relative to the local folder or absolute paths, don't use ../ to navigate to upper levels to allow to run the test locally
	projects, err := project.LoadProjectsToDeploy(fs, "project1", apis, "./test-resources/duplicate-name-test")
	assert.NilError(t, err)

	errors := execute(environment, projects, true, "", false)
	assert.Equal(t, errors != nil, true)
	assert.ErrorContains(t, errors[0], "duplicate UID 'calculated-metrics-log/metric' found in")
}

func TestExecutePassOnDifferentApis(t *testing.T) {
	environment := environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")

	apis := testGetExecuteApis()

	path := util.ReplacePathSeparators("./test-resources/duplicate-name-test")
	fs := util.CreateTestFileSystem()
	projects, err := project.LoadProjectsToDeploy(fs, "project2", apis, path)
	assert.NilError(t, err)

	errors := execute(environment, projects, true, "", false)
	for _, err := range errors {
		assert.NilError(t, err)
	}
}

func TestExecuteFailOnDuplicateNamesInDifferentProjects(t *testing.T) {
	environment := environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")

	apis := testGetExecuteApis()

	path := util.ReplacePathSeparators("./test-resources/duplicate-name-test")
	fs := util.CreateTestFileSystem()
	projects, err := project.LoadProjectsToDeploy(fs, "project1, project2", apis, path)
	assert.NilError(t, err)

	errors := execute(environment, projects, true, "", false)
	assert.ErrorContains(t, errors[0], "duplicate UID 'calculated-metrics-log/metric' found in")
}

func TestExecutePassOnDuplicateNamesInDifferentEnvironments(t *testing.T) {
	environmentDev := environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")
	environmentProd := environment.NewEnvironment("prod", "Prod", "", "https://url/to/prod/environment", "PROD")

	apis := testGetExecuteApis()

	path := util.ReplacePathSeparators("./test-resources/duplicate-name-test")
	fs := util.CreateTestFileSystem()
	projects, err := project.LoadProjectsToDeploy(fs, "project5", apis, path)
	assert.NilError(t, err)

	errors := execute(environmentDev, projects, true, "", false)
	for _, err := range errors {
		assert.NilError(t, err)
	}
	errors = execute(environmentProd, projects, true, "", false)
	for _, err := range errors {
		assert.NilError(t, err)
	}
}

// TODO (CDF-6511) Currently here UnmarshallYaml logs fatal, only ever returns nil errors!
// func TestInvalidEnvironmentFileResultsInError(t *testing.T) {
// 	_, err := environment.LoadEnvironmentList("", "test-resources/invalid-environmentsfile.yaml")
// 	assert.Assert(t, err != nil, "Expected error return")
// }

// TODO (CDF-6511) add tests when execute failures of single environments don't crash program anymore
