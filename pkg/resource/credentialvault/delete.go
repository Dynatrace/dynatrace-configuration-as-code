/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package credentialvault

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
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
	slog.InfoContext(ctx, "Deleting credential vault entries", log.TypeAttr(config.CredentialVaultID), slog.Int("count", len(dps)))

	errCount := 0
	for _, dp := range dps {
		err := d.deleteSingle(ctx, dp)
		if err != nil {
			errCount++
		}
	}
	if errCount > 0 {
		return fmt.Errorf("failed to delete %d credential vault entries", errCount)
	}
	return nil
}

func (d Deleter) deleteSingle(ctx context.Context, dp pointer.DeletePointer) error {
	logger := slog.With(log.TypeAttr(dp.Type), slog.String("id", dp.OriginObjectId))
	id := dp.OriginObjectId

	if id == "" {
		coordinate := dp.AsCoordinate()
		logger = slog.With(log.CoordinateAttr(coordinate))
		name := dp.Identifier

		var err error
		id, err = findEntryByName(ctx, d.source, name)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to get credential vault entry by name", slog.String("name", name), log.ErrorAttr(err))
			return err
		}

		if id == "" {
			logger.DebugContext(ctx, "No credential vault item found with name", slog.String("name", name))
			return nil
		}

		logger = logger.With(slog.String("id", id))
	}

	_, err := d.source.Delete(ctx, id)
	if err != nil && !api.IsNotFoundError(err) {
		logger.ErrorContext(ctx, "Failed to delete credential vault entry", log.ErrorAttr(err))
		return fmt.Errorf("failed to delete entry with id '%s': %w", id, err)
	}

	logger.DebugContext(ctx, "Credential vault entry deleted successfully")
	return nil
}

func (d Deleter) DeleteAll(ctx context.Context) error {
	slog.InfoContext(ctx, "Deleting all credential vault entries", log.TypeAttr(config.CredentialVaultID))

	items, err := d.source.List(ctx)
	if err != nil {
		return err
	}

	errCount := 0
	for _, i := range items.All() {
		var e entry
		if err := json.Unmarshal(i, &e); err != nil {
			errCount++
			continue
		}
		err := d.deleteSingle(ctx, pointer.DeletePointer{Type: string(config.CredentialVaultID), OriginObjectId: e.ID})
		if err != nil {
			errCount++
		}
	}

	if errCount > 0 {
		slog.ErrorContext(ctx, "Failed to delete some credential vault entries", slog.Int("count", errCount))
		return fmt.Errorf("failed to delete %d credential vault entries", errCount)
	}

	return nil
}
