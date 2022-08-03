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
	"testing"

	"gotest.tools/assert"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/v2/runner"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

var skipDeploymentFolder = AbsOrPanicFromSlash("test-resources/skip-deployment-project/")
var skipDeploymentEnvironmentsFile = AbsOrPanicFromSlash("test-resources/test-environments.yaml")

func TestValidationSkipDeployment(t *testing.T) {
	t.Setenv("CONFIG_V1", "1")

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectA",
		skipDeploymentFolder,
	})
	err := cmd.Execute()
	assert.NilError(t, err)

}

func TestValidationSkipDeploymentWithBrokenDependency(t *testing.T) {
	t.Setenv("CONFIG_V1", "1")

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectB",
		skipDeploymentFolder,
	})
	err := cmd.Execute()
	assert.Error(t, err, "dry run found 3 errors. check logs")
}

func TestValidationSkipDeploymentWithOverridingDependency(t *testing.T) {
	t.Setenv("CONFIG_V1", "1")

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectC",
		skipDeploymentFolder,
	})
	err := cmd.Execute()

	assert.NilError(t, err)
}

func TestValidationSkipDeploymentWithOverridingFlagValue(t *testing.T) {
	t.Setenv("CONFIG_V1", "1")

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectE",
		skipDeploymentFolder,
	})
	err := cmd.Execute()

	assert.NilError(t, err)
}

func TestValidationSkipDeploymentInterProjectWithMissingDependency(t *testing.T) {
	t.Setenv("CONFIG_V1", "1")

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectD",
		skipDeploymentFolder,
	})
	err := cmd.Execute()

	assert.Error(t, err, "dry run found 1 errors. check logs")
}
