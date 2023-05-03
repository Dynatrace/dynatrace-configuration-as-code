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
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	"golang.org/x/exp/maps"
	"strings"
	"sync"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

// Downloader is responsible for downloading classic Dynatrace APIs
type Downloader struct {
	apisToDownload api.APIs
	// apiFilters contains logic to filter specific apis based on
	// custom logic implemented in the apiFilter
	apiFilters map[string]apiFilter

	// client is the actual rest client used to call
	// the dynatrace APIs
	client dtclient.Client
}

// WithAPIFilters sets the api filters for the Downloader
func WithAPIFilters(apiFilters map[string]apiFilter) func(*Downloader) {
	return func(d *Downloader) {
		d.apiFilters = apiFilters
	}
}

func WithAPIs(apis api.APIs) func(*Downloader) {
	return func(d *Downloader) {
		d.apisToDownload = apis
	}
}

// NewDownloader creates a new Downloader
func NewDownloader(client dtclient.Client, opts ...func(*Downloader)) *Downloader {
	c := &Downloader{
		apisToDownload: api.NewAPIs(),
		apiFilters:     apiFilters,
		client:         client,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (d *Downloader) Download(projectName string, specificAPIs ...config.ClassicApiType) (project.ConfigsPerType, error) {
	if len(specificAPIs) == 0 {
		return d.downloadAll(projectName)
	}
	var apis []string
	for _, s := range specificAPIs {
		apis = append(apis, s.Api)
	}
	return d.downloadSpecific(projectName, apis)
}

func (d *Downloader) downloadAll(projectName string) (project.ConfigsPerType, error) {
	configs := d.downloadAPIs(d.apisToDownload, projectName)
	return configs, nil
}

func (d *Downloader) downloadSpecific(projectName string, specificAPIs []string) (project.ConfigsPerType, error) {
	if ok, unknownApis := validateSpecificAPIs(d.apisToDownload, specificAPIs); !ok {
		err := fmt.Errorf("requested APIs '%v' are not known", strings.Join(unknownApis, ","))
		log.Error("%v. Please consult our documentation for known API names.", err)
		return nil, err
	}

	apisToDownload := getApisToDownload(d.apisToDownload, specificAPIs)
	if len(apisToDownload) == 0 {
		return nil, fmt.Errorf("no APIs to download")
	}

	configs := d.downloadAPIs(apisToDownload, projectName)
	return configs, nil
}

func validateSpecificAPIs(a api.APIs, apiNames []string) (valid bool, unknownAPIs []string) {
	for _, v := range apiNames {
		if !a.Contains(v) {
			unknownAPIs = append(unknownAPIs, v)
		}
	}
	return len(unknownAPIs) == 0, unknownAPIs
}

func getApisToDownload(apis api.APIs, specificAPIs []string) api.APIs {
	if len(specificAPIs) > 0 {
		return apis.Filter(api.RetainByName(specificAPIs), skipDownloadFilter)
	}
	return apis.Filter(skipDownloadFilter, removeDeprecatedEndpoints)
}

func skipDownloadFilter(api api.API) bool {
	if api.SkipDownload {
		log.Info("API can not be downloaded and needs manual creation: '%v'.", api.ID)
		return true
	}
	return false
}

func removeDeprecatedEndpoints(api api.API) bool {
	if api.DeprecatedBy != "" {
		log.Warn("API %q is deprecated by %q and will not be downloaded", api.ID, api.DeprecatedBy)
		return true
	}
	return false
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
			if err != nil {
				log.Error("\tFailed to fetch configs of type '%v', skipping download of this type. Reason: %v", currentApi.ID, err)
				return
			}
			// filter all configs we do not want to download. All remaining will be downloaded
			configsToDownload = d.filterConfigsToSkip(currentApi, configsToDownload)

			if len(configsToDownload) == 0 {
				log.Debug("\tNo configs of type '%v' to download", currentApi.ID)
				return
			}

			log.Debug("\tFound %d configs of type '%v' to download", len(configsToDownload), currentApi.ID)
			cfgs := d.downloadConfigsOfAPI(currentApi, configsToDownload, projectName)

			log.Debug("\tFinished downloading all configs of type '%v'", currentApi.ID)
			if len(cfgs) > 0 {
				mutex.Lock()
				results[currentApi.ID] = cfgs
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
				log.Error("Error fetching config '%v' in api '%v': %v", value.Id, api.ID, err)
				return
			}

			if !d.skipPersist(api, downloadedJson) {
				log.Debug("\tSkipping persisting config %v (%v) in API %v", value.Id, value.Name, api.ID)
				return
			}

			c, err := d.createConfigForDownloadedJson(downloadedJson, api, value, projectName)
			if err != nil {
				log.Error("Error creating config for %v in api %v: %v", value.Id, api.ID, err)
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
	params["name"] = &valueParam.ValueParameter{Value: templ.Name()}

	coord := coordinate.Coordinate{
		Project:  projectId,
		ConfigId: templ.Id(),
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
	templ := template.NewDownloadTemplate(value.Id, value.Name, string(bytes))
	return templ, nil
}

func (d *Downloader) findConfigsToDownload(currentApi api.API) ([]dtclient.Value, error) {
	if currentApi.SingleConfiguration {
		log.Debug("\tFetching singleton-configuration '%v'", currentApi.ID)

		// singleton-config. We use the api-id as mock-id
		singletonConfigToDownload := dtclient.Value{Id: currentApi.ID, Name: currentApi.ID}
		return []dtclient.Value{singletonConfigToDownload}, nil
	}
	log.Debug("\tFetching all '%v' configs", currentApi.ID)
	return d.client.ListConfigs(currentApi)
}

func (d *Downloader) skipPersist(a api.API, json map[string]interface{}) bool {
	if cases := d.apiFilters[a.ID]; cases.shouldConfigBePersisted != nil {
		return cases.shouldConfigBePersisted(json)
	}
	return true
}
func (d *Downloader) skipDownload(a api.API, value dtclient.Value) bool {
	if cases := d.apiFilters[a.ID]; cases.shouldBeSkippedPreDownload != nil {
		return cases.shouldBeSkippedPreDownload(value)
	}

	return false
}

func (d *Downloader) filterConfigsToSkip(a api.API, value []dtclient.Value) []dtclient.Value {
	valuesToDownload := make([]dtclient.Value, 0, len(value))

	for _, value := range value {
		if !d.skipDownload(a, value) {
			valuesToDownload = append(valuesToDownload, value)
		} else {
			log.Debug("Skipping download of config  '%v' of API '%v'", value.Id, a.ID)
		}
	}

	return valuesToDownload
}
