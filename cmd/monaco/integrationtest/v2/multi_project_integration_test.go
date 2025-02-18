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
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
)

var multiProjectFolder = "test-resources/integration-multi-project/"
var multiProjectManifest = multiProjectFolder + "manifest.yaml"
var multiProjectSpecificEnvironment = ""

// Tests all environments with all projects
func TestIntegrationMultiProject(t *testing.T) {

	RunIntegrationWithCleanup(t, multiProjectFolder, multiProjectManifest, multiProjectSpecificEnvironment, "MultiProject", func(fs afero.Fs, _ TestContext) {
		// This causes a POST for all configs:
		err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", multiProjectManifest))
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{}, multiProjectSpecificEnvironment, true)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProject(t *testing.T) {
	err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --dry-run --verbose", multiProjectManifest))
	assert.NoError(t, err)
}

// tests a single project with dependencies
func TestIntegrationMultiProjectSingleProject(t *testing.T) {

	RunIntegrationWithCleanup(t, multiProjectFolder, multiProjectManifest, multiProjectSpecificEnvironment, "MultiProjectOnProject", func(fs afero.Fs, _ TestContext) {
		err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=star-trek --verbose", multiProjectManifest))
		assert.NoError(t, err)

		// Validate Star Trek sub-projects were deployed
		integrationtest.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{"star-trek.star-wars", "star-trek.star-gate"}, multiProjectSpecificEnvironment, true)

		// Validate movies project was not deployed
		integrationtest.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{"movies.science fiction.the-hitchhikers-guide-to-the-galaxy"}, multiProjectSpecificEnvironment, false)
	})
}

func TestIntegrationMultiProject_ReturnsErrorOnInvalidProjectDefinitions(t *testing.T) {
	invalidManifest := multiProjectFolder + "invalid-manifest-with-duplicate-projects.yaml"
	err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --verbose", invalidManifest))
	assert.Error(t, err)
}
