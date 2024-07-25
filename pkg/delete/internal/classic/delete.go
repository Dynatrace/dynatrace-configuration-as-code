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
	"errors"
	"fmt"
	"net/http"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"golang.org/x/net/context"
)

// Delete removes the given pointer.DeletePointer entries from the environment the supplied client dtclient.Client connects to
func Delete(ctx context.Context, client client.ConfigClient, dps []pointer.DeletePointer) error {
	var err error

	for _, dp := range dps {
		log := log.WithCtxFields(ctx).WithFields(field.Coordinate(dp.AsCoordinate()))
		theAPI := api.NewAPIs()[dp.Type]
		var parentID string
		var e error
		if theAPI.HasParent() {
			parentID, e = resolveIdentifier(ctx, client, theAPI.Parent, toIdentifier(dp.Scope, "", ""))
			if e != nil && !is404(e) {
				log.WithFields(field.Error(e)).Error("unable to resolve config ID: %v", e)
				err = errors.Join(err, e)
				continue
			} else if parentID == "" {
				log.Debug("parent doesn't exist - no need for action")
				continue
			}
		}

		a := theAPI.ApplyParentObjectID(parentID)
		id := dp.OriginObjectId
		if id == "" {
			id, e = resolveIdentifier(ctx, client, &a, toIdentifier(dp.Identifier, dp.ActionType, dp.Domain))
			if e != nil && !is404(e) {
				log.WithFields(field.Error(e)).Error("unable to resolve config ID: %v", e)
				err = errors.Join(err, e)
				continue
			} else if id == "" {
				log.Debug("config doesn't exist - no need for action")
				continue
			}
		}

		if e := client.DeleteConfigById(ctx, a, id); e != nil && !is404(e) {
			log.WithFields(field.Error(e)).Error("failed to delete config: %v", e)
			err = errors.Join(err, e)
		}
		log.Debug("successfully deleted")
	}
	return err
}

type identifier map[string]any

func toIdentifier(identifier, actionType, domain string) identifier {
	return map[string]any{
		"name":       identifier,
		"actionType": actionType,
		"domain":     domain,
	}
}

func is404(err error) bool {
	var v coreapi.APIError
	return errors.As(err, &v) && v.StatusCode == http.StatusNotFound
}

// resolveIdentifier get the actual ID from DT and update entries with it
func resolveIdentifier(ctx context.Context, client client.ConfigClient, theAPI *api.API, identifier identifier) (string, error) {
	knownValues, err := client.ListConfigs(ctx, *theAPI)
	if err != nil {
		return "", err
	}

	id, err := findUniqueID(knownValues, identifier, theAPI.CheckEqualFunc)
	if err != nil {
		return "", err
	}

	return id, nil
}

func findUniqueID(knownValues []dtclient.Value, identifier identifier, checkEqualFn func(map[string]any, map[string]any) bool) (string, error) {
	type resolvedID = string
	var knownByName []resolvedID
	var knownByID resolvedID

	for i := range knownValues {
		if checkEqualFn != nil {
			if checkEqualFn(knownValues[i].CustomFields, identifier) {
				knownByName = append(knownByName, knownValues[i].Id)
			}
		} else if identifier["name"] == knownValues[i].Name {
			knownByName = append(knownByName, knownValues[i].Id)
		} else if identifier["name"] == knownValues[i].Id {
			knownByID = knownValues[i].Id
		}
	}

	if len(knownByName) == 0 {
		return knownByID, nil
	}
	if len(knownByName) == 1 { //unique identifier-id pair
		return knownByName[0], nil
	}
	//multiple configs with this name found -> error
	return "", fmt.Errorf("unable to find unique config - matching IDs are %s", knownByName)
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
func DeleteAll(ctx context.Context, client client.ConfigClient, apis api.APIs) error {

	errs := 0

	for _, a := range apis {
		logger := log.WithCtxFields(ctx).WithFields(field.Type(a.ID))
		if a.HasParent() {
			logger.Debug("Skipping %q, will be deleted by the parent api %q", a.ID, a.Parent)
		}
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
			err := client.DeleteConfigById(ctx, a, v.Id)

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
