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
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/errors"
	"strings"
	"sync"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

// Downloader is responsible for downloading Settings 2.0 objects
type Downloader struct {
	// client is the settings 2.0 client to be used by the Downloader
	client dtclient.SettingsClient

	// filters specifies which settings 2.0 objects need special treatment under
	// certain conditions and need to be skipped
	filters Filters
}

// WithFilters sets specific settings filters for settings 2.0 object that needs to be filtered following
// to some custom criteria
func WithFilters(filters Filters) func(*Downloader) {
	return func(d *Downloader) {
		d.filters = filters
	}
}

// NewDownloader creates a new downloader for Settings 2.0 objects
func NewDownloader(client dtclient.SettingsClient, opts ...func(*Downloader)) *Downloader {
	d := &Downloader{
		client:  client,
		filters: defaultSettingsFilters,
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

func (d *Downloader) Download(projectName string, schemaIDs ...config.SettingsType) (v2.ConfigsPerType, error) {
	if len(schemaIDs) == 0 {
		return d.downloadAll(projectName)
	}
	var schemas []string
	for _, s := range schemaIDs {
		schemas = append(schemas, s.SchemaId)
	}
	return d.downloadSpecific(projectName, schemas)
}

func (d *Downloader) downloadAll(projectName string) (v2.ConfigsPerType, error) {
	log.Debug("Fetching all schemas to download")

	// get ALL schemas
	schemas, err := d.client.ListSchemas()
	if err != nil {
		log.Error("Failed to fetch all known schemas. Skipping settings download. Reason: %s", err)
		return nil, err
	}
	// convert to list of IDs
	var ids []string
	for _, i := range schemas {
		ids = append(ids, i.SchemaId)
	}

	result := d.download(ids, projectName)
	return result, nil
}

func (d *Downloader) downloadSpecific(projectName string, schemaIDs []string) (v2.ConfigsPerType, error) {
	if ok, unknownSchemas := validateSpecificSchemas(d.client, schemaIDs); !ok {
		err := fmt.Errorf("requested settings-schema(s) '%v' are not known", strings.Join(unknownSchemas, ","))
		log.Error("%v. Please consult the documentation for available schemas and verify they are available in your environment.", err)
		return nil, err
	}
	log.Debug("Settings to download: \n - %v", strings.Join(schemaIDs, "\n - "))
	result := d.download(schemaIDs, projectName)
	return result, nil
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
			objects, err := d.client.ListSettings(s, dtclient.ListSettingsOptions{})
			if err != nil {
				var errMsg string
				var respErr clientErrors.RespError
				if errors.As(err, &respErr) {
					errMsg = respErr.ConcurrentError()
				} else {
					errMsg = err.Error()
				}
				log.Error("Failed to fetch all settings for schema %s: %v", s, errMsg)
				return
			}
			log.Info("Downloaded %d settings for schema %s", len(objects), s)
			if len(objects) == 0 {
				return
			}

			cfgs := d.convertAllObjects(objects, projectName)
			downloadMutex.Lock()
			results[s] = cfgs
			downloadMutex.Unlock()

			log.Debug("Finished downloading all (%d) settings for schema %s", len(objects), s)
		}(schema)
	}
	wg.Wait()

	return results
}

func (d *Downloader) convertAllObjects(objects []dtclient.DownloadSettingsObject, projectName string) []config.Config {
	result := make([]config.Config, 0, len(objects))
	for _, o := range objects {

		// try to unmarshall settings value
		var contentUnmarshalled map[string]interface{}
		if err := json.Unmarshal(o.Value, &contentUnmarshalled); err != nil {
			log.Error("Unable to unmarshal JSON value of settings 2.0 object: %v", err)
			return result
		}
		// skip discarded settings objects
		if shouldDiscard, reason := d.filters.Get(o.SchemaId).ShouldDiscard(contentUnmarshalled); shouldDiscard {
			log.Warn("Downloaded setting of schema %q will be discarded. Reason: %s", o.SchemaId, reason)
			continue
		}

		indentedJson := jsonutils.MarshalIndent(o.Value)
		// construct config object with generated config ID
		configId := idutils.GenerateUUIDFromString(o.ObjectId)
		c := config.Config{
			Template: template.NewDownloadTemplate(configId, configId, string(indentedJson)),
			Coordinate: coordinate.Coordinate{
				Project:  projectName,
				Type:     o.SchemaId,
				ConfigId: configId,
			},
			Type: config.SettingsType{
				SchemaId:      o.SchemaId,
				SchemaVersion: o.SchemaVersion,
			},
			Parameters: map[string]parameter.Parameter{
				config.NameParameter:  &value.ValueParameter{Value: configId},
				config.ScopeParameter: &value.ValueParameter{Value: o.Scope},
			},
			Skip:           false,
			OriginObjectId: o.ObjectId,
		}
		result = append(result, c)
	}
	return result
}

func validateSpecificSchemas(c dtclient.SettingsClient, schemas []string) (valid bool, unknownSchemas []string) {
	if len(schemas) == 0 {
		return true, nil
	}

	schemaList, err := c.ListSchemas()
	if err != nil {
		log.Error("failed to query available Settings Schemas: %v", err)
		return false, schemas
	}
	knownSchemas := make(map[string]struct{}, len(schemaList))
	for _, s := range schemaList {
		knownSchemas[s.SchemaId] = struct{}{}
	}

	for _, s := range schemas {
		if _, exists := knownSchemas[s]; !exists {
			unknownSchemas = append(unknownSchemas, s)
		}
	}
	return len(unknownSchemas) == 0, unknownSchemas
}
