//go:build unit

// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package classic

import (
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestDownloadConfigs_FailedToFindConfigsToDownload(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).Return([]dtclient.Value{}, fmt.Errorf("NO"))

	testAPI := api.API{ID: "API_ID", URLPath: "API_PATH", NonUniqueName: true}
	apiMap := api.APIs{"API_ID": testAPI}

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project")
	assert.NoError(t, err)
	assert.Len(t, configurations, 0)
}

func TestDownload_NoConfigsToDownloadFound(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).Return([]dtclient.Value{}, nil)

	testAPI := api.API{ID: "API_ID", URLPath: "API_PATH", NonUniqueName: true}

	apiMap := api.APIs{"API_ID": testAPI}

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project")
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

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project")
	assert.NoError(t, err)
	assert.Len(t, configurations, 2)
}

func TestDownload_SingleConfigurationAPI(t *testing.T) {
	client := dtclient.NewMockClient(gomock.NewController(t))
	client.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", SingleConfiguration: true, NonUniqueName: true}
	apiMap := api.APIs{"API_ID_1": testAPI1}

	downloader := NewDownloader(client, WithAPIs(apiMap))

	configurations, err := downloader.Download("project")
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

	downloader := NewDownloader(c, WithAPIs(apiMap))
	configurations, err := downloader.Download("project")
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_ConfigsDownloaded_WithEmptyFilter(t *testing.T) {
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

	downloader := NewDownloader(c, WithAPIs(apiMap), WithAPIContentFilters(map[string]contentFilter{}))

	configurations, err := downloader.Download("project")
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

	apiFilters := map[string]contentFilter{"API_ID_1": {
		shouldConfigBePersisted: func(_ map[string]interface{}) bool {
			return false
		},
	}}

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}
	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	downloader := NewDownloader(c, WithAPIs(apiMap), WithAPIContentFilters(apiFilters))

	configurations, err := downloader.Download("project")
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

	apiFilters := map[string]contentFilter{
		"API_ID_1": {
			shouldBeSkippedPreDownload: func(_ dtclient.Value) bool {
				return true
			},
		},
		"API_ID_2": {
			shouldConfigBePersisted: func(_ map[string]interface{}) bool {
				return false
			},
		},
	}

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}
	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	downloader := NewDownloader(c, WithAPIs(apiMap), WithAPIContentFilters(apiFilters))

	type flags struct {
		downloadFilterFF        bool
		downloadFilterConfigsFF bool
	}
	tests := []struct {
		name                  string
		given                 flags
		wantDownloadedConfigs int
	}{
		{
			"downloads nothing if filters active",
			flags{
				downloadFilterFF:        true,
				downloadFilterConfigsFF: true,
			},
			0,
		},
		{
			"downloads all if base filter off",
			flags{
				downloadFilterFF:        false,
				downloadFilterConfigsFF: true,
			},
			2,
		},
		{
			"downloads all if configs filter off",
			flags{
				downloadFilterFF:        true,
				downloadFilterConfigsFF: false,
			},
			2,
		},
		{
			"downloads all if both filters off",
			flags{
				downloadFilterFF:        false,
				downloadFilterConfigsFF: false,
			},
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(featureflags.DownloadFilter().EnvName(), strconv.FormatBool(tt.given.downloadFilterFF))
			t.Setenv(featureflags.DownloadFilterClassicConfigs().EnvName(), strconv.FormatBool(tt.given.downloadFilterConfigsFF))

			configurations, err := downloader.Download("project")
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

	apiFilters := map[string]contentFilter{"API_ID_1": {
		shouldBeSkippedPreDownload: func(_ dtclient.Value) bool {
			return true
		},
	}}

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", NonUniqueName: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2", NonUniqueName: false}
	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	downloader := NewDownloader(c, WithAPIs(apiMap), WithAPIContentFilters(apiFilters))

	configurations, err := downloader.Download("project")
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_EmptyAPIMap_ResultsInError(t *testing.T) {
	client := dtclient.NewMockClient(gomock.NewController(t))
	downloader := NewDownloader(client, WithAPIs(api.APIs{}))

	configurations, err := downloader.Download("project")
	assert.ErrorContains(t, err, "no APIs to download")
	assert.Len(t, configurations, 0)
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

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project")
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

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project")
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
}

func TestDownload_DeprecatedConfigsAreSkipped(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(1)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", DeprecatedBy: "API_ID_2"}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2"}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project")
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
	_, exists := configurations["API_ID_2"]
	assert.True(t, exists)
}

func TestDownloadSpecific_DeprecatedConfigsAreNotSkippedIfRequested(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(1)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", DeprecatedBy: "API_ID_2"}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2"}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project", config.ClassicApiType{Api: "API_ID_1"})
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
	_, exists := configurations["API_ID_1"]
	assert.True(t, exists)
}

func TestDownload_SkipDownloadConfigsAreSkipped(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(1)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", SkipDownload: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2"}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project")
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
	_, exists := configurations["API_ID_2"]
	assert.True(t, exists)
}

func TestDownloadSpecific_SkipDownloadConfigsAreSkippedEvenIfRequested(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, a api.API) ([]dtclient.Value, error) {
		if a.ID == "API_ID_1" {
			return []dtclient.Value{{Id: "API_ID_1", Name: "API_NAME_1"}}, nil
		} else if a.ID == "API_ID_2" {
			return []dtclient.Value{{Id: "API_ID_2", Name: "API_NAME_2"}}, nil
		}
		return nil, nil
	}).Times(1)
	c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Return([]byte("{}"), nil)

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", SkipDownload: true}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2"}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	downloader := NewDownloader(c, WithAPIs(apiMap))

	configurations, err := downloader.Download("project", config.ClassicApiType{Api: "API_ID_1"}, config.ClassicApiType{Api: "API_ID_2"})
	assert.NoError(t, err)
	assert.Len(t, configurations, 1)
	_, exists := configurations["API_ID_2"]
	assert.True(t, exists)
}

func TestDownloadSpecific_ReturnsErrorIfUnknownAPIsAreRequested(t *testing.T) {

	testAPI1 := api.API{ID: "API_ID_1", URLPath: "API_PATH_1", DeprecatedBy: "API_ID_2"}
	testAPI2 := api.API{ID: "API_ID_2", URLPath: "API_PATH_2"}

	apiMap := api.APIs{"API_ID_1": testAPI1, "API_ID_2": testAPI2}

	downloader := NewDownloader(nil, WithAPIs(apiMap))

	_, err := downloader.Download("project", config.ClassicApiType{Api: "API_ID_42"})
	assert.ErrorContains(t, err, "API_ID_42")
	assert.ErrorContains(t, err, "not known")
}

func TestGetApisToDownload(t *testing.T) {
	type given struct {
		apis         api.APIs
		specificAPIs []string
	}
	type expected struct {
		apis []string
	}
	tests := []struct {
		name     string
		given    given
		expected expected
	}{
		{
			name: "filter all specific defined api",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				specificAPIs: []string{"api_1"},
			},
			expected: expected{
				apis: []string{"api_1"},
			},
		}, {
			name: "if deprecated api is defined, do not filter it",
			given: given{
				apis: api.APIs{
					"api_1":          api.API{ID: "api_1"},
					"api_2":          api.API{ID: "api_2"},
					"deprecated_api": api.API{ID: "deprecated_api", DeprecatedBy: "new_api"},
				},
				specificAPIs: []string{"api_1", "deprecated_api"},
			},
			expected: expected{
				apis: []string{"api_1", "deprecated_api"},
			},
		},
		{
			name: "if specific api is not requested, filter deprecated apis",
			given: given{
				apis: api.APIs{
					"api_1":          api.API{ID: "api_1"},
					"api_2":          api.API{ID: "api_2"},
					"deprecated_api": api.API{ID: "deprecated_api", DeprecatedBy: "new_api"},
				},
				specificAPIs: []string{},
			},
			expected: expected{
				apis: []string{"api_1", "api_2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := filterAPIs(tt.given.apis, tt.given.specificAPIs)
			for _, e := range tt.expected.apis {
				assert.Contains(t, actual, e)
			}
		})
	}
}
