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
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestDownloader_Download(t *testing.T) {
	t.Run("download buckets - OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/platform/storage/management/v1/bucket-definitions":
				wfData, _ := os.ReadFile("./testdata/buckets.json")
				rw.Write(wfData)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()

		baseUrl, _ := url.Parse(server.URL)
		bucketClient := buckets.NewClient(rest.NewClient(baseUrl, server.Client()))
		downloader := NewDownloader(bucketClient)
		result, err := downloader.Download("projectName")
		assert.Len(t, result, 1)
		assert.Len(t, result["bucket"], 2) // there should be 2 buckets (default bucket shall be skipped)
		assert.Equal(t, coordinate.Coordinate{"projectName", "bucket", "372852f8-86d5-3d6d-8feb-6b537aef6bf4"}, result["bucket"][0].Coordinate)
		assert.Equal(t, &value.ValueParameter{"Default metrics (15 months)"}, result["bucket"][0].Parameters[config.NameParameter])
		assert.Equal(t, coordinate.Coordinate{"projectName", "bucket", "6e2cd4d7-9ac5-3294-a9ce-277da9bd200c"}, result["bucket"][1].Coordinate)
		assert.Equal(t, &value.ValueParameter{"another name"}, result["bucket"][1].Parameters[config.NameParameter])

		assert.NoError(t, err)
	})

	t.Run("download buckets - fetch buckets fails - no error returned", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/platform/storage/management/v1/bucket-definitions":
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()

		baseUrl, _ := url.Parse(server.URL)
		bucketClient := buckets.NewClient(rest.NewClient(baseUrl, server.Client()))
		downloader := NewDownloader(bucketClient)
		result, err := downloader.Download("projectName")
		assert.Len(t, result, 0)
		assert.NoError(t, err)
	})

	t.Run("download buckets - fetch buckets fails on API error - no error returned", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/platform/storage/management/v1/bucket-definitions":
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte("{}"))
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()

		baseUrl, _ := url.Parse(server.URL)
		bucketClient := buckets.NewClient(rest.NewClient(baseUrl, server.Client()))
		downloader := NewDownloader(bucketClient)
		result, err := downloader.Download("projectName")
		assert.Len(t, result, 0)
		assert.NoError(t, err)
	})
}
