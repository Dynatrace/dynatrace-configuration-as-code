//go:build integration
// +build integration

/**
 * @license
 * Copyright 2021 Dynatrace LLC
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

package v2

import (
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

var multiProjectFolder = "test-resources/integration-multi-project/"
var multiProjectManifest = multiProjectFolder + "manifest.yaml"
var oneProjectManifest = multiProjectFolder + "manifest-one-project.yaml"
var oneProjectManifestNotDeployed = multiProjectFolder + "manifest-one-project-not-deployed.yaml"
var multiProjectSpecificEnvironment = ""

// Tests all environments with all projects
func TestIntegrationMultiProject(t *testing.T) {

	RunIntegrationWithCleanup(t, multiProjectFolder, multiProjectManifest, multiProjectSpecificEnvironment, "MultiProject", func(fs afero.Fs) {

		// This causes a POST for all configs:
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", multiProjectManifest})
		err := cmd.Execute()

		assert.NilError(t, err)

		AssertAllConfigsAvailability(t, fs, multiProjectManifest, multiProjectSpecificEnvironment, true)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProject(t *testing.T) {

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", multiProjectManifest})
	err := cmd.Execute()

	assert.NilError(t, err)
}

// tests a single project with dependencies
func TestIntegrationMultiProjectSingleProject(t *testing.T) {

	RunIntegrationWithCleanup(t, multiProjectFolder, multiProjectManifest, multiProjectSpecificEnvironment, "MultiProjectOnProject", func(fs afero.Fs) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy",
			"--verbose",
			"-p", "star-trek",
			multiProjectManifest})
		err := cmd.Execute()

		assert.NilError(t, err)

		AssertAllConfigsAvailability(t, fs, oneProjectManifest, multiProjectSpecificEnvironment, true)
		AssertAllConfigsAvailability(t, fs, oneProjectManifestNotDeployed, multiProjectSpecificEnvironment, false)
	})
}
