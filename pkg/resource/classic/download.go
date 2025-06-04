/*
 * @license
 * Copyright 2025 Dynatrace LLC
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
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/maps"

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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type downloadSource interface {
	Get(context.Context, api.API, string) ([]byte, error)
	List(context.Context, api.API) ([]dtclient.Value, error)
}

type DownloadAPI struct {
	configSource   downloadSource
	apisToDownload api.APIs
	filters        ContentFilters
}

func NewDownloadAPI(configSource downloadSource, apisToDownload api.APIs, filters ContentFilters) *DownloadAPI {
	return &DownloadAPI{configSource, apisToDownload, filters}
}

func (a DownloadAPI) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	log.InfoContext(ctx, "Downloading configuration objects")
	log.DebugContext(ctx, "APIs to download: \n - %v", strings.Join(maps.Keys(a.apisToDownload), "\n - "))
	results := make(project.ConfigsPerType, len(a.apisToDownload))
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(a.apisToDownload))

	log.DebugContext(ctx, "Fetching configs to download")
	startTime := time.Now()
	for _, currentApi := range a.apisToDownload {
		go func() {
			defer wg.Done()

			foundValues, err := findConfigsToDownload(ctx, a.configSource, currentApi, a.filters)
			if err != nil {
				log.WithFields(field.Error(err), field.Type(currentApi.ID)).ErrorContext(ctx, "Failed to fetch configs of type '%s', skipping download of this type. Reason: %v", currentApi.ID, err)
				return
			}

			foundValues = checkAndRemoveValuesWithDuplicateIDs(currentApi, foundValues)

			foundValues = filterConfigsToSkip(currentApi, foundValues, a.filters)
			if len(foundValues) == 0 {
				log.WithFields(field.Type(currentApi.ID)).DebugContext(ctx, "No configs of type '%s' to download", currentApi.ID)
				return
			}

			log.WithFields(field.Type(currentApi.ID)).DebugContext(ctx, "Found %d configs of type '%s' to download", len(foundValues), currentApi.ID)
			if configs := downloadConfigs(ctx, a.configSource, currentApi, foundValues, projectName, a.filters); len(configs) > 0 {
				mutex.Lock()
				results[currentApi.ID] = configs
				mutex.Unlock()
			}
		}()
	}
	wg.Wait()
	duration := time.Since(startTime).Truncate(1 * time.Second)
	log.DebugContext(ctx, "Finished fetching all configs in %v", duration)

	return results, nil
}

func checkAndRemoveValuesWithDuplicateIDs(api api.API, originalValues values) values {
	seenIDs := make(map[string]struct{}, len(originalValues))
	filteredValues := make(values, 0, len(originalValues))

	for _, v := range originalValues {
		if _, alreadySeen := seenIDs[v.value.Id]; alreadySeen {
			log.Warn("Received multiple '%s' configs with the same ID '%s'; skipping duplicate named '%s'", api.ID, v.value.Id, v.value.Name)
			continue
		}

		seenIDs[v.value.Id] = struct{}{}
		filteredValues = append(filteredValues, v)
	}
	return filteredValues
}

func downloadConfigs(ctx context.Context, configSource downloadSource, api api.API, configsToDownload values, projectName string, filters ContentFilters) []config.Config {
	var results []config.Config

	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(configsToDownload))
	for _, v := range configsToDownload {
		go func() {
			defer wg.Done()

			dlConfigs, err := download(ctx, configSource, api, v)
			if err != nil {
				log.WithFields(field.Type(api.ID), field.F("value", v), field.Error(err)).WarnContext(ctx, "Error fetching config '%s' in api '%s': %v", v.value.Id, api.ID, err)
				return
			}

			for _, dlConfig := range dlConfigs {
				if api.TweakResponseFunc != nil {
					api.TweakResponseFunc(dlConfig)
				}

				c, err := createConfigObject(dlConfig, api, v, projectName)
				if err != nil {
					log.WithFields(field.Type(api.ID), field.F("value", v), field.Error(err)).WarnContext(ctx, "Error creating config for '%s' in api '%s': %v", v.value.Id, api.ID, err)
					return
				}

				content, err := c.Template.Content()
				if err != nil {
					return
				}

				if !shouldPersist(api, content, filters) {
					log.DebugContext(ctx, "\tSkipping persisting config %v (%v) in API %v", v.value.Id, v.value.Name, api.ID)
					continue
				}

				mutex.Lock()
				results = append(results, c)
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

func (v value) id() string {
	return v.value.Id + v.parentConfigId
}

// findConfigsToDownload tries to identify all values that should be downloaded from a Dynatrace environment for
// the given API
func findConfigsToDownload(ctx context.Context, configSource downloadSource, apiToDownload api.API, filters ContentFilters) (values, error) {
	if apiToDownload.SingleConfiguration && !apiToDownload.HasParent() {
		log.WithFields(field.Type(apiToDownload.ID)).DebugContext(ctx, "\tFetching singleton-configuration '%v'", apiToDownload.ID)

		// singleton-config. We use the api-id as mock-id
		singletonConfigToDownload := dtclient.Value{Id: apiToDownload.ID, Name: apiToDownload.ID}
		return values{{value: singletonConfigToDownload}}, nil
	}
	log.WithFields(field.Type(apiToDownload.ID)).DebugContext(ctx, "\tFetching all '%v' configs", apiToDownload.ID)

	if apiToDownload.HasParent() {
		var res values
		parentAPIValues, err := configSource.List(ctx, *apiToDownload.Parent)
		if err != nil {
			return values{}, err
		}
		for _, parentAPIValue := range parentAPIValues {

			if skipDownload(*apiToDownload.Parent, parentAPIValue, filters) {
				continue
			}

			if apiToDownload.SingleConfiguration {
				vv := dtclient.Value{Id: parentAPIValue.Id, Name: parentAPIValue.Name, Owner: parentAPIValue.Owner}
				res = append(res, value{value: vv, parentConfigId: parentAPIValue.Id})
				continue
			}

			apiValues, err := configSource.List(ctx, apiToDownload.ApplyParentObjectID(parentAPIValue.Id))
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
	vals, err := configSource.List(ctx, apiToDownload)
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
	return featureflags.DownloadFilter.Enabled() && featureflags.DownloadFilterClassicConfigs.Enabled()
}

func download(ctx context.Context, configSource downloadSource, theApi api.API, value value) ([]map[string]any, error) {
	id := value.value.Id

	// check if we should skip the id to enforce to read/download "all" configs instead of a single one
	if theApi.HasParent() && theApi.ID != api.UserActionAndSessionPropertiesMobile {
		id = ""
	}

	response, err := configSource.Get(ctx, theApi.ApplyParentObjectID(value.parentConfigId), id)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal(response, &data)
	if err != nil {
		return nil, err
	}

	vals, found := data[theApi.PropertyNameOfGetAllResponse]
	if !found {
		return []map[string]any{data}, nil
	}

	var res []map[string]any
	err = mapstructure.Decode(vals, &res)
	if err != nil {
		return []map[string]any{}, err
	}

	if theApi.CheckEqualFunc != nil {
		res = slices.DeleteFunc(res, func(m map[string]any) bool {
			var remove bool
			for _, r := range res {
				remove = !theApi.CheckEqualFunc(m, r)
			}
			return remove
		})
	}

	if theApi.PropertyNameOfIdentifier != "" {
		return slices.DeleteFunc(res, func(m map[string]interface{}) bool {
			return m[theApi.PropertyNameOfIdentifier].(string) != value.value.Id
		}), nil
	}

	return res, nil
}

func createConfigObject(mappedJson map[string]interface{}, theApi api.API, value value, projectId string) (config.Config, error) {
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
	templ := template.NewInMemoryTemplate(value.id(), string(bytes))
	return templ, nil
}
