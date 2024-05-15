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

package automation

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	automationAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"golang.org/x/net/context"
)

type client interface {
	Delete(ctx context.Context, resourceType automationAPI.ResourceType, id string) (automation.Response, error)
	List(ctx context.Context, resourceType automationAPI.ResourceType) (automation.ListResponse, error)
}

func Delete(ctx context.Context, c client, automationResource config.AutomationResource, entries []pointer.DeletePointer) error {

	logger := log.WithCtxFields(ctx).WithFields(field.Type(string(automationResource)))
	logger.Info("Deleting %d config(s) of type %q...", len(entries), automationResource)

	deleteErrs := 0

	for _, e := range entries {

		logger := logger.WithFields(field.Coordinate(e.AsCoordinate()))

		id := e.OriginObjectId
		if id == "" {
			id = idutils.GenerateUUIDFromCoordinate(e.AsCoordinate())
		}

		logger.Debug("Deleting %v with id %q.", automationResource, id)

		resourceType, err := automationutils.ClientResourceTypeFromConfigType(automationResource)
		if err != nil {
			logger.WithFields(field.Error(err)).Error("Failed to delete %v with ID %q: %v", automationResource, id, err)
			deleteErrs++
		}

		_, err = c.Delete(ctx, resourceType, id)
		if err != nil {
			var apiErr api.APIError
			if errors.As(err, &apiErr) {
				if apiErr.StatusCode != http.StatusNotFound {
					logger.WithFields(field.Error(err)).Error("Failed to delete %v with ID %q - rejected by API: %v", automationResource, id, err)
					deleteErrs++
				}
			} else {
				logger.WithFields(field.Error(err)).Error("Failed to delete %v with ID %q - network error: %v", automationResource, id, err)
				deleteErrs++
			}
		}
	}

	if deleteErrs > 0 {
		return fmt.Errorf("failed to delete %d Automation objects(s) of type %q", deleteErrs, automationResource)
	}

	return nil
}

// DeleteAll collects and deletes automations resources using the given automation client.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - c (automationClient): An implementation of the automationClient interface for performing automation-related operations.
//
// Returns:
//   - error: After all deletions where attempted an error is returned if any attempt failed.
func DeleteAll(ctx context.Context, c client) error {
	errs := 0

	resources := []config.AutomationResource{config.Workflow, config.SchedulingRule, config.BusinessCalendar}
	for _, resource := range resources {
		logger := log.WithCtxFields(ctx).WithFields(field.Type(string(resource)))

		t, err := automationutils.ClientResourceTypeFromConfigType(resource)
		if err != nil {
			logger.Error("Failed to delete Automation objects of type %q: %v", err)
			errs++
			continue
		}

		logger.Info("Collecting Automation objects of type %q...", resource)
		resp, err := c.List(ctx, t)
		if err != nil {
			var apiErr api.APIError
			if errors.As(err, &apiErr) {
				logger.WithFields(field.Error(err)).Error("Failed to collect Automation objects of type %q - rejected by API: %v", resource, err)
				errs++
				continue
			} else {
				logger.Error("Failed to collect Automation objects of type %q - network error: %v", resource, err)
				errs++
				continue
			}
		}

		objects, err := automationutils.DecodeListResponse(resp)
		if err != nil {
			logger.WithFields(field.Error(err)).Error("ailed to collect Automation objects of type %q: %v", resource, err)
			errs++
			continue
		}

		logger.Info("Deleting %d objects of type %q...", len(objects), resource)
		for _, o := range objects {
			logger := logger.WithFields(field.F("object", o))
			logger.Debug("Deleting Automation object with id %q...", o.ID)
			_, err := c.Delete(ctx, t, o.ID)
			if err != nil {
				var apiErr api.APIError
				if errors.As(err, &apiErr) {
					if apiErr.StatusCode != http.StatusNotFound {
						logger.WithFields(field.Error(err)).Error("Failed to delete %v with ID %q - rejected by API: %v", resource, o.ID, err)
						errs++
					}
				} else {
					logger.WithFields(field.Error(err)).Error("Failed to delete %v with ID %q - network error: %v", resource, o.ID, err)
					errs++
				}
			}
		}
	}

	if errs > 0 {
		return fmt.Errorf("failed to delete %d Automation object(s)", errs)
	}

	return nil
}
