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

package scope

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestIntegrationScopeParameters(t *testing.T) {
	configFolder := "testdata/integration-scope-parameters/"
	manifest := configFolder + "/manifest.yaml"

	envVars := map[string]string{
		"SCOPE_TEST_ENV_VAR": "environment",
	}

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifest),
			runner.WithSuffix("ScopeParameters"),
			runner.WithEnvVars(envVars),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			// This causes Creation of all Settings
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
			assert.NoError(t, err)

			// This causes an Update of all Settings
			err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
			assert.NoError(t, err)
		})
}

// Tests a dry run (validation)
func TestIntegrationScopeParameterValidation(t *testing.T) {

	configFolder := "testdata/integration-scope-parameters/"
	manifest := configFolder + "manifest.yaml"

	envVar := "SCOPE_TEST_ENV_VAR"
	t.Setenv(envVar, "environment")

	err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --dry-run --verbose", manifest))
	assert.NoError(t, err)
}
