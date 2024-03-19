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
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
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
	}

	RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifest, environment, "ScopedConfigs", envVars, func(fs afero.Fs, testContext TestContext) {

		// deploy with sharing turned off and assert state
		t.Setenv(integrationtest.AddSuffix(dashboardSharedEnvName, testContext.suffix), "false")
		runDeployCommand(t, fs, manifest, environment)
		integrationtest.AssertAllConfigsAvailabilityAndHook(t, fs, manifest, []string{}, environment, true, getAssertSharedStateHook(false))

		// deploy with sharing turned on and assert state
		t.Setenv(integrationtest.AddSuffix(dashboardSharedEnvName, testContext.suffix), "true")
		runDeployCommand(t, fs, manifest, environment)
		integrationtest.AssertAllConfigsAvailabilityAndHook(t, fs, manifest, []string{}, environment, true, getAssertSharedStateHook(true))
	})
}

func runDeployCommand(t *testing.T, fs afero.Fs, manifestPath string, specificEnvironment string) {
	cmd := runner.BuildCli(fs)
	cmd.SetArgs([]string{"deploy", "--verbose", manifestPath, "--environment", specificEnvironment})
	err := cmd.Execute()
	require.NoError(t, err)
}

func getAssertSharedStateHook(expectShared bool) func(t *testing.T, clients *client.ClientSet, c config.Config, props parameter.Properties) {
	return func(t *testing.T, clients *client.ClientSet, c config.Config, props parameter.Properties) {
		classicApiType, ok := c.Type.(config.ClassicApiType)
		if !ok {
			return
		}

		theApi := api.NewAPIs()[classicApiType.Api]
		id, ok := props[config.IdParameter].(string)
		require.True(t, ok, "expected to get a ID for config")
		name, ok := props[config.NameParameter].(string)
		require.True(t, ok, "expected to get a name for config")

		if (theApi.ID == api.Dashboard) && (strings.HasPrefix(name, "Application monitoring")) {
			jsonBytes, err := clients.Classic().ReadConfigById(theApi, id)
			require.NoError(t, err)

			assertDashboardSharedState(t, jsonBytes, expectShared)
		} else if theApi.ID == api.DashboardShareSettings {
			scope, ok := props[config.ScopeParameter].(string)
			require.True(t, ok, "expected to get a scope for config")

			theApi = theApi.ApplyParentObjectID(scope)
			jsonBytes, err := clients.Classic().ReadConfigById(theApi, "")
			require.NoError(t, err)

			assertDashboardShareSettingsEnabledState(t, jsonBytes, expectShared)
		}
	}
}

func assertDashboardSharedState(t *testing.T, jsonBytes []byte, expectShared bool) {
	var resultMap map[string]interface{}
	err := json.Unmarshal(jsonBytes, &resultMap)
	require.NoError(t, err)

	dashboardMetadata, ok := resultMap["dashboardMetadata"].(map[string]interface{})
	require.True(t, ok, "expected to get dashboard metadata")

	shared, ok := dashboardMetadata["shared"].(bool)
	require.True(t, ok, "expected to get shared")

	assert.EqualValues(t, expectShared, shared, "expected dashboard shared = %t", expectShared)
}

func assertDashboardShareSettingsEnabledState(t *testing.T, jsonBytes []byte, expectEnabled bool) {
	var resultMap map[string]interface{}
	err := json.Unmarshal(jsonBytes, &resultMap)
	require.NoError(t, err)

	enabled, ok := resultMap["enabled"].(bool)
	require.True(t, ok, "expected to get enabled")

	assert.EqualValues(t, expectEnabled, enabled, "expected dashboard enabled = %t", expectEnabled)
}
