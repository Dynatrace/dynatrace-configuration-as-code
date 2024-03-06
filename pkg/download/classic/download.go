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

func Download(client dtclient.Client, projectName string, apisToDownload api.APIs, filters ContentFilters) (projectv2.ConfigsPerType, error) {
	log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
	results := make(projectv2.ConfigsPerType, len(apisToDownload))
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(apisToDownload))

	log.Debug("Fetching configs to download")
	startTime := time.Now()
	for _, currentApi := range apisToDownload {
		go func() {
			defer wg.Done()
			lg := log.WithFields(field.Type(currentApi.ID))
			downloadedConfigs := downloadConfigs(client, currentApi, projectName, filters)
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

func downloadConfigs(client dtclient.Client, api api.API, projectName string, filters ContentFilters) []downloadedConfig {
	var results []downloadedConfig
	logger := log.WithFields(field.Type(api.ID))
	foundValues, err := findConfigsToDownload(client, api, filters)
	if err != nil {
		logger.WithFields(field.Error(err)).Error("Failed to fetch configs of type '%v', skipping download of this type. Reason: %v", api.ID, err)
		return results
	}

	foundValues = filterConfigsToSkip(api, foundValues, filters)
	if len(foundValues) == 0 {
		logger.Debug("No configs of type '%v' to download", api.ID)
		return results
	}

	logger.Debug("Found %d configs of type %q to download", len(foundValues), api.ID)

	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(foundValues))
	for _, v := range foundValues {
		go func() {
			defer wg.Done()

			downloadedJsons, err := downloadAndUnmarshalConfig(client, api, v)
			if err != nil {
				log.WithFields(field.Type(api.ID), field.F("value", v), field.Error(err)).Error("Error fetching config '%v' in api '%v': %v", v.value.Id, api.ID, err)
				return
			}

			for _, downloadedJson := range downloadedJsons {
				if api.TweakResponseFunc != nil {
					api.TweakResponseFunc(downloadedJson)
				}

				c, err := createConfigForDownloadedJson(downloadedJson, api, v, projectName)
				if err != nil {
					log.WithFields(field.Type(api.ID), field.F("value", v), field.Error(err)).Error("Error creating config for %v in api %v: %v", v.value.Id, api.ID, err)
					return
				}

				c1 := downloadedConfig{
					Config: c,
					value:  v.value,
				}

				mutex.Lock()
				results = append(results, c1)
				mutex.Unlock()
			}
		}()
	}
	wg.Wait()
	return results
}

// values represents values that basically hold IDs and values of dynatrace objects
// to be downloaded
type values []value

// value is a wrapper around the clients dtclient.Value and adds additional information to it
type value struct {
	// value holds the id and name of the found dynatrace entity
	value dtclient.Value
	// parentConfigId optionally holds the id of the parent dynatrace entity id.
	// If parentConfigId is empty, means that there is no parent
	parentConfigId string
}

func (v value) ID() string {
	if v.value.Id == v.parentConfigId {
		return v.value.Id
	}
	return v.value.Id + v.parentConfigId
}

// findConfigsToDownload tries to identify all values that should be downloaded from a Dynatrace environment for
// the given API
func findConfigsToDownload(client dtclient.Client, apiToDownload api.API, filters ContentFilters) (values, error) {
	if apiToDownload.SingleConfiguration && !apiToDownload.HasParent() {
		log.WithFields(field.Type(apiToDownload.ID)).Debug("\tFetching singleton-configuration '%v'", apiToDownload.ID)

		// singleton-config. We use the api-id as mock-id
		singletonConfigToDownload := dtclient.Value{Id: apiToDownload.ID, Name: apiToDownload.ID}
		return values{{value: singletonConfigToDownload}}, nil
	}
	log.WithFields(field.Type(apiToDownload.ID)).Debug("\tFetching all '%v' configs", apiToDownload.ID)

	if apiToDownload.HasParent() {
		var res values
		parentAPIValues, err := client.ListConfigs(context.TODO(), *apiToDownload.Parent)
		if err != nil {
			return values{}, err
		}
		for _, parentAPIValue := range parentAPIValues {

			if skipDownload(*apiToDownload.Parent, parentAPIValue, filters) {
				continue
			}

			if apiToDownload.SingleConfiguration {
				vv := dtclient.Value{Id: parentAPIValue.Id, Name: parentAPIValue.Id, Owner: parentAPIValue.Owner}
				res = append(res, value{value: vv, parentConfigId: parentAPIValue.Id})
				continue
			}

			apiValues, err := client.ListConfigs(context.TODO(), apiToDownload.Resolve(parentAPIValue.Id))
			if err != nil {
				return values{}, err
			}
			for _, vv := range apiValues {
				res = append(res, value{value: vv, parentConfigId: parentAPIValue.Id})
			}
		}
		return res, nil
	}

	var res values
	vals, err := client.ListConfigs(context.TODO(), apiToDownload)
	for _, v := range vals {
		res = append(res, value{value: v})
	}
	if err != nil {
		return values{}, err
	}

	return res, nil
}

func filterConfigsToSkip(a api.API, vals values, filters ContentFilters) values {
	var valuesToDownload values

	for _, v := range vals {
		if !skipDownload(a, v.value, filters) {
			valuesToDownload = append(valuesToDownload, v)
		} else {
			log.WithFields(field.Type(a.ID), field.F("value", v)).Debug("Skipping download of config  '%v' of API '%v'", v.value.Id, a.ID)
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

func downloadAndUnmarshalConfig(client dtclient.Client, theApi api.API, value value) ([]map[string]interface{}, error) {
	id := value.value.Id

	// check if we should skip the id to enforce to read/download "all" configs instead of a single one
	if theApi.HasParent() && theApi.ID != api.UserActionAndSessionPropertiesMobile {
		id = ""
	}

	response, err := client.ReadConfigById(theApi.Resolve(value.parentConfigId), id)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal(response, &data)
	if err != nil {
		return nil, err
	}

	if values, ok := data[theApi.PropertyNameOfGetAllResponse]; ok {
		var res []map[string]any
		err := mapstructure.Decode(values, &res)
		return res, err
	}

	return []map[string]any{data}, nil
}

func createConfigForDownloadedJson(mappedJson map[string]interface{}, theApi api.API, value value, projectId string) (config.Config, error) {
	templ, err := createTemplate(mappedJson, value, theApi.ID)
	if err != nil {
		return config.Config{}, err
	}

	params := map[string]parameter.Parameter{}
	params["name"] = &valueParam.ValueParameter{Value: value.value.Name}

	// we use the id (key) of user-action-and-session-properties-mobile as it is its unique identifier
	if theApi.ID == api.UserActionAndSessionPropertiesMobile {
		params["name"] = &valueParam.ValueParameter{Value: value.value.Id}
	}

	if theApi.HasParent() {
		params[config.ScopeParameter] = reference.New(projectId, theApi.Parent.ID, value.parentConfigId, "id")
	}

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

func createTemplate(mappedJson map[string]interface{}, value value, apiId string) (tmpl template.Template, err error) {
	mappedJson = sanitizeProperties(mappedJson, apiId)
	bytes, err := json.MarshalIndent(mappedJson, "", "  ")
	if err != nil {
		return nil, err
	}
	templ := template.NewInMemoryTemplate(value.ID(), string(bytes))
	return templ, nil
}
