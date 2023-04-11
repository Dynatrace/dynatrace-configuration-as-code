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

package automation_test

import (
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpsertWorkflow(t *testing.T) {
	jsonData := []byte(`{"id" : "91cc8988-2223-404a-a3f5-5f1a839ecd45", "data" : "some-data"}`)

	t.Run("Upsert - with invalid JSON payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Fail(t, "server was called but shouldn't")
		}))
		defer server.Close()
		workflowClient := automation.NewClient(server.URL, server.Client(), automation.Workflows)
		wf, err := workflowClient.Upsert("id", []byte{})
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("Upsert - Create - with missing ID field", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Fail(t, "unexpected call to server")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client(), automation.Workflows)
		wf, err := workflowClient.Upsert("", jsonData)
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("Upsert - Create - OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut {
				assert.True(t, strings.HasSuffix(req.URL.String(), "some-monaco-generated-ID"))
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			if req.Method == http.MethodPost {
				var data map[string]interface{}
				bytes, _ := io.ReadAll(req.Body)
				_ = json.Unmarshal(bytes, &data)
				assert.Equal(t, "some-monaco-generated-ID", data["id"])
				rw.Write(bytes)
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client(), automation.Workflows)
		wf, err := workflowClient.Upsert("some-monaco-generated-ID", jsonData)
		assert.NotNil(t, wf)
		assert.NoError(t, err)
	})

	t.Run("Upsert - API returns different ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut {
				assert.True(t, strings.HasSuffix(req.URL.String(), "some-monaco-generated-ID"))
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			if req.Method == http.MethodPost {
				var data map[string]interface{}
				bytes, _ := io.ReadAll(req.Body)
				_ = json.Unmarshal(bytes, &data)
				assert.Equal(t, "some-monaco-generated-ID", data["id"])
				rw.Write(jsonData)
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client(), automation.Workflows)
		wf, err := workflowClient.Upsert("some-monaco-generated-ID", jsonData)
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("Upsert - Create - with failing HTTP POST call", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut {
				rw.WriteHeader(http.StatusNotFound)
				return
			}

			if req.Method == http.MethodPost {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			assert.Fail(t, "unexpected HTTP method call (expected POST)")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client(), automation.Workflows)
		wf, err := workflowClient.Upsert("some-monaco-generated-ID", jsonData)
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("Upsert - Update - with failing HTTP PUT call", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client(), automation.Workflows)
		wf, err := workflowClient.Upsert("some-monaco-generated-ID", jsonData)
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("Upsert - Update - OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut {
				// check for absence of Id field
				var data map[string]interface{}
				bytes, _ := io.ReadAll(req.Body)
				_ = json.Unmarshal(bytes, &data)
				_, ok := data["id"]
				assert.False(t, ok)

				rw.Write(jsonData)
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client(), automation.Workflows)
		wf, err := workflowClient.Upsert("some-monaco-generated-ID", jsonData)
		assert.NotNil(t, wf)
		assert.NoError(t, err)
	})

}
