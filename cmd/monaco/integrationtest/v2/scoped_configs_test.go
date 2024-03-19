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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeployScopedConfigurations(t *testing.T) {

	dashboardSharedEnvName := "DASHBOARD_SHARED"
	configFolder := "test-resources/scoped-configs/"
	environment := "classic_env"
	manifest := configFolder + "manifest.yaml"
	envVars := map[string]string{
		featureflags.MRumProperties().EnvName():         "true",
		featureflags.DashboardShareSettings().EnvName(): "true",
		dashboardSharedEnvName:                          "false",
	}

	RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifest, environment, "ScopedConfigs", envVars, func(fs afero.Fs, _ TestContext) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest, "--environment", environment})
		err := cmd.Execute()

		assert.NoError(t, err)
		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{}, environment, true)
	})
}
