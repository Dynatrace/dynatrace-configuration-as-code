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
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

var multiProjectFolder = AbsOrPanicFromSlash("test-resources/integration-multi-project/")
var multiProjectFolderWithoutSlash = AbsOrPanicFromSlash("test-resources/integration-multi-project")
var multiProjectEnvironmentsFile = filepath.Join(multiProjectFolder, "environments.yaml")

// Tests all environments with all projects
func TestIntegrationMultiProject(t *testing.T) {
	RunLegacyIntegrationWithCleanup(t, multiProjectFolder, multiProjectEnvironmentsFile, "MultiProject", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", multiProjectEnvironmentsFile, fs)
		assert.Check(t, len(errs) == 0, "didn't expect errors loading test environments")

		projects, err := project.LoadProjectsToDeploy(fs, "", api.NewV1Apis(), multiProjectFolder)
		assert.NilError(t, err)

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			"--environments", multiProjectEnvironmentsFile,
			multiProjectFolder,
		})
		err = cmd.Execute()

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.NilError(t, err)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProject(t *testing.T) {
	t.Setenv("CONFIG_V1", "1")

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--environments", multiProjectEnvironmentsFile,
		"--dry-run",
		multiProjectFolder,
	})
	err := cmd.Execute()

	assert.NilError(t, err)
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProjectWithoutEndingSlashInPath(t *testing.T) {
	t.Setenv("CONFIG_V1", "1")

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--environments", multiProjectEnvironmentsFile,
		"--dry-run",
		multiProjectFolderWithoutSlash,
	})
	err := cmd.Execute()

	assert.NilError(t, err)
}

// tests a single project with dependencies
func TestIntegrationMultiProjectSingleProject(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, multiProjectFolder, multiProjectEnvironmentsFile, "MultiProjectSingleProject", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", multiProjectEnvironmentsFile, fs)
		FailOnAnyError(errs, "loading of environments failed")

		projects, err := project.LoadProjectsToDeploy(fs, "star-trek", api.NewV1Apis(), multiProjectFolder)
		assert.NilError(t, err)

		cmd := runner.BuildCli(fs)

		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			"--environments", multiProjectEnvironmentsFile,
			"-p", "star-trek",
			multiProjectFolder,
		})
		err = cmd.Execute()

		AssertAllConfigsAvailability(projects, t, environments, true)

		assert.NilError(t, err)
	})
}
