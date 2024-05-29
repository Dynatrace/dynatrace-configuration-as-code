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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestIntegrationScopeParameters(t *testing.T) {
	configFolder := "test-resources/integration-scope-parameters/"
	manifest := configFolder + "/manifest.yaml"
	specificEnvironment := ""

	envVars := map[string]string{
		"SCOPE_TEST_ENV_VAR": "environment",
	}

	RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifest, specificEnvironment, "ScopeParameters", envVars, func(fs afero.Fs, _ TestContext) {

		// This causes Creation of all Settings
		cmd := runner.BuildCmd(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest})
		err := cmd.Execute()

		assert.NoError(t, err)

		// This causes an Update of all Settings
		cmd = runner.BuildCmd(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest})
		err = cmd.Execute()
		assert.NoError(t, err)
	})
}

// Tests a dry run (validation)
func TestIntegrationScopeParameterValidation(t *testing.T) {

	configFolder := "test-resources/integration-scope-parameters/"
	manifest := configFolder + "manifest.yaml"

	envVar := "SCOPE_TEST_ENV_VAR"
	t.Setenv(envVar, "environment")

	cmd := runner.BuildCmd(testutils.CreateTestFileSystem())
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.NoError(t, err)
}
