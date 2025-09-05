/**
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

package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
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

type DownloadSource interface {
	ListSchemas(context.Context) (dtclient.SchemaList, error)
	List(context.Context, string, dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error)
	GetPermission(context.Context, string) (dtclient.PermissionObject, error)
}

type DownloadAPI struct {
	settingsSource        DownloadSource
	filters               Filters
	specificSchemas       []string
	classicSettingsSource bool
}

func NewDownloadAPI(settingsSource DownloadSource, filters Filters, specificSchemas []string, usePlatform bool) *DownloadAPI {
	return &DownloadAPI{settingsSource, filters, specificSchemas, usePlatform}
}

func (a DownloadAPI) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	slog.InfoContext(ctx, "Downloading settings objects")
	if len(a.specificSchemas) == 0 {
		return a.downloadAll(ctx, projectName, a.filters)
	}

	return a.downloadSpecific(ctx, projectName, a.specificSchemas, a.filters)
}

func (a DownloadAPI) downloadAll(ctx context.Context, projectName string, filters Filters) (project.ConfigsPerType, error) {
	slog.DebugContext(ctx, "Fetching all schemas to download")
	schemas, err := a.fetchAllSchemas(ctx)
	if err != nil {
		return nil, err
	}

	return a.download(ctx, schemas, projectName, filters), nil
}

func (a DownloadAPI) downloadSpecific(ctx context.Context, projectName string, schemaIDs []string, filters Filters) (project.ConfigsPerType, error) {
	schemas, err := a.fetchSchemas(ctx, schemaIDs)
	if err != nil {
		return project.ConfigsPerType{}, err
	}

	if ok, unknownSchemaIDs := validateSpecificSchemas(schemas, schemaIDs); !ok {
		err := fmt.Errorf("requested settings-schema(s) '%v' are not known", strings.Join(unknownSchemaIDs, ","))
		slog.ErrorContext(ctx, "Some settings schemas are unknown", slog.Any("unknownSchemaIds", unknownSchemaIDs))
		return nil, err
	}

	slog.DebugContext(ctx, "Settings to download", slog.Any("schemaIds", schemaIDs))
	result := a.download(ctx, schemas, projectName, filters)
	return result, nil
}

func (a DownloadAPI) fetchAllSchemas(ctx context.Context) ([]schema, error) {
	dlSchemas, err := a.settingsSource.ListSchemas(ctx)
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

func (a DownloadAPI) fetchSchemas(ctx context.Context, schemaIds []string) ([]schema, error) {
	dlSchemas, err := a.settingsSource.ListSchemas(ctx)
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

func (a DownloadAPI) download(ctx context.Context, schemas []schema, projectName string, filters Filters) project.ConfigsPerType {
	results := make(project.ConfigsPerType, len(schemas))
	downloadMutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(schemas))
	for _, sc := range schemas {
		go func(s schema) {
			defer wg.Done()

			lg := slog.With(log.TypeAttr(s.id))

			lg.DebugContext(ctx, "Downloading all settings for schema")
			objects, err := a.settingsSource.List(ctx, s.id, dtclient.ListSettingsOptions{})
			if err != nil {
				lg.ErrorContext(ctx, "Failed to fetch all settings for schema", log.ErrorAttr(err))
				return
			}

			permissions, err := a.getPermissions(ctx, s, objects, lg)
			if err != nil {
				lg.ErrorContext(ctx, "Failed to fetch settings permissions for schema", log.ErrorAttr(err))
				return
			}

			cfgs := convertAllObjects(objects, permissions, projectName, sc.ordered, filters)
			downloadMutex.Lock()
			results[s.id] = cfgs
			downloadMutex.Unlock()

			switch len(objects) {
			case 0:
				lg.DebugContext(ctx, "Did not find any settings to download for schema")
			case len(cfgs):
				lg.InfoContext(ctx, "Downloaded settings for schema", slog.Int("count", len(cfgs)))
			default:
				lg.InfoContext(ctx, "Downloaded settings for schema. Skipped persisting unmodifiable settings", slog.Int("count", len(cfgs)), slog.Int("skipCount", len(objects)-len(cfgs)))
			}
		}(sc)
	}
	wg.Wait()

	return results
}

func (a DownloadAPI) getPermissions(ctx context.Context, s schema, objects []dtclient.DownloadSettingsObject, lg *slog.Logger) (map[string]dtclient.PermissionObject, error) {
	if s.ownerBasedAccessControl == nil || !*s.ownerBasedAccessControl {
		return nil, nil
	}

	if a.classicSettingsSource {
		lg.WarnContext(ctx, "Skipped getting permissions as download is using classic credentials")
		return nil, nil
	}

	return getObjectsPermission(ctx, a.settingsSource, objects)
}

func getObjectsPermission(ctx context.Context, settingsSource DownloadSource, objects []dtclient.DownloadSettingsObject) (map[string]dtclient.PermissionObject, error) {
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

func convertAllObjects(settingsObjects []dtclient.DownloadSettingsObject, permissions map[string]dtclient.PermissionObject, projectName string, ordered bool, filters Filters) []config.Config {
	result := make([]config.Config, 0, len(settingsObjects))

	var previousConfigForScope = make(map[string]*config.Config)

	for _, settingsObject := range settingsObjects {
		if shouldFilterUnmodifiableSettings() && !settingsObject.IsModifiable() && len(settingsObject.GetModifiablePaths()) == 0 {
			slog.Debug("Discarded unmodifiable default settings object", log.TypeAttr(settingsObject.SchemaId), slog.Any("object", settingsObject))
			continue
		}

		// try to unmarshall settings value
		var contentUnmarshalled map[string]interface{}
		if err := json.Unmarshal(settingsObject.Value, &contentUnmarshalled); err != nil {
			slog.Error("Unable to unmarshal JSON value of settings object", log.ErrorAttr(err), log.TypeAttr(settingsObject.SchemaId), slog.Any("object", settingsObject))
			return result
		}
		// skip discarded settings settingsObjects
		if shouldDiscard, reason := filters.Get(settingsObject.SchemaId).ShouldDiscard(contentUnmarshalled); shouldFilterSettings() && shouldDiscard {
			slog.Debug("Discarded setting object", slog.String("reason", reason), log.TypeAttr(settingsObject.SchemaId), slog.Any("object", settingsObject))
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
