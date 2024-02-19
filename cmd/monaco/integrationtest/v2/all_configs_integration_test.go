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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/internal/test"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAllConfigs(t *testing.T) {
	tests := []struct {
		name, environment string
	}{
		{
			name:        "tests all known configs against classic url",
			environment: "classic_env",
		},
		{
			name:        "tests all known configs against platform url",
			environment: "platform_env",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configFolder := "test-resources/integration-all-configs/"
			manifest := configFolder + "manifest.yaml"

			envVars := map[string]string{featureflags.Experimental().EnvName(): "true"}

			RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifest, tc.environment, "AllConfigs", envVars, func(fs afero.Fs, _ TestContext) {
				{
					// This causes a POST for all configs:
					_, err := test.Monacof("deploy %s --environment %s", manifest, tc.environment).WithFs(fs).Run()
					require.NoError(t, err)
				}
				{
					// This causes a PUT for all configs:
					_, err := test.Monacof("deploy %s --environment %s", manifest, tc.environment).WithFs(fs).Run()
					require.NoError(t, err)
				}
			})
		})
	}
}

// Tests a dry run (validation)
func TestIntegrationValidationAllConfigs(t *testing.T) {

	t.Setenv("UNIQUE_TEST_SUFFIX", "can-be-nonunique-for-validation")
	t.Setenv(featureflags.Experimental().EnvName(), "true")

	configFolder := "test-resources/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	_, err := test.Monacof("monaco deploy %s --dry-run", manifest).Run()
	assert.NoError(t, err)
}
