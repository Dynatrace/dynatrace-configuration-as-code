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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"strings"
	"sync"
)

type schema struct {
	id      string
	ordered bool
}

func Download(client client.SettingsClient, projectName string, filters Filters, schemaIDs ...config.SettingsType) (v2.ConfigsPerType, error) {
	if len(schemaIDs) == 0 {
		return downloadAll(client, projectName, filters)
	}
	var schemas []string
	for _, s := range schemaIDs {
		schemas = append(schemas, s.SchemaId)
	}
	return downloadSpecific(client, projectName, schemas, filters)
}

func downloadAll(client client.SettingsClient, projectName string, filters Filters) (v2.ConfigsPerType, error) {
	log.Debug("Fetching all schemas to download")
	schemas, err := fetchAllSchemas(client)
	if err != nil {
		return nil, err
	}

	return download(client, schemas, projectName, filters), nil
}

func downloadSpecific(client client.SettingsClient, projectName string, schemaIDs []string, filters Filters) (v2.ConfigsPerType, error) {
	schemas, err := fetchSchemas(client, schemaIDs)
	if err != nil {
		return v2.ConfigsPerType{}, err
	}

	if ok, unknownSchemas := validateSpecificSchemas(schemas, schemaIDs); !ok {
		err := fmt.Errorf("requested settings-schema(s) '%v' are not known", strings.Join(unknownSchemas, ","))
		log.WithFields(field.F("unknownSchemas", unknownSchemas), field.Error(err)).Error("%v. Please consult the documentation for available schemas and verify they are available in your environment.", err)
		return nil, err
	}

	log.Debug("Settings to download: \n - %v", strings.Join(schemaIDs, "\n - "))
	result := download(client, schemas, projectName, filters)
	return result, nil
}

func fetchAllSchemas(cl client.SettingsClient) ([]schema, error) {
	dlSchemas, err := cl.ListSchemas(context.TODO())
	if err != nil {
		return nil, err
	}

	var schemas []schema
	for _, s := range dlSchemas {
		schemas = append(schemas, schema{
			id:      s.SchemaId,
			ordered: s.Ordered,
		})

	}
	return schemas, nil
}

func fetchSchemas(cl client.SettingsClient, schemaIds []string) ([]schema, error) {
	dlSchemas, err := cl.ListSchemas(context.TODO())
	if err != nil {
		return nil, err
	}

	schemaIdSet := make(map[string]struct{})
	for _, id := range schemaIds {
		schemaIdSet[id] = struct{}{}
	}

	var schemas []schema
	for _, s := range dlSchemas {
		if _, exists := schemaIdSet[s.SchemaId]; exists {
			schemas = append(schemas, schema{
				id:      s.SchemaId,
				ordered: s.Ordered,
			})
		}
	}

	return schemas, nil
}

func download(client client.SettingsClient, schemas []schema, projectName string, filters Filters) v2.ConfigsPerType {
	results := make(v2.ConfigsPerType, len(schemas))
	downloadMutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(schemas))
	for _, sc := range schemas {
		go func(s schema) {
			defer wg.Done()

			lg := log.WithFields(field.Type(s.id))

			lg.Debug("Downloading all settings for schema '%s'", s.id)
			objects, err := client.ListSettings(context.TODO(), s.id, dtclient.ListSettingsOptions{})
			if err != nil {
				var errMsg string
				var respErr coreapi.APIError
				if errors.As(err, &respErr) {
					errMsg = asConcurrentError(respErr)
				} else {
					errMsg = err.Error()
				}
				lg.WithFields(field.Error(err)).Error("Failed to fetch all settings for schema '%s': %v", s.id, errMsg)
				return
			}

			cfgs := convertAllObjects(objects, projectName, sc.ordered, filters)
			downloadMutex.Lock()
			results[s.id] = cfgs
			downloadMutex.Unlock()

			lg = lg.WithFields(field.F("configsDownloaded", len(cfgs)))
			switch len(objects) {
			case 0:
				lg.Debug("Did not find any settings to download for schema '%s'", s.id)
			case len(cfgs):
				lg.Info("Downloaded %d settings for schema '%s'", len(cfgs), s.id)
			default:
				lg.Info("Downloaded %d settings for schema '%s'. Skipped persisting %d unmodifiable setting(s)", len(cfgs), s.id, len(objects)-len(cfgs))
			}
		}(sc)
	}
	wg.Wait()

	return results
}

func asConcurrentError(err coreapi.APIError) string {
	if err.StatusCode != 403 {
		return err.Error()
	}

	concurrentDownloadLimit := environment.GetEnvValueInt(environment.ConcurrentRequestsEnvKey)
	additionalMessage := fmt.Sprintf("\n\n    A 403 error code probably means too many requests.\n    Reduce the number of concurrent requests by setting the %q environment variable (current value: %d). \n    Then wait a few minutes and retry ", environment.ConcurrentRequestsEnvKey, concurrentDownloadLimit)
	return fmt.Sprintf("%s\n%s", err.Error(), additionalMessage)
}

func convertAllObjects(objects []dtclient.DownloadSettingsObject, projectName string, ordered bool, filters Filters) []config.Config {
	result := make([]config.Config, 0, len(objects))
	var previousConfig *config.Config = nil
	for _, o := range objects {

		if shouldFilterUnmodifiableSettings() && o.ModificationInfo != nil && !o.ModificationInfo.Modifiable && len(o.ModificationInfo.ModifiablePaths) == 0 {
			log.WithFields(field.Type(o.SchemaId), field.F("object", o)).Debug("Discarded settings object %q (%s). Reason: Unmodifiable default setting.", o.ObjectId, o.SchemaId)
			continue
		}

		// try to unmarshall settings value
		var contentUnmarshalled map[string]interface{}
		if err := json.Unmarshal(o.Value, &contentUnmarshalled); err != nil {
			log.WithFields(field.Type(o.SchemaId), field.F("object", o)).Error("Unable to unmarshal JSON value of settings 2.0 object: %v", err)
			return result
		}
		// skip discarded settings objects
		if shouldDiscard, reason := filters.Get(o.SchemaId).ShouldDiscard(contentUnmarshalled); shouldFilterSettings() && shouldDiscard {
			log.WithFields(field.Type(o.SchemaId), field.F("object", o)).Debug("Discarded setting object %q (%s). Reason: %s", o.ObjectId, o.SchemaId, reason)
			continue
		}

		indentedJson := jsonutils.MarshalIndent(o.Value)
		// construct config object with generated config ID
		configId := idutils.GenerateUUIDFromString(o.ObjectId)
		c := config.Config{
			Template: template.NewInMemoryTemplate(configId, string(indentedJson)),
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
				config.ScopeParameter: &value.ValueParameter{Value: o.Scope},
			},
			Skip:           false,
			OriginObjectId: o.ObjectId,
		}

		if ordered && (previousConfig != nil) {
			c.Parameters[config.InsertAfterParameter] = reference.NewWithCoordinate(previousConfig.Coordinate, "id")
		}
		result = append(result, c)
		previousConfig = &c

	}
	return result
}

func shouldFilterSettings() bool {
	return featureflags.Permanent[featureflags.DownloadFilter].Enabled() && featureflags.Permanent[featureflags.DownloadFilterSettings].Enabled()
}

func shouldFilterUnmodifiableSettings() bool {
	return shouldFilterSettings() && featureflags.Permanent[featureflags.DownloadFilterSettingsUnmodifiable].Enabled()
}

func validateSpecificSchemas(schemas []schema, schemaIDs []string) (valid bool, unknownSchemas []string) {
	if len(schemaIDs) == 0 {
		return true, nil
	}

	knownSchemas := make(map[string]struct{}, len(schemas))
	for _, s := range schemas {
		knownSchemas[s.id] = struct{}{}
	}

	for _, s := range schemaIDs {
		if _, exists := knownSchemas[s]; !exists {
			unknownSchemas = append(unknownSchemas, s)
		}
	}
	return len(unknownSchemas) == 0, unknownSchemas
}
