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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"strconv"

	"testing"
)

func TestDownloadConfigs_FailedToFindConfigsToDownload_(t *testing.T) {
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

func TestDownload_ConfigsDownloaded_(t *testing.T) {
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

func TestDownload_KeyUserActionMobile_(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(context.TODO(), api.NewAPIs()["application-mobile"]).Return([]dtclient.Value{{Id: "some-application-id", Name: "some-application-name"}}, nil)
	c.EXPECT().ListConfigs(context.TODO(), api.NewAPIs()["key-user-actions-mobile"].Resolve("some-application-id")).Return([]dtclient.Value{{Id: "abc", Name: "abc"}}, nil)
	c.EXPECT().ReadConfigById(gomock.Any(), "").Return([]byte(`{"keyUserActions": [{"name": "abc"}]}`), nil)

	apiMap := api.APIs{"key-user-actions-mobile": api.NewAPIs()["key-user-actions-mobile"]}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)

	assert.Len(t, configurations, 1)
	gotConfig := configurations["key-user-actions-mobile"][0]
	assert.Len(t, configurations["key-user-actions-mobile"], 1)
	assert.Equal(t, reference.New("project", "application-mobile", "some-application-id", "id"), gotConfig.Parameters[config.ScopeParameter])
	assert.Len(t, gotConfig.Parameters, 2)
	assert.Equal(t, valueParam.New("abc"), gotConfig.Parameters[config.NameParameter])
	assert.Equal(t, config.ClassicApiType{Api: "key-user-actions-mobile"}, gotConfig.Type)
	assert.Equal(t, coordinate.Coordinate{Project: "project", Type: "key-user-actions-mobile", ConfigId: "abcsome-application-id"}, gotConfig.Coordinate)
	assert.False(t, gotConfig.Skip)
}

func TestDownload_SingleConfigurationAPI_(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", SingleConfiguration: true, NonUniqueName: true}
	apiMap := api.APIs{"API_ID_1": testAPI1}

	configurations, err := Download(c, "project", apiMap, ApiContentFilters)
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_ErrorFetchingConfig_(t *testing.T) {
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

func TestDownload_ConfigsDownloaded_WithEmptyFilte_(t *testing.T) {
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

func TestDownload_SkipConfigThatShouldNotBePersisted_(t *testing.T) {

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
