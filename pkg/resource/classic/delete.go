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

package classic

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type DeleteSource interface {
	Delete(ctx context.Context, api api.API, id string) error
	List(ctx context.Context, api api.API) ([]dtclient.Value, error)
}

type Deleter struct {
	source DeleteSource
}

func NewDeleter(source DeleteSource) *Deleter {
	return &Deleter{source}
}

// Delete removes the given pointer.DeletePointer entries from the environment the supplied client dtclient.Client connects to
func (d Deleter) Delete(ctx context.Context, dps []pointer.DeletePointer) error {
	if len(dps) == 0 {
		return nil
	}
	apiType := dps[0].Type
	logger := slog.With(log.TypeAttr(apiType))
	logger.InfoContext(ctx, "Deleting configurations ...", slog.Int("count", len(dps)))

	var err error
	for _, dp := range dps {
		logger := logger.With(log.CoordinateAttr(dp.AsCoordinate()))
		theAPI := api.NewAPIs()[dp.Type]
		var parentID string
		var e error
		if theAPI.HasParent() {
			parentID, e = d.resolveIdentifier(ctx, theAPI.Parent, toIdentifier(dp.Scope, "", ""))
			if e != nil && !coreapi.IsNotFoundError(e) {
				logger.ErrorContext(ctx, "Unable to resolve config ID", log.ErrorAttr(e))
				err = errors.Join(err, e)
				continue
			} else if parentID == "" {
				logger.DebugContext(ctx, "Parent doesn't exist - no need for action")
				continue
			}
		}

		a := theAPI.ApplyParentObjectID(parentID)
		id := dp.OriginObjectId
		if id == "" {
			id, e = d.resolveIdentifier(ctx, &a, toIdentifier(dp.Identifier, dp.ActionType, dp.Domain))
			if e != nil && !coreapi.IsNotFoundError(e) {
				logger.ErrorContext(ctx, "Unable to resolve config ID", log.ErrorAttr(e))
				err = errors.Join(err, e)
				continue
			} else if id == "" {
				logger.DebugContext(ctx, "Config doesn't exist - no need for action")
				continue
			}
		}

		if e := d.source.Delete(ctx, a, id); e != nil && !coreapi.IsNotFoundError(e) {
			logger.ErrorContext(ctx, "Failed to delete config", log.ErrorAttr(e))
			err = errors.Join(err, e)
			continue
		}
		logger.DebugContext(ctx, "Config deleted successfully")
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

// resolveIdentifier get the actual ID from DT and update entries with it
func (d Deleter) resolveIdentifier(ctx context.Context, theAPI *api.API, identifier identifier) (string, error) {
	knownValues, err := d.source.List(ctx, *theAPI)
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
