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

package classic

import (
	"context"
	"fmt"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDownloadConfigs_FailedToFindConfigsToDownload(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).Return([]dtclient.Value{}, fmt.Errorf("NO"))

	testAPI := api.API{ID: "API_ID", URLPath: "API_PATH", NonUniqueName: true}
	apiMap := api.APIs{"API_ID": testAPI}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 0)
}

func TestDownload_NoConfigsToDownloadFound_(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).Return([]dtclient.Value{}, nil)

	testAPI := api.API{ID: "API_ID", URLPath: "API_PATH", NonUniqueName: true}

	apiMap := api.APIs{"API_ID": testAPI}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 0)
}

func TestDownload_ConfigsDownloaded(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 2)
}

func TestDownload_KeyUserActionMobile(t *testing.T) {
	standardAPIs := api.NewAPIs()
	apiMap := api.APIs{api.KeyUserActionsMobile: standardAPIs[api.KeyUserActionsMobile],
		api.ApplicationMobile: standardAPIs[api.ApplicationMobile],
	}

	applicationId := "some-application-id"

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(context.TODO(), apiMap[api.ApplicationMobile]).Return([]dtclient.Value{{Id: applicationId, Name: "some-application-name"}}, nil).Times(2)
	c.EXPECT().ListConfigs(context.TODO(), apiMap[api.KeyUserActionsMobile].Resolve(applicationId)).Return([]dtclient.Value{{Id: "abc", Name: "abc"}}, nil).Times(1)
	c.EXPECT().ReadConfigById(apiMap[api.ApplicationMobile], applicationId).Return([]byte(`{"keyUserActions": [{"name": "abc"}]}`), nil).Times(1)
	c.EXPECT().ReadConfigById(apiMap[api.KeyUserActionsMobile].Resolve(applicationId), "").Return([]byte(`{}`), nil).Times(1)

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
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

func TestDownload_KeyUserActionWeb(t *testing.T) {

	c := dtclient.NewMockClient(gomock.NewController(t))
	ctx := context.TODO()
	apis := api.NewAPIs()
	c.EXPECT().ListConfigs(ctx, matcher.EqAPI(apis["application-web"])).Return([]dtclient.Value{{Id: "applicationID", Name: "web-application"}}, nil)
	c.EXPECT().ListConfigs(ctx, matcher.EqAPI((apis["key-user-actions-web"].Resolve("applicationID")))).Return([]dtclient.Value{{Id: "APPLICATION_METHOD-ID", Name: "the_name"}}, nil)
	c.EXPECT().ReadConfigById(gomock.Any(), "").Return([]byte(`{"keyUserActionList":[{"name":"the_name","actionType":"Load","domain":"dt.com","meIdentifier":"APPLICATION_METHOD-ID"}]}`), nil)

	apiMap := api.NewAPIs().Filter(api.RetainByName([]string{"key-user-actions-web"}))

	configurations, err := Download(c, "project", apiMap, map[string]ContentFilter{})
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
	gotConfig := configurations["key-user-actions-web"][0]
	assert.Len(t, configurations["key-user-actions-web"], 1)
	assert.Equal(t, reference.New("project", "application-web", "applicationID", "id"), gotConfig.Parameters[config.ScopeParameter])
	assert.Len(t, gotConfig.Parameters, 2)
	assert.Equal(t, valueParam.New("the_name"), gotConfig.Parameters[config.NameParameter])
	assert.Equal(t, config.ClassicApiType{Api: "key-user-actions-web"}, gotConfig.Type)
	assert.Equal(t, coordinate.Coordinate{Project: "project", Type: "key-user-actions-web", ConfigId: "APPLICATION_METHOD-IDapplicationID"}, gotConfig.Coordinate)
	assert.False(t, gotConfig.Skip)
}

func TestDownload_SingleConfigurationAPI(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", SingleConfiguration: true, NonUniqueName: true}
	apiMap := api.APIs{"API_ID_1": testAPI1}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_ErrorFetchingConfig(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).DoAndReturn(func(a api.API, id string) (json []byte, err error) {
		if a.ID == "API_ID_1" {
			return []byte("{}"), fmt.Errorf("NO")
		}
		return []byte("{}"), nil
	}).Times(2)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_ConfigsDownloaded_WithEmptyFile(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: true}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 2)
}

func TestDownload_SkipConfigThatShouldNotBePersisted(t *testing.T) {

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil).Times(2)

	filters := map[string]ContentFilter{"API_ID_1": {
		ShouldConfigBePersisted: func(_ map[string]interface{}) bool {
			return false
		},
	}}

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}
	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations, err := Download(c, "project", apiMap, filters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_SkipConfigBeforeDownload(t *testing.T) {

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).AnyTimes()
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil).AnyTimes()

	filters := map[string]ContentFilter{
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

	apiMap := api.APIs{
		"API_ID_1": api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true},
		"API_ID_2": api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false},
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
			t.Setenv(featureflags.DownloadFilterClassicConfigs().EnvName(), strconv.FormatBool(tt.withFiltering))
			t.Setenv(featureflags.DownloadFilter().EnvName(), strconv.FormatBool(tt.withFiltering))

			configurations, err := Download(c, "project", apiMap, filters)
			assert.NoError(t, err)
			assert.Len(t, configurations, tt.wantDownloadedConfigs)
		})
	}
}

func TestDownload_FilteringCanBeTurnedOffViaFeatureFlags(t *testing.T) {

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	filters := map[string]ContentFilter{"API_ID_1": {
		ShouldBeSkippedPreDownload: func(_ dtclient.Value) bool {
			return true
		},
	}}

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}
	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations, err := Download(c, "project", apiMap, filters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_APIWithoutAnyConfigAvailableAreNotDownloaded(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{}, nil
		}
		return nil, nil
	}).Times(2)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}
	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_MalformedResponseFromAnAPI(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(2)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("-1"), nil)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}
	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

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

	contentFilters := map[string]ContentFilter{
		"PARENT_API_ID": {
			ShouldBeSkippedPreDownload: func(value dtclient.Value) bool { return true },
		},
	}

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(parentAPI)).Return([]dtclient.Value{{Id: "PARENT_ID_1", Name: "PARENT_NAME_1"}}, nil).Times(2)

	configurations, err := Download(c, "project", apiMap, contentFilters)
	require.NoError(t, err)
	assert.Len(t, configurations, 0, "Expected no configurations as everything is skipped")
}

func TestDownload_SingleConfigurationChild(t *testing.T) {
	parentAPI := api.API{
		ID:            "PARENT_API_ID",
		URLPath:       "PARENT_API_PATH",
		NonUniqueName: true}

	apiMap := api.APIs{
		"PARENT_API_ID": parentAPI,
		"CHILD_API_ID": api.API{ID: "CHILD_API_ID",
			URLPath:             "CHILD_API_PATH",
			NonUniqueName:       false,
			Parent:              &parentAPI,
			SingleConfiguration: true}}

	contentFilters := map[string]ContentFilter{}

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(parentAPI)).Return([]dtclient.Value{{Id: "PARENT_ID_1", Name: "PARENT_NAME_1"}}, nil).Times(2)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil).AnyTimes()

	configurations, err := Download(c, "project", apiMap, contentFilters)
	require.NoError(t, err)
	require.Len(t, configurations, 2, "Expected two configurations")
	require.Len(t, configurations["PARENT_API_ID"], 1)
	require.Len(t, configurations["CHILD_API_ID"], 1)
	assert.Equal(t, configurations["PARENT_API_ID"][0].Coordinate.ConfigId, configurations["CHILD_API_ID"][0].Coordinate.ConfigId, "Single child config should have the same config ID as parent")
}
