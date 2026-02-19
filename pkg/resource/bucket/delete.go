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

package bucket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/buckettools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type DeleteSource interface {
	Get(ctx context.Context, bucketName string) (api.Response, error)
	Delete(ctx context.Context, id string) (api.Response, error)
	List(ctx context.Context) (buckets.ListResponse, error)
}

type Deleter struct {
	source DeleteSource
}

func NewDeleter(source DeleteSource) *Deleter {
	return &Deleter{source: source}
}

type deleteItem struct {
	bucketName string
	coordinate *coordinate.Coordinate // optional. Only used for logs.
}

func (d Deleter) Delete(ctx context.Context, entries []pointer.DeletePointer) error {
	if len(entries) == 0 {
		return nil
	}
	deleteItems := convertDeletePointerToDeleteItem(entries)
	logger := slog.With(log.TypeAttr(config.BucketTypeID))
	logger.InfoContext(ctx, "Deleting Grail buckets", slog.Int("count", len(deleteItems)))

	if errorCount := d.delete(ctx, deleteItems, logger); errorCount > 0 {
		return fmt.Errorf("failed to delete %d Grail buckets", errorCount)
	}

	return nil
}

// DeleteAll collects and deletes objects of type "bucket".
//
// Parameters:
//   - ctx (context.Context): The context for the operation.
//
// Returns:
//   - error: After all deletions where attempted an error is returned if any attempt failed.
func (d Deleter) DeleteAll(ctx context.Context) error {
	logger := slog.With(log.TypeAttr(config.BucketTypeID))
	logger.InfoContext(ctx, "Deleting all Grail buckets")

	response, err := d.source.List(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to collect Grail buckets", log.ErrorAttr(err))
		return err
	}

	deleteItems, errorCountParse := parseResponseToDeleteItem(ctx, response.All(), logger)
	errorCountDelete := d.delete(ctx, deleteItems, logger)
	errorCount := errorCountParse + errorCountDelete

	if errorCount > 0 {
		logger.ErrorContext(ctx, "Failed to delete some Grail buckets", slog.Int("count", errorCount))
		return fmt.Errorf("failed to delete %d Grail buckets", errorCount)
	}

	return nil
}

func convertDeletePointerToDeleteItem(entries []pointer.DeletePointer) []deleteItem {
	deleteItems := make([]deleteItem, len(entries))
	for i, e := range entries {
		cord := e.AsCoordinate()
		bucketName := e.OriginObjectId

		if e.OriginObjectId == "" {
			bucketName = idutils.GenerateBucketName(cord)
		}

		deleteItems[i] = deleteItem{
			bucketName: bucketName,
			coordinate: &cord,
		}
	}
	return deleteItems
}

func parseResponseToDeleteItem(ctx context.Context, bucketResponses [][]byte, logger *slog.Logger) ([]deleteItem, int) {
	var bucketName struct {
		BucketName string `json:"bucketName"`
	}
	deleteItems := make([]deleteItem, 0)
	errCount := 0

	for _, obj := range bucketResponses {
		if err := json.Unmarshal(obj, &bucketName); err != nil {
			logger.ErrorContext(ctx, "Failed to parse Grail bucket JSON", log.ErrorAttr(err))
			errCount++
			continue
		}
		deleteItems = append(deleteItems, deleteItem{
			bucketName: bucketName.BucketName,
		})
	}
	return deleteItems, errCount
}

func (d Deleter) delete(ctx context.Context, deleteItems []deleteItem, baseLogger *slog.Logger) int {
	errorCount := 0

	for _, delItem := range deleteItems {
		bucketName := delItem.bucketName
		// exclude builtin bucket names, they cannot be deleted anyway
		if buckettools.IsDefault(bucketName) {
			continue
		}

		logger := baseLogger.With(slog.String("name", bucketName))
		if delItem.coordinate != nil {
			logger = logger.With(log.CoordinateAttr(*delItem.coordinate))
		}

		logger.DebugContext(ctx, "Deleting Grail bucket")

		bucketExists, err := buckets.AwaitActiveOrNotFound(ctx, d.source, bucketName, maxRetryDuration, durationBetweenRetries)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to determine state of Grail bucket", log.ErrorAttr(err))
			errorCount++
			continue
		}

		if !bucketExists {
			logger.DebugContext(ctx, "Grail bucket doesn't exist - no need for action")
			continue
		}

		_, err = d.source.Delete(ctx, bucketName)
		if err != nil && !api.IsNotFoundError(err) {
			logger.ErrorContext(ctx, "Failed to delete Grail bucket", log.ErrorAttr(err))
			errorCount++
		}
	}
	return errorCount
}
