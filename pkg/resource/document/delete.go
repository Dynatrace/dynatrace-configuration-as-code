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

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type source interface {
	List(ctx context.Context, filter string) (documents.ListResponse, error)
	Delete(ctx context.Context, id string) (api.Response, error)
}

type Deleter struct {
	documentSource source
}

func NewDeleter(documentSource source) *Deleter {
	return &Deleter{documentSource}
}

func (d Deleter) Delete(ctx context.Context, dps []pointer.DeletePointer) error {
	errCount := 0
	for _, dp := range dps {
		err := d.deleteSingle(ctx, dp)
		if err != nil {
			log.With(log.TypeAttr(dp.Type), log.CoordinateAttr(dp.AsCoordinate())).ErrorContext(ctx, "Failed to delete entry: %v", err)
			errCount++
		}
	}
	if errCount > 0 {
		return fmt.Errorf("failed to delete %d document objects(s)", errCount)
	}
	return nil
}

func (d Deleter) deleteSingle(ctx context.Context, dp pointer.DeletePointer) error {
	logger := log.With(log.TypeAttr(dp.Type), log.CoordinateAttr(dp.AsCoordinate()))
	var id string
	if dp.OriginObjectId != "" {
		id = dp.OriginObjectId
	}
	if id == "" {
		var err error
		extID := idutils.GenerateExternalID(dp.AsCoordinate())

		id, err = d.tryGetDocumentIDByExternalID(ctx, extID)
		if err != nil {
			return err
		}
	}

	if id == "" {
		logger.DebugContext(ctx, "no action needed")
		return nil
	}

	_, err := d.documentSource.Delete(ctx, id)
	if err != nil && !api.IsNotFoundError(err) {
		return fmt.Errorf("failed to delete entry with id '%s': %w", id, err)
	}

	logger.DebugContext(ctx, "Config with ID '%s' successfully deleted", id)
	return nil
}

func (d Deleter) tryGetDocumentIDByExternalID(ctx context.Context, externalId string) (string, error) {
	switch listResponse, err := d.documentSource.List(ctx, fmt.Sprintf("externalId=='%s'", externalId)); {
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
	listResponse, err := d.documentSource.List(ctx, fmt.Sprintf("type='%s' or type='%s'", documents.Dashboard, documents.Notebook))
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
	return retErr
}
