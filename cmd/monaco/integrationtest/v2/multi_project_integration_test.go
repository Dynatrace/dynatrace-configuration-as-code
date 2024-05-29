//go:build integration

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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/spf13/afero"
)

var multiProjectFolder = "test-resources/integration-multi-project/"
var multiProjectManifest = multiProjectFolder + "manifest.yaml"
var multiProjectSpecificEnvironment = ""

// Tests all environments with all projects
func TestIntegrationMultiProject(t *testing.T) {

	RunIntegrationWithCleanup(t, multiProjectFolder, multiProjectManifest, multiProjectSpecificEnvironment, "MultiProject", func(fs afero.Fs, _ TestContext) {

		// This causes a POST for all configs:
		cmd := runner.BuildCmd(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", multiProjectManifest})
		err := cmd.Execute()

		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{}, multiProjectSpecificEnvironment, true)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProject(t *testing.T) {

	cmd := runner.BuildCmd(testutils.CreateTestFileSystem())
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", multiProjectManifest})
	err := cmd.Execute()

	assert.NoError(t, err)
}

// tests a single project with dependencies
func TestIntegrationMultiProjectSingleProject(t *testing.T) {

	RunIntegrationWithCleanup(t, multiProjectFolder, multiProjectManifest, multiProjectSpecificEnvironment, "MultiProjectOnProject", func(fs afero.Fs, _ TestContext) {

		cmd := runner.BuildCmd(fs)
		cmd.SetArgs([]string{"deploy",
			"--verbose",
			"-p", "star-trek",
			multiProjectManifest})
		err := cmd.Execute()

		assert.NoError(t, err)

		// Validate Star Trek sub-projects were deployed
		integrationtest.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{"star-trek.star-wars", "star-trek.star-gate"}, multiProjectSpecificEnvironment, true)

		// Validate movies project was not deployed
		integrationtest.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{"movies.science fiction.the-hitchhikers-guide-to-the-galaxy"}, multiProjectSpecificEnvironment, false)
	})
}

func TestIntegrationMultiProject_ReturnsErrorOnInvalidProjectDefinitions(t *testing.T) {

	invalidManifest := multiProjectFolder + "invalid-manifest-with-duplicate-projects.yaml"

	cmd := runner.BuildCmd(testutils.CreateTestFileSystem())
	cmd.SetArgs([]string{"deploy", "--verbose", invalidManifest})
	err := cmd.Execute()

	assert.Error(t, err)
}
