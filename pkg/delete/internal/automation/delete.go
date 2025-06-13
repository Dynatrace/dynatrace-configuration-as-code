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
	"context"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type client interface {
	Delete(ctx context.Context, resourceType automation.ResourceType, id string) (api.Response, error)
	List(ctx context.Context, resourceType automation.ResourceType) (api.PagedListResponse, error)
}

func Delete(ctx context.Context, c client, automationResource config.AutomationResource, entries []pointer.DeletePointer) error {
	logger := log.WithFields(field.Type(string(automationResource)))
	logger.InfoContext(ctx, "Deleting %d config(s) of type %q...", len(entries), automationResource)

	deleteErrs := 0
	for _, e := range entries {
		deleteErrs += deleteSingle(ctx, c, e)
	}

	if deleteErrs > 0 {
		return fmt.Errorf("failed to delete %d Automation objects(s) of type %q", deleteErrs, automationResource)
	}
	return nil
}

func deleteSingle(ctx context.Context, c client, dp pointer.DeletePointer) int {
	logger := log.WithFields(field.Type(dp.Type), field.Coordinate(dp.AsCoordinate()))

	id := dp.OriginObjectId
	if id == "" {
		id = idutils.GenerateUUIDFromCoordinate(dp.AsCoordinate())
	}

	logger.DebugContext(ctx, "Deleting %v with id %q.", dp.Type, id)

	resourceType, err := automationutils.ClientResourceTypeFromConfigType(config.AutomationResource(dp.Type))
	if err != nil {
		logger.WithFields(field.Error(err)).ErrorContext(ctx, "Failed to delete %v with ID %q: %v", dp.Type, id, err)
		return 1
	}
	_, err = c.Delete(ctx, resourceType, id)
	if err != nil {
		if !api.IsNotFoundError(err) {
			logger.WithFields(field.Error(err)).ErrorContext(ctx, "Failed to delete %v with ID '%s': %v", dp.Type, id, err)
			return 1
		}
	}
	logger.DebugContext(ctx, "Automation object with id %q deleted", id)
	return 0
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
	errCount := 0

	resources := []config.AutomationResource{config.Workflow, config.SchedulingRule, config.BusinessCalendar}
	for _, resource := range resources {
		logger := log.WithFields(field.Type(string(resource)))

		t, err := automationutils.ClientResourceTypeFromConfigType(resource)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to delete Automation objects of type '%s': %v", resource, err)
			errCount++
			continue
		}

		logger.InfoContext(ctx, "Collecting Automation objects of type %q...", resource)
		resp, err := c.List(ctx, t)
		if err != nil {
			logger.WithFields(field.Error(err)).ErrorContext(ctx, "Failed to collect Automation objects of type '%s': %v", resource, err)
			errCount++
			continue
		}

		objects, err := automationutils.DecodeListResponse(resp)
		if err != nil {
			logger.WithFields(field.Error(err)).ErrorContext(ctx, "Failed to collect Automation objects of type '%s': %v", resource, err)
			errCount++
			continue
		}

		logger.InfoContext(ctx, "Deleting %d objects of type %q...", len(objects), resource)
		for _, o := range objects {
			errCount += deleteSingle(ctx, c, pointer.DeletePointer{Type: automationTypesToResources[t], OriginObjectId: o.ID})
		}
	}

	if errCount > 0 {
		return fmt.Errorf("failed to delete %d Automation object(s)", errCount)
	}

	return nil
}

var automationTypesToResources = map[automation.ResourceType]string{
	automation.Workflows:         "workflow",
	automation.BusinessCalendars: "business-calendar",
	automation.SchedulingRules:   "scheduling-rule",
}
