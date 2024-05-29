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
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var folder = AbsOrPanicFromSlash("test-resources/integration-multi-environment/")
var environmentsFile = filepath.Join(folder, "environments.yaml")

// Tests all environments with all projects
func TestIntegrationMultiEnvironment(t *testing.T) {
	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironment", func(fs afero.Fs, manifest string) {
		err := monaco.RunWithFSf(fs, "monaco deploy --verbose %s", manifest)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{}, "", true)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiEnvironment(t *testing.T) {
	RunLegacyIntegrationWithoutCleanup(t, folder, environmentsFile, "validationMultiEnv", func(fs afero.Fs, manifest string) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --verbose --dry-run", manifest)
		assert.NoError(t, err)
	})
}

// tests a single project
func TestIntegrationMultiEnvironmentSingleProject(t *testing.T) {
	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleProject", func(fs afero.Fs, manifestFile string) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --verbose --project=cinema-infrastructure", manifestFile)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{"cinema-infrastructure"}, "", true)
	})
}

// Tests a single project with dependency
func TestIntegrationMultiEnvironmentSingleProjectWithDependency(t *testing.T) {
	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleProjectWithDependency", func(fs afero.Fs, manifestFile string) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --project=star-trek --verbose", manifestFile)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{"star-trek"}, "", true)
	})
}

// tests a single environment
func TestIntegrationMultiEnvironmentSingleEnvironment(t *testing.T) {
	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleEnvironment", func(fs afero.Fs, manifestFile string) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --environment=environment2 --verbose", manifestFile)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{"star-trek"}, "environment2", true)
	})
}
