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
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	assert2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/assert"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestDeployScopedConfigurations(t *testing.T) {

	dashboardSharedEnvName := "DASHBOARD_SHARED"
	configFolder := "testdata/scoped-configs/"
	environment := "classic_env"
	manifestPath := configFolder + "manifest.yaml"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifestPath),
			runner.WithSuffix("ScopedConfigs"),
			runner.WithEnvironment(environment),
		},
		func(fs afero.Fs, testContext runner.TestContext) {

			// deploy with sharing turned off and assert state
			runner.SetTestEnvVar(t, dashboardSharedEnvName, "false", testContext.Suffix)
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy --verbose %s --environment %s", manifestPath, environment))
			require.NoError(t, err)

			assert2.AssertAllConfigsAvailability(t, fs, manifestPath, nil, environment, true)
			assertOverallDashboardSharedState(t, fs, testContext, manifestPath, environment, false)

			// deploy with sharing turned on and assert state
			runner.SetTestEnvVar(t, dashboardSharedEnvName, "true", testContext.Suffix)
			err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy --verbose %s --environment %s", manifestPath, environment))
			require.NoError(t, err)

			assertOverallDashboardSharedState(t, fs, testContext, manifestPath, environment, true)
		})
}

func assertOverallDashboardSharedState(t *testing.T, fs afero.Fs, testContext runner.TestContext, manifestPath string, environment string, expectShared bool) {
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	assert.Empty(t, errs)

	environmentDefinition := man.Environments.SelectedEnvironments[environment]
	clientSet := runner.CreateDynatraceClients(t, environmentDefinition)
	apis := api.NewAPIs()

	dashboardAPI := apis[api.Dashboard]
	dashboardName := runner.AddSuffix("Application monitoring", testContext.Suffix)
	exists, dashboardID, err := clientSet.ConfigClient.ExistsWithName(context.TODO(), dashboardAPI, dashboardName)

	require.NoError(t, err, "expect to be able to get dashboard by name")
	require.True(t, exists, "dashboard must exist")

	dashboardJSONBytes, err := clientSet.ConfigClient.Get(context.TODO(), dashboardAPI, dashboardID)
	require.NoError(t, err, "expect to be able to get dashboard by ID")
	assertDashboardSharedState(t, dashboardJSONBytes, expectShared)

	dashboardShareSettingsAPI := apis[api.DashboardShareSettings].ApplyParentObjectID(dashboardID)
	dashboardShareSettingsJSONBytes, err := clientSet.ConfigClient.Get(context.TODO(), dashboardShareSettingsAPI, "")
	require.NoError(t, err, "expect to be able to get dashboard shared settings by ID")
	assertDashboardShareSettingsEnabledState(t, dashboardShareSettingsJSONBytes, expectShared)
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

	assert.EqualValues(t, expectEnabled, enabled, "expected dashboard-share-settings enabled = %t", expectEnabled)
}
