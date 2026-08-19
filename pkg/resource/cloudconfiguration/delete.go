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

package cloudconfiguration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type DeleteSource interface {
	List(ctx context.Context) (api.ListResponse, error)
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
	slog.InfoContext(ctx, "Deleting Cloud Configurations", log.TypeAttr(config.CloudConfigurationID), slog.Int("count", len(dps)))

	errCount := 0
	for _, dp := range dps {
		if err := d.deleteSingle(ctx, dp); err != nil {
			errCount++
		}
	}
	if errCount > 0 {
		return fmt.Errorf("failed to delete %d cloud configuration(s)", errCount)
	}
	return nil
}

func (d Deleter) deleteSingle(ctx context.Context, dp pointer.DeletePointer) error {
	id := dp.OriginObjectId
	if id == "" {
		slog.WarnContext(ctx, "Skipping cloud configuration deletion: no ID available", log.TypeAttr(dp.Type), slog.String("identifier", dp.Identifier))
		return nil
	}

	_, err := d.source.Delete(ctx, id)
	if err != nil && !api.IsNotFoundError(err) {
		slog.ErrorContext(ctx, "Failed to delete cloud configuration", log.TypeAttr(dp.Type), slog.String("id", id), log.ErrorAttr(err))
		return fmt.Errorf("failed to delete cloud configuration with id '%s': %w", id, err)
	}

	slog.DebugContext(ctx, "Cloud configuration deleted", slog.String("id", id))
	return nil
}

func (d Deleter) DeleteAll(ctx context.Context) error {
	slog.InfoContext(ctx, "Deleting all Cloud Configurations", log.TypeAttr(config.CloudConfigurationID))

	items, err := d.source.List(ctx)
	if err != nil {
		return err
	}

	errCount := 0
	for _, raw := range items.Objects {
		id, err := idFromResponse(api.Response{Data: raw})
		if err != nil {
			errCount++
			continue
		}
		if err := d.deleteSingle(ctx, pointer.DeletePointer{Type: string(config.CloudConfigurationID), OriginObjectId: id}); err != nil {
			errCount++
		}
	}

	if errCount > 0 {
		return fmt.Errorf("failed to delete %d cloud configuration(s)", errCount)
	}
	return nil
}
