//go:build unit

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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

func TestDownloader_Download(t *testing.T) {
	t.Run("download buckets - OK and escapes go-templates", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/platform/storage/management/v1/bucket-definitions":
				wfData, err := os.ReadFile("./testdata/buckets.json")
				assert.NoError(t, err)
				rw.Write(wfData)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()

		baseUrl, err := url.Parse(server.URL)
		assert.NoError(t, err)
		bucketApi := NewDownloadAPI(buckets.NewClient(rest.NewClient(baseUrl, server.Client())))
		result, err := bucketApi.Download(t.Context(), "projectName")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Len(t, result["bucket"], 2) // there should be 2 buckets (default bucket shall be skipped)
		expectedTemplate0 := `{
  "displayName": "{{.displayName}}",
  "metricInterval": "` + "{{`{{`}}.PT1M{{`}}`}}" + `",
  "retentionDays": 462,
  "table": "metrics"
}`
		expectedDisplayName0 := "Default metrics (15 months)"
		assertBucketConfig(t, result["bucket"][0], "bucket_name", expectedTemplate0, &expectedDisplayName0)

		expectedTemplate1 := `{
  "metricInterval": "PT2M",
  "retentionDays": 31,
  "table": "metrics"
}`
		assertBucketConfig(t, result["bucket"][1], "another name", expectedTemplate1, nil)
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
		bucketApi := NewDownloadAPI(buckets.NewClient(rest.NewClient(baseUrl, server.Client())))
		result, err := bucketApi.Download(t.Context(), "projectName")
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
		bucketApi := NewDownloadAPI(buckets.NewClient(rest.NewClient(baseUrl, server.Client())))
		result, err := bucketApi.Download(t.Context(), "projectName")
		assert.Len(t, result, 0)
		assert.NoError(t, err)
	})
}

func assertBucketConfig(t *testing.T, gotConfig config.Config, expectedBucketName, expectedTemplate string, expectedDisplayName *string) {
	assert.Equal(t, coordinate.Coordinate{Project: "projectName", Type: "bucket", ConfigId: expectedBucketName}, gotConfig.Coordinate)
	assert.Equal(t, template.NewInMemoryTemplate(expectedBucketName, expectedTemplate), gotConfig.Template)
	assert.Equal(t, expectedBucketName, gotConfig.OriginObjectId)

	if expectedDisplayName != nil {
		param, exists := gotConfig.Parameters[displayName]
		assert.Truef(t, exists, "should have the parameter %q set", displayName)
		val, err := param.ResolveValue(parameter.ResolveContext{})
		assert.NoError(t, err)
		assert.Equal(t, *expectedDisplayName, val)
	} else {
		_, exists := gotConfig.Parameters[displayName]
		assert.Falsef(t, exists, "should not have the parameter %q set", displayName)
	}
}

func Test_convertObject(t *testing.T) {
	t.Run("test", func(t *testing.T) {

		given := []byte(`
{
            "bucketName": "bucketName",
            "table": "logs",
            "status": "active",
            "retentionDays": 35,
            "version": 2,
            "updatable": false
        }`)

		actual, _ := convertObject(given, "project")

		assert.Equal(t, nil, actual.Parameters["displayName"])
	})
}
