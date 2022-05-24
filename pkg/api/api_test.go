//go:build unit
// +build unit

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
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"gotest.tools/assert"
)

var testDevEnvironment = environment.NewEnvironment("development", "Dev", "", "https://url/to/dev/environment", "DEV")
var testAlertingProfileApi = NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false)
var testManagementZoneApi = NewStandardApi("management-zone", "/api/config/v1/managementZones", false)
var testDashboardApi = NewStandardApi("dashboard", "/api/config/v1/dashboards", true)
var testReportsApi = NewStandardApi("reports", "/api/config/v1/reports", false)

var hostsAutoUpdateApiId = "hosts-auto-update"
var testHostsAutoUpdateApi = NewSingleConfigurationApi(hostsAutoUpdateApiId, "/api/config/v1/hosts/autoupdate")

func TestGetUrl(t *testing.T) {

	url := testManagementZoneApi.GetUrl(testDevEnvironment)
	assert.Equal(t, "https://url/to/dev/environment/api/config/v1/managementZones", url)
}

func TestCreateApis(t *testing.T) {
	apis := make(map[string]Api)
	apis = NewApis()

	notification, ok := apis["notification"]
	assert.Assert(t, ok, "Expected `notification` key in Apis")
	assert.Equal(t, notification.GetUrl(testDevEnvironment), "https://url/to/dev/environment/api/config/v1/notifications", "Expected to get `notification` API url")
}

func TestCreateApisResultsInError(t *testing.T) {
	apis := make(map[string]Api)
	apis = NewApis()

	_, ok := apis["notexistingkey"]
	assert.Assert(t, !ok, "Expected error on `notexistingkey` key in createApis")
}

func TestIfFolderContainsApiInPath(t *testing.T) {
	assert.Equal(t, ContainsApiName("trillian"), false, "Check if `trillian` is an API")
	assert.Equal(t, ContainsApiName("extension"), true, "Check if `extension` is an API")
	assert.Equal(t, ContainsApiName("/project/sub-project/extension/subfolder"), true, "Check if `extension` is an API")
	assert.Equal(t, ContainsApiName("/project/sub-project"), false, "Check if `extension` is an API")
}

func TestIsSingleConfigurationApi(t *testing.T) {
	isSingleConfigurationApi := testDashboardApi.IsSingleConfigurationApi()
	assert.Equal(t, false, isSingleConfigurationApi)

	isSingleConfigurationApi = testHostsAutoUpdateApi.IsSingleConfigurationApi()
	assert.Equal(t, true, isSingleConfigurationApi)
}

func TestIsNonUniqueNameApi(t *testing.T) {
	isNonUniqueNameApi := testDashboardApi.IsNonUniqueNameApi()
	assert.Equal(t, true, isNonUniqueNameApi)

	isNonUniqueNameApi = testHostsAutoUpdateApi.IsNonUniqueNameApi()
	assert.Equal(t, false, isNonUniqueNameApi)
}

func TestNewIdValue(t *testing.T) {
	value := testHostsAutoUpdateApi.NewIdValue()
	assert.Equal(t, hostsAutoUpdateApiId, value.Name)
	assert.Equal(t, hostsAutoUpdateApiId, value.Id)
}
