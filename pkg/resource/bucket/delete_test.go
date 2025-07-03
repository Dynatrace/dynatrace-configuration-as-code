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

package bucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

//#region Client
type client struct {
	get    func(ctx context.Context, bucketName string) (api.Response, error)
	delete func(ctx context.Context, bucketName string) (api.Response, error)
	list   func(ctx context.Context) (buckets.ListResponse, error)
}

func (c client) Get(ctx context.Context, bucketName string) (api.Response, error) {
	return c.get(ctx, bucketName)
}

func (c client) Delete(ctx context.Context, bucketName string) (api.Response, error) {
	return c.delete(ctx, bucketName)
}

func (c client) List(ctx context.Context) (buckets.ListResponse, error) {
	return c.list(ctx)
}

//#endregion

func TestDeleteBuckets(t *testing.T) {
	activeBucketResponse := []byte(`{
		 "bucketName": "bucket name",
		 "table": "metrics",
		 "displayName": "Default metrics (15 months)",
		 "status": "active",
		 "retentionDays": 462,
		 "metricInterval": "PT1M",
		 "version": 1
	}`)
	deletingBucketResponse := []byte(`{
			 "bucketName": "bucket name",
			 "table": "metrics",
			 "displayName": "Default metrics (15 months)",
			 "status": "deleting",
			 "retentionDays": 462,
			 "metricInterval": "PT1M",
			 "version": 1
		}`)
	updatingBucketResponse := []byte(`{
			 "bucketName": "bucket name",
			 "table": "metrics",
			 "displayName": "Default metrics (15 months)",
			 "status": "updating",
			 "retentionDays": 462,
			 "metricInterval": "PT1M",
			 "version": 1
		}`)

	t.Run("should succeed with one retry for the stable call", func(t *testing.T) {
		getCalls := 0
		c := client{
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				getCalls++
				if getCalls > 1 {
					return api.Response{Data: activeBucketResponse}, nil
				}
				return api.Response{Data: updatingBucketResponse}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{Data: deletingBucketResponse}, nil
			},
		}

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:       "bucket",
				Project:    "project",
				Identifier: "id1",
			},
		}
		errs := bucket.Delete(t.Context(), c, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
		assert.Equal(t, getCalls, 2, "number of GET calls should be 2")
	})

	t.Run("succeeds if object does not exist during delete API call", func(t *testing.T) {
		c := client{
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{Data: activeBucketResponse}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{}, api.APIError{StatusCode: http.StatusNotFound}
			},
		}

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:       "bucket",
				Project:    "project",
				Identifier: "id1",
			},
		}
		errs := bucket.Delete(t.Context(), c, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("succeeds if object does not exist during bucket stable check", func(t *testing.T) {
		c := client{
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{}, api.APIError{StatusCode: http.StatusNotFound}
			},
		}

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:       "bucket",
				Project:    "project",
				Identifier: "id1",
			},
		}
		errs := bucket.Delete(t.Context(), c, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("errors on HTTP error", func(t *testing.T) {
		c := client{
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{Data: activeBucketResponse}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{}, api.APIError{StatusCode: http.StatusBadRequest}
			},
		}

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:       "bucket",
				Project:    "project",
				Identifier: "id1",
			},
		}
		err := bucket.Delete(t.Context(), c, entriesToDelete)
		assert.Error(t, err, "there should be one delete error")
	})

	t.Run("succeeds via 'objectId' identification", func(t *testing.T) {
		objectID := "origin_object_ID"
		c := client{
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				require.Equal(t, bucketName, objectID)
				return api.Response{Data: activeBucketResponse}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				require.Equal(t, bucketName, objectID)
				return api.Response{Data: deletingBucketResponse}, nil
			},
		}

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:           "bucket",
				OriginObjectId: objectID,
			},
		}
		errs := bucket.Delete(t.Context(), c, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})
}
