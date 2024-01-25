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

package classic

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"golang.org/x/net/context"
	"strings"
)

type deleteValue struct {
	pointer.DeletePointer
	ID   string
	Name string
}

// Delete removes the given pointer.DeletePointer entries from the environment the supplied client dtclient.Client connects to
func Delete(ctx context.Context, client dtclient.Client, theApi api.API, entries []pointer.DeletePointer, targetApi string) error {
	logger := log.WithCtxFields(ctx).WithFields(field.Type(theApi.ID))

	deleteErrs := 0
	var err error
	var delValues []deleteValue

	// if the api is *not* a subpath api, we can just list all configs that exist for a given api and then filter the items that need to be deleted
	if !theApi.SubPathAPI {
		var values []dtclient.Value
		values, err = client.ListConfigs(ctx, theApi)
		if err != nil {
			logger.WithFields(field.Error(err)).Error("Failed to fetch existing configs of API type %q - skipping deletion: %v", theApi.ID, err)
			return err
		}

		delValues, err = filterValuesToDelete(logger, entries, values, theApi.ID)

	} else {
		// for sub-path APIs, it is a bit more complex. we need to query all entries of each scope defined we can delete it.

		// map all entries by scope, so we can later filter them by scope
		scopedMapped := map[string][]pointer.DeletePointer{}
		for _, entry := range entries {
			scopedMapped[entry.Scope] = append(scopedMapped[entry.Scope], entry)
		}

		for scope, scopeEntries := range scopedMapped {
			a := theApi.Resolve(scope)

			var values []dtclient.Value
			values, err = client.ListConfigs(ctx, a)
			if err != nil {
				logger.WithFields(field.Error(err)).Error("Failed to fetch existing configs for api %q (scope: %s): %w", a.ID, scope, err)
				deleteErrs++
				continue
			}

			var vals []deleteValue
			vals, err = filterValuesToDelete(logger, scopeEntries, values, theApi.ID)

			delValues = append(delValues, vals...)
		}
	}

	if err != nil {
		deleteErrs++
	}

	if len(delValues) == 0 {
		logger.Debug("No values found to delete for type %q.", targetApi)
		return err
	}

	logger.Info("Deleting %d config(s) of type %q...", len(delValues), theApi.ID)

	for _, v := range delValues {
		vLog := logger.WithFields(field.Coordinate(v.AsCoordinate()), field.F("value", v))

		a := theApi
		if a.SubPathAPI {
			a = a.Resolve(v.DeletePointer.Scope)
		}

		vLog.Debug("Deleting %s with ID %s", targetApi, v.ID)
		if err := client.DeleteConfigById(a, v.ID); err != nil {
			vLog.Error("Failed to delete %s with ID %s: %v", a.ID, v.ID, err)
			deleteErrs++
		}
	}

	if deleteErrs > 0 {
		return fmt.Errorf("failed to delete %d config(s) of type %q", deleteErrs, theApi.ID)
	}

	return nil
}

// DeleteAll collects and deletes all classic API configuration objects using the provided ConfigClient.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - client (dtclient.ConfigClient): An implementation of the ConfigClient interface for managing configuration objects.
//   - apis (api.APIs): A list of APIs for which configuration values need to be collected and deleted.
//
// Returns:
//   - error: After all deletions where attempted an error is returned if any attempt failed.
func DeleteAll(ctx context.Context, client dtclient.ConfigClient, apis api.APIs) error {

	errs := 0

	for _, a := range apis {
		logger := log.WithCtxFields(ctx).WithFields(field.Type(a.ID))
		logger.Info("Collecting configs of type %q...", a.ID)
		values, err := client.ListConfigs(ctx, a)
		if err != nil {
			errs++
			continue
		}

		logger.Info("Deleting %d configs of type %q...", len(values), a.ID)

		for _, v := range values {
			logger := logger.WithFields(field.F("value", v))
			logger.Debug("Deleting config %s:%s...", a.ID, v.Id)
			err := client.DeleteConfigById(a, v.Id)

			if err != nil {
				logger.WithFields(field.Error(err)).Error("Failed to delete %s with ID %s: %v", a.ID, v.Id, err)
				errs++
			}
		}
	}

	if errs > 0 {
		return fmt.Errorf("failed to delete %d config(s)", errs)
	}

	return nil
}

// filterValuesToDelete filters the given values for only values we want to delete.
// We first search the names of the config-to-be-deleted, and if we find it, return them.
// If we don't find it, we look if the name is actually an id, and if we find it, return them.
// If a given name is found multiple times, we return an error for each name.
func filterValuesToDelete(logger loggers.Logger, entries []pointer.DeletePointer, existingValues []dtclient.Value, apiName string) ([]deleteValue, error) {

	toDeleteByDelPtr := make(map[pointer.DeletePointer][]dtclient.Value, len(entries))
	valuesById := make(map[string]dtclient.Value, len(existingValues))

	for _, v := range existingValues {
		valuesById[v.Id] = v

		for _, entry := range entries {
			if toDeleteByDelPtr[entry] == nil {
				toDeleteByDelPtr[entry] = []dtclient.Value{}
			}

			if v.Name == entry.Identifier {
				toDeleteByDelPtr[entry] = append(toDeleteByDelPtr[entry], v)
			}
		}
	}

	result := make([]deleteValue, 0, len(entries))
	filterErr := false

	for delPtr, valuesToDelete := range toDeleteByDelPtr {

		switch len(valuesToDelete) {
		case 1:
			result = append(result, deleteValue{
				DeletePointer: delPtr,
				ID:            valuesToDelete[0].Id,
				Name:          valuesToDelete[0].Name,
			})
		case 0:
			v, found := valuesById[delPtr.Identifier]

			if found {
				result = append(result, deleteValue{
					DeletePointer: delPtr,
					ID:            v.Id,
					Name:          v.Name,
				})
			} else {
				logger.WithFields(field.F("expectedID", delPtr.Identifier)).Debug("No config of type %s found with the name or ID %q", apiName, delPtr.Identifier)
			}

		default:
			// multiple configs with this name found -> error
			matches := strings.Builder{}
			for i, v := range valuesToDelete {
				matches.WriteString(v.Id)
				if i < len(valuesToDelete)-1 {
					matches.WriteString(", ")
				}
			}
			logger.WithFields(field.F("expectedID", delPtr.Identifier)).Error("Unable to delete unique config - multiple configs of type %q found with the name %q. Please manually delete the desired configuration(s) with IDs: %s", apiName, delPtr.Identifier, matches.String())
			filterErr = true
		}
	}

	if filterErr {
		return result, fmt.Errorf("failed to identify all configurations to be deleted")
	}

	return result, nil
}
