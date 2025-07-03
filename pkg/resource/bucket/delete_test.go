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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

func TestDeleteBuckets(t *testing.T) {
	var activeBucketResponse = []byte(`{
		 "bucketName": "bucket name",
		 "table": "metrics",
		 "displayName": "Default metrics (15 months)",
		 "status": "active",
		 "retentionDays": 462,
		 "metricInterval": "PT1M",
		 "version": 1
	}`)

	t.Run("TestDeleteBuckets", func(t *testing.T) {
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

		getCalls := 0
		mux := http.NewServeMux()
		mux.HandleFunc("GET /platform/storage/management/v1/bucket-definitions/{bucketName}", func(rw http.ResponseWriter, req *http.Request) {
			if getCalls > 0 {
				rw.Write(activeBucketResponse)
			} else {
				rw.Write(updatingBucketResponse)
			}
			getCalls++
		})
		mux.HandleFunc("DELETE /platform/storage/management/v1/bucket-definitions/{bucketName}", func(rw http.ResponseWriter, req *http.Request) {
			rw.Write(deletingBucketResponse)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

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

	t.Run("TestDeleteBuckets - No Error if object does not exist during DELETE", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /platform/storage/management/v1/bucket-definitions/{bucketName}", func(rw http.ResponseWriter, req *http.Request) {
			rw.Write(activeBucketResponse)
		})
		mux.HandleFunc("DELETE /platform/storage/management/v1/bucket-definitions/{bucketName}", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusNotFound)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

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

	t.Run("TestDeleteBuckets - No Error if object does not exist", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /platform/storage/management/v1/bucket-definitions/{bucketName}", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusNotFound)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

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

	t.Run("TestDeleteBuckets - Returns Error on HTTP error", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /platform/storage/management/v1/bucket-definitions/{bucketName}", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			rw.Write(activeBucketResponse)
		})
		mux.HandleFunc("DELETE /platform/storage/management/v1/bucket-definitions/{bucketName}", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusBadRequest)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

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

	t.Run("identification via 'objectId'", func(t *testing.T) {
		deletingBucketResponse := []byte(`{
 "bucketName": "bucket name",
 "table": "metrics",
 "displayName": "Default metrics (15 months)",
 "status": "deleting",
 "retentionDays": 462,
 "metricInterval": "PT1M",
 "version": 1
}`)

		mux := http.NewServeMux()
		mux.HandleFunc("GET /platform/storage/management/v1/bucket-definitions/origin_object_ID", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			rw.Write(activeBucketResponse)
		})
		mux.HandleFunc("DELETE /platform/storage/management/v1/bucket-definitions/origin_object_ID", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			rw.Write(deletingBucketResponse)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:           "bucket",
				OriginObjectId: "origin_object_ID",
			},
		}
		errs := bucket.Delete(t.Context(), c, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

}
