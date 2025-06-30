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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
)

func TestDeleteBuckets(t *testing.T) {
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

		getCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "bucket-definitions") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/project_id1"))
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				return
			}
			if req.Method == http.MethodGet && getCalls < 5 {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/project_id1"))
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				getCalls++
				return
			} else if req.Method == http.MethodGet {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))
		a := bucket.NewDeleter(c)
		entriesToDelete := []pointer.DeletePointer{
			{
				Type:       "bucket",
				Project:    "project",
				Identifier: "id1",
			},
		}

		errs := a.Delete(t.Context(), entriesToDelete)

		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteBuckets - No Error if object does not exist", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "bucket-definitions") {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))
		a := bucket.NewDeleter(c)

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:       "bucket",
				Project:    "project",
				Identifier: "id1",
			},
		}

		errs := a.Delete(t.Context(), entriesToDelete)

		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteBuckets - Returns Error on HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "bucket-definitions") {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))
		a := bucket.NewDeleter(c)

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:       "bucket",
				Project:    "project",
				Identifier: "id1",
			},
		}

		err := a.Delete(t.Context(), entriesToDelete)

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

		getCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "bucket-definitions") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/origin_object_ID"))
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				return
			}
			if req.Method == http.MethodGet && getCalls < 5 {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/origin_object_ID"))
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				getCalls++
				return
			} else if req.Method == http.MethodGet {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))
		a := bucket.NewDeleter(c)

		entriesToDelete := []pointer.DeletePointer{
			{
				Type:           "bucket",
				OriginObjectId: "origin_object_ID",
			},
		}

		errs := a.Delete(t.Context(), entriesToDelete)

		assert.Empty(t, errs, "errors should be empty")
	})
}

func TestDeleteAll(t *testing.T) {

	t.Run("one listed bucket is deleted", func(t *testing.T) {
		deletingBucketResponse := []byte(`{
 "bucketName": "bucket-name",
 "table": "metrics",
 "displayName": "Default metrics (15 months)",
 "status": "deleting",
 "retentionDays": 462,
 "metricInterval": "PT1M",
 "version": 1
}`)
		listBucketsResponse := []byte(`{
"buckets": [
	{
		"bucketName": "bucket-name",
 		"table": "metrics",
 		"displayName": "Default metrics (15 months)",
 		"status": "deleting",
 		"retentionDays": 462,
 		"metricInterval": "PT1M",
 		"version": 1
	}
]}`)

		getCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet && req.RequestURI == "/platform/storage/management/v1/bucket-definitions" {
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(listBucketsResponse)
				return
			}

			if req.Method == http.MethodDelete && req.RequestURI == "/platform/storage/management/v1/bucket-definitions/bucket-name" {
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				return
			}
			if req.Method == http.MethodGet && req.RequestURI == "/platform/storage/management/v1/bucket-definitions/bucket-name" {
				if getCalls < 5 {
					rw.WriteHeader(http.StatusOK)
					_, _ = rw.Write(deletingBucketResponse)
					getCalls++
					return
				} else {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))
		a := bucket.NewDeleter(c)

		errs := a.DeleteAll(t.Context())

		assert.Empty(t, errs, "errors should be empty")
	})
	t.Run("default bucket is skipped", func(t *testing.T) {
		listBucketsResponse := []byte(`{
"buckets": [
	{
		"bucketName": "default_bucket",
 		"table": "metrics",
 		"displayName": "Default metrics (15 months)",
 		"status": "deleting",
 		"retentionDays": 462,
 		"metricInterval": "PT1M",
 		"version": 1
	}
]}`)

		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet && req.RequestURI == "/platform/storage/management/v1/bucket-definitions" {
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(listBucketsResponse)
				return
			}

			if req.Method == http.MethodDelete && req.RequestURI == "/platform/storage/management/v1/bucket-definitions/default_bucket" {
				assert.Fail(t, "Default buckets are skipped and should not be deleted")
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))
		a := bucket.NewDeleter(c)

		errs := a.DeleteAll(t.Context())

		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("deletion continues on error", func(t *testing.T) {
		deletingBucketResponse := []byte(`{
 "bucketName": "bucket-name",
 "table": "metrics",
 "displayName": "Default metrics (15 months)",
 "status": "deleting",
 "retentionDays": 462,
 "metricInterval": "PT1M",
 "version": 1
}`)
		listBucketsResponse := []byte(`{
"buckets": [
	{
		"bucketName": "invalid-bucket",
 		"table": "metrics",
 		"displayName": "Default metrics (15 months)",
 		"status": "deleting",
 		"retentionDays": 462,
 		"metricInterval": "PT1M",
 		"version": 1
	},
	{
		"bucketName": "bucket-name",
 		"table": "metrics",
 		"displayName": "Default metrics (15 months)",
 		"status": "deleting",
 		"retentionDays": 462,
 		"metricInterval": "PT1M",
 		"version": 1
	}
]}`)

		getCalls := 0
		deleteCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet && req.RequestURI == "/platform/storage/management/v1/bucket-definitions" {
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(listBucketsResponse)
				return
			}

			if req.Method == http.MethodDelete && req.RequestURI == "/platform/storage/management/v1/bucket-definitions/invalid-bucket" {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			if req.Method == http.MethodDelete && req.RequestURI == "/platform/storage/management/v1/bucket-definitions/bucket-name" {
				deleteCalled = true
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				return
			}
			if req.Method == http.MethodGet && req.RequestURI == "/platform/storage/management/v1/bucket-definitions/bucket-name" {
				if getCalls < 5 {
					rw.WriteHeader(http.StatusOK)
					_, _ = rw.Write(deletingBucketResponse)
					getCalls++
					return
				} else {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))
		a := bucket.NewDeleter(c)

		_ = a.DeleteAll(t.Context())

		assert.True(t, deleteCalled)
	})
}
