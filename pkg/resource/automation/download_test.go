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

package automation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

// serveFile reads a testdata file and writes it to the response, failing the test on any error.
func serveFile(t *testing.T, rw http.ResponseWriter, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	_, err = rw.Write(data)
	require.NoError(t, err)
}

type serverOptions struct {
	// businessCalsErrStatus, if non-zero, is returned instead of serving the list file.
	businessCalsErrStatus int
	// workflowGetErrStatus maps a workflow ID to the HTTP status that should be returned
	// for its GET route instead of serving the file.
	workflowGetErrStatus map[string]int
}

type serverOption func(*serverOptions)

func withBusinessCalsStatus(status int) serverOption {
	return func(o *serverOptions) { o.businessCalsErrStatus = status }
}

func withWorkflowExportStatus(id string, status int) serverOption {
	return func(o *serverOptions) {
		if o.workflowGetErrStatus == nil {
			o.workflowGetErrStatus = make(map[string]int)
		}
		o.workflowGetErrStatus[id] = status
	}
}

// newAutomationServer registers all List routes and per-ID routes on a ServeMux.
// Use the option helpers to override the default (serve-from-file) behaviour for individual routes.
func newAutomationServer(t *testing.T, opts ...serverOption) *httptest.Server {
	t.Helper()

	o := &serverOptions{}
	for _, opt := range opts {
		opt(o)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /platform/automation/v1/workflows", func(rw http.ResponseWriter, req *http.Request) {
		serveFile(t, rw, "./testdata/listWorkflows.json")
	})
	mux.HandleFunc("GET /platform/automation/v1/business-calendars", func(rw http.ResponseWriter, req *http.Request) {
		if o.businessCalsErrStatus != 0 {
			rw.WriteHeader(o.businessCalsErrStatus)
			return
		}
		serveFile(t, rw, "./testdata/listBusinessCals.json")
	})
	mux.HandleFunc("GET /platform/automation/v1/scheduling-rules", func(rw http.ResponseWriter, req *http.Request) {
		serveFile(t, rw, "./testdata/listSchedulingRules.json")
	})
	mux.HandleFunc("GET /platform/automation/v1/workflows/{id}", func(rw http.ResponseWriter, req *http.Request) {
		id := req.PathValue("id")
		if status, ok := o.workflowGetErrStatus[id]; ok {
			rw.WriteHeader(status)
			return
		}
		serveFile(t, rw, "./testdata/getWorkflow-"+id+".json")
	})

	return httptest.NewServer(mux)
}

// newAutomationClient builds an automation API client pointed at the given server.
func newAutomationClient(t *testing.T, server *httptest.Server) *DownloadAPI {
	t.Helper()
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	return NewDownloadAPI(automation.NewClient(rest.NewClient(serverURL, server.Client())))
}

func TestDownloader_Download(t *testing.T) {
	t.Run("download all resources and escape Jinja", func(t *testing.T) {
		server := newAutomationServer(t)
		defer server.Close()

		result, err := newAutomationClient(t, server).Download(t.Context(), "projectName")
		require.NoError(t, err)
		require.Len(t, result, 3)
		require.Len(t, result[string(config.Workflow)], 3)
		require.Len(t, result[string(config.SchedulingRule)], 6)
		require.Len(t, result[string(config.BusinessCalendar)], 2)

		wfContent, err := result[string(config.Workflow)][0].Template.Content()
		require.NoError(t, err)
		ruleContent, err := result[string(config.SchedulingRule)][0].Template.Content()
		require.NoError(t, err)
		calContent, err := result[string(config.BusinessCalendar)][0].Template.Content()
		require.NoError(t, err)

		type description struct {
			Description string `json:"description"`
		}
		var wfDesc, ruleDesc, calDesc description
		const expectedDescription = "{{`{{`}}execution().id{{`}}`}}"

		require.NoError(t, json.Unmarshal([]byte(wfContent), &wfDesc))
		require.NoError(t, json.Unmarshal([]byte(ruleContent), &ruleDesc))
		require.NoError(t, json.Unmarshal([]byte(calContent), &calDesc))
		assert.Equal(t, expectedDescription, wfDesc.Description)
		assert.Equal(t, expectedDescription, ruleDesc.Description)
		assert.Equal(t, expectedDescription, calDesc.Description)
	})
}

func TestDownloader_Download_FailsToDownloadSpecificResource(t *testing.T) {
	server := newAutomationServer(t, withBusinessCalsStatus(http.StatusBadRequest))
	defer server.Close()

	result, err := newAutomationClient(t, server).Download(t.Context(), "projectName")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Len(t, result[string(config.Workflow)], 3)
	assert.Len(t, result[string(config.SchedulingRule)], 6)
}

func TestDownloader_Download_SingleWorkflowExportFails(t *testing.T) {
	const failingID = "12345678-1234-1234-1234-123456789092"

	server := newAutomationServer(t, withWorkflowExportStatus(failingID, http.StatusInternalServerError))
	defer server.Close()

	result, err := newAutomationClient(t, server).Download(t.Context(), "projectName")
	require.NoError(t, err)
	// The failing workflow is skipped, but the rest are still downloaded
	assert.Len(t, result, 3)
	workflows := result[string(config.Workflow)]
	assert.Len(t, workflows, 2)
	for _, wf := range workflows {
		assert.NotEqual(t, failingID, wf.OriginObjectId)
	}
	assert.Len(t, result[string(config.SchedulingRule)], 6)
	assert.Len(t, result[string(config.BusinessCalendar)], 2)
}

func Test_createTemplateFromRawJSON(t *testing.T) {
	type want struct {
		t    template.Template
		name string
	}

	tests := []struct {
		name  string
		given automationutils.Response
		want  want
	}{
		{
			"sets template ID to object ID",
			automationutils.Response{
				ID:   "42",
				Data: []byte(`{ "id": "42", "workflow_name": "My Workflow", "important": "data" }`),
			},
			want{
				t: template.NewInMemoryTemplate("42", `{
  "important": "data",
  "workflow_name": "My Workflow"
}`),
			},
		},
		{
			"works if reply is not valid JSON",
			automationutils.Response{
				ID:   "42",
				Data: []byte(`{ "id": "42`),
			},
			want{
				t: template.NewInMemoryTemplate("42", `{ "id": "42`),
			},
		},
		{
			"strips modificationInfo from template",
			automationutils.Response{
				ID:   "42",
				Data: []byte(`{ "id": "42", "title": "My Workflow", "modificationInfo": { "createdBy": "user@example.com", "lastModifiedBy": "user@example.com" } }`),
			},
			want{
				t: template.NewInMemoryTemplate("42", `{
  "title": "{{.name}}"
}`),
				name: "My Workflow",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, gotExtractedName := createTemplateFromRawJSON(tt.given, "DOES NOT MATTER FOR TEST", "SOME PROJECT")
			assert.Equalf(t, tt.want.t, gotT, "createTemplateFromRawJSON(%v)", tt.given)
			if tt.want.name != "" {
				require.NotNilf(t, gotExtractedName, "createTemplateFromRawJSON(%v)", tt.given)
				assert.Equalf(t, tt.want.name, *gotExtractedName, "createTemplateFromRawJSON(%v)", tt.given)
			} else {
				assert.Nil(t, gotExtractedName, "expected no name to be extracted")
			}
		})
	}
}
