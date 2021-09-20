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
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

const multiProjectFolder = "test-resources/integration-multi-project/"
const multiProjectFolderWithoutSlash = "test-resources/integration-multi-project"
const multiProjectEnvironmentsFile = multiProjectFolder + "environments.yaml"

// Tests all environments with all projects
func TestIntegrationMultiProject(t *testing.T) {

	RunIntegrationWithCleanup(t, multiProjectFolder, multiProjectEnvironmentsFile, "MultiProject", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", multiProjectEnvironmentsFile, fs)
		assert.Check(t, len(errs) == 0, "didn't expect errors loading test environments")

		projects, err := project.LoadProjectsToDeploy(fs, "", api.NewApis(), multiProjectFolder)
		assert.NilError(t, err)

		statusCode := RunImpl([]string{
			"monaco",
			"--environments", multiProjectEnvironmentsFile,
			multiProjectFolder,
		}, fs)

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.Equal(t, statusCode, 0)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProject(t *testing.T) {

	statusCode := RunImpl([]string{
		"monaco",
		"--environments", multiProjectEnvironmentsFile,
		"--dry-run",
		multiProjectFolder,
	}, util.CreateTestFileSystem())

	assert.Equal(t, statusCode, 0)
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProjectWithoutEndingSlashInPath(t *testing.T) {

	statusCode := RunImpl([]string{
		"monaco",
		"--environments", multiProjectEnvironmentsFile,
		"--dry-run",
		multiProjectFolderWithoutSlash,
	}, util.CreateTestFileSystem())

	assert.Equal(t, statusCode, 0)
}

// tests a single project with dependencies
func TestIntegrationMultiProjectSingleProject(t *testing.T) {

	RunIntegrationWithCleanup(t, multiProjectFolder, multiProjectEnvironmentsFile, "MultiProjectSingleProject", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", multiProjectEnvironmentsFile, fs)
		FailOnAnyError(errs, "loading of environments failed")

		projects, err := project.LoadProjectsToDeploy(fs, "star-trek", api.NewApis(), multiProjectFolder)
		assert.NilError(t, err)

		assert.Equal(t, projects[0].GetId(), "test-resources/integration-multi-project/cinema-infrastructure", "Check if dependent project `cinema-infrastructure` is loaded and will be deployed first.")

		statusCode := RunImpl([]string{
			"monaco",
			"--environments", multiProjectEnvironmentsFile,
			"-p", "star-trek",
			multiProjectFolder,
		}, fs)

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.Equal(t, statusCode, 0)
	})
}
