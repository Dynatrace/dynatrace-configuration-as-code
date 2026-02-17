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

package slo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type DeleteSource interface {
	List(ctx context.Context) (api.PagedListResponse, error)
	Delete(ctx context.Context, id string) (api.Response, error)
}

type Deleter struct {
	source DeleteSource
}

func NewDeleter(source DeleteSource) *Deleter {
	return &Deleter{source}
}

func (d Deleter) Delete(ctx context.Context, dps []pointer.DeletePointer) error {
	if len(dps) == 0 {
		return nil
	}
	slog.InfoContext(ctx, "Deleting SLOs ...", log.TypeAttr(config.ServiceLevelObjectiveID), slog.Int("count", len(dps)))

	errCount := 0
	for _, dp := range dps {
		err := d.deleteSingle(ctx, dp)
		if err != nil {
			errCount++
		}
	}
	if errCount > 0 {
		return fmt.Errorf("failed to delete %d %s object(s)", errCount, config.ServiceLevelObjectiveID)
	}
	return nil
}

func (d Deleter) deleteSingle(ctx context.Context, dp pointer.DeletePointer) error {
	logger := slog.With(log.TypeAttr(dp.Type), slog.String("id", dp.OriginObjectId))
	id := dp.OriginObjectId

	if id == "" {
		coordinate := dp.AsCoordinate()
		logger = slog.With(log.CoordinateAttr(coordinate))
		extID := idutils.GenerateExternalID(coordinate)

		var err error
		id, err = d.findEntryWithExternalID(ctx, extID)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to get SLO by external ID", slog.String("externalId", extID), log.ErrorAttr(err))
			return err
		}

		if id == "" {
			logger.DebugContext(ctx, "No SLO found with external ID", slog.String("externalId", extID))
			return nil
		}

		logger = logger.With(slog.String("id", id))
	}

	_, err := d.source.Delete(ctx, id)
	if err != nil && !api.IsNotFoundError(err) {
		logger.ErrorContext(ctx, "Failed to delete SLO", log.ErrorAttr(err))
		return fmt.Errorf("failed to delete entry with id '%s': %w", id, err)
	}

	logger.DebugContext(ctx, "SLO deleted successfully")
	return nil
}

func (d Deleter) findEntryWithExternalID(ctx context.Context, externalID string) (string, error) {
	items, err := d.source.List(ctx)
	if err != nil {
		return "", err
	}

	var found []entry
	for _, i := range items.All() {
		var e entry
		if err := json.Unmarshal(i, &e); err != nil {
			return "", err
		}
		if e.ExternalID == externalID {
			found = append(found, e)
		}
	}

	switch {
	case len(found) == 0:
		return "", nil
	case len(found) > 1:
		var ids []string
		for _, i := range found {
			ids = append(ids, i.ID)
		}
		return "", fmt.Errorf("found more than one %s with same externalId (%s); matching IDs: %s", config.ServiceLevelObjectiveID, externalID, ids)
	default:
		return found[0].ID, nil
	}
}

func (d Deleter) DeleteAll(ctx context.Context) error {
	slog.InfoContext(ctx, "Deleting all SLOs ...", log.TypeAttr(config.ServiceLevelObjectiveID))

	items, err := d.source.List(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, i := range items.All() {
		var e entry
		if err := json.Unmarshal(i, &e); err != nil {
			errs = append(errs, err)
			continue
		}
		err := d.deleteSingle(ctx, pointer.DeletePointer{Type: string(config.ServiceLevelObjectiveID), OriginObjectId: e.ID})
		if err != nil {
			errs = append(errs, err)
		}
	}

	retErr := errors.Join(errs...)
	if retErr != nil {
		slog.ErrorContext(ctx, "Failed to delete all SLOs", log.ErrorAttr(retErr))
	}

	return retErr
}

type entry struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
}
