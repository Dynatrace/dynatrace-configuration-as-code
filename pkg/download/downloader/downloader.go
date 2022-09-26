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

package downloader

import (
	"encoding/json"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"sync"
)

// DownloadAllConfigs downloads all specified apis from a given environment.
//
// See package documentation for implementation details.
func DownloadAllConfigs(apisToDownload api.ApiMap, client rest.DynatraceClient, projectName string) project.ConfigsPerApis {

	// apis & mutex to lock it
	apis := make(project.ConfigsPerApis, len(apisToDownload))
	apisMutex := sync.Mutex{}

	// during dev: we can use it to greatly improve the speed like this, but in prod we might spam the dt api with too many concurrent requests.
	// We could use a limiter, e.g. https://pkg.go.dev/github.com/tidwall/limiter?utm_source=godoc#New, or https://github.com/korovkin/limiter
	// We could use a channel mechanism directly https://calmops.com/golang/golang-limit-total-number-of-goroutines/
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(apisToDownload))

	log.Debug("Fetching configs to download")

	for _, currentApi := range apisToDownload {
		currentApi := currentApi // prevent data race

		go func() {

			// download the configs for each api and fill them in the map
			configs := downloadConfigForApi(currentApi, client, projectName)

			if len(configs) > 0 {
				apisMutex.Lock()
				apis[currentApi.GetId()] = configs
				apisMutex.Unlock()
			}
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()

	log.Debug("Finished fetching all configs")
	return apis
}

func downloadConfigForApi(currentApi api.Api, client rest.DynatraceClient, projectName string) []config.Config {
	var configsToDownload []api.Value
	var err error

	apiId := currentApi.GetId()

	if !currentApi.IsSingleConfigurationApi() {
		log.Debug("\tFetching all '%v' configs", apiId)
		configsToDownload, err = client.List(currentApi)

		if err != nil {
			log.Error("\tFailed to fetch configs of type '%v', skipping download of this type. Reason: %v", apiId, err)
			return []config.Config{}
		}
	} else {
		log.Debug("\tFetching singleton-configuration '%v'", apiId)

		// singleton-config. We use the api-id as mock-id
		singletonConfigToDownload := api.Value{Id: currentApi.GetId(), Name: currentApi.GetId()}
		configsToDownload = []api.Value{singletonConfigToDownload}
	}

	// filter all configs we do not want to download. All remaining will be downloaded
	configsToDownload = filterConfigsToSkip(currentApi, configsToDownload)

	if len(configsToDownload) == 0 {
		log.Debug("\tNo configs of type '%v' to download", apiId)
		return []config.Config{}
	}
	configs := createConfigsForApi(currentApi, configsToDownload, client, projectName)

	log.Debug("\tFound %d configs of type '%v' to download", len(configsToDownload), apiId)
	log.Debug("\tFinished downloading all configs of type '%v'", apiId)

	return configs

}

// filterConfigsToSkip filters the configs to download to not needed configs. E.g. dashboards from Dynatrace are presets - we can discard them immediately before downloading
func filterConfigsToSkip(a api.Api, value []api.Value) []api.Value {
	valuesToDownload := make([]api.Value, 0, len(value))

	for _, value := range value {
		if !shouldConfigBeSkipped(a, value) {
			valuesToDownload = append(valuesToDownload, value)
		} else {
			log.Debug("Skipping download of config  '%v' of API '%v'", value.Id, a.GetId())
		}
	}

	return valuesToDownload
}

func createConfigsForApi(theApi api.Api, values []api.Value, client rest.DynatraceClient, projectId string) []config.Config {
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
	templ, err := createTemplate(mappedJson, value)
	if err != nil {
		return config.Config{}, err
	}

	params := map[string]parameter.Parameter{}
	params["name"] = &valueParam.ValueParameter{Value: templ.Name()}

	coord := coordinate.Coordinate{
		Project: projectId,
		Config:  templ.Id(),
		Api:     theApi.GetId(),
	}

	return config.Config{
		Template:   templ,
		Coordinate: coord,
		References: []coordinate.Coordinate{},
		Skip:       false,
		Parameters: params,
	}, nil
}

func createTemplate(mappedJson map[string]interface{}, value api.Value) (tmpl template.Template, err error) {

	mappedJson = sanitizeProperties(mappedJson)
	bytes, err := json.MarshalIndent(mappedJson, "", "  ")
	if err != nil {
		return nil, err
	}

	templ := template.NewDownloadTemplate(value.Id, value.Name, string(bytes))

	return templ, nil
}
