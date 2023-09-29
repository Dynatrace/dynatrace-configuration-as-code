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
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
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
		downloader := NewDownloader(httpClient)
		result, err := downloader.Download("projectName")
		assert.Len(t, result, 3)
		assert.Len(t, result[string(config.Workflow)], 3)
		assert.Len(t, result[string(config.SchedulingRule)], 6)
		assert.Len(t, result[string(config.BusinessCalendar)], 2)
		assert.NoError(t, err)
	})

	t.Run("download specific resource", func(t *testing.T) {
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
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		httpClient := automation.NewClient(rest.NewClient(serverURL, server.Client()))
		downloader := NewDownloader(httpClient)
		result, err := downloader.Download("projectName",
			config.AutomationType{Resource: config.Workflow}, config.AutomationType{Resource: config.BusinessCalendar})
		assert.Len(t, result, 2)
		assert.Len(t, result[string(config.Workflow)], 3)
		assert.Len(t, result[string(config.SchedulingRule)], 0)
		assert.Len(t, result[string(config.BusinessCalendar)], 2)
		assert.NoError(t, err)
	})

	t.Run("download workflow resource with jinja template", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/platform/automation/v1/workflows":
				wfData, _ := os.ReadFile("./testdata/listWorkflowsWithJinja.json")
				rw.Write(wfData)
			default:
				assert.Fail(t, "unexpect call to server with path "+req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		httpClient := automation.NewClient(rest.NewClient(serverURL, server.Client()))

		downloader := NewDownloader(httpClient)
		result, err := downloader.Download("projectName", config.AutomationType{Resource: config.Workflow})

		assert.Len(t, result, 1)
		assert.Len(t, result[string(config.Workflow)], 1)
		assert.Contains(t, result[string(config.Workflow)][0].Template.Content(), "{{`{{`}}")
		assert.Contains(t, result[string(config.Workflow)][0].Template.Content(), "{{`}}`}}")
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
	downloader := NewDownloader(httpClient)
	result, err := downloader.Download("projectName")
	assert.Len(t, result, 2)
	assert.Len(t, result[string(config.Workflow)], 3)
	assert.Len(t, result[string(config.SchedulingRule)], 6)
	assert.NoError(t, err)
}

func Test_convertObject(t *testing.T) {
	t.Run("if a title is present, extract it as a name", func(t *testing.T) {
		given := []byte(`{ "id": "42", "title": "My Workflow", "lastExecution": { "some": "details" }, "important": "data" }`)
		actual, err := convertObject(given)

		assert.NoError(t, err)
		assert.NotNil(t, actual.Template)
		assert.Equal(t, "42", actual.Template.Id())
		assert.Equal(t, "My Workflow", actual.Template.Name())
		assert.NotNil(t, actual.Parameters[config.NameParameter])
	})

	t.Run("if a title isn't present, name is ID", func(t *testing.T) {
		given := []byte(`{ "id": "42", "lastExecution": { "some": "details" }, "important": "data" }`)
		actual, err := convertObject(given)

		assert.NoError(t, err)
		assert.NotNil(t, actual.Template)
		assert.Equal(t, "42", actual.Template.Id())
		assert.Equal(t, "42", actual.Template.Name())
		assert.Nil(t, actual.Parameters[config.NameParameter])
	})
}
