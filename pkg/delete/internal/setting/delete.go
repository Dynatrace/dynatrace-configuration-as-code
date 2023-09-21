/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package setting

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"golang.org/x/net/context"
)

func Delete(ctx context.Context, c dtclient.Client, entries []pointer.DeletePointer) error {

	if len(entries) == 0 {
		return nil
	}
	schema := entries[0].Type

	logger := log.WithCtxFields(ctx).WithFields(field.Type(schema))
	logger.Info("Deleting %d settings objects(s) of schema %q...", len(entries), schema)

	deleteErrs := 0
	for _, e := range entries {

		logger := logger.WithFields(field.Coordinate(e.AsCoordinate()))

		if e.Project == "" {
			logger.Warn("Generating legacy externalID - this will fail to identify a newer Settings object. Consider defining a 'project' for this delete entry.")
		}
		externalID, err := idutils.GenerateExternalID(e.AsCoordinate())

		if err != nil {
			logger.Error("Unable to generate externalID, Setting will not be deleted: %v", err)
			deleteErrs++
			continue
		}
		// get settings objects with matching external ID
		objects, err := c.ListSettings(ctx, e.Type, dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId == externalID }})
		if err != nil {
			logger.Error("Could not fetch settings object: %v", err)
			deleteErrs++
			continue
		}

		if len(objects) == 0 {
			logger.Debug("No settings object found to delete")
			continue
		}

		for _, obj := range objects {
			if obj.ModificationInfo != nil && !obj.ModificationInfo.Deletable {
				logger.WithFields(field.F("object", obj)).Warn("Requested settings object with ID %s is not deletable.", obj.ObjectId)
				continue
			}

			logger.Debug("Deleting settings object with objectId %q.", obj.ObjectId)
			err := c.DeleteSettings(obj.ObjectId)
			if err != nil {
				logger.Error("Failed to delete settings object with object ID %s: %v", obj.ObjectId, err)
				deleteErrs++
			}
		}
	}

	if deleteErrs > 0 {
		return fmt.Errorf("failed to delete %d settings objects(s) of schema %q", deleteErrs, schema)
	}

	return nil
}

// DeleteAll collects and deletes settings objects using the provided SettingsClient.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - c (dtclient.SettingsClient): An implementation of the SettingsClient interface for managing settings objects.
//
// Returns:
//   - error: After all deletions where attempted an error is returned if any attempt failed.
func DeleteAll(ctx context.Context, c dtclient.SettingsClient) error {
	errs := 0

	schemas, err := c.ListSchemas()
	if err != nil {
		return fmt.Errorf("failed to fetch settings schemas. No settings will be deleted. Reason: %w", err)
	}

	schemaIds := make([]string, len(schemas))
	for i := range schemas {
		schemaIds[i] = schemas[i].SchemaId
	}

	logger := log.WithCtxFields(ctx)
	logger.Debug("Deleting settings of schemas %v...", schemaIds)

	for _, s := range schemaIds {
		logger := logger.WithFields(field.Type(s))
		logger.Info("Collecting objects of type %q...", s)

		settings, err := c.ListSettings(ctx, s, dtclient.ListSettingsOptions{DiscardValue: true})
		if err != nil {
			logger.WithFields(field.Error(err)).Error("Failed to collect object for schema %q: %v", s, err)
			errs++
			continue
		}

		logger.Info("Deleting %d objects of type %q...", len(settings), s)
		for _, setting := range settings {
			if setting.ModificationInfo != nil && !setting.ModificationInfo.Deletable {
				continue
			}

			logger.WithFields(field.F("object", setting)).Debug("Deleting settings object with objectId %q...", setting.ObjectId)
			err := c.DeleteSettings(setting.ObjectId)
			if err != nil {
				logger.Error("Failed to delete settings object with object ID %s: %v", setting.ObjectId, err)
				errs++
			}
		}
	}

	if errs > 0 {
		return fmt.Errorf("failed to delete %d setting(s)", errs)
	}

	return nil
}
