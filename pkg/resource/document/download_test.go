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

package document_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/document"
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
			Type:     "document",
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
			Type:     "document",
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

	expectedLaunchpad1Config := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "document",
			ConfigId: "1d10690f-7e21-4757-a8bd-bf3a723efc4a",
		},
		OriginObjectId: "1d10690f-7e21-4757-a8bd-bf3a723efc4a",
		Type:           config.DocumentType{Kind: config.LaunchpadKind, Private: true},
		Template:       template.NewInMemoryTemplate("1d10690f-7e21-4757-a8bd-bf3a723efc4a", "{}"),
		Parameters: config.Parameters{
			config.NameParameter: &value.ValueParameter{Value: "My super awesome launchpad"},
		},
		Skip:        false,
		Environment: "",
		Group:       "",
	}
	expectedLaunchpad2Config := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "document",
			ConfigId: "801b0ef7-6d87-4107-9bba-b2f75b5ec290",
		},
		OriginObjectId: "801b0ef7-6d87-4107-9bba-b2f75b5ec290",
		Type:           config.DocumentType{Kind: config.LaunchpadKind, Private: true},
		Template:       template.NewInMemoryTemplate("801b0ef7-6d87-4107-9bba-b2f75b5ec290", "{}"),
		Parameters: config.Parameters{
			config.NameParameter: &value.ValueParameter{Value: "Another super cool Launchpad document"},
		},
		Skip:        false,
		Environment: "",
		Group:       "",
	}

	t.Run("download of all document kinds in one go works", func(t *testing.T) {

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
					assert.Equal(t, "type=='dashboard'", request.URL.Query().Get("filter"))
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getDashboardDocument.txt")
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
					assert.Equal(t, "type=='notebook'", request.URL.Query().Get("filter"))
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getNotebookDocument.txt")
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
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/listLaunchpadDocuments.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents", request.URL.Path)
					assert.Equal(t, "type=='launchpad'", request.URL.Query().Get("filter"))
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getLaunchpadDocument-1.txt")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
						ContentType:  "multipart/form-data;boundary=WYwl3-IPtOFH1PoqAJPJK8NCSfoAmaYvjfxD",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents/1d10690f-7e21-4757-a8bd-bf3a723efc4a", request.URL.Path)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getLaunchpadDocument-2.txt")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
						ContentType:  "multipart/form-data;boundary=7JMn-qLHCElqdRsKLhKfTjCYZPd7oJaWx4tu52bd",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents/801b0ef7-6d87-4107-9bba-b2f75b5ec290", request.URL.Path)
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		documentClient := documents.NewClient(rest.NewClient(server.URL(), server.Client()))
		documentApi := document.NewDownloadAPI(documentClient)
		result, err := documentApi.Download(t.Context(), "project")
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		// expect one dashboard and one notebook and 2 launchpads
		require.Len(t, result["document"], 4)

		dashboardConfig := result["document"][0]
		assert.Empty(t, cmp.Diff(expectedDashboardConfig, dashboardConfig, templateComparer))

		notebookConfig := result["document"][1]
		assert.Empty(t, cmp.Diff(expectedNotebookConfig, notebookConfig, templateComparer))

		launchpad1 := result["document"][2]
		assert.Empty(t, cmp.Diff(expectedLaunchpad1Config, launchpad1, templateComparer))

		launchpad2 := result["document"][3]
		assert.Empty(t, cmp.Diff(expectedLaunchpad2Config, launchpad2, templateComparer))
	})

	t.Run("no error downloading documents with faulty client", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{})
		defer server.Close()

		documentClient := documents.NewClient(rest.NewClient(server.URL(), server.FaultyClient()))
		documentApi := document.NewDownloadAPI(documentClient)
		result, err := documentApi.Download(t.Context(), "project")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.True(t, true)

		// expect no dashboards or notebooks
		require.Len(t, result["document"], 0)
	})

	t.Run("other documents are written even if one fails", func(t *testing.T) {

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
					assert.Equal(t, "type=='dashboard'", request.URL.Query().Get("filter"))
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
					assert.Equal(t, "type=='notebook'", request.URL.Query().Get("filter"))
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getNotebookDocument.txt")
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
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/listLaunchpadDocuments.json")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents", request.URL.Path)
					assert.Equal(t, "type=='launchpad'", request.URL.Query().Get("filter"))
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getLaunchpadDocument-1.txt")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
						ContentType:  "multipart/form-data;boundary=WYwl3-IPtOFH1PoqAJPJK8NCSfoAmaYvjfxD",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents/1d10690f-7e21-4757-a8bd-bf3a723efc4a", request.URL.Path)
				},
			},
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					data, err := os.ReadFile("./testdata/getLaunchpadDocument-2.txt")
					assert.NoError(t, err)

					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: string(data),
						ContentType:  "multipart/form-data;boundary=7JMn-qLHCElqdRsKLhKfTjCYZPd7oJaWx4tu52bd",
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/platform/document/v1/documents/801b0ef7-6d87-4107-9bba-b2f75b5ec290", request.URL.Path)
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		documentClient := documents.NewClient(rest.NewClient(server.URL(), server.Client()))
		documentApi := document.NewDownloadAPI(documentClient)
		result, err := documentApi.Download(t.Context(), "project")
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		// expect one notebook and two launchpads (no dashboard)
		require.Len(t, result["document"], 3)
		notebookConfig := result["document"][0]
		assert.Empty(t, cmp.Diff(expectedNotebookConfig, notebookConfig, templateComparer))

		launchpad1 := result["document"][1]
		assert.Empty(t, cmp.Diff(expectedLaunchpad1Config, launchpad1, templateComparer))

		launchpad2 := result["document"][2]
		assert.Empty(t, cmp.Diff(expectedLaunchpad2Config, launchpad2, templateComparer))
	})

}
