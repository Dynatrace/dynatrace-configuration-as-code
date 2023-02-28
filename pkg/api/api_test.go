//go:build unit

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

package api

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/environment/v1"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testDevEnvironment = v1.NewEnvironment("development", "Dev", "", "https://url/to/dev/environment", "DEV")
var testManagementZoneApi = NewStandardApi("management-zone", "/api/config/v1/managementZones", false, "", false)
var testDashboardApi = NewStandardApi("dashboard", "/api/config/v1/dashboards", true, "dashboard-v2", false)

var hostsAutoUpdateApiId = "hosts-auto-update"
var testHostsAutoUpdateApi = NewSingleConfigurationApi(hostsAutoUpdateApiId, "/api/config/v1/hosts/autoupdate", "", false)

func TestGetUrl(t *testing.T) {

	url := testManagementZoneApi.GetUrl(testDevEnvironment.GetEnvironmentUrl())
	assert.Equal(t, "https://url/to/dev/environment/api/config/v1/managementZones", url)
}

func TestCreateApis(t *testing.T) {
	apis := NewApis()

	assert.Contains(t, apis, "notification", "Expected `notification` key in KnownApis")
	assert.Equal(t, "https://url/to/dev/environment/api/config/v1/notifications", apis["notification"].GetUrl(testDevEnvironment.GetEnvironmentUrl()), "Expected to get `notification` API url")
}

func TestCreateApisResultsInError(t *testing.T) {
	apis := NewApis()

	assert.NotContainsf(t, apis, "notexistingkey", "Expected error on `notexistingkey` key in createApis")
}

func TestIfFolderContainsApiInPath(t *testing.T) {
	apis := NewApis()
	assert.False(t, apis.ContainsApiName("trillian"), "Check if `trillian` is an API")
	assert.True(t, apis.ContainsApiName("extension"), "Check if `extension` is an API")
	assert.True(t, apis.ContainsApiName("/project/sub-project/extension/subfolder"), "Check if `extension` is an API")
	assert.False(t, apis.ContainsApiName("/project/sub-project"), "Check if `extension` is an API")
}

func TestIsSingleConfigurationApi(t *testing.T) {
	isSingleConfigurationApi := testDashboardApi.IsSingleConfigurationApi()
	assert.False(t, isSingleConfigurationApi)

	isSingleConfigurationApi = testHostsAutoUpdateApi.IsSingleConfigurationApi()
	assert.True(t, isSingleConfigurationApi)
}

func TestIsNonUniqueNameApi(t *testing.T) {
	isNonUniqueNameApi := testDashboardApi.IsNonUniqueNameApi()
	assert.True(t, isNonUniqueNameApi)

	isNonUniqueNameApi = testHostsAutoUpdateApi.IsNonUniqueNameApi()
	assert.False(t, isNonUniqueNameApi)
}

func TestContains(t *testing.T) {
	apis := NewApis()
	assert.True(t, apis.Contains("alerting-profile"))
	assert.False(t, apis.Contains("something"))
}
