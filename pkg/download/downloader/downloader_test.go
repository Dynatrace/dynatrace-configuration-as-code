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

package downloader

import (
	"errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
	"os"
	"testing"
)

func TestGetDownloadLimit(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			"no env supplied",
			"",
			defaultConcurrentDownloads,
		},
		{
			"env invalid",
			"invalid",
			defaultConcurrentDownloads,
		},
		{
			"negative",
			"-1",
			defaultConcurrentDownloads,
		},
		{
			"valid env",
			"1000",
			1000,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := os.Setenv(concurrentRequestsEnvKey, test.envValue)
			assert.NilError(t, err)

			limit := getConcurrentDownloadLimit()
			assert.Equal(t, limit, test.expected)
		})
	}
}

func TestDownloadAllConfigs(t *testing.T) {
	t.Run("empty api map returns nothing and does not call the download function", func(t *testing.T) {
		callback := func(currentApi api.Api, client rest.DynatraceClient, projectName string, _ findConfigsToDownloadFunc, _ filterConfigsToSkipFunc, _ downloadConfigsOfApiFunc) []config.Config {
			t.Error("callback should not have been called")
			return nil
		}

		downloadAllConfigs(api.ApiMap{}, nil, "", callback)
	})

	t.Run("one api is getting downloaded and inserted", func(t *testing.T) {
		a, _ := api.CreateAPIMockWithId(t, "id")
		c := config.Config{}

		callback := func(currentApi api.Api, client rest.DynatraceClient, projectName string, _ findConfigsToDownloadFunc, _ filterConfigsToSkipFunc, _ downloadConfigsOfApiFunc) []config.Config {
			assert.Equal(t, currentApi, a)

			return []config.Config{c}
		}

		configsPerApi := downloadAllConfigs(api.ApiMap{"id": a}, nil, "", callback)
		assert.Equal(t, len(configsPerApi), 1, "should contain one element")
		configs, found := configsPerApi["id"]
		assert.Equal(t, found, true, "api should be present in the result")
		assert.DeepEqual(t, configs, []config.Config{c})
	})

	t.Run("one api without configs is not inserted", func(t *testing.T) {
		a, _ := api.CreateAPIMockFactory(t)

		callback := func(currentApi api.Api, client rest.DynatraceClient, projectName string, _ findConfigsToDownloadFunc, _ filterConfigsToSkipFunc, _ downloadConfigsOfApiFunc) []config.Config {
			assert.Equal(t, currentApi, a)

			return []config.Config{}
		}

		configsPerApi := downloadAllConfigs(api.ApiMap{"id": a}, nil, "", callback)
		assert.Equal(t, len(configsPerApi), 0, "result should be empty")
	})

	t.Run("multiple apis produce the correct result", func(t *testing.T) {
		a1, _ := api.CreateAPIMockWithId(t, "api-1")
		a2, _ := api.CreateAPIMockWithId(t, "api-2")
		a3, _ := api.CreateAPIMockWithId(t, "api-3")

		c1 := config.Config{}
		c2 := config.Config{}

		callback := func(currentApi api.Api, client rest.DynatraceClient, projectName string, _ findConfigsToDownloadFunc, _ filterConfigsToSkipFunc, _ downloadConfigsOfApiFunc) []config.Config {
			switch currentApi.GetId() { // return different results for different apis
			case "api-1":
				return []config.Config{}
			case "api-2":
				return []config.Config{c1}
			case "api-3":
				return []config.Config{c1, c2}
			}

			t.Error("unknown api encountered")

			return nil
		}

		configsPerApi := downloadAllConfigs(api.ApiMap{"api-1": a1, "api-2": a2, "api-3": a3}, nil, "", callback)

		assert.Equal(t, len(configsPerApi), 2, "should contain two elements")

		configs, found := configsPerApi["api-2"]
		assert.Equal(t, found, true, "api should be present in the result")
		assert.DeepEqual(t, configs, []config.Config{c1})

		configs, found = configsPerApi["api-3"]
		assert.Equal(t, found, true, "api should be present in the result")
		assert.DeepEqual(t, configs, []config.Config{c1, c2})
	})
}

func TestDownloadConfigForApi(t *testing.T) {

	t.Run("find configs returns an error and empty configs are returned", func(t *testing.T) {
		a, _ := api.CreateAPIMockWithId(t, "api-1")

		var find findConfigsToDownloadFunc = func(currentApi api.Api, client rest.DynatraceClient) ([]api.Value, error) {
			return []api.Value{}, errors.New("some-reason")
		}

		var filter filterConfigsToSkipFunc = func(a api.Api, values []api.Value) []api.Value {
			t.Error("filter should never be called")
			return nil
		}

		var download downloadConfigsOfApiFunc = func(a api.Api, values []api.Value, client rest.DynatraceClient, s string) []config.Config {
			t.Error("download should never be called")
			return nil
		}

		configs := downloadConfigForApi(a, nil, "", find, filter, download)
		assert.Equal(t, len(configs), 0, "empty array should be returned")
	})

	t.Run("if filter filters nothing, an empty array is returned", func(t *testing.T) {
		a, _ := api.CreateAPIMockWithId(t, "api-1")

		vals := []api.Value{{}, {}}

		var filterCalled bool // check that filter actually has been called

		var find findConfigsToDownloadFunc = func(currentApi api.Api, client rest.DynatraceClient) ([]api.Value, error) {
			return vals, nil
		}

		var filter filterConfigsToSkipFunc = func(a api.Api, values []api.Value) []api.Value {
			filterCalled = true
			assert.DeepEqual(t, values, vals)
			return []api.Value{}
		}

		var download downloadConfigsOfApiFunc = func(a api.Api, values []api.Value, client rest.DynatraceClient, s string) []config.Config {
			return nil
		}

		configs := downloadConfigForApi(a, nil, "", find, filter, download)
		assert.Equal(t, filterCalled, true, "filter function has not been called")
		assert.Equal(t, len(configs), 0, "configs should be empty")
	})

	t.Run("download is called with the correct values", func(t *testing.T) {
		a, _ := api.CreateAPIMockWithId(t, "api-1")

		vals := []api.Value{{}, {}, {}}
		confs := []config.Config{{}, {}, {}}

		var find findConfigsToDownloadFunc = func(currentApi api.Api, client rest.DynatraceClient) ([]api.Value, error) {
			return vals, nil
		}

		var filter filterConfigsToSkipFunc = func(a api.Api, values []api.Value) []api.Value {
			return vals
		}

		var download downloadConfigsOfApiFunc = func(a api.Api, values []api.Value, client rest.DynatraceClient, s string) []config.Config {
			assert.DeepEqual(t, values, vals)
			return confs
		}

		configs := downloadConfigForApi(a, nil, "", find, filter, download)
		assert.DeepEqual(t, configs, confs)
	})

	t.Run("project name and client is forwarded correctly", func(t *testing.T) {
		a, _ := api.CreateAPIMockWithId(t, "api-1")
		c := rest.CreateDynatraceClientMockFactory(t)

		var downloadHasBeenCalled bool

		var find findConfigsToDownloadFunc = func(currentApi api.Api, client rest.DynatraceClient) ([]api.Value, error) {
			return nil, nil
		}

		var filter filterConfigsToSkipFunc = func(a api.Api, values []api.Value) []api.Value {
			return []api.Value{{}}
		}

		var download downloadConfigsOfApiFunc = func(a api.Api, values []api.Value, client rest.DynatraceClient, s string) []config.Config {
			assert.Equal(t, a.GetId(), "api-1")
			assert.Equal(t, s, "project-name")
			assert.Equal(t, client, c)

			downloadHasBeenCalled = true

			return nil
		}

		downloadConfigForApi(a, c, "project-name", find, filter, download)
		assert.Equal(t, downloadHasBeenCalled, true, "download has not been called")
	})
}

func TestFindConfigsToDownload(t *testing.T) {
	t.Run("singleton-apis return the config without invoking the client", func(t *testing.T) {
		c := rest.NewMockDynatraceClient(gomock.NewController(t))

		a := api.NewMockApi(gomock.NewController(t))
		a.EXPECT().IsSingleConfigurationApi().Return(true)
		a.EXPECT().GetId().AnyTimes().Return("api-id")

		download, err := findConfigsToDownload(a, c)

		assert.NilError(t, err)
		assert.DeepEqual(t, download, []api.Value{{Id: "api-id", Name: "api-id"}})
	})

	t.Run("non-singletons fetch values from the client and return them", func(t *testing.T) {
		vals := []api.Value{{}}

		a := api.NewMockApi(gomock.NewController(t))
		a.EXPECT().IsSingleConfigurationApi().Return(false)
		a.EXPECT().GetId().AnyTimes().Return("api-id")

		c := rest.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().List(a).Return(vals, nil)

		download, err := findConfigsToDownload(a, c)
		assert.NilError(t, err)
		assert.DeepEqual(t, download, vals)
	})

	t.Run("non-singletons fetch errors are returned", func(t *testing.T) {
		a := api.NewMockApi(gomock.NewController(t))
		a.EXPECT().IsSingleConfigurationApi().Return(false)
		a.EXPECT().GetId().AnyTimes().Return("api-id")

		c := rest.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().List(a).Return(nil, errors.New("error"))

		_, err := findConfigsToDownload(a, c)
		assert.Error(t, err, "error")
	})
}
