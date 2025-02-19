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
	"strings"
	"sync"

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
)

type schema struct {
	id      string
	ordered bool
}

func Download(ctx context.Context, client client.SettingsClient, projectName string, filters Filters, schemaIDs ...config.SettingsType) (v2.ConfigsPerType, error) {
	if len(schemaIDs) == 0 {
		return downloadAll(ctx, client, projectName, filters)
	}
	var schemas []string
	for _, s := range schemaIDs {
		schemas = append(schemas, s.SchemaId)
	}
	return downloadSpecific(ctx, client, projectName, schemas, filters)
}

func downloadAll(ctx context.Context, client client.SettingsClient, projectName string, filters Filters) (v2.ConfigsPerType, error) {
	log.Debug("Fetching all schemas to download")
	schemas, err := fetchAllSchemas(ctx, client)
	if err != nil {
		return nil, err
	}

	return download(ctx, client, schemas, projectName, filters), nil
}

func downloadSpecific(ctx context.Context, client client.SettingsClient, projectName string, schemaIDs []string, filters Filters) (v2.ConfigsPerType, error) {
	schemas, err := fetchSchemas(ctx, client, schemaIDs)
	if err != nil {
		return v2.ConfigsPerType{}, err
	}

	if ok, unknownSchemas := validateSpecificSchemas(schemas, schemaIDs); !ok {
		err := fmt.Errorf("requested settings-schema(s) '%v' are not known", strings.Join(unknownSchemas, ","))
		log.WithFields(field.F("unknownSchemas", unknownSchemas), field.Error(err)).Error("%v. Please consult the documentation for available schemas and verify they are available in your environment.", err)
		return nil, err
	}

	log.Debug("Settings to download: \n - %v", strings.Join(schemaIDs, "\n - "))
	result := download(ctx, client, schemas, projectName, filters)
	return result, nil
}

func fetchAllSchemas(ctx context.Context, cl client.SettingsClient) ([]schema, error) {
	dlSchemas, err := cl.ListSchemas(ctx)
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

func fetchSchemas(ctx context.Context, cl client.SettingsClient, schemaIds []string) ([]schema, error) {
	dlSchemas, err := cl.ListSchemas(ctx)
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

func download(ctx context.Context, client client.SettingsClient, schemas []schema, projectName string, filters Filters) v2.ConfigsPerType {
	results := make(v2.ConfigsPerType, len(schemas))
	downloadMutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(schemas))
	for _, sc := range schemas {
		go func(s schema) {
			defer wg.Done()

			lg := log.WithFields(field.Type(s.id))

			lg.Debug("Downloading all settings for schema '%s'", s.id)
			objects, err := client.List(ctx, s.id, dtclient.ListSettingsOptions{})
			if err != nil {
				var errMsg string
				var apiErr coreapi.APIError
				if errors.As(err, &apiErr) {
					errMsg = asConcurrentErrMsg(apiErr)
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

func asConcurrentErrMsg(err coreapi.APIError) string {
	if err.StatusCode != 403 {
		return err.Error()
	}

	concurrentDownloadLimit := environment.GetEnvValueInt(environment.ConcurrentRequestsEnvKey)
	additionalMessage := fmt.Sprintf("\n\n    A 403 error code probably means too many requests.\n    Reduce the number of concurrent requests by setting the %q environment variable (current value: %d). \n    Then wait a few minutes and retry ", environment.ConcurrentRequestsEnvKey, concurrentDownloadLimit)
	return fmt.Sprintf("%s\n%s", err.Error(), additionalMessage)
}

func convertAllObjects(settingsObjects []dtclient.DownloadSettingsObject, projectName string, ordered bool, filters Filters) []config.Config {
	result := make([]config.Config, 0, len(settingsObjects))

	var previousConfigForScope = make(map[string]*config.Config)

	for _, settingsObject := range settingsObjects {
		if shouldFilterUnmodifiableSettings() && !settingsObject.IsModifiable() && len(settingsObject.GetModifiablePaths()) == 0 {
			log.WithFields(field.Type(settingsObject.SchemaId), field.F("object", settingsObject)).Debug("Discarded settings object %q (%s). Reason: Unmodifiable default setting.", settingsObject.ObjectId, settingsObject.SchemaId)
			continue
		}

		// try to unmarshall settings value
		var contentUnmarshalled map[string]interface{}
		if err := json.Unmarshal(settingsObject.Value, &contentUnmarshalled); err != nil {
			log.WithFields(field.Type(settingsObject.SchemaId), field.F("object", settingsObject)).Error("Unable to unmarshal JSON value of settings 2.0 object: %v", err)
			return result
		}
		// skip discarded settings settingsObjects
		if shouldDiscard, reason := filters.Get(settingsObject.SchemaId).ShouldDiscard(contentUnmarshalled); shouldFilterSettings() && shouldDiscard {
			log.WithFields(field.Type(settingsObject.SchemaId), field.F("object", settingsObject)).Debug("Discarded setting object %q (%s). Reason: %s", settingsObject.ObjectId, settingsObject.SchemaId, reason)
			continue
		}

		indentedJson := jsonutils.MarshalIndent(settingsObject.Value)
		// construct config object with generated config ID
		configId := idutils.GenerateUUIDFromString(settingsObject.ObjectId)
		scope := settingsObject.Scope
		c := config.Config{
			Template: template.NewInMemoryTemplate(configId, string(indentedJson)),
			Coordinate: coordinate.Coordinate{
				Project:  projectName,
				Type:     settingsObject.SchemaId,
				ConfigId: configId,
			},
			Type: config.SettingsType{
				SchemaId:      settingsObject.SchemaId,
				SchemaVersion: settingsObject.SchemaVersion,
			},
			Parameters: map[string]parameter.Parameter{
				config.ScopeParameter: &value.ValueParameter{Value: scope},
			},
			Skip:           false,
			OriginObjectId: settingsObject.ObjectId,
		}

		insertAfterConfig, found := previousConfigForScope[scope]
		if settingsObject.IsMovable() && ordered && found {
			c.Parameters[config.InsertAfterParameter] = reference.NewWithCoordinate(insertAfterConfig.Coordinate, "id")
		}
		result = append(result, c)
		previousConfigForScope[scope] = &c

	}
	return result
}

func shouldFilterSettings() bool {
	return featureflags.DownloadFilter.Enabled() && featureflags.DownloadFilterSettings.Enabled()
}

func shouldFilterUnmodifiableSettings() bool {
	return shouldFilterSettings() && featureflags.DownloadFilterSettingsUnmodifiable.Enabled()
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
