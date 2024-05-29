//go:build integration_v1

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
	"strings"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var skipDeploymentFolder = AbsOrPanicFromSlash("test-resources/skip-deployment-project/")
var skipDeploymentEnvironmentsFile = AbsOrPanicFromSlash("test-resources/test-environments.yaml")

func TestValidationSkipDeployment(t *testing.T) {
	t.Setenv("TEST_TOKEN", "mock test token")

	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, t.Name(), func(fs afero.Fs, manifest string) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --project=projectA --dry-run --verbose", manifest)
		assert.NoError(t, err)
	})

}

func TestValidationSkipDeploymentWithBrokenDependency_GraphBasedDoesNotReturnErrorAsDependenciesAreIgnored(t *testing.T) {
	t.Setenv("TEST_TOKEN", "mock test token")

	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, "SkipDeployment", func(fs afero.Fs, manifest string) {

		logOutput := strings.Builder{}
		cmd := runner.BuildCmdWithLogSpy(fs, &logOutput)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
			"--project", "projectB",
		})
		err := cmd.Execute()
		assert.NoError(t, err, "children of skipped configs should not result in an error")

		runLog := logOutput.String()
		assert.Contains(t, runLog, "Skipping deployment of projectB:management-zone:mg-zone-b, as it depends on projectB:auto-tag:application-tagging-b which was skipped")
	})
}

func TestValidationSkipDeploymentWithOverridingDependency(t *testing.T) {
	t.Setenv("TEST_TOKEN", "mock test token")

	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, t.Name(), func(fs afero.Fs, manifest string) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --project=projectC --dry-run --verbose", manifest)
		assert.NoError(t, err)
	})
}

func TestValidationSkipDeploymentWithOverridingFlagValue(t *testing.T) {
	t.Setenv("TEST_TOKEN", "mock test token")

	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, t.Name(), func(fs afero.Fs, manifest string) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --project=projectE --dry-run --verbose", manifest)
		assert.NoError(t, err)
	})
}

func TestValidationSkipDeploymentInterProjectWithMissingDependency_GraphBasedDoesNotReturnErrorAsDependenciesAreIgnored(t *testing.T) {
	t.Setenv("TEST_TOKEN", "mock test token")

	RunLegacyIntegrationWithoutCleanup(t, skipDeploymentFolder, skipDeploymentEnvironmentsFile, t.Name(), func(fs afero.Fs, manifest string) {
		logOutput := strings.Builder{}
		cmd := runner.BuildCmdWithLogSpy(fs, &logOutput)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
			"--project", "projectD",
		})
		err := cmd.Execute()

		assert.NoError(t, err)

		runLog := logOutput.String()
		assert.Contains(t, runLog, "Skipping deployment of projectA:management-zone:mg-zone, as it depends on projectA:auto-tag:application-tagging which was skipped")
	})
}
