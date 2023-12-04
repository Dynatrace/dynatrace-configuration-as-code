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
			configsToDownload, err := d.findConfigsToDownload(currentApi)
			remoteCount := len(configsToDownload)
			if err != nil {
				log.WithFields(field.Type(currentApi.ID), field.Error(err)).Error("\tFailed to fetch configs of type '%v', skipping download of this type. Reason: %v", currentApi.ID, err)
				return
			}
			// filter all configs we do not want to download. All remaining will be downloaded
			configsToDownload = d.filterConfigsToSkip(currentApi, configsToDownload)

			if len(configsToDownload) == 0 {
				log.WithFields(field.Type(currentApi.ID)).Debug("\tNo configs of type '%v' to download", currentApi.ID)
				return
			}

			log.WithFields(field.Type(currentApi.ID), field.F("configsToDownload", len(configsToDownload))).Debug("\tFound %d configs of type '%v' to download", len(configsToDownload), currentApi.ID)
			cfgs := d.downloadConfigsOfAPI(currentApi, configsToDownload, projectName)

			log.WithFields(field.Type(currentApi.ID), field.F("configsDownloaded", len(cfgs))).Debug("\tFinished downloading all configs of type '%v'", currentApi.ID)
			if len(cfgs) > 0 {
				mutex.Lock()
				results[currentApi.ID] = cfgs
				mutex.Unlock()
			}

			if remoteCount == len(cfgs) {
				log.WithFields(field.Type(currentApi.ID)).Info("Downloaded %d configurations for API %q", len(cfgs), currentApi.ID)
			} else {
				log.WithFields(field.Type(currentApi.ID)).Info("Downloaded %d configurations for API %q. Skipped persisting %d unmodifiable config(s).", len(cfgs), currentApi.ID, remoteCount-len(cfgs))
			}

		}()
	}
	log.Debug("Started all downloads")
	wg.Wait()

	duration := time.Since(startTime).Truncate(1 * time.Second)
	log.Debug("Finished fetching all configs in %v", duration)

	return results
}

func (d *Downloader) downloadConfigsOfAPI(api api.API, values []dtclient.Value, projectName string) []config.Config {
	results := make([]config.Config, 0, len(values))
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(values))

	for _, value := range values {
		value := value
		go func() {
			defer wg.Done()
			downloadedJson, err := d.downloadAndUnmarshalConfig(api, value)
			if err != nil {
				log.WithFields(field.Type(api.ID), field.F("value", value), field.Error(err)).Error("Error fetching config '%v' in api '%v': %v", value.Id, api.ID, err)
				return
			}

			if !d.shouldPersist(api, downloadedJson) {
				log.WithFields(field.Type(api.ID), field.F("value", value)).Debug("\tSkipping persisting config %v (%v) in API %v", value.Id, value.Name, api.ID)
				return
			}

			c, err := d.createConfigForDownloadedJson(downloadedJson, api, value, projectName)
			if err != nil {
				log.WithFields(field.Type(api.ID), field.F("value", value), field.Error(err)).Error("Error creating config for %v in api %v: %v", value.Id, api.ID, err)
				return
			}

			mutex.Lock()
			results = append(results, c)
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
		Skip:       false,
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

func (d *Downloader) shouldPersist(a api.API, json map[string]interface{}) bool {
	if d.filter {
		if cases := d.apiContentFilters[a.ID]; cases.ShouldConfigBePersisted != nil {
			return cases.ShouldConfigBePersisted(json)
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
