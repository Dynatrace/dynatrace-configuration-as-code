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
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDownloader_Download(t *testing.T) {
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
	result, err := downloader.Download("projectName")
	assert.Len(t, result, 3)
	assert.Len(t, result[string(config.Workflow)], 3)
	assert.Len(t, result[string(config.SchedulingRule)], 6)
	assert.Len(t, result[string(config.BusinessCalendar)], 2)
	assert.NoError(t, err)
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
	assert.Len(t, result, 0)
	assert.Len(t, result[string(config.Workflow)], 0)
	assert.Len(t, result[string(config.SchedulingRule)], 0)
	assert.Len(t, result[string(config.BusinessCalendar)], 0)
	assert.Error(t, err)
}

func TestDownloader_Download_Specific_ResouceTypes(t *testing.T) {
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
	assert.Len(t, result[string(config.Workflow)], 3)
	assert.Len(t, result[string(config.SchedulingRule)], 0)
	assert.Len(t, result[string(config.BusinessCalendar)], 2)
	assert.NoError(t, err)
}
