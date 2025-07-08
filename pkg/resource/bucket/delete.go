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

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/buckettools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type DeleteSource interface {
	Get(ctx context.Context, bucketName string) (api.Response, error)
	Delete(ctx context.Context, id string) (api.Response, error)
	List(ctx context.Context) (buckets.ListResponse, error)
}

type Deleter struct {
	bucketSource DeleteSource
}

func NewDeleter(bucketSource DeleteSource) *Deleter {
	return &Deleter{bucketSource: bucketSource}
}

type bucketDelete struct {
	bucketName string
	coordinate *coordinate.Coordinate
}

func (d Deleter) Delete(ctx context.Context, entries []pointer.DeletePointer) error {
	logger := log.With(log.TypeAttr("bucket"))
	bucketDeletes := deletePointerToBucketDelete(entries)

	if errorCount := d.delete(ctx, bucketDeletes, logger); errorCount > 0 {
		return fmt.Errorf("failed to delete %d Grail bucket configurations", errorCount)
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
	logger := log.With(log.TypeAttr("bucket"))
	logger.InfoContext(ctx, "Collecting Grail bucket configurations...")

	response, err := d.bucketSource.List(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to collect Grail bucket configurations: %v", err)
		return err
	}

	bucketDeletes, errorCountParse := responsesToBucketDelete(ctx, response.All(), logger)
	errorCountDelete := d.delete(ctx, bucketDeletes, logger)
	errorCount := errorCountParse + errorCountDelete

	if errorCount > 0 {
		return fmt.Errorf("failed to delete %d Grail bucket configuration(s)", errorCount)
	}

	return nil
}

func deletePointerToBucketDelete(entries []pointer.DeletePointer) []bucketDelete {
	bucketsDeletes := make([]bucketDelete, len(entries))
	for i, e := range entries {
		cord := e.AsCoordinate()
		bucketName := e.OriginObjectId

		if e.OriginObjectId == "" {
			bucketName = idutils.GenerateBucketName(cord)
		}

		bucketsDeletes[i] = bucketDelete{
			bucketName: bucketName,
			coordinate: &cord,
		}
	}
	return bucketsDeletes
}

func responsesToBucketDelete(ctx context.Context, bucketResponses [][]byte, logger *log.Slogger) ([]bucketDelete, int) {
	var bucketName struct {
		BucketName string `json:"bucketName"`
	}
	bucketDeletes := make([]bucketDelete, 0)
	errCount := 0

	for _, obj := range bucketResponses {
		if err := json.Unmarshal(obj, &bucketName); err != nil {
			logger.ErrorContext(ctx, "Failed to parse Grail bucket JSON: %v", err)
			errCount++
			continue
		}
		bucketDeletes = append(bucketDeletes, bucketDelete{
			bucketName: bucketName.BucketName,
		})
	}
	return bucketDeletes, errCount
}

func (d Deleter) delete(ctx context.Context, bucketDeletes []bucketDelete, baseLogger *log.Slogger) int {
	errorCount := 0
	baseLogger.InfoContext(ctx, `Deleting %d config(s) of type 'bucket'...`, len(bucketDeletes))

	for _, bucketDeleteEntry := range bucketDeletes {
		bucketName := bucketDeleteEntry.bucketName
		// exclude builtin bucket names, they cannot be deleted anyway
		if buckettools.IsDefault(bucketName) {
			continue
		}

		logger := baseLogger
		if bucketDeleteEntry.coordinate != nil {
			logger = logger.With(log.CoordinateAttr(*bucketDeleteEntry.coordinate))
		}

		logger.DebugContext(ctx, "Deleting Grail buckets '%s'", bucketName)
		bucketExists, err := buckets.AwaitActiveOrNotFound(ctx, d.bucketSource, bucketName, maxRetryDuration, durationBetweenRetries)

		if err != nil {
			logger.With(log.ErrorAttr(err)).ErrorContext(ctx, "Failed to delete Grail buckets '%s': %v", bucketName, err)
			errorCount++
			continue
		}

		if !bucketExists {
			// bucket already deleted
			continue
		}

		_, err = d.bucketSource.Delete(ctx, bucketName)

		if err != nil && !api.IsNotFoundError(err) {
			logger.With(log.ErrorAttr(err)).ErrorContext(ctx, "Failed to delete Grail bucket '%s': %v", bucketName, err)
			errorCount++
		}
	}
	return errorCount
}
