/**
 * @license
 * Copyright 2022 Dynatrace LLC
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

package downloader

import (
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

const defaultConcurrentDownloads = 50
const concurrentRequestsEnvKey = "CONCURRENT_REQUESTS"

// DownloadAllConfigs downloads all specified apis from a given environment.
//
// See package documentation for implementation details.
func DownloadAllConfigs(apisToDownload api.ApiMap, client rest.DynatraceClient, projectName string) project.ConfigsPerType {
	return downloadAllConfigs(apisToDownload, client, projectName, downloadConfigForApi)
}

type downloadConfigForApiFunc func(api.Api, rest.DynatraceClient, string, findConfigsToDownloadFunc, filterConfigsToSkipFunc, downloadConfigsOfApiFunc) []config.Config

func downloadAllConfigs(
	apisToDownload api.ApiMap,
	client rest.DynatraceClient,
	projectName string,
	downloadConfigForApi downloadConfigForApiFunc,
) project.ConfigsPerType {

	// apis & mutex to lock it
	apis := make(project.ConfigsPerType, len(apisToDownload))
	apisMutex := sync.Mutex{}

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(apisToDownload))

	client = rest.LimitClientParallelRequests(client, getConcurrentDownloadLimit())

	log.Debug("Fetching configs to download")

	startTime := time.Now()

	for _, currentApi := range apisToDownload {
		currentApi := currentApi // prevent data race

		go func() {

			// download the configs for each api and fill them in the map
			configs := downloadConfigForApi(currentApi, client, projectName, findConfigsToDownload, filterConfigsToSkip, downloadConfigsOfApi)

			if len(configs) > 0 {
				apisMutex.Lock()
				apis[currentApi.GetId()] = configs
				apisMutex.Unlock()
			}

			waitGroup.Done()
		}()
	}

	log.Debug("Started all downloads")
	waitGroup.Wait()

	duration := time.Now().Sub(startTime).Truncate(1 * time.Second)
	log.Debug("Finished fetching all configs in %v", duration)

	return apis
}

func getConcurrentDownloadLimit() int {
	concurrentRequests, err := strconv.Atoi(os.Getenv(concurrentRequestsEnvKey))
	if err != nil || concurrentRequests < 0 {
		return defaultConcurrentDownloads
	}

	return concurrentRequests
}

type (
	findConfigsToDownloadFunc func(currentApi api.Api, client rest.DynatraceClient) ([]api.Value, error)
	filterConfigsToSkipFunc   func(api.Api, []api.Value) []api.Value
	downloadConfigsOfApiFunc  func(api.Api, []api.Value, rest.DynatraceClient, string) []config.Config
)

func downloadConfigForApi(
	currentApi api.Api,
	client rest.DynatraceClient,
	projectName string,
	findConfigsToDownload findConfigsToDownloadFunc,
	filterConfigsToSkip filterConfigsToSkipFunc,
	downloadConfigsOfApi downloadConfigsOfApiFunc,
) []config.Config {

	configsToDownload, err := findConfigsToDownload(currentApi, client)
	if err != nil {
		log.Error("\tFailed to fetch configs of type '%v', skipping download of this type. Reason: %v", currentApi.GetId(), err)
		return []config.Config{}
	}

	// filter all configs we do not want to download. All remaining will be downloaded
	configsToDownload = filterConfigsToSkip(currentApi, configsToDownload)

	if len(configsToDownload) == 0 {
		log.Debug("\tNo configs of type '%v' to download", currentApi.GetId())
		return []config.Config{}
	}

	log.Debug("\tFound %d configs of type '%v' to download", len(configsToDownload), currentApi.GetId())
	configs := downloadConfigsOfApi(currentApi, configsToDownload, client, projectName)

	log.Debug("\tFinished downloading all configs of type '%v'", currentApi.GetId())

	return configs

}

func findConfigsToDownload(currentApi api.Api, client rest.DynatraceClient) ([]api.Value, error) {

	if currentApi.IsSingleConfigurationApi() {
		log.Debug("\tFetching singleton-configuration '%v'", currentApi.GetId())

		// singleton-config. We use the api-id as mock-id
		singletonConfigToDownload := api.Value{Id: currentApi.GetId(), Name: currentApi.GetId()}
		return []api.Value{singletonConfigToDownload}, nil
	}

	log.Debug("\tFetching all '%v' configs", currentApi.GetId())
	return client.List(currentApi)
}

func downloadConfigsOfApi(theApi api.Api, values []api.Value, client rest.DynatraceClient, projectId string) []config.Config {
	configs := make([]config.Config, 0, len(values))
	configsMutex := sync.Mutex{}

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(values))

	for _, value := range values {

		value := value

		go func() {
			conf, skipConfig := downloadConfig(theApi, value, client, projectId)

			if !skipConfig {
				configsMutex.Lock()
				configs = append(configs, conf)
				configsMutex.Unlock()
			}

			waitGroup.Done()
		}()
	}

	waitGroup.Wait()

	return configs
}

func downloadConfig(theApi api.Api, value api.Value, client rest.DynatraceClient, projectId string) (conf config.Config, skipConfig bool) {
	return downloadConfigForTesting(theApi, value, client, projectId, shouldConfigBePersisted)
}

type shouldConfigBePersistedFunc func(a api.Api, json map[string]interface{}) bool

func downloadConfigForTesting(
	theApi api.Api,
	value api.Value,
	client rest.DynatraceClient,
	projectId string,
	shouldConfigBePersisted shouldConfigBePersistedFunc,
) (conf config.Config, skipConfig bool) {
	// download json and check if we should skip it
	downloadedJson, err := downloadAndUnmarshalConfig(theApi, value, client)
	if err != nil {
		log.Error("Error fetching config '%v' in api '%v': %v", value.Id, theApi.GetId(), err)
		return config.Config{}, true
	}

	if !shouldConfigBePersisted(theApi, downloadedJson) {
		log.Debug("\tSkipping persisting config %v (%v) in API %v", value.Id, value.Name, theApi.GetId())
		return config.Config{}, true
	}

	c, err := createConfigForDownloadedJson(downloadedJson, theApi, value, projectId)

	if err != nil {
		log.Error("Error creating config for %v in api %v: %v", value.Id, theApi.GetId(), err)
		return config.Config{}, true
	}

	return c, false
}

func downloadAndUnmarshalConfig(theApi api.Api, value api.Value, client rest.DynatraceClient) (map[string]interface{}, error) {
	response, err := client.ReadById(theApi, value.Id)

	if err != nil {
		return nil, err
	}

	var data map[string]interface{}

	err = json.Unmarshal(response, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func createConfigForDownloadedJson(mappedJson map[string]interface{}, theApi api.Api, value api.Value, projectId string) (config.Config, error) {
	templ, err := createTemplate(mappedJson, value, theApi.GetId())
	if err != nil {
		return config.Config{}, err
	}

	params := map[string]parameter.Parameter{}
	params["name"] = &valueParam.ValueParameter{Value: templ.Name()}

	coord := coordinate.Coordinate{
		Project:  projectId,
		ConfigId: templ.Id(),
		Type:     theApi.GetId(),
	}

	return config.Config{
		Template:   templ,
		Coordinate: coord,
		References: []coordinate.Coordinate{},
		Skip:       false,
		Parameters: params,
	}, nil
}

func createTemplate(mappedJson map[string]interface{}, value api.Value, apiId string) (tmpl template.Template, err error) {

	mappedJson = sanitizeProperties(mappedJson, apiId)
	bytes, err := json.MarshalIndent(mappedJson, "", "  ")
	if err != nil {
		return nil, err
	}

	templ := template.NewDownloadTemplate(value.Id, value.Name, string(bytes))

	return templ, nil
}
