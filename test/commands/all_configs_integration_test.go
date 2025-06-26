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

package commands

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

// tests all configs for a single environment
func TestIntegrationAllConfigsClassic(t *testing.T) {
	configFolder := "testdata/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	// flags are needed because the configs are still read and invalid types result in an error
	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")
	targetEnvironment := "classic_env"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifest),
			runner.WithSuffix("AllConfigs"),
			runner.WithEnvironment(targetEnvironment),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			// This causes a POST for all configs:
			runDeployCommand(t, fs, manifest, targetEnvironment)

			// This causes a PUT for all configs:
			runDeployCommand(t, fs, manifest, targetEnvironment)
		})
}

func TestIntegrationAllConfigsPlatformWithOAuth(t *testing.T) {
	configFolder := "testdata/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")

	targetEnvironment := "platform_oauth_env"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifest),
			runner.WithSuffix("AllConfigs"),
			runner.WithEnvironment(targetEnvironment),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			// This causes a POST for all configs:
			runDeployCommand(t, fs, manifest, targetEnvironment)

			// This causes a PUT for all configs:
			runDeployCommand(t, fs, manifest, targetEnvironment)
		})
}

func TestIntegrationAllConfigsPlatformWithToken(t *testing.T) {
	configFolder := "testdata/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")
	t.Setenv(featureflags.PlatformToken.EnvName(), "true")

	targetEnvironment := "platform_token_env"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifest),
			runner.WithSuffix("AllConfigs"),
			runner.WithEnvironment(targetEnvironment),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			// This causes a POST for all configs:
			runDeployCommand(t, fs, manifest, targetEnvironment)

			// This causes a PUT for all configs:
			runDeployCommand(t, fs, manifest, targetEnvironment)
		})
}

func runDeployCommand(t *testing.T, fs afero.Fs, manifest, specificEnvironment string) {
	t.Helper()

	// This causes a POST for all configs:
	err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --verbose", manifest, specificEnvironment))
	assert.NoError(t, err)
}

// Tests a dry run (validation)
func TestIntegrationValidationAllConfigs(t *testing.T) {
	t.Setenv("UNIQUE_TEST_SUFFIX", "can-be-nonunique-for-validation")
	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")
	t.Setenv(featureflags.PlatformToken.EnvName(), "true")
	fs := afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())

	err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --dry-run --verbose", "testdata/integration-all-configs/manifest.yaml"))
	assert.NoError(t, err)
}
