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

package settings

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type DeleteSource interface {
	ListSchemas(ctx context.Context) (dtclient.SchemaList, error)
	List(ctx context.Context, schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error)
	Delete(ctx context.Context, objectID string) error
}

type Deleter struct {
	source DeleteSource
}

func NewDeleter(source DeleteSource) *Deleter {
	return &Deleter{source}
}

func (d Deleter) Delete(ctx context.Context, entries []pointer.DeletePointer) error {
	if len(entries) == 0 {
		return nil
	}
	schema := entries[0].Type

	logger := log.With(log.TypeAttr(schema))
	logger.InfoContext(ctx, "Deleting %d settings object(s) of schema %q...", len(entries), schema)

	deleteErrs := 0
	for _, e := range entries {
		logger := logger.With(log.CoordinateAttr(e.AsCoordinate()))

		filterFunc, err := getFilter(e)
		if err != nil {
			logger.ErrorContext(ctx, "Setting will not be deleted: %v", err)
			deleteErrs++
			continue
		}

		settingsObjects, err := d.source.List(ctx, e.Type, dtclient.ListSettingsOptions{DiscardValue: true, Filter: filterFunc})
		if err != nil {
			logger.ErrorContext(ctx, "Could not fetch settings object: %v", err)
			deleteErrs++
			continue
		}

		if len(settingsObjects) == 0 {
			if e.OriginObjectId != "" {
				logger.DebugContext(ctx, "No settings object found to delete. Could not find object with matching object id.")
				continue
			}
			logger.DebugContext(ctx, "No settings object found to delete. Could not find object with matching external id.")
			continue
		}

		for _, settingsObject := range settingsObjects {
			if !settingsObject.IsDeletable() {
				logger.With(slog.Any("object", settingsObject)).WarnContext(ctx, "Requested settings object with ID %s is not deletable.", settingsObject.ObjectId)
				continue
			}

			logger.DebugContext(ctx, "Deleting settings object with objectId %q.", settingsObject.ObjectId)
			err := d.source.Delete(ctx, settingsObject.ObjectId)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to delete settings object with object ID %s: %v", settingsObject.ObjectId, err)
				deleteErrs++
			}
		}
	}

	if deleteErrs > 0 {
		return fmt.Errorf("failed to delete %d settings object(s) of schema %q", deleteErrs, schema)
	}

	return nil
}

func getFilter(deletePointer pointer.DeletePointer) (dtclient.ListSettingsFilter, error) {
	if deletePointer.OriginObjectId != "" {
		return func(o dtclient.DownloadSettingsObject) bool { return o.ObjectId == deletePointer.OriginObjectId }, nil
	}

	externalID, err := idutils.GenerateExternalIDForSettingsObject(deletePointer.AsCoordinate())
	if err != nil {
		return nil, fmt.Errorf("unable to generate external id: %w", err)
	}
	return func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId == externalID }, nil

}

// DeleteAll collects and deletes settings objects using the provided SettingsClient.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - c (dtclient.SettingsClient): An implementation of the SettingsClient interface for managing settings objects.
//
// Returns:
//   - error: After all deletions where attempted an error is returned if any attempt failed.
func (d Deleter) DeleteAll(ctx context.Context) error {
	errCount := 0

	schemas, err := d.source.ListSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch settings schemas. No settings will be deleted. Reason: %w", err)
	}

	schemaIds := make([]string, len(schemas))
	for i := range schemas {
		schemaIds[i] = schemas[i].SchemaId
	}

	log.DebugContext(ctx, "Deleting settings of schemas %v...", schemaIds)

	for _, s := range schemaIds {
		logger := log.With(log.TypeAttr(s))
		logger.InfoContext(ctx, "Collecting objects of type %q...", s)

		settingsObjects, err := d.source.List(ctx, s, dtclient.ListSettingsOptions{DiscardValue: true})
		if err != nil {
			logger.With(log.ErrorAttr(err)).ErrorContext(ctx, "Failed to collect object for schema %q: %v", s, err)
			errCount++
			continue
		}

		logger.InfoContext(ctx, "Deleting %d objects of type %q...", len(settingsObjects), s)
		for _, settingsObject := range settingsObjects {
			if !settingsObject.IsDeletable() {
				continue
			}

			logger.With(slog.Any("object", settingsObject)).DebugContext(ctx, "Deleting settings object with object ID '%s'...", settingsObject.ObjectId)
			err := d.source.Delete(ctx, settingsObject.ObjectId)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to delete settings object with object ID '%s': %v", settingsObject.ObjectId, err)
				errCount++
			}
		}
	}

	if errCount > 0 {
		returnedError := fmt.Errorf("failed to delete %d setting(s)", errCount)
		log.ErrorContext(ctx, "Failed to delete all Settings 2.0 objects: %v", returnedError)
		return returnedError
	}

	return nil
}
