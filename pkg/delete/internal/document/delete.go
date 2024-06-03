/*
 * @license
 * Copyright 2024 Dynatrace LLC
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
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type client interface {
	List(ctx context.Context, filter string) (documents.ListResponse, error)
	Delete(ctx context.Context, id string) (documents.Response, error)
}

func Delete(ctx context.Context, c client, dps []pointer.DeletePointer) error {
	errCount := 0
	for _, dp := range dps {
		err := deleteSingle(ctx, c, dp)
		if err != nil {
			log.WithCtxFields(ctx).WithFields(field.Type(dp.Type), field.Coordinate(dp.AsCoordinate())).Error("Failed to delete entry: %v", err)
			errCount++
		}
	}
	if errCount > 0 {
		return fmt.Errorf("failed to delete %d document objects(s)", errCount)
	}
	return nil
}

func deleteSingle(ctx context.Context, c client, dp pointer.DeletePointer) error {
	logger := log.WithCtxFields(ctx).WithFields(field.Type(dp.Type), field.Coordinate(dp.AsCoordinate()))
	var id string
	if dp.OriginObjectId != "" {
		id = dp.OriginObjectId
	}
	if id == "" {
		extID, err := idutils.GenerateExternalIDForDocument(dp.AsCoordinate())
		if err != nil {
			return fmt.Errorf("unable to generate externalID: %w", err)
		}

		id, err = tryGetDocumentIDByExternalID(ctx, c, extID)
		if err != nil {
			return err
		}
	}

	if id == "" {
		logger.Debug("no action needed")
		return nil
	}

	_, err := c.Delete(ctx, id)
	if err != nil && !isAPIErrorStatusNotFound(err) {
		logger.Error("Failed to delete entry with id '%s' - %v", id, err)
		return err
	}

	logger.Debug("Config with ID '%s' successfully deleted", id)
	return nil
}

func tryGetDocumentIDByExternalID(ctx context.Context, c client, externalId string) (string, error) {
	switch listResponse, err := c.List(ctx, fmt.Sprintf("externalId=='%s'", externalId)); {
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

func isAPIErrorStatusNotFound(err error) bool {
	var apiErr api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.StatusCode == http.StatusNotFound
}
