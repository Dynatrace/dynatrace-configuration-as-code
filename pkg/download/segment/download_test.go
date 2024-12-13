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

package segment_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	coreLib "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/segment"
)

func TestDownloader_Download(t *testing.T) {
	t.Run("download segments works", func(t *testing.T) {
		t.Setenv(featureflags.Temporary[featureflags.Segments].EnvName(), "true")
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/listResponse.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/storage/filter-segments/v1/filter-segments:lean", request.URL.Path)
					assert.Equal(t, "add-fields=EXTERNALID", request.URL.RawQuery)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/uid_1_getResponse.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/storage/filter-segments/v1/filter-segments/uid_1", request.URL.Path)
					assert.Equal(t, "add-fields=INCLUDES&add-fields=VARIABLES&add-fields=EXTERNALID&add-fields=RESOURCECONTEXT", request.URL.RawQuery)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/uid_2_getResponse.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/storage/filter-segments/v1/filter-segments/uid_2", request.URL.Path)
					assert.Equal(t, "add-fields=INCLUDES&add-fields=VARIABLES&add-fields=EXTERNALID&add-fields=RESOURCECONTEXT", request.URL.RawQuery)
				},
			},
		})
		defer server.Close()

		client := coreLib.NewClient(rest.NewClient(server.URL(), server.Client()))
		result, err := segment.Download(client, "project")

		assert.NoError(t, err)
		assert.Len(t, result, 1)

		require.Len(t, result[string(config.SegmentID)], 2, "all listed segments should be downloaded")
	})

	t.Run("segment without uio is ignored", func(t *testing.T) {
		t.Setenv(featureflags.Temporary[featureflags.Segments].EnvName(), "true")
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/listResponse.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/storage/filter-segments/v1/filter-segments:lean", request.URL.Path)
					assert.Equal(t, "add-fields=EXTERNALID", request.URL.RawQuery)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/uid_1_getResponse_wo_uid.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/storage/filter-segments/v1/filter-segments/uid_1", request.URL.Path)
					assert.Equal(t, "add-fields=INCLUDES&add-fields=VARIABLES&add-fields=EXTERNALID&add-fields=RESOURCECONTEXT", request.URL.RawQuery)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/uid_2_getResponse.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/storage/filter-segments/v1/filter-segments/uid_2", request.URL.Path)
					assert.Equal(t, "add-fields=INCLUDES&add-fields=VARIABLES&add-fields=EXTERNALID&add-fields=RESOURCECONTEXT", request.URL.RawQuery)
				},
			},
		})
		defer server.Close()

		client := coreLib.NewClient(rest.NewClient(server.URL(), server.Client()))
		result, err := segment.Download(client, "project")

		assert.NoError(t, err)
		assert.Len(t, result, 1)

		assert.Len(t, result[string(config.SegmentID)], 1, "all listed segments should be downloaded")
		assert.Equal(t, "uid_2", result[string(config.SegmentID)][0].OriginObjectId)
	})

	t.Run("no error downloading segments with faulty client", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{})
		defer server.Close()

		client := coreLib.NewClient(rest.NewClient(server.URL(), server.FaultyClient()))
		result, err := segment.Download(client, "project")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}
