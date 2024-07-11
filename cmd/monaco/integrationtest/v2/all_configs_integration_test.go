//go:build integration

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

package v2

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/spf13/afero"
)

// tests all configs for a single environment
func TestIntegrationAllConfigsClassic(t *testing.T) {
	specificEnvironment := "classic_env"

	runAllConfigsTest(t, specificEnvironment)
}

func TestIntegrationAllConfigsPlatform(t *testing.T) {
	specificEnvironment := "platform_env"

	runAllConfigsTest(t, specificEnvironment)
}

func runAllConfigsTest(t *testing.T, specificEnvironment string) {
	configFolder := "test-resources/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	envVars := map[string]string{
		featureflags.UserActionSessionPropertiesMobile().EnvName(): "true",
		featureflags.KeyUserActionsMobile().EnvName():              "true",
		featureflags.KeyUserActionsWeb().EnvName():                 "true",
		featureflags.OpenPipeline().EnvName():                      "true"}

	RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifest, specificEnvironment, "AllConfigs", envVars, func(fs afero.Fs, _ TestContext) {

		// This causes a POST for all configs:

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest, "--environment", specificEnvironment})
		err := cmd.Execute()

		assert.NoError(t, err)

		// This causes a PUT for all configs:

		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest, "--environment", specificEnvironment})
		err = cmd.Execute()
		assert.NoError(t, err)

	})
}

// Tests a dry run (validation)
func TestIntegrationValidationAllConfigs(t *testing.T) {

	t.Setenv("UNIQUE_TEST_SUFFIX", "can-be-nonunique-for-validation")
	t.Setenv(featureflags.UserActionSessionPropertiesMobile().EnvName(), "true")
	t.Setenv(featureflags.KeyUserActionsMobile().EnvName(), "true")
	t.Setenv(featureflags.KeyUserActionsWeb().EnvName(), "true")
	t.Setenv(featureflags.OpenPipeline().EnvName(), "true")

	configFolder := "test-resources/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	cmd := runner.BuildCli(testutils.CreateTestFileSystem())
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.NoError(t, err)
}
