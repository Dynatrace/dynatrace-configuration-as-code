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
	manifest2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"path/filepath"
	"testing"

	"gotest.tools/assert"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
	"github.com/spf13/afero"
)

var folder = AbsOrPanicFromSlash("test-resources/integration-multi-environment/")
var environmentsFile = filepath.Join(folder, "environments.yaml")

// Tests all environments with all projects
func TestIntegrationMultiEnvironment(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironment", func(fs afero.Fs, manifest string) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
		})
		err := cmd.Execute()
		assert.NilError(t, err)

		AssertAllConfigsAvailableInManifest(t, fs, manifest)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationMultiEnvironment(t *testing.T) {
	RunLegacyIntegrationWithoutCleanup(t, folder, environmentsFile, "validationMultiEnv", func(fs afero.Fs, manifest string) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--dry-run",
		})
		err := cmd.Execute()

		assert.NilError(t, err)
	})
}

// tests a single project
func TestIntegrationMultiEnvironmentSingleProject(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleProject", func(fs afero.Fs, manifestFile string) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifestFile,
			"-p", "cinema-infrastructure",
		})
		err := cmd.Execute()
		assert.NilError(t, err)

		t.Log("Asserting available configs")
		manifest := loadManifest(t, fs, manifestFile)
		projects := map[string]manifest2.ProjectDefinition{
			"cinema-infrastructure": manifest.Projects["cinema-infrastructure"],
		}

		AssertAllConfigsAvailable(t, fs, manifestFile, manifest, projects, manifest.Environments)

	})
}

// Tests a single project with dependency
func TestIntegrationMultiEnvironmentSingleProjectWithDependency(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleProjectWithDependency", func(fs afero.Fs, manifestFile string) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifestFile,
			"-p", "star-trek",
		})
		err := cmd.Execute()
		assert.NilError(t, err)

		manifest := loadManifest(t, fs, manifestFile)
		projects := map[string]manifest2.ProjectDefinition{
			"star-trek": manifest.Projects["star-trek"],
		}

		AssertAllConfigsAvailable(t, fs, manifestFile, manifest, projects, manifest.Environments)
	})
}

// tests a single environment
func TestIntegrationMultiEnvironmentSingleEnvironment(t *testing.T) {

	RunLegacyIntegrationWithCleanup(t, folder, environmentsFile, "MultiEnvironmentSingleEnvironment", func(fs afero.Fs, manifestFile string) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifestFile,
		})
		err := cmd.Execute()
		assert.NilError(t, err)

		manifest := loadManifest(t, fs, manifestFile)

		// remove environment odt69781, just keep dav48679
		delete(manifest.Environments, "odt69781")

		projects := map[string]manifest2.ProjectDefinition{
			"star-trek": manifest.Projects["star-trek"],
		}

		AssertAllConfigsAvailable(t, fs, manifestFile, manifest, projects, manifest.Environments)

	})
}
