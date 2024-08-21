/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package classic_test

import (
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"strconv"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils/matcher"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/classic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDownload_KeyUserActionMobile(t *testing.T) {
	standardAPIs := api.NewAPIs()
	apiMap := api.APIs{api.KeyUserActionsMobile: standardAPIs[api.KeyUserActionsMobile],
		api.ApplicationMobile: standardAPIs[api.ApplicationMobile],
	}

	applicationId := "some-application-id"

	c := client.NewMockDynatraceClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(context.TODO(), apiMap[api.ApplicationMobile]).Return([]dtclient.Value{{Id: applicationId, Name: "some-application-name"}}, nil).Times(2)
	c.EXPECT().ListConfigs(context.TODO(), apiMap[api.KeyUserActionsMobile].ApplyParentObjectID(applicationId)).Return([]dtclient.Value{{Id: "abc", Name: "abc"}}, nil).Times(1)
	c.EXPECT().ReadConfigById(context.TODO(), apiMap[api.ApplicationMobile], applicationId).Return([]byte(`{"keyUserActions": [{"name": "abc"}]}`), nil).Times(1)
	c.EXPECT().ReadConfigById(context.TODO(), apiMap[api.KeyUserActionsMobile].ApplyParentObjectID(applicationId), "").Return([]byte(`{}`), nil).Times(1)

	configurations, err := classic.Download(c, "project", apiMap, classic.ApiContentFilters)
	require.NoError(t, err)
	assert.Len(t, configurations, 2, "Expected two configurations downloaded")

	require.Len(t, configurations[api.KeyUserActionsMobile], 1)
	gotKeyUserActionsMobileConfig := configurations[api.KeyUserActionsMobile][0]

	assert.Equal(t, reference.New("project", api.ApplicationMobile, applicationId, "id"), gotKeyUserActionsMobileConfig.Parameters[config.ScopeParameter])
	assert.Len(t, gotKeyUserActionsMobileConfig.Parameters, 2)
	assert.Equal(t, valueParam.New("abc"), gotKeyUserActionsMobileConfig.Parameters[config.NameParameter])
	assert.Equal(t, config.ClassicApiType{Api: api.KeyUserActionsMobile}, gotKeyUserActionsMobileConfig.Type)
	assert.Equal(t, coordinate.Coordinate{Project: "project", Type: api.KeyUserActionsMobile, ConfigId: "abcsome-application-id"}, gotKeyUserActionsMobileConfig.Coordinate)
	assert.False(t, gotKeyUserActionsMobileConfig.Skip)
}

func apiGet(a string) api.API {
	return api.NewAPIs()[a]
}

func toAPIs(apis ...api.API) api.APIs {
	ret := make(map[string]api.API)
	for _, a := range apis {
		ret[a.ID] = a
	}
	return ret
}

func TestDownload_KeyUserActionWeb(t *testing.T) {
	c := client.NewMockDynatraceClient(gomock.NewController(t))
	ctx := context.TODO()
	c.EXPECT().ListConfigs(ctx, matcher.EqAPI(apiGet(api.ApplicationWeb))).Return([]dtclient.Value{{Id: "applicationID", Name: "web application name"}}, nil)
	c.EXPECT().ListConfigs(ctx, matcher.EqAPI((apiGet(api.KeyUserActionsWeb).ApplyParentObjectID("applicationID")))).Return([]dtclient.Value{{Id: "APPLICATION_METHOD-ID", Name: "the_name"}}, nil)
	c.EXPECT().ReadConfigById(ctx, gomock.Any(), "").Return([]byte(`{"keyUserActionList":[{"name":"the_name","actionType":"Load","domain":"dt.com","meIdentifier":"APPLICATION_METHOD-ID"}]}`), nil)

	apiMap := api.NewAPIs().Filter(api.RetainByName([]string{api.KeyUserActionsWeb}))

	configurations, err := classic.Download(c, "project", apiMap, map[string]classic.ContentFilter{})
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
	gotConfig := configurations[api.KeyUserActionsWeb][0]
	assert.Len(t, configurations[api.KeyUserActionsWeb], 1)
	assert.Equal(t, reference.New("project", api.ApplicationWeb, "applicationID", "id"), gotConfig.Parameters[config.ScopeParameter])
	assert.Len(t, gotConfig.Parameters, 2)
	assert.Equal(t, valueParam.New("the_name"), gotConfig.Parameters[config.NameParameter])
	assert.Equal(t, config.ClassicApiType{Api: api.KeyUserActionsWeb}, gotConfig.Type)
	assert.Equal(t, coordinate.Coordinate{Project: "project", Type: api.KeyUserActionsWeb, ConfigId: "APPLICATION_METHOD-IDapplicationID"}, gotConfig.Coordinate)
	assert.False(t, gotConfig.Skip)
}

func TestDownload_KeyUserActionWeb_Uniqnes(t *testing.T) {
	c := client.NewMockDynatraceClient(gomock.NewController(t))
	ctx := context.TODO()
	c.EXPECT().ListConfigs(ctx, matcher.EqAPI(apiGet(api.ApplicationWeb))).Return([]dtclient.Value{{Id: "applicationID", Name: "web application name"}}, nil)
	c.EXPECT().ListConfigs(ctx, matcher.EqAPI((apiGet(api.KeyUserActionsWeb).ApplyParentObjectID("applicationID")))).Return([]dtclient.Value{{Id: "APPLICATION_METHOD-ID", Name: "the_name"}, {Id: "APPLICATION_METHOD-ID2", Name: "the_name"}, {Id: "APPLICATION_METHOD-ID3", Name: "the_name"}}, nil)
	c.EXPECT().ReadConfigById(ctx, matcher.EqAPI(apiGet(api.KeyUserActionsWeb).ApplyParentObjectID("applicationID")), "").Return([]byte(`{
"keyUserActionList":[
  {"name":"the_name","actionType":"Load","domain":"dt.com","meIdentifier":"APPLICATION_METHOD-ID"},
  {"name":"the_name","actionType":"Load","domain":"dt2.com","meIdentifier":"APPLICATION_METHOD-ID2"},
  {"name":"the_name","actionType":"Custom","domain":"dt.com","meIdentifier":"APPLICATION_METHOD-ID3"}
]}`), nil).Times(3)

	apiMap := api.NewAPIs().Filter(api.RetainByName([]string{api.KeyUserActionsWeb}))

	configurations, err := classic.Download(c, "project", apiMap, map[string]classic.ContentFilter{})
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
	assert.Len(t, configurations[api.KeyUserActionsWeb], 3)
}

func TestDownload_SkipConfigThatShouldNotBePersisted(t *testing.T) {
	api1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	api2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}

	c := client.NewMockDynatraceClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(api1)).Return([]dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil)
	c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(api2)).Return([]dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("{}"), nil).Times(2)

	filters := map[string]classic.ContentFilter{"API_ID_1": {
		ShouldConfigBePersisted: func(_ map[string]interface{}) bool {
			return false
		},
	}}

	configurations, err := classic.Download(c, "project", toAPIs(api1, api2), filters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_SkipConfigBeforeDownload(t *testing.T) {
	api1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	api2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}

	filters := map[string]classic.ContentFilter{
		"API_ID_1": {
			ShouldBeSkippedPreDownload: func(_ dtclient.Value) bool {
				return true
			},
		},
		"API_ID_2": {
			ShouldConfigBePersisted: func(_ map[string]interface{}) bool {
				return false
			},
		},
	}

	tests := []struct {
		name                  string
		withFiltering         bool
		wantDownloadedConfigs int
	}{
		{
			"downloads nothing if filters active - default configuration",
			true,
			0,
		},
		{
			"downloads all if filtering is off",
			false,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockDynatraceClient(gomock.NewController(t))
			c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(api1)).Return([]dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil)
			c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(api2)).Return([]dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil)
			c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("{}"), nil).AnyTimes()

			t.Setenv(featureflags.Permanent[featureflags.DownloadFilterClassicConfigs].EnvName(), strconv.FormatBool(tt.withFiltering))
			t.Setenv(featureflags.Permanent[featureflags.DownloadFilter].EnvName(), strconv.FormatBool(tt.withFiltering))

			configurations, err := classic.Download(c, "project", toAPIs(api1, api2), filters)
			assert.NoError(t, err)
			assert.Len(t, configurations, tt.wantDownloadedConfigs)
		})
	}
}

func TestDownload_FilteringCanBeTurnedOffViaFeatureFlags(t *testing.T) {
	api1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	api2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}

	c := client.NewMockDynatraceClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(api1)).Return([]dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil)
	c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(api2)).Return([]dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	filters := map[string]classic.ContentFilter{"API_ID_1": {
		ShouldBeSkippedPreDownload: func(_ dtclient.Value) bool {
			return true
		},
	}}

	configurations, err := classic.Download(c, "project", toAPIs(api1, api2), filters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func Test_generalCases(t *testing.T) {
	api1 := api.API{ID: "API_1", URLPath: "url_1", NonUniqueName: true}
	api2 := api.API{ID: "API_2", URLPath: "url_2", NonUniqueName: false}

	tests := []struct {
		name           string
		mockList       []listMockData
		mockConfigByID []readConfigByIDData
		expectedKeys   []string // the tick is to have only one entry per an api, and to check which API is present in resulut
	}{
		{
			name: "ReadConfigById (GET by ID) returns empty configuration - works",
			mockList: []listMockData{
				{api: api1, response: []dtclient.Value{{Id: "ID_1", Name: "NAME_1"}}},
				{api: api2, response: []dtclient.Value{{Id: "ID_2", Name: "NAME_2"}}},
			},
			mockConfigByID: []readConfigByIDData{
				{id: "ID_1", response: "{}"},
				{id: "ID_2", response: "{}"},
			},
			expectedKeys: []string{"API_1", "API_2"},
		},
		{
			name: "ReadConfigById (GET by ID) details returns NO configuration - works",
			mockList: []listMockData{
				{api: api1, response: []dtclient.Value{{Id: "ID_1", Name: "NAME_1"}}},
				{api: api2, response: []dtclient.Value{{Id: "ID_2", Name: "NAME_2"}}},
			},
			mockConfigByID: []readConfigByIDData{
				{id: "ID_1"},
				{id: "ID_2"},
			},
		},
		{
			name: "ReadConfigById (GET by ID) returns error - works",
			mockList: []listMockData{
				{api: api1, response: []dtclient.Value{{Id: "ID_1", Name: "NAME_1"}}},
				{api: api2, response: []dtclient.Value{{Id: "ID_2", Name: "NAME_2"}}},
			},
			mockConfigByID: []readConfigByIDData{
				{id: "ID_1", err: fmt.Errorf("some HTTP error")},
				{id: "ID_2", err: fmt.Errorf("some HTTP error")},
			},
		},
		{
			name: "ListConfigs returns nothing - works",
			mockList: []listMockData{
				{api: api1, response: []dtclient.Value{{Id: "ID_1", Name: "NAME_1"}}},
				{api: api2},
			},
			mockConfigByID: []readConfigByIDData{
				{id: "ID_1", response: "{}"},
			},
			expectedKeys: []string{"API_1"},
		},
		{
			name: "ListConfigs returns an empty list - works",
			mockList: []listMockData{
				{api: api1, response: []dtclient.Value{{Id: "ID_1", Name: "NAME_1"}}},
				{api: api2, response: []dtclient.Value{}},
			},
			mockConfigByID: []readConfigByIDData{
				{id: "ID_1", response: "{}"},
			},
			expectedKeys: []string{"API_1"},
		},
		{
			name: "malformed response from an API - ignored",
			mockList: []listMockData{
				{api: api1, response: []dtclient.Value{{Id: "ID_1", Name: "NAME_1"}}},
				{api: api2, response: []dtclient.Value{{Id: "ID_2", Name: "NAME_2"}}}},
			mockConfigByID: []readConfigByIDData{
				{id: "ID_1", response: "{}"},
				{id: "ID_2", response: "not a JSON - ignore"},
			},
			expectedKeys: []string{"API_1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := client.NewMockDynatraceClient(gomock.NewController(t))
			for _, m := range tc.mockList {
				c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(m.api)).Return(m.response, m.err)
			}
			for _, m := range tc.mockConfigByID {
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any(), m.id).Return([]byte(m.response), m.err)
			}

			actual, err := classic.Download(c, "project", toAPIs(api1, api2), classic.ApiContentFilters)

			require.NoError(t, err)
			require.Len(t, actual, len(tc.expectedKeys))
			for _, k := range tc.expectedKeys {
				assert.Contains(t, actual, k)
			}
		})
	}
}

type (
	listMockData struct {
		api      api.API
		response []dtclient.Value
		err      error
	}
	readConfigByIDData struct {
		id, response string
		err          error
	}
)

func TestDownload_SkippedParentsSkipChildren(t *testing.T) {
	parentAPI := api.API{
		ID:            "PARENT_API_ID",
		URLPath:       "PARENT_API_PATH",
		NonUniqueName: true}

	apiMap := api.APIs{
		"PARENT_API_ID": parentAPI,
		"CHILD_API_ID": api.API{ID: "CHILD_API_ID",
			URLPath:       "CHILD_API_PATH",
			NonUniqueName: false,
			Parent:        &parentAPI}}

	contentFilters := map[string]classic.ContentFilter{
		"PARENT_API_ID": {
			ShouldBeSkippedPreDownload: func(value dtclient.Value) bool { return true },
		},
	}

	c := client.NewMockDynatraceClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(parentAPI)).Return([]dtclient.Value{{Id: "PARENT_ID_1", Name: "PARENT_NAME_1"}}, nil).Times(2)

	configurations, err := classic.Download(c, "project", apiMap, contentFilters)
	require.NoError(t, err)
	assert.Len(t, configurations, 0, "Expected no configurations as everything is skipped")
}
