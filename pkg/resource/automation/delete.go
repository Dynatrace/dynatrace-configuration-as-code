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

package automation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type DeleteSource interface {
	Delete(ctx context.Context, resourceType automation.ResourceType, id string) (api.Response, error)
	List(ctx context.Context, resourceType automation.ResourceType) (api.PagedListResponse, error)
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
	automationResource := entries[0].Type

	slog.InfoContext(ctx, "Deleting automation objects ...", log.TypeAttr(automationResource), slog.Int("count", len(entries)))

	deleteErrs := 0
	for _, e := range entries {
		deleteErrs += d.deleteSingle(ctx, e)
	}

	if deleteErrs > 0 {
		return fmt.Errorf("failed to delete %d automation object(s) of type %q", deleteErrs, automationResource)
	}
	return nil
}

func (d Deleter) deleteSingle(ctx context.Context, dp pointer.DeletePointer) int {
	var logger *slog.Logger
	id := dp.OriginObjectId
	if id == "" {
		id = idutils.GenerateUUIDFromCoordinate(dp.AsCoordinate())
		logger = slog.With(log.CoordinateAttr(dp.AsCoordinate()), slog.String("id", id))
	} else {
		logger = slog.With(log.TypeAttr(dp.Type), slog.String("id", id))
	}

	resourceType, err := automationutils.ClientResourceTypeFromConfigType(config.AutomationResource(dp.Type))
	if err != nil {
		logger.ErrorContext(ctx, "Failed to delete automation object", log.ErrorAttr(err))
		return 1
	}
	_, err = d.source.Delete(ctx, resourceType, id)
	if err != nil {
		if api.IsNotFoundError(err) {
			logger.DebugContext(ctx, "Automation object doesn't exist - no need for action")
			return 0
		}
		logger.ErrorContext(ctx, "Failed to delete automation object", log.ErrorAttr(err))
		return 1
	}
	logger.DebugContext(ctx, "Automation object deleted successfully")
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
func (d Deleter) DeleteAll(ctx context.Context) error {
	slog.InfoContext(ctx, "Deleting all automation objects ... ")
	errCount := 0

	resources := []config.AutomationResource{config.Workflow, config.SchedulingRule, config.BusinessCalendar}
	for _, resource := range resources {
		logger := slog.With(log.TypeAttr(string(resource)))

		t, err := automationutils.ClientResourceTypeFromConfigType(resource)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to delete automation objects", log.ErrorAttr(err))
			errCount++
			continue
		}

		logger.InfoContext(ctx, "Collecting automation objects ...")
		resp, err := d.source.List(ctx, t)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to collect automation objects", log.ErrorAttr(err))
			errCount++
			continue
		}

		objects, err := automationutils.DecodeListResponse(resp)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to collect automation objects", log.ErrorAttr(err))
			errCount++
			continue
		}

		logger.InfoContext(ctx, "Deleting automation objects", slog.Int("count", len(objects)))
		for _, o := range objects {
			errCount += d.deleteSingle(ctx, pointer.DeletePointer{Type: string(resourceTypeToAutomationResource[t]), OriginObjectId: o.ID})
		}
	}

	if errCount > 0 {
		slog.ErrorContext(ctx, "Failed to delete some automation objects", slog.Int("count", errCount))
		return fmt.Errorf("failed to delete %d automation object(s)", errCount)
	}

	return nil
}
