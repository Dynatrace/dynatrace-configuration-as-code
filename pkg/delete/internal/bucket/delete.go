/*
 * @license
 * Copyright 2023 Dynatrace LLC
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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type client interface {
	Delete(ctx context.Context, id string) (buckets.Response, error)
	List(ctx context.Context) (buckets.ListResponse, error)
}

func Delete(ctx context.Context, c client, entries []pointer.DeletePointer) error {
	logger := log.WithCtxFields(ctx).WithFields(field.Type("bucket"))
	logger.Info(`Deleting %d config(s) of type "bucket"...`, len(entries))

	deleteErrs := 0
	for _, e := range entries {

		logger := logger.WithFields(field.Coordinate(e.AsCoordinate()))

		bucketName := e.OriginObjectId
		if e.OriginObjectId == "" {
			bucketName = idutils.GenerateBucketName(e.AsCoordinate())
		}

		logger.Debug("Deleting bucket '%s'", bucketName)
		_, err := c.Delete(ctx, bucketName)
		if err != nil {
			if !api.IsNotFoundError(err) {
				logger.WithFields(field.Error(err)).ErrorContext(ctx, "Failed to delete Grail Bucket '%s': %v", bucketName, err)
				deleteErrs++
			}

		}
	}

	if deleteErrs > 0 {
		return fmt.Errorf("failed to delete %d Grail Bucket configurations", deleteErrs)
	}

	return nil
}

// AllBuckets collects and deletes objects of type "bucket" using the provided bucketClient.
//
// Parameters:
//   - ctx (context.Context): The context for the operation.
//   - c (bucketClient): The bucketClient used for listing and deleting objects.
//
// Returns:
//   - error: After all deletions where attempted an error is returned if any attempt failed.
func DeleteAll(ctx context.Context, c client) error {
	logger := log.WithCtxFields(ctx).WithFields(field.Type("bucket"))
	logger.Info("Collecting Grail Bucket configurations...")

	response, err := c.List(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to collect Grail Bucket configurations: %v", err)
		return err
	}

	logger.Info("Deleting %d objects of type %q...", len(response.All()), "bucket")
	errCount := 0
	for _, obj := range response.All() {
		var bucketName struct {
			BucketName string `json:"bucketName"`
		}

		if err := json.Unmarshal(obj, &bucketName); err != nil {
			logger.ErrorContext(ctx, "Failed to parse bucket JSON: %v", err)
			errCount++
			continue
		}

		// exclude builtin bucket names, they cannot be deleted anyway
		if buckettools.IsDefault(bucketName.BucketName) {
			continue
		}

		_, err := c.Delete(ctx, bucketName.BucketName)
		if err != nil {
			if !api.IsNotFoundError(err) {
				logger.ErrorContext(ctx, "Failed to delete Grail Bucket '%s': %v", bucketName.BucketName, err)
				errCount++
				continue
			}

		}
	}

	if errCount > 0 {
		return fmt.Errorf("failed to delete %d Grail Bucket configuration(s)", errCount)
	}

	return nil
}
