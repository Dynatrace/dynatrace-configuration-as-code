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
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	projectv2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/maps"
	"strings"
	"sync"
	"time"
)

type downloadedConfig struct {
	config.Config
	value dtclient.Value
}

type downloaderMap map[string]func(dtclient.Client, api.API, string, ContentFilters) func() []downloadedConfig

var downloaders = downloaderMap{
	"key-user-actions-mobile": downloadKUAMobile, // key user actions for mobile applications
	"key-user-actions-web":    downloadKuaWeb,    // key user actions for web applications
	"any":                     downloadAny,       // generically treated configurations
}

func (df *downloaderMap) Get(client dtclient.Client, api api.API, projectName string, filters ContentFilters) func() []downloadedConfig {
	if fn, ok := downloaders[api.ID]; ok {
		return fn(client, api, projectName, filters)
	}
	return downloaders["any"](client, api, projectName, filters)
}

func Download(client dtclient.Client, projectName string, apisToDownload api.APIs, filters ContentFilters) (projectv2.ConfigsPerType, error) {
	log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
	results := make(projectv2.ConfigsPerType, len(apisToDownload))
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(apisToDownload))

	log.Debug("Fetching configs to download")
	startTime := time.Now()
	for _, currentApi := range apisToDownload {
		currentApi := currentApi // prevent data race

		go func() {
			defer wg.Done()
			lg := log.WithFields(field.Type(currentApi.ID))
			downloadedConfigs := downloaders.Get(client, currentApi, projectName, filters)()
			var configsToPersist []downloadedConfig
			for _, c := range downloadedConfigs {
				content, err := c.Template.Content()
				if err != nil {
					return
				}
				if shouldPersist(currentApi, content, filters) {
					configsToPersist = append(configsToPersist, c)
				} else {
					lg.Debug("\tSkipping persisting config %v (%v) in API %v", c.value.Id, c.value.Name, currentApi.ID)
				}
			}
			if len(configsToPersist) > 0 {
				mutex.Lock()
				results[currentApi.ID] = getConfigsFromCustomConfigs(configsToPersist)
				mutex.Unlock()
			}
		}()
	}
	log.Debug("Started all downloads")
	wg.Wait()

	duration := time.Since(startTime).Truncate(1 * time.Second)
	log.Debug("Finished fetching all configs in %v", duration)

	return results, nil
}

func getConfigsFromCustomConfigs(customConfigs []downloadedConfig) []config.Config {
	var finalConfigs []config.Config
	for _, c := range customConfigs {
		finalConfigs = append(finalConfigs, c.Config)
	}
	return finalConfigs
}

func downloadKua(client dtclient.Client, theApi api.API, projectName string, appType string, _ ContentFilters) func() []downloadedConfig {
	return func() []downloadedConfig {
		var configs []downloadedConfig
		apps, err := client.ListConfigs(context.TODO(), api.NewAPIs()[appType])
		if err != nil {
			return configs
		}
		for _, a := range apps {
			kuas, err := downloadAndUnmarshalConfig(client, theApi.Resolve(a.Id), dtclient.Value{})
			if theApi.TweakResponseFunc != nil {
				theApi.TweakResponseFunc(kuas)
			}

			if err != nil {
				return configs
			}
			var keyUserActions dtclient.KeyUserActionsMobileResponse
			mapstructure.Decode(kuas, &keyUserActions)

			var arr []map[string]any
			mapstructure.Decode(kuas[theApi.PropertyNameOfGetAllResponse], &arr)
			for _, content := range arr {
				value := dtclient.Value{Id: content["name"].(string), Name: content["name"].(string)}
				cfg, err := createConfigForDownloadedJson(content, theApi, value, projectName)
				if err != nil {
					return configs
				}
				cfg.Parameters[config.ScopeParameter] = reference.New(projectName, appType, a.Id, "id")
				configs = append(configs, downloadedConfig{Config: cfg, value: value})

			}
		}
		return configs
	}
}

func downloadKUAMobile(client dtclient.Client, theApi api.API, projectName string, filters ContentFilters) func() []downloadedConfig {
	return downloadKua(client, theApi, projectName, "application-mobile", filters)
}

func downloadKuaWeb(client dtclient.Client, theApi api.API, projectName string, filters ContentFilters) func() []downloadedConfig {
	return downloadKua(client, theApi, projectName, "application-web", filters)
}

func downloadAny(client dtclient.Client, api api.API, projectName string, filters ContentFilters) func() []downloadedConfig {
	return func() []downloadedConfig {
		var results []downloadedConfig
		logger := log.WithFields(field.Type(api.ID))
		values, err := findConfigsToDownload(client, api)
		if err != nil {
			logger.WithFields(field.Error(err)).Error("Failed to fetch configs of type '%v', skipping download of this type. Reason: %v", api.ID, err)
			return results
		}

		values = filterConfigsToSkip(api, values, filters)

		if len(values) == 0 {
			logger.Debug("No configs of type '%v' to download", api.ID)
			return results
		}

		logger.Debug("Found %d configs of type %q to download", len(values), api.ID)

		mutex := sync.Mutex{}
		wg := sync.WaitGroup{}
		wg.Add(len(values))
		for _, value := range values {
			value := value
			go func() {
				defer wg.Done()
				downloadedJson, err := downloadAndUnmarshalConfig(client, api, value)
				if api.TweakResponseFunc != nil {
					api.TweakResponseFunc(downloadedJson)
				}

				if err != nil {
					log.WithFields(field.Type(api.ID), field.F("value", value), field.Error(err)).Error("Error fetching config '%v' in api '%v': %v", value.Id, api.ID, err)
					return
				}
				c, err := createConfigForDownloadedJson(downloadedJson, api, value, projectName)
				if err != nil {
					log.WithFields(field.Type(api.ID), field.F("value", value), field.Error(err)).Error("Error creating config for %v in api %v: %v", value.Id, api.ID, err)
					return
				}

				c1 := downloadedConfig{
					Config: c,
					value:  value,
				}

				mutex.Lock()
				results = append(results, c1)
				mutex.Unlock()

			}()
		}
		wg.Wait()
		return results
	}
}

func findConfigsToDownload(client dtclient.Client, currentApi api.API) ([]dtclient.Value, error) {
	if currentApi.SingleConfiguration {
		log.WithFields(field.Type(currentApi.ID)).Debug("\tFetching singleton-configuration '%v'", currentApi.ID)

		// singleton-config. We use the api-id as mock-id
		singletonConfigToDownload := dtclient.Value{Id: currentApi.ID, Name: currentApi.ID}
		return []dtclient.Value{singletonConfigToDownload}, nil
	}
	log.WithFields(field.Type(currentApi.ID)).Debug("\tFetching all '%v' configs", currentApi.ID)
	return client.ListConfigs(context.TODO(), currentApi)
}

func filterConfigsToSkip(a api.API, value []dtclient.Value, filters ContentFilters) []dtclient.Value {
	valuesToDownload := make([]dtclient.Value, 0, len(value))

	for _, value := range value {
		if !skipDownload(a, value, filters) {
			valuesToDownload = append(valuesToDownload, value)
		} else {
			log.WithFields(field.Type(a.ID), field.F("value", value)).Debug("Skipping download of config  '%v' of API '%v'", value.Id, a.ID)
		}
	}

	return valuesToDownload
}

func shouldPersist(a api.API, jsonStr string, filters ContentFilters) bool {
	if shouldFilter() {
		if cases := filters[a.ID]; cases.ShouldConfigBePersisted != nil {
			var unmarshalledJSON map[string]any
			_ = json.Unmarshal([]byte(jsonStr), &unmarshalledJSON)
			return cases.ShouldConfigBePersisted(unmarshalledJSON)
		}
	}
	return true
}

func skipDownload(a api.API, value dtclient.Value, filters ContentFilters) bool {
	if shouldFilter() {
		if cases := filters[a.ID]; cases.ShouldBeSkippedPreDownload != nil {
			return cases.ShouldBeSkippedPreDownload(value)
		}
	}
	return false
}

func shouldFilter() bool {
	return featureflags.DownloadFilter().Enabled() && featureflags.DownloadFilterClassicConfigs().Enabled()
}

func downloadAndUnmarshalConfig(client dtclient.Client, theApi api.API, value dtclient.Value) (map[string]interface{}, error) {
	response, err := client.ReadConfigById(theApi, value.Id)

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

func createConfigForDownloadedJson(mappedJson map[string]interface{}, theApi api.API, value dtclient.Value, projectId string) (config.Config, error) {
	templ, err := createTemplate(mappedJson, value, theApi.ID)
	if err != nil {
		return config.Config{}, err
	}

	params := map[string]parameter.Parameter{}
	params["name"] = &valueParam.ValueParameter{Value: value.Name}

	coord := coordinate.Coordinate{
		Project:  projectId,
		ConfigId: templ.ID(),
		Type:     theApi.ID,
	}

	return config.Config{
		Type:       config.ClassicApiType{Api: theApi.ID},
		Template:   templ,
		Coordinate: coord,
		Parameters: params,
	}, nil
}

func createTemplate(mappedJson map[string]interface{}, value dtclient.Value, apiId string) (tmpl template.Template, err error) {
	mappedJson = sanitizeProperties(mappedJson, apiId)
	bytes, err := json.MarshalIndent(mappedJson, "", "  ")
	if err != nil {
		return nil, err
	}
	templ := template.NewInMemoryTemplate(value.Id, string(bytes))
	return templ, nil
}
