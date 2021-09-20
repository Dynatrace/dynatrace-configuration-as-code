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

	"gotest.tools/assert"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
)

const folder = "test-resources/integration-multi-environment/"
const environmentsFile = folder + "environments.yaml"

// Tests all environments with all projects
func TestIntegrationMultiEnvironment(t *testing.T) {

	RunIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironment", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", environmentsFile, fs)
		assert.Check(t, len(errs) == 0, "didn't expect errors loading test environments")

		projects, err := project.LoadProjectsToDeploy(fs, "", api.NewApis(), folder)
		assert.NilError(t, err)

		statusCode := RunImpl([]string{
			"monaco",
			"--environments", environmentsFile,
			folder,
		}, fs)

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.Equal(t, statusCode, 0)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiEnvironment(t *testing.T) {

	statusCode := RunImpl([]string{
		"monaco",
		"--environments", environmentsFile,
		"--dry-run",
		folder,
	}, util.CreateTestFileSystem())

	assert.Equal(t, statusCode, 0)
}

// tests a single project
func TestIntegrationMultiEnvironmentSingleProject(t *testing.T) {

	RunIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleProject", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", environmentsFile, fs)
		FailOnAnyError(errs, "loading of environments failed")

		projects, err := project.LoadProjectsToDeploy(fs, "cinema-infrastructure", api.NewApis(), folder)
		assert.NilError(t, err)

		statusCode := RunImpl([]string{
			"monaco",
			"--environments", environmentsFile,
			"-p", "cinema-infrastructure",
			folder,
		}, fs)

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.Equal(t, statusCode, 0)
	})
}

// Tests a single project with dependency
func TestIntegrationMultiEnvironmentSingleProjectWithDependency(t *testing.T) {

	RunIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleProjectWithDependency", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", environmentsFile, fs)
		FailOnAnyError(errs, "loading of environments failed")

		projects, err := project.LoadProjectsToDeploy(fs, "star-trek", api.NewApis(), folder)
		assert.NilError(t, err)

		assert.Check(t, len(projects) == 2, "Projects should be star-trek and the dependency cinema-infrastructure")

		statusCode := RunImpl([]string{
			"monaco",
			"--environments", environmentsFile,
			"-p", "star-trek",
			folder,
		}, fs)

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.Equal(t, statusCode, 0)
	})
}

// tests a single environment
func TestIntegrationMultiEnvironmentSingleEnvironment(t *testing.T) {

	RunIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleEnvironment", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", environmentsFile, fs)
		FailOnAnyError(errs, "loading of environments failed")

		projects, err := project.LoadProjectsToDeploy(fs, "star-trek", api.NewApis(), folder)
		assert.NilError(t, err)

		// remove environment odt69781, just keep dav48679
		delete(environments, "odt69781")

		statusCode := RunImpl([]string{
			"monaco",
			"--environments", environmentsFile,
			folder,
		}, fs)

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.Equal(t, statusCode, 0)
	})
}
