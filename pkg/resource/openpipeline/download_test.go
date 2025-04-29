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

package openpipeline

import (
	"net/http"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

func TestDownloader_Download(t *testing.T) {
	templateComparer := cmp.Comparer(func(a, b template.Template) bool {
		cA, _ := a.Content()
		cB, _ := b.Content()
		return assert.Empty(t, cmp.Diff(cA, cB))
	})

	expectedOPLogsConfig := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "openpipeline",
			ConfigId: "logs",
		},
		Type:       config.OpenPipelineType{Kind: "logs"},
		Template:   template.NewInMemoryTemplate("logs", "{\n  \"id\": \"logs\"\n}"),
		Parameters: config.Parameters{},
		Skip:       false,
	}

	expectedOPEventsConfig := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "openpipeline",
			ConfigId: "events",
		},
		Type:       config.OpenPipelineType{Kind: "events"},
		Template:   template.NewInMemoryTemplate("events", "{\n  \"id\": \"events\"\n}"),
		Parameters: config.Parameters{},
		Skip:       false,
	}

	t.Run("download openpipeline config works", func(t *testing.T) {

		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/listConfigs.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/openpipeline/v1/configurations", request.URL.Path)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getLogs.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
						ContentType:  "application/json",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/openpipeline/v1/configurations/logs", request.URL.Path)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getEvents.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/openpipeline/v1/configurations/events", request.URL.Path)
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		opClient := openpipeline.NewClient(rest.NewClient(server.URL(), server.Client()))
		result, err := Download(t.Context(), opClient, "project")
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		// expect two configs
		require.Len(t, result["openpipeline"], 2)

		opLogsConfig := result["openpipeline"][0]
		assert.Empty(t, cmp.Diff(expectedOPLogsConfig, opLogsConfig, templateComparer))

		opEventsConfig := result["openpipeline"][1]
		assert.Empty(t, cmp.Diff(expectedOPEventsConfig, opEventsConfig, templateComparer))
	})

	t.Run("no error downloading openpipeline configs with faulty client", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{})
		defer server.Close()

		opClient := openpipeline.NewClient(rest.NewClient(server.URL(), server.FaultyClient()))
		result, err := Download(t.Context(), opClient, "project")
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		// expect no dashboards or notebooks
		require.Len(t, result["openpipeline"], 0)
	})
}
