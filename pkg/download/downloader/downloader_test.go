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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
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
