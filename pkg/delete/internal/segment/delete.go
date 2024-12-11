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

package segment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type client interface {
	List(ctx context.Context) (segments.Response, error)
	Delete(ctx context.Context, id string) (segments.Response, error)
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
		return fmt.Errorf("failed to delete %d %s objects(s)", errCount, config.Segment{})
	}
	return nil
}

func deleteSingle(ctx context.Context, c client, dp pointer.DeletePointer) error {
	logger := log.WithCtxFields(ctx).WithFields(field.Type(dp.Type), field.Coordinate(dp.AsCoordinate()))

	id := dp.OriginObjectId
	if id == "" {
		var err error
		id, err = findEntryWithExternalID(ctx, c, dp)
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
		return fmt.Errorf("failed to delete entry with id '%s' - %w", id, err)
	}

	logger.Debug("Config with ID '%s' successfully deleted", id)
	return nil
}

func findEntryWithExternalID(ctx context.Context, c client, dp pointer.DeletePointer) (string, error) {
	listResp, err := c.List(ctx)
	if err != nil {
		return "", err
	}

	var items []struct {
		Uid        string `json:"uid"`
		ExternalId string `json:"externalId"`
	}
	if err = json.Unmarshal(listResp.Data, &items); err != nil {
		return "", fmt.Errorf("problem with reading recieved data: %w", err)
	}

	extID, err := idutils.GenerateExternalIDForDocument(dp.AsCoordinate())
	if err != nil {
		return "", fmt.Errorf("unable to generate externalID: %w", err)
	}

	var foundUid []string
	for _, item := range items {
		if item.ExternalId == extID {
			foundUid = append(foundUid, item.Uid)
		}
	}

	switch {
	case len(foundUid) == 0:
		return "", nil
	case len(foundUid) > 1:
		return "", fmt.Errorf("found more than one %s with same externalId (%s); matching IDs: %s", config.SegmentID, extID, foundUid)
	default:
		return foundUid[0], nil
	}
}

func isAPIErrorStatusNotFound(err error) bool {
	var apiErr api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.StatusCode == http.StatusNotFound
}
