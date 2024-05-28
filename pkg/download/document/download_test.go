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

package document

import (
	"net/http"
	"os"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_Download(t *testing.T) {
	templateComparer := cmp.Comparer(func(a, b template.Template) bool {
		cA, _ := a.Content()
		cB, _ := b.Content()
		return assert.Empty(t, cmp.Diff(cA, cB))
	})

	expectedDashboardConfig := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "dashboard-document",
			ConfigId: "12345678-1234-1234-1234-0123456789ab",
		},
		OriginObjectId: "12345678-1234-1234-1234-0123456789ab",
		Type:           config.DocumentType{Kind: config.DashboardKind},
		Template:       template.NewInMemoryTemplate("12345678-1234-1234-1234-0123456789ab", "{}"),
		Parameters: config.Parameters{
			config.NameParameter: &value.ValueParameter{Value: "Getting started"},
		},
		Skip:        false,
		Environment: "",
		Group:       "",
	}

	expectedNotebookConfig := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "notebook-document",
			ConfigId: "23456781-1234-1234-1234-0123456789ab",
		},
		OriginObjectId: "23456781-1234-1234-1234-0123456789ab",
		Type:           config.DocumentType{Kind: config.NotebookKind, Private: true},
		Template:       template.NewInMemoryTemplate("23456781-1234-1234-1234-0123456789ab", "{}"),
		Parameters: config.Parameters{
			config.NameParameter: &value.ValueParameter{Value: "Getting started"},
		},
		Skip:        false,
		Environment: "",
		Group:       "",
	}

	t.Run("download dashboard and notebook documents works", func(t *testing.T) {

		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/listDashboardDocuments.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents", request.URL.Path)
					assert.Equal(t, "filter=type%3D%3D%27dashboard%27", request.URL.RawQuery)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getDashboardDocument.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
						ContentType:  "multipart/form-data;boundary=LGaFEwDyfzC3cW23idF7YWRXxPuNGk",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents/12345678-1234-1234-1234-0123456789ab", request.URL.Path)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/listNotebookDocuments.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents", request.URL.Path)
					assert.Equal(t, "filter=type%3D%3D%27notebook%27", request.URL.RawQuery)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getNotebookDocument.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
						ContentType:  "multipart/form-data;boundary=LGaFEwDyfzC3cW23idF7YWRXxPuNGk",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents/23456781-1234-1234-1234-0123456789ab", request.URL.Path)
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		documentClient := documents.NewClient(rest.NewClient(server.URL(), server.Client()))
		result, err := Download(documentClient, "project")
		assert.NoError(t, err)
		assert.Len(t, result, 2)

		// expect one dashboard
		require.Len(t, result["dashboard-document"], 1)
		dashboardConfig := result["dashboard-document"][0]
		assert.Empty(t, cmp.Diff(expectedDashboardConfig, dashboardConfig, templateComparer))

		// expect one notebook
		require.Len(t, result["notebook-document"], 1)
		notebookConfig := result["notebook-document"][0]
		assert.Empty(t, cmp.Diff(expectedNotebookConfig, notebookConfig, templateComparer))
	})

	t.Run("no error downloading documents with faulty client", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{})
		defer server.Close()

		documentClient := documents.NewClient(rest.NewClient(server.URL(), server.FaultyClient()))
		result, err := Download(documentClient, "project")
		assert.NoError(t, err)
		assert.Len(t, result, 2)

		// expect no dashboards
		require.Len(t, result["dashboard-document"], 0)

		// expect no notebook
		require.Len(t, result["notebook-document"], 0)
	})

	t.Run("notebook download still works if dashboard download fails", func(t *testing.T) {

		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusBadRequest,
						ResponseBody: "{}",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents", request.URL.Path)
					assert.Equal(t, "filter=type%3D%3D%27dashboard%27", request.URL.RawQuery)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/listNotebookDocuments.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents", request.URL.Path)
					assert.Equal(t, "filter=type%3D%3D%27notebook%27", request.URL.RawQuery)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getNotebookDocument.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
						ContentType:  "multipart/form-data;boundary=LGaFEwDyfzC3cW23idF7YWRXxPuNGk",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents/23456781-1234-1234-1234-0123456789ab", request.URL.Path)
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		documentClient := documents.NewClient(rest.NewClient(server.URL(), server.Client()))
		result, err := Download(documentClient, "project")
		assert.NoError(t, err)
		assert.Len(t, result, 2)

		// expect no dashboards
		require.Len(t, result["dashboard-document"], 0)

		// expect one notebook
		require.Len(t, result["notebook-document"], 1)
		notebookConfig := result["notebook-document"][0]
		assert.Empty(t, cmp.Diff(expectedNotebookConfig, notebookConfig, templateComparer))
	})

}
