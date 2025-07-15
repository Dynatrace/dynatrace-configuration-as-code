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
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
)

// #region Client
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

var (
	activeBucketResponse = []byte(`{
		 "bucketName": "bucket name",
		 "table": "metrics",
		 "displayName": "Default metrics (15 months)",
		 "status": "active",
		 "retentionDays": 462,
		 "metricInterval": "PT1M",
		 "version": 1
	}`)
	deletingBucketResponse = []byte(`{
		 "bucketName": "bucket name",
		 "table": "metrics",
		 "displayName": "Default metrics (15 months)",
		 "status": "deleting",
		 "retentionDays": 462,
		 "metricInterval": "PT1M",
		 "version": 1
	}`)
	updatingBucketResponse = []byte(`{
		 "bucketName": "bucket name",
		 "table": "metrics",
		 "displayName": "Default metrics (15 months)",
		 "status": "updating",
		 "retentionDays": 462,
		 "metricInterval": "PT1M",
		 "version": 1
	}`)
)

func TestDelete(t *testing.T) {

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
		errs := bucket.NewDeleter(c).Delete(t.Context(), entriesToDelete)
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
		errs := bucket.NewDeleter(c).Delete(t.Context(), entriesToDelete)
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
		errs := bucket.NewDeleter(c).Delete(t.Context(), entriesToDelete)
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
		err := bucket.NewDeleter(c).Delete(t.Context(), entriesToDelete)
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
		errs := bucket.NewDeleter(c).Delete(t.Context(), entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("errors if the stable check errors", func(t *testing.T) {
		c := client{
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{}, errors.New("custom error")
			},
		}

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:       "bucket",
				Project:    "project",
				Identifier: "id1",
			},
		}
		err := bucket.NewDeleter(c).Delete(t.Context(), entriesToDelete)
		assert.Error(t, err)
	})
}

func TestDeleteAll(t *testing.T) {
	listBucketResponse := [][]byte{[]byte(`{"bucketName": "bucket1"}`), []byte(`{"bucketName": "bucket2"}`)}

	t.Run("calls delete of all buckets", func(t *testing.T) {
		deleteCalls := 0
		c := client{
			list: func(ctx context.Context) (buckets.ListResponse, error) {
				return buckets.ListResponse{{Objects: listBucketResponse}}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				deleteCalls++
				return api.Response{Data: deletingBucketResponse}, nil
			},
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{Data: activeBucketResponse}, nil
			},
		}
		err := bucket.NewDeleter(c).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, 2, deleteCalls)
	})

	t.Run("should not call delete of all buckets if the stable check resulted in not existing", func(t *testing.T) {
		deleteCalled := false
		getCalls := 0
		c := client{
			list: func(ctx context.Context) (buckets.ListResponse, error) {
				return buckets.ListResponse{{Objects: listBucketResponse}}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				deleteCalled = true
				return api.Response{}, nil
			},
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				getCalls++
				return api.Response{}, api.APIError{StatusCode: http.StatusNotFound}
			},
		}
		err := bucket.NewDeleter(c).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.False(t, deleteCalled)
		assert.Equal(t, 2, getCalls)
	})

	t.Run("errors if stable check errors", func(t *testing.T) {
		deleteCalled := false
		getCalls := 0
		c := client{
			list: func(ctx context.Context) (buckets.ListResponse, error) {
				return buckets.ListResponse{{Objects: listBucketResponse}}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				deleteCalled = true
				return api.Response{}, nil
			},
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				getCalls++
				return api.Response{}, api.APIError{StatusCode: http.StatusInternalServerError}
			},
		}
		err := bucket.NewDeleter(c).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.False(t, deleteCalled)
		assert.Equal(t, 2, getCalls)
	})

	t.Run("ignores NotFound errors on delete", func(t *testing.T) {
		deleteCalls := 0
		c := client{
			list: func(ctx context.Context) (buckets.ListResponse, error) {
				return buckets.ListResponse{{Objects: listBucketResponse}}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				deleteCalls++
				return api.Response{}, api.APIError{StatusCode: http.StatusNotFound}
			},
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{Data: activeBucketResponse}, nil
			},
		}
		err := bucket.NewDeleter(c).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, 2, deleteCalls)
	})

	t.Run("should error if delete errors", func(t *testing.T) {
		deleteCalls := 0
		c := client{
			list: func(ctx context.Context) (buckets.ListResponse, error) {
				return buckets.ListResponse{{Objects: listBucketResponse}}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				deleteCalls++
				return api.Response{}, api.APIError{StatusCode: http.StatusInternalServerError}
			},
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				return api.Response{Data: activeBucketResponse}, nil
			},
		}
		err := bucket.NewDeleter(c).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.Equal(t, 2, deleteCalls)
	})

	t.Run("errors if list errored", func(t *testing.T) {
		customErr := errors.New("custom error")
		c := client{
			list: func(ctx context.Context) (buckets.ListResponse, error) {
				return buckets.ListResponse{}, customErr
			},
		}

		err := bucket.NewDeleter(c).DeleteAll(t.Context())
		assert.ErrorIs(t, err, customErr)
	})

	t.Run("errors on invalid response data", func(t *testing.T) {
		c := client{
			list: func(ctx context.Context) (buckets.ListResponse, error) {
				return buckets.ListResponse{{Objects: [][]byte{[]byte("invalid json")}}}, nil
			},
		}

		err := bucket.NewDeleter(c).DeleteAll(t.Context())
		assert.Error(t, err)
	})

	t.Run("should not delete default buckets", func(t *testing.T) {
		getCalled := false
		deleteCalled := false
		c := client{
			list: func(ctx context.Context) (buckets.ListResponse, error) {
				return buckets.ListResponse{{Objects: [][]byte{[]byte(`{"bucketName": "default_name"}`)}}}, nil
			},
			get: func(ctx context.Context, bucketName string) (api.Response, error) {
				getCalled = true
				return api.Response{}, nil
			},
			delete: func(ctx context.Context, bucketName string) (api.Response, error) {
				deleteCalled = true
				return api.Response{}, nil
			},
		}

		err := bucket.NewDeleter(c).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.False(t, getCalled)
		assert.False(t, deleteCalled)
	})
}
