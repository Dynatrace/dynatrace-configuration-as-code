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

package segment

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
	List(ctx context.Context) (api.Response, error)
	Delete(ctx context.Context, id string) (api.Response, error)
}

type Deleter struct {
	source DeleteSource
}

func NewDeleter(source DeleteSource) *Deleter {
	return &Deleter{source: source}
}

func (d Deleter) Delete(ctx context.Context, dps []pointer.DeletePointer) error {
	if len(dps) == 0 {
		return nil
	}
	slog.InfoContext(ctx, "Deleting segments", log.TypeAttr(config.SegmentID), slog.Int("count", len(dps)))

	errCount := 0
	for _, dp := range dps {
		err := d.deleteSingle(ctx, dp)
		if err != nil {
			errCount++
		}
	}
	if errCount > 0 {
		return fmt.Errorf("failed to delete %d %s object(s)", errCount, config.SegmentID)
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
		id, err = d.tryGetSegmentIDByExternalID(ctx, extID)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to get segment by external ID", slog.String("externalId", extID), log.ErrorAttr(err))
			return err
		}

		if id == "" {
			logger.DebugContext(ctx, "No segment found with external ID", slog.String("externalId", extID))
			return nil
		}

		logger = logger.With(slog.String("id", id))
	}

	_, err := d.source.Delete(ctx, id)
	if err != nil && !api.IsNotFoundError(err) {
		logger.ErrorContext(ctx, "Failed to delete segment", log.ErrorAttr(err))
		return fmt.Errorf("failed to delete entry with id '%s': %w", id, err)
	}

	logger.DebugContext(ctx, "Segment deleted successfully")
	return nil
}

func (d Deleter) tryGetSegmentIDByExternalID(ctx context.Context, extID string) (string, error) {
	items, err := d.list(ctx)
	if err != nil {
		return "", err
	}

	var foundUUIDs []string
	for _, i := range items {
		if i.ExternalID == extID {
			foundUUIDs = append(foundUUIDs, i.UID)
		}
	}

	switch {
	case len(foundUUIDs) == 0:
		return "", nil
	case len(foundUUIDs) > 1:
		return "", fmt.Errorf("found more than one %s with same externalId (%s); matching IDs: %s", config.SegmentID, extID, foundUUIDs)
	default:
		return foundUUIDs[0], nil
	}
}

func (d Deleter) DeleteAll(ctx context.Context) error {
	slog.InfoContext(ctx, "Deleting all segments", log.TypeAttr(config.SegmentID))

	items, err := d.list(ctx)
	if err != nil {
		return err
	}

	var retErr error
	for _, i := range items {
		err := d.deleteSingle(ctx, pointer.DeletePointer{Type: string(config.SegmentID), OriginObjectId: i.UID})
		if err != nil {
			retErr = errors.Join(retErr, err)
		}
	}

	if retErr != nil {
		slog.ErrorContext(ctx, "Failed to delete all segments", log.ErrorAttr(retErr))
	}

	return retErr
}

type items []struct {
	UID        string `json:"uid"`
	ExternalID string `json:"externalId"`
}

func (d Deleter) list(ctx context.Context) (items, error) {
	listResp, err := d.source.List(ctx)
	if err != nil {
		return nil, err
	}

	var items items
	if err = json.Unmarshal(listResp.Data, &items); err != nil {
		return nil, fmt.Errorf("problem with reading received data: %w", err)
	}

	return items, nil
}
