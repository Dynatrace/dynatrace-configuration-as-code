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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

func TestDownloader_Download(t *testing.T) {
	t.Run("download all resource", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/platform/automation/v1/workflows":
				wfData, _ := os.ReadFile("./testdata/listWorkflows.json")
				rw.Write(wfData)
			case "/platform/automation/v1/business-calendars":
				wfData, _ := os.ReadFile("./testdata/listBusinessCals.json")
				rw.Write(wfData)
			case "/platform/automation/v1/scheduling-rules":
				wfData, _ := os.ReadFile("./testdata/listSchedulingRules.json")
				rw.Write(wfData)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		httpClient := automation.NewClient(rest.NewClient(serverURL, server.Client()))
		automationApi := NewDownloadAPI(httpClient)
		result, err := automationApi.Download(t.Context(), "projectName")
		assert.Len(t, result, 3)
		assert.Len(t, result[string(config.Workflow)], 3)
		assert.Len(t, result[string(config.SchedulingRule)], 6)
		assert.Len(t, result[string(config.BusinessCalendar)], 2)
		assert.NoError(t, err)
	})
}

func TestDownloader_Download_FailsToDownloadSpecificResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/platform/automation/v1/workflows":
			wfData, _ := os.ReadFile("./testdata/listWorkflows.json")
			rw.Write(wfData)
		case "/platform/automation/v1/business-calendars":
			rw.WriteHeader(http.StatusBadRequest)
		case "/platform/automation/v1/scheduling-rules":
			wfData, _ := os.ReadFile("./testdata/listSchedulingRules.json")
			rw.Write(wfData)
		default:
			assert.Fail(t, "unexpect call to server with path "+req.URL.Path)
		}

	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)
	httpClient := automation.NewClient(rest.NewClient(serverURL, server.Client()))
	automationApi := NewDownloadAPI(httpClient)
	result, err := automationApi.Download(t.Context(), "projectName")
	assert.Len(t, result, 2)
	assert.Len(t, result[string(config.Workflow)], 3)
	assert.Len(t, result[string(config.SchedulingRule)], 6)
	assert.NoError(t, err)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, gotExtractedName := createTemplateFromRawJSON(tt.given, "DOES NOT MATTER FOR TEST", "SOME PROJECT")
			assert.Equalf(t, tt.want.t, gotT, "createTemplateFromRawJSON(%v)", tt.given)
			if tt.want.name != "" {
				assert.Equalf(t, tt.want.name, *gotExtractedName, "createTemplateFromRawJSON(%v)", tt.given)
			} else {
				assert.Nil(t, gotExtractedName, "expected no name to be extracted")
			}

		})
	}
}
