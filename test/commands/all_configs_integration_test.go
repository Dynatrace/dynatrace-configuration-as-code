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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
)

// tests all configs for a single environment
func TestIntegrationAllConfigsClassic(t *testing.T) {
	configFolder := "testdata/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	// flags are needed because the configs are still read and invalid types result in an error
	t.Setenv(featureflags.OpenPipeline.EnvName(), "true")
	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")
	t.Setenv(featureflags.AccessControlSettings.EnvName(), "true")
	targetEnvironment := "classic_env"

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithManifestPath(manifest),
			v2.WithSuffix("AllConfigs"),
			v2.WithEnvironment(targetEnvironment),
		},
		func(fs afero.Fs, _ v2.TestContext) {
			// This causes a POST for all configs:
			runDeployCommand(t, fs, manifest, targetEnvironment)

			// This causes a PUT for all configs:
			runDeployCommand(t, fs, manifest, targetEnvironment)
		})
}

func TestIntegrationAllConfigsPlatform(t *testing.T) {
	configFolder := "testdata/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	t.Setenv(featureflags.OpenPipeline.EnvName(), "true")
	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")
	t.Setenv(featureflags.AccessControlSettings.EnvName(), "true")

	targetEnvironment := "platform_env"

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithManifestPath(manifest),
			v2.WithSuffix("AllConfigs"),
			v2.WithEnvironment(targetEnvironment),
		},
		func(fs afero.Fs, _ v2.TestContext) {
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
	t.Setenv(featureflags.OpenPipeline.EnvName(), "true")
	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")
	t.Setenv(featureflags.AccessControlSettings.EnvName(), "true")

	fs := afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())

	err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --dry-run --verbose", "testdata/integration-all-configs/manifest.yaml"))
	assert.NoError(t, err)
}
