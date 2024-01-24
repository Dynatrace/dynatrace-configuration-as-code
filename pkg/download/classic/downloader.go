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

package classic

import (
	"context"
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"golang.org/x/exp/maps"
	"strings"
	"sync"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type (
	// Downloader is responsible for downloading classic Dynatrace APIs. To create it sound, use NewDownloader construction function.
	Downloader struct {
		apisToDownload api.APIs

		filter bool

		// apiContentFilters contains rules to filter specific apis based on
		// custom logic implemented in the ContentFilter
		apiContentFilters map[string]ContentFilter

		// client is the actual rest client used to call
		// the dynatrace APIs
		client dtclient.Client
	}

	Option func(downloader *Downloader)

	downloadedConfig struct {
		config.Config
		value dtclient.Value
	}
)

// NewDownloader creates a new sound Downloader.
func NewDownloader(client dtclient.Client, opts ...Option) *Downloader {
	c := &Downloader{
		apisToDownload:    api.NewAPIs(),
		filter:            true,
		apiContentFilters: apiContentFilters,
		client:            client,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// WithAPIs sets the endpoints from which Downloader is going to download. During settings, it checks does the given endpoints are known.
func WithAPIs(apis api.APIs) Option {
	return func(d *Downloader) {
		d.apisToDownload = apis
	}
}

func WithAPIContentFilters(apiFilters map[string]ContentFilter) Option {
	return func(d *Downloader) {
		d.apiContentFilters = apiFilters
	}
}

func WithFiltering(b bool) Option {
	return func(d *Downloader) {
		d.filter = b
	}
}

func (d *Downloader) Download(projectName string, _ ...config.ClassicApiType) (project.ConfigsPerType, error) {
	log.Info("Downloading configuration APIs from %d endpoints", len(d.apisToDownload))
	configs := d.downloadAPIs(d.apisToDownload, projectName)
	return configs, nil
}

func (d *Downloader) downloadAPIs(apisToDownload api.APIs, projectName string) project.ConfigsPerType {
	log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
	results := make(project.ConfigsPerType, len(apisToDownload))
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

			var downloadedConfigs []downloadedConfig
			if currentApi.ID == "key-user-actions-mobile" {
				downloadedConfigs = d.downloadKeyUserActions(projectName)
			} else {
				downloadedConfigs = d.downloadConfigsOfAPI(currentApi, projectName)
			}

			var configsToPersist []downloadedConfig
			for _, c := range downloadedConfigs {
				content, err := c.Template.Content()
				if err != nil {
					return
				}
				if d.shouldPersist(currentApi, content) {
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

	return results
}

func getConfigsFromCustomConfigs(customConfigs []downloadedConfig) []config.Config {
	var finalConfigs []config.Config
	for _, c := range customConfigs {
		finalConfigs = append(finalConfigs, c.Config)
	}
	return finalConfigs
}

func (d *Downloader) downloadConfigsOfAPI(api api.API, projectName string) []downloadedConfig {
	var results []downloadedConfig
	logger := log.WithFields(field.Type(api.ID))
	values, err := d.findConfigsToDownload(api)
	if err != nil {
		logger.WithFields(field.Error(err)).Error("Failed to fetch configs of type '%v', skipping download of this type. Reason: %v", api.ID, err)
		return results
	}

	values = d.filterConfigsToSkip(api, values)

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
			downloadedJson, err := d.downloadAndUnmarshalConfig(api, value)
			if api.TweakResponseFunc != nil {
				api.TweakResponseFunc(downloadedJson)
			}

			if err != nil {
				log.WithFields(field.Type(api.ID), field.F("value", value), field.Error(err)).Error("Error fetching config '%v' in api '%v': %v", value.Id, api.ID, err)
				return
			}
			c, err := d.createConfigForDownloadedJson(downloadedJson, api, value, projectName)
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

func (d *Downloader) downloadAndUnmarshalConfig(theApi api.API, value dtclient.Value) (map[string]interface{}, error) {
	response, err := d.client.ReadConfigById(theApi, value.Id)

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

func (d *Downloader) createConfigForDownloadedJson(mappedJson map[string]interface{}, theApi api.API, value dtclient.Value, projectId string) (config.Config, error) {
	templ, err := d.createTemplate(mappedJson, value, theApi.ID)
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

func (d *Downloader) createTemplate(mappedJson map[string]interface{}, value dtclient.Value, apiId string) (tmpl template.Template, err error) {
	mappedJson = sanitizeProperties(mappedJson, apiId)
	bytes, err := json.MarshalIndent(mappedJson, "", "  ")
	if err != nil {
		return nil, err
	}
	templ := template.NewInMemoryTemplate(value.Id, string(bytes))
	return templ, nil
}

func (d *Downloader) findConfigsToDownload(currentApi api.API) ([]dtclient.Value, error) {
	if currentApi.SingleConfiguration {
		log.WithFields(field.Type(currentApi.ID)).Debug("\tFetching singleton-configuration '%v'", currentApi.ID)

		// singleton-config. We use the api-id as mock-id
		singletonConfigToDownload := dtclient.Value{Id: currentApi.ID, Name: currentApi.ID}
		return []dtclient.Value{singletonConfigToDownload}, nil
	}
	log.WithFields(field.Type(currentApi.ID)).Debug("\tFetching all '%v' configs", currentApi.ID)
	return d.client.ListConfigs(context.TODO(), currentApi)
}

func (d *Downloader) shouldPersist(a api.API, jsonStr string) bool {
	if d.filter {
		if cases := d.apiContentFilters[a.ID]; cases.ShouldConfigBePersisted != nil {
			var unmarshalledJSON map[string]any
			_ = json.Unmarshal([]byte(jsonStr), &unmarshalledJSON)
			return cases.ShouldConfigBePersisted(unmarshalledJSON)
		}
	}
	return true
}

func (d *Downloader) skipDownload(a api.API, value dtclient.Value) bool {
	if d.filter {
		if cases := d.apiContentFilters[a.ID]; cases.ShouldBeSkippedPreDownload != nil {
			return cases.ShouldBeSkippedPreDownload(value)
		}
	}
	return false
}

func (d *Downloader) filterConfigsToSkip(a api.API, value []dtclient.Value) []dtclient.Value {
	valuesToDownload := make([]dtclient.Value, 0, len(value))

	for _, value := range value {
		if !d.skipDownload(a, value) {
			valuesToDownload = append(valuesToDownload, value)
		} else {
			log.WithFields(field.Type(a.ID), field.F("value", value)).Debug("Skipping download of config  '%v' of API '%v'", value.Id, a.ID)
		}
	}

	return valuesToDownload
}
