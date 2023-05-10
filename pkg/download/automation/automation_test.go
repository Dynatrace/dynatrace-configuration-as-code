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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDownloader_Download(t *testing.T) {
	t.Run("download all automation resources", func(t *testing.T) {
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
				t.Fatal("NO")
			}
		}))
		defer server.Close()

		httpClient := automation.NewClient(server.URL, server.Client())
		downloader := NewDownloader(httpClient)

		actual, err := downloader.Download("projectName")

		assert.Len(t, actual, 3)
		assert.Len(t, actual[string(config.Workflow)], 4)
		assert.Len(t, actual[string(config.SchedulingRule)], 6)
		assert.Len(t, actual[string(config.BusinessCalendar)], 2)

		assert.NoError(t, err)
	})

	t.Run("jinja must be excaped", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/platform/automation/v1/workflows":
				rw.Write([]byte(`
{
    "count": 1,
    "results":[{
        "id": "12345678-1234-1234-1234-123456789999",
        "title": "jinja workflow",
        "tasks": {
            "runJavascript1": {
                "name": "run_javascript_1",
                "input": {
                    "script": "// optional import of sdk modules\nimport { metadataClient } from '@dynatrace-sdk/client-metadata';\nimport { executionsClient } from '@dynatrace-sdk/client-automation';\n\nexport default async function ({ execution_id }) {\n  // your code goes here\n  const me = await metadataClient.getUserInfo();\n  console.log('Automated script execution on behalf of', me.userName);\n\n  console.log({{ event() }})\n  // get the current execution\n  const ex = await executionsClient.getExecution({ id: execution_id });\n\n  return { ...me, triggeredBy: ex.trigger };\n}"
                },
                "action": "dynatrace.automations:run-javascript",
                "position": {
                    "x": 0,
                    "y": 1
                },
                "description": "Build a custom task running js Code",
                "predecessors": []
            }
        },
        "taskDefaults": {},
        "usages": [],
        "lastExecution": null,
        "description": "",
        "labels": {},
        "version": 1,
        "actor": "ed6a9c8f-06f0-4508-9b8e-c47bbe67c83d",
        "owner": "ed6a9c8f-06f0-4508-9b8e-c47bbe67c83d",
        "isPrivate": true,
        "triggerType": "Manual",
        "schemaVersion": 3,
        "trigger": {}
    }]
}
`))
			default:
				t.Fatal("NO")
			}
		}))
		defer server.Close()

		httpClient := automation.NewClient(server.URL, server.Client())
		downloader := NewDownloader(httpClient)

		actual, err := downloader.Download("projectName", config.AutomationType{Resource: config.Workflow})

		for _, r := range actual {
			for _, c := range r {
				assert.Contains(t, c.Template.Content(), `\{\{`)
				assert.Contains(t, c.Template.Content(), `\}\}`)
			}
		}
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
	httpClient := automation.NewClient(server.URL, server.Client())
	downloader := NewDownloader(httpClient)
	result, err := downloader.Download("projectName")
	assert.Len(t, result, 2)
	assert.Len(t, result[string(config.Workflow)], 4)
	assert.Len(t, result[string(config.SchedulingRule)], 6)
	assert.NoError(t, err)
}

func TestDownloader_Download_Specific_ResourceTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/platform/automation/v1/workflows":
			wfData, _ := os.ReadFile("./testdata/listWorkflows.json")
			rw.Write(wfData)
		case "/platform/automation/v1/business-calendars":
			wfData, _ := os.ReadFile("./testdata/listBusinessCals.json")
			rw.Write(wfData)
		case "/platform/automation/v1/scheduling-rules":
			assert.Fail(t, "unexpect call to server with path "+req.URL.Path)
		default:
			assert.Fail(t, "unexpect call to server with path "+req.URL.Path)
		}

	}))
	defer server.Close()
	httpClient := automation.NewClient(server.URL, server.Client())
	downloader := NewDownloader(httpClient)
	result, err := downloader.Download("projectName",
		config.AutomationType{Resource: config.Workflow}, config.AutomationType{Resource: config.BusinessCalendar})
	assert.Len(t, result, 2)
	assert.Len(t, result[string(config.Workflow)], 4)
	assert.Len(t, result[string(config.SchedulingRule)], 0)
	assert.Len(t, result[string(config.BusinessCalendar)], 2)
	assert.NoError(t, err)
}

func Test_createTemplateFromRawJSON(t *testing.T) {
	type want struct {
		t    template.Template
		name string
	}
	tests := []struct {
		name  string
		given automation.Response
		want  want
	}{
		{
			"sanitizes template as expected",
			automation.Response{
				ID:   "42",
				Data: []byte(`{ "id": "42", "title": "My Workflow", "lastExecution": { "some": "details" }, "important": "data" }`),
			},
			want{
				t: template.NewDownloadTemplate("42", "My Workflow", `{
  "important": "data",
  "title": "{{.name}}"
}`),
				name: "My Workflow",
			},
		},
		{
			"defaults name to ID if title is not found",
			automation.Response{
				ID:   "42",
				Data: []byte(`{ "id": "42", "workflow_name": "My Workflow", "important": "data" }`),
			},
			want{
				t: template.NewDownloadTemplate("42", "42", `{
  "important": "data",
  "workflow_name": "My Workflow"
}`),
				name: "42",
			},
		},
		{
			"works if reply is not valid JSON",
			automation.Response{
				ID:   "42",
				Data: []byte(`{ "id": "42`),
			},
			want{
				t:    template.NewDownloadTemplate("42", "42", `{ "id": "42`),
				name: "42",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, gotExtractedName := createTemplateFromRawJSON(tt.given, "DOES NOT MATTER FOR TEST")
			assert.Equalf(t, tt.want.t, gotT, "createTemplateFromRawJSON(%v)", tt.given)
			assert.Equalf(t, tt.want.name, gotExtractedName, "createTemplateFromRawJSON(%v)", tt.given)
		})
	}
}
