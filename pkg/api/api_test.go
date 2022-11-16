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
	"reflect"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"gotest.tools/assert"
)

var testDevEnvironment = environment.NewEnvironment("development", "Dev", "", "https://url/to/dev/environment", "DEV")
var testManagementZoneApi = NewStandardApi("management-zone", "/api/config/v1/managementZones", false, "", false)
var testDashboardApi = NewStandardApi("dashboard", "/api/config/v1/dashboards", true, "dashboard-v2", false)

var hostsAutoUpdateApiId = "hosts-auto-update"
var testHostsAutoUpdateApi = NewSingleConfigurationApi(hostsAutoUpdateApiId, "/api/config/v1/hosts/autoupdate", "", false)

func TestGetUrl(t *testing.T) {

	url := testManagementZoneApi.GetUrlFromEnvironmentUrl(testDevEnvironment.GetEnvironmentUrl())
	assert.Equal(t, "https://url/to/dev/environment/api/config/v1/managementZones", url)
}

func TestCreateApis(t *testing.T) {
	apis := make(map[string]Api)
	apis = NewApis()

	notification, ok := apis["notification"]
	assert.Assert(t, ok, "Expected `notification` key in KnownApis")
	assert.Equal(t, notification.GetUrlFromEnvironmentUrl(testDevEnvironment.GetEnvironmentUrl()), "https://url/to/dev/environment/api/config/v1/notifications", "Expected to get `notification` API url")
}

func TestCreateApisResultsInError(t *testing.T) {
	apis := make(map[string]Api)
	apis = NewApis()

	_, ok := apis["notexistingkey"]
	assert.Assert(t, !ok, "Expected error on `notexistingkey` key in createApis")
}

func TestIfFolderContainsApiInPath(t *testing.T) {
	apis := NewApis()
	assert.Equal(t, apis.ContainsApiName("trillian"), false, "Check if `trillian` is an API")
	assert.Equal(t, apis.ContainsApiName("extension"), true, "Check if `extension` is an API")
	assert.Equal(t, apis.ContainsApiName("/project/sub-project/extension/subfolder"), true, "Check if `extension` is an API")
	assert.Equal(t, apis.ContainsApiName("/project/sub-project"), false, "Check if `extension` is an API")
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

func TestIsDeprecatedApi(t *testing.T) {
	isDeprecatedApi := testDashboardApi.IsDeprecatedApi()
	assert.Equal(t, true, isDeprecatedApi)

	isDeprecatedApi = testManagementZoneApi.IsDeprecatedApi()
	assert.Equal(t, false, isDeprecatedApi)
}

func TestApiMap_Filter(t *testing.T) {

	skip := createSkipableApi(t, true)
	dontSkip := createSkipableApi(t, false)

	tests := []struct {
		name   string
		m      ApiMap
		filter func(api Api) bool
		want   ApiMap
		want1  ApiMap
	}{
		{
			"split nothing",
			nil,
			func(a Api) bool { return true },
			ApiMap{},
			ApiMap{},
		},
		{
			"split only first",
			ApiMap{"a": skip, "b": dontSkip},
			func(a Api) bool { return true },
			ApiMap{},
			ApiMap{"a": skip, "b": dontSkip},
		},
		{
			"split only second",
			ApiMap{"a": skip, "b": dontSkip},
			func(a Api) bool { return false },
			ApiMap{"a": skip, "b": dontSkip},
			ApiMap{},
		},
		{
			"split by download second",
			ApiMap{"a": skip, "b": dontSkip},
			func(a Api) bool { return a.ShouldSkipDownload() },
			ApiMap{"b": dontSkip},
			ApiMap{"a": skip},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.m.Filter(tt.filter)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Filter() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func createSkipableApi(t *testing.T, skip bool) Api {
	api, finish := CreateAPIMockFactory(t)
	finish() // don't care about verify

	api.EXPECT().ShouldSkipDownload().Return(skip)
	return api
}

func TestApiMap_FilterApisByName(t *testing.T) {
	tests := []struct {
		name            string
		m               ApiMap
		args            []string
		wantApis        ApiMap
		wantUnknownApis []string
	}{
		{
			"empty values",
			ApiMap{},
			[]string{},
			ApiMap{},
			[]string{},
		},
		{
			"empty map, non empty keys",
			ApiMap{},
			[]string{"a"},
			ApiMap{},
			[]string{"a"},
		},
		{
			"non empty map, empty values",
			createApiMapWithKeys("a"),
			[]string{},
			createApiMapWithKeys("a"),
			[]string{},
		},
		{
			"full matching values",
			createApiMapWithKeys("a"),
			[]string{"a"},
			createApiMapWithKeys("a"),
			[]string{},
		},
		{
			"partially matching values",
			createApiMapWithKeys("a"),
			[]string{"a", "b"},
			createApiMapWithKeys("a"),
			[]string{"b"},
		},
		{
			"partially matching values with more keys in map",
			createApiMapWithKeys("a", "c"),
			[]string{"a", "b"},
			createApiMapWithKeys("a"),
			[]string{"b"},
		},
		{
			"filtering map",
			createApiMapWithKeys("a", "c"),
			[]string{"a"},
			createApiMapWithKeys("a"),
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApis, gotUnknownApis := tt.m.FilterApisByName(tt.args)
			if !reflect.DeepEqual(gotApis, tt.wantApis) {
				t.Errorf("FilterApisByName() gotApis = %v, want %v", gotApis, tt.wantApis)
			}
			if !reflect.DeepEqual(gotUnknownApis, tt.wantUnknownApis) {
				t.Errorf("FilterApisByName() gotUnknownApis = %v, want %v", gotUnknownApis, tt.wantUnknownApis)
			}
		})
	}
}

func createApiMapWithKeys(keys ...string) ApiMap {
	m := make(ApiMap, len(keys))

	for _, k := range keys {
		m[k] = nil
	}

	return m
}
