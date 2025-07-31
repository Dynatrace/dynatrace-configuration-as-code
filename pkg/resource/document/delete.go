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

package document

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type DeleteSource interface {
	List(ctx context.Context, filter string) (documents.ListResponse, error)
	Delete(ctx context.Context, id string) (api.Response, error)
}

type Deleter struct {
	source DeleteSource
}

func NewDeleter(source DeleteSource) *Deleter {
	return &Deleter{source}
}

func (d Deleter) Delete(ctx context.Context, dps []pointer.DeletePointer) error {
	errCount := 0
	for _, dp := range dps {
		err := d.deleteSingle(ctx, dp)
		if err != nil {
			errCount++
		}
	}
	if errCount > 0 {
		return fmt.Errorf("failed to delete %d document object(s)", errCount)
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
		id, err = d.tryGetDocumentIDByExternalID(ctx, extID)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to get document by external ID", slog.String("externalId", extID), log.ErrorAttr(err))
			return err
		}

		if id == "" {
			logger.DebugContext(ctx, "No document found with external ID", slog.String("externalId", extID))
			return nil
		}

		logger = logger.With(slog.String("id", id))
	}

	_, err := d.source.Delete(ctx, id)
	if err != nil && !api.IsNotFoundError(err) {
		logger.ErrorContext(ctx, "Failed to delete document", log.ErrorAttr(err))
		return fmt.Errorf("failed to delete entry with id '%s': %w", id, err)
	}

	logger.DebugContext(ctx, "Document deleted successfully")
	return nil
}

func (d Deleter) tryGetDocumentIDByExternalID(ctx context.Context, externalId string) (string, error) {
	switch listResponse, err := d.source.List(ctx, fmt.Sprintf("externalId=='%s'", externalId)); {
	case err != nil:
		return "", err
	case len(listResponse.Responses) == 0:
		return "", nil
	case len(listResponse.Responses) > 1:
		var ids []string
		for _, r := range listResponse.Responses {
			ids = append(ids, r.ID)
		}
		return "", fmt.Errorf("found more than one document with same externalId (%s); matching document IDs: %s", externalId, ids)
	default:
		return listResponse.Responses[0].ID, nil
	}
}

func (d Deleter) DeleteAll(ctx context.Context) error {
	listResponse, err := d.source.List(ctx, getFilterForAllSupportedDocumentTypes())
	if err != nil {
		return err
	}

	var retErr error
	for _, x := range listResponse.Responses {
		err := d.deleteSingle(ctx, pointer.DeletePointer{Type: x.Type, OriginObjectId: x.ID})
		if err != nil {
			retErr = errors.Join(retErr, err)
		}
	}

	if retErr != nil {
		slog.ErrorContext(ctx, "Failed to delete all document configurations", log.ErrorAttr(retErr))
	}

	return retErr
}

func getFilterForAllSupportedDocumentTypes() string {
	return strings.Join(getMatchersForAllSupportedDocumentTypes(), " or ")
}

func getMatchersForAllSupportedDocumentTypes() []string {
	matchers := make([]string, len(supportedDocumentTypes))
	for i, t := range supportedDocumentTypes {
		matchers[i] = fmt.Sprintf("type='%s'", t)
	}
	return matchers
}
