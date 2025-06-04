//go:build integration

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package multiproject

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	assert2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/assert"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

var multiProjectFolder = "testdata/integration-multi-project/"
var multiProjectManifest = multiProjectFolder + "manifest.yaml"
var multiProjectSpecificEnvironment = ""

// Tests all environments with all projects
func TestIntegrationMultiProject(t *testing.T) {

	runner.Run(t, multiProjectFolder,
		runner.Options{
			runner.WithManifestPath(multiProjectManifest),
			runner.WithSuffix("MultiProject"),
			runner.WithEnvironment(multiProjectSpecificEnvironment),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			// This causes a POST for all configs:
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", multiProjectManifest))
			assert.NoError(t, err)

			assert2.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{}, multiProjectSpecificEnvironment, true)
		})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiProject(t *testing.T) {
	err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --dry-run --verbose", multiProjectManifest))
	assert.NoError(t, err)
}

// tests a single project with dependencies
func TestIntegrationMultiProjectSingleProject(t *testing.T) {

	runner.Run(t, multiProjectFolder,
		runner.Options{
			runner.WithManifestPath(multiProjectManifest),
			runner.WithSuffix("MultiProjectOnProject"),
			runner.WithEnvironment(multiProjectSpecificEnvironment),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=star-trek --verbose", multiProjectManifest))
			assert.NoError(t, err)

			// Validate Star Trek sub-projects were deployed
			assert2.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{"star-trek.star-wars", "star-trek.star-gate"}, multiProjectSpecificEnvironment, true)

			// Validate movies project was not deployed
			assert2.AssertAllConfigsAvailability(t, fs, multiProjectManifest, []string{"movies.science fiction.the-hitchhikers-guide-to-the-galaxy"}, multiProjectSpecificEnvironment, false)
		})
}

func TestIntegrationMultiProject_ReturnsErrorOnInvalidProjectDefinitions(t *testing.T) {
	invalidManifest := multiProjectFolder + "invalid-manifest-with-duplicate-projects.yaml"
	err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --verbose", invalidManifest))
	assert.Error(t, err)
}
