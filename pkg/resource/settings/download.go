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
	"slices"
	"strings"
	"sync"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type schema struct {
	id                      string
	ordered                 bool
	ownerBasedAccessControl *bool
}

type Source interface {
	ListSchemas(context.Context) (dtclient.SchemaList, error)
	List(context.Context, string, dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error)
	GetPermission(context.Context, string) (dtclient.PermissionObject, error)
}

type API struct {
	settingsSource  Source
	filters         Filters
	specificSchemas []string
}

func NewAPI(settingsSource Source, filters Filters, specificSchemas []string) *API {
	return &API{settingsSource, filters, specificSchemas}
}

func (a API) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	log.Info("Downloading settings objects")
	if len(a.specificSchemas) == 0 {
		return downloadAll(ctx, a.settingsSource, projectName, a.filters)
	}

	return downloadSpecific(ctx, a.settingsSource, projectName, a.specificSchemas, a.filters)
}

func downloadAll(ctx context.Context, settingsSource Source, projectName string, filters Filters) (project.ConfigsPerType, error) {
	log.Debug("Fetching all schemas to download")
	schemas, err := fetchAllSchemas(ctx, settingsSource)
	if err != nil {
		return nil, err
	}

	return download(ctx, settingsSource, schemas, projectName, filters), nil
}

func downloadSpecific(ctx context.Context, settingsSource Source, projectName string, schemaIDs []string, filters Filters) (project.ConfigsPerType, error) {
	schemas, err := fetchSchemas(ctx, settingsSource, schemaIDs)
	if err != nil {
		return project.ConfigsPerType{}, err
	}

	if ok, unknownSchemas := validateSpecificSchemas(schemas, schemaIDs); !ok {
		err := fmt.Errorf("requested settings-schema(s) '%v' are not known", strings.Join(unknownSchemas, ","))
		log.WithFields(field.F("unknownSchemas", unknownSchemas), field.Error(err)).Error("%v. Please consult the documentation for available schemas and verify they are available in your environment.", err)
		return nil, err
	}

	log.Debug("Settings to download: \n - %v", strings.Join(schemaIDs, "\n - "))
	result := download(ctx, settingsSource, schemas, projectName, filters)
	return result, nil
}

func fetchAllSchemas(ctx context.Context, cl Source) ([]schema, error) {
	dlSchemas, err := cl.ListSchemas(ctx)
	if err != nil {
		return nil, err
	}

	var schemas []schema
	for _, s := range dlSchemas {
		schemas = append(schemas, schema{
			id:                      s.SchemaId,
			ordered:                 s.Ordered,
			ownerBasedAccessControl: s.OwnerBasedAccessControl,
		})

	}
	return schemas, nil
}

func fetchSchemas(ctx context.Context, cl Source, schemaIds []string) ([]schema, error) {
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
				id:                      s.SchemaId,
				ordered:                 s.Ordered,
				ownerBasedAccessControl: s.OwnerBasedAccessControl,
			})
		}
	}

	return schemas, nil
}

func download(ctx context.Context, settingsSource Source, schemas []schema, projectName string, filters Filters) project.ConfigsPerType {
	results := make(project.ConfigsPerType, len(schemas))
	downloadMutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(schemas))
	for _, sc := range schemas {
		go func(s schema) {
			defer wg.Done()

			lg := log.WithFields(field.Type(s.id))

			lg.Debug("Downloading all settings for schema '%s'", s.id)
			objects, err := settingsSource.List(ctx, s.id, dtclient.ListSettingsOptions{})
			if err != nil {
				errMsg := extractApiErrorMessage(err)
				lg.WithFields(field.Error(err)).Error("Failed to fetch all settings for schema '%s': %v", s.id, errMsg)
				return
			}

			permissions := make(map[string]dtclient.PermissionObject)
			if s.ownerBasedAccessControl != nil && *s.ownerBasedAccessControl && featureflags.AccessControlSettings.Enabled() {
				var permErr error
				permissions, permErr = getObjectsPermission(ctx, settingsSource, objects)
				if permErr != nil {
					errMsg := extractApiErrorMessage(permErr)
					lg.WithFields(field.Error(permErr)).Error("Failed to fetch settings permissions for schema '%s': %v", s.id, errMsg)
					return
				}
			}

			cfgs := convertAllObjects(objects, permissions, projectName, sc.ordered, filters)
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

func extractApiErrorMessage(err error) string {
	var apiErr coreapi.APIError
	if errors.As(err, &apiErr) {
		return asConcurrentErrMsg(apiErr)
	}
	return err.Error()
}

func getObjectsPermission(ctx context.Context, settingsSource Source, objects []dtclient.DownloadSettingsObject) (map[string]dtclient.PermissionObject, error) {
	type result struct {
		Permission dtclient.PermissionObject
		ObjectId   string
		Err        error
	}
	errs := make([]error, 0)
	resChan := make(chan result, len(objects))
	defer close(resChan)

	permissions := make(map[string]dtclient.PermissionObject)
	for _, obj := range objects {
		go func(ctx context.Context, obj dtclient.DownloadSettingsObject) {
			permission, err := settingsSource.GetPermission(ctx, obj.ObjectId)
			resChan <- result{permission, obj.ObjectId, err}
		}(ctx, obj)
	}

	for i := 0; i < len(objects); i++ {
		res := <-resChan
		if res.Err != nil {
			errs = append(errs, res.Err)
			continue
		}
		permissions[res.ObjectId] = res.Permission
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return permissions, nil
}

func asConcurrentErrMsg(err coreapi.APIError) string {
	if err.StatusCode != 403 {
		return err.Error()
	}

	concurrentDownloadLimit := environment.GetEnvValueInt(environment.ConcurrentRequestsEnvKey)
	additionalMessage := fmt.Sprintf("\n\n    A 403 error code probably means too many requests.\n    Reduce the number of concurrent requests by setting the %q environment variable (current value: %d). \n    Then wait a few minutes and retry ", environment.ConcurrentRequestsEnvKey, concurrentDownloadLimit)
	return fmt.Sprintf("%s\n%s", err.Error(), additionalMessage)
}

func convertAllObjects(settingsObjects []dtclient.DownloadSettingsObject, permissions map[string]dtclient.PermissionObject, projectName string, ordered bool, filters Filters) []config.Config {
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
				SchemaId:          settingsObject.SchemaId,
				SchemaVersion:     settingsObject.SchemaVersion,
				AllUserPermission: getObjectPermission(permissions, settingsObject.ObjectId),
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

func getObjectPermission(permissions map[string]dtclient.PermissionObject, objectID string) *config.AllUserPermissionKind {
	if p, exists := permissions[objectID]; exists && p.Accessor != nil && p.Accessor.Type == dtclient.AllUsers {
		if slices.Contains(p.Permissions, dtclient.Write) {
			return pointer.Pointer(config.WritePermission)
		}
		if slices.Contains(p.Permissions, dtclient.Read) {
			return pointer.Pointer(config.ReadPermission)
		}
		return pointer.Pointer(config.NonePermission)
	}
	return nil
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
