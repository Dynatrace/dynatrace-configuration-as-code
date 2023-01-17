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

package settings

import (
	"encoding/json"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	v2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"sync"
)

// Downloader is responsible for downloading Settings 2.0 objects
type Downloader struct {
	client rest.SettingsClient
}

// NewSettingsDownloader creates a new downloader for Settings 2.0 objects
func NewSettingsDownloader(client rest.SettingsClient) *Downloader {
	return &Downloader{
		client: client,
	}
}

// Download downloads all settings 2.0 objects for the given schema IDs

func Download(client rest.SettingsClient, schemaIDs []string, projectName string) v2.ConfigsPerType {
	return NewSettingsDownloader(client).Download(schemaIDs, projectName)
}

// DownloadAll downloads all settings 2.0 objects for a given project
func DownloadAll(client rest.SettingsClient, projectName string) v2.ConfigsPerType {
	return NewSettingsDownloader(client).DownloadAll(projectName)
}

// Download downloads all settings 2.0 objects for the given schema IDs and a given project
// The returned value is a map of settings 2.0 objects with the schema ID as keys
func (d *Downloader) Download(schemaIDs []string, projectName string) v2.ConfigsPerType {
	return d.download(schemaIDs, projectName)
}

// DownloadAll downloads all settings 2.0 objects for a given project.
// The returned value is a map of settings 2.0 objects with the schema ID as keys
func (d *Downloader) DownloadAll(projectName string) v2.ConfigsPerType {
	log.Debug("Fetching all schemas to download")

	// get ALL schemas
	schemas, err := d.client.ListSchemas()
	if err != nil {
		log.Error("Failed to fetch all known schemas. Skipping settings download. Reason: %s", err)
		return nil
	}
	// convert to list of IDs
	var ids []string
	for _, i := range schemas {
		ids = append(ids, i.SchemaId)
	}

	return d.download(ids, projectName)
}

func (d *Downloader) download(schemas []string, projectName string) v2.ConfigsPerType {
	results := make(v2.ConfigsPerType, len(schemas))
	downloadMutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(schemas))
	for _, schema := range schemas {
		go func(s string) {
			defer wg.Done()
			log.Debug("Downloading all settings for schema %s", s)
			objects, err := d.client.ListSettings(s, rest.ListSettingsOptions{})
			if err != nil {
				log.Error("Failed to fetch all settings for schema %s: %v", s, err)
				return
			}
			if len(objects) == 0 {
				return
			}
			configs := d.convertAllObjects(objects, projectName)
			downloadMutex.Lock()
			results[s] = configs
			downloadMutex.Unlock()
		}(schema)
	}
	wg.Wait()

	return results
}

func (d *Downloader) convertAllObjects(objects []rest.DownloadSettingsObject, projectName string) []config.Config {
	result := make([]config.Config, 0, len(objects))

	for _, o := range objects {
		result = append(result, d.convertObject(o, projectName))
	}

	return result
}

func (d *Downloader) convertObject(o rest.DownloadSettingsObject, projectName string) config.Config {

	var content string
	if bytes, err := json.MarshalIndent(o.Value, "", "  "); err == nil {
		content = string(bytes)
	} else {
		log.Warn("Failed to indent settings template. Reason: %s", err)
		content = string(o.Value)
	}

	configId := util.GenerateUuidFromName(o.ObjectId)

	templ := template.NewDownloadTemplate(configId, configId, content)

	return config.Config{
		Template: templ,
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     o.SchemaId,
			ConfigId: configId,
		},
		Type: config.Type{
			SchemaId:      o.SchemaId,
			SchemaVersion: o.SchemaVersion,
		},
		Parameters: map[string]parameter.Parameter{
			config.NameParameter:  &value.ValueParameter{Value: configId},
			config.ScopeParameter: &value.ValueParameter{Value: o.Scope},
		},
		Skip: false,
	}

}
