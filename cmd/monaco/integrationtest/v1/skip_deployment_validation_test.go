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
	"github.com/spf13/afero"
	"testing"

	"gotest.tools/assert"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
)

var skipDeploymentFolder = AbsOrPanicFromSlash("test-resources/skip-deployment-project/")
var skipDeploymentEnvironmentsFile = AbsOrPanicFromSlash("test-resources/test-environments.yaml")

func TestValidationSkipDeployment(t *testing.T) {

	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, t.Name(), func(fs afero.Fs, manifest string) {
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			"--dry-run",
			manifest,
			"--project", "projectA",
		})
		err := cmd.Execute()
		assert.NilError(t, err)
	})

}

func TestValidationSkipDeploymentWithBrokenDependency(t *testing.T) {
	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, "SkipDeployment", func(fs afero.Fs, manifest string) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
			"--project", "projectB",
		})
		err := cmd.Execute()
		assert.Error(t, err, "errors during Validation")
	})
}

func TestValidationSkipDeploymentWithOverridingDependency(t *testing.T) {

	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, t.Name(), func(fs afero.Fs, manifest string) {
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
			"--project", "projectC",
		})
		err := cmd.Execute()

		assert.NilError(t, err)
	})
}

func TestValidationSkipDeploymentWithOverridingFlagValue(t *testing.T) {
	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, t.Name(), func(fs afero.Fs, manifest string) {
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
			"--project", "projectE",
		})
		err := cmd.Execute()

		assert.NilError(t, err)
	})
}

func TestValidationSkipDeploymentInterProjectWithMissingDependency(t *testing.T) {
	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, t.Name(), func(fs afero.Fs, manifest string) {
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
			"--project", "projectD",
		})
		err := cmd.Execute()

		assert.Error(t, err, "errors during Validation")
	})
}
