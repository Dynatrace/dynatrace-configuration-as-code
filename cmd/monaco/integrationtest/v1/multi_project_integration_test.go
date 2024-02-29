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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/spf13/afero"
)

var multiProjectFolder = AbsOrPanicFromSlash("test-resources/integration-multi-project/")
var multiProjectFolderWithoutSlash = AbsOrPanicFromSlash("test-resources/integration-multi-project")
var multiProjectEnvironmentsFile = filepath.Join(multiProjectFolder, "environments.yaml")

// Tests all environments with all projects
func TestIntegrationMultiProject(t *testing.T) {
	RunLegacyIntegrationWithCleanup(t, multiProjectFolder, multiProjectEnvironmentsFile, "MultiProject", func(fs afero.Fs, manifest string) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
		})
		err := cmd.Execute()
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{}, "", true)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProject(t *testing.T) {
	RunLegacyIntegrationWithoutCleanup(t, multiProjectFolder, multiProjectEnvironmentsFile, "validMultiProj", func(fs afero.Fs, manifest string) {
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
		})
		err := cmd.Execute()

		assert.NoError(t, err)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProjectWithoutEndingSlashInPath(t *testing.T) {
	RunLegacyIntegrationWithoutCleanup(t, multiProjectFolderWithoutSlash, multiProjectEnvironmentsFile, "validMultiProj", func(fs afero.Fs, manifest string) {
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
		})
		err := cmd.Execute()

		assert.NoError(t, err)
	})

}

// tests a single project with dependencies
func TestIntegrationMultiProjectSingleProject(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, multiProjectFolder, multiProjectEnvironmentsFile, "MultiProjectSingleProject", func(fs afero.Fs, manifestFile string) {

		cmd := runner.BuildCli(fs)

		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifestFile,
			"-p", "star-trek",
		})
		err := cmd.Execute()
		assert.NoError(t, err)

		t.Log("Asserting available configs")

		integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{"star-trek.star-wars"}, "", true)
	})
}
