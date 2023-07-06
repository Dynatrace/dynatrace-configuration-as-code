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
	"context"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/automation"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAutomationClientGet(t *testing.T) {
	jsonData := []byte(`{ "id" : "91cc8988-2223-404a-a3f5-5f1a839ecd45", "data" : "some-data1" }`)
	t.Run("GET - OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet {
				rw.Write(jsonData)
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Get(context.TODO(), automation.Workflows, "91cc8988-2223-404a-a3f5-5f1a839ecd45")
		assert.NotNil(t, wf)
		assert.Equal(t, jsonData, wf.Data)
		assert.NoError(t, err)
	})

	t.Run("GET - URL parse error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet {
				rw.Write(jsonData)
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Get(context.TODO(), automation.Workflows, "\n")
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("GET - Request error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Get(context.TODO(), automation.Workflows, "91cc8988-2223-404a-a3f5-5f1a839ecd45")
		assert.Nil(t, wf)
		assert.Error(t, err)

		server.Close()
		wf, err = workflowClient.Get(context.TODO(), automation.Workflows, "91cc8988-2223-404a-a3f5-5f1a839ecd45")
		assert.Nil(t, wf)
		assert.Error(t, err)
	})
}

func TestAutomationClientList(t *testing.T) {

	jsonData := []byte(`{"count" : 2, "results" : [ { "id" : "91cc8988-2223-404a-a3f5-5f1a839ecd45", "data" : "some-data1"}, { "id" : "91cc8988-2223-404a-a3f5-5f1a839ecd46", "data" : "some-data2"} ]}`)
	t.Run("List - OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet {
				rw.Write(jsonData)
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.List(context.TODO(), automation.Workflows)
		assert.NotNil(t, wf)
		assert.NoError(t, err)
	})
	t.Run("List - HTTP GET fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.List(context.TODO(), automation.Workflows)
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("List - HTTP GET returns garbage data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet {
				rw.Write([]byte("lskdlskejsdlfrkdlvdkedjgokdfjgldffk"))
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.List(context.TODO(), automation.Workflows)
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("List - test pagination", func(t *testing.T) {
		data := []byte(`{"count" : 4, "results" : [ {"id" : "91cc8988-2223-404a-a3f5-5f1a839ecd45", "data" : "some-data1"} ]}`)
		noCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet {
				rw.Write(data)
				rw.WriteHeader(http.StatusOK)
				noCalls++
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.List(context.TODO(), automation.Workflows)
		assert.Equal(t, noCalls, 4, "There should be 4 cals")
		assert.NotNil(t, wf)
		assert.NoError(t, err)
	})

	t.Run("List - admin access fails - subsequent calls without admin access pass", func(t *testing.T) {
		data := []byte(`{"count" : 4, "results" : [ {"id" : "91cc8988-2223-404a-a3f5-5f1a839ecd45", "data" : "some-data1"} ]}`)
		noCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet && noCalls == 0 {
				assert.Equal(t, req.URL.Query().Get("adminAccess"), "true")
				rw.WriteHeader(http.StatusForbidden)
				noCalls++
				return
			}
			if req.Method == http.MethodGet {
				assert.Equal(t, req.URL.Query().Get("adminAccess"), "false")
				rw.Write(data)
				rw.WriteHeader(http.StatusOK)
				noCalls++
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.List(context.TODO(), automation.Workflows)
		assert.Equal(t, noCalls, 5, "There should be 5 cals")
		assert.NotNil(t, wf)
		assert.NoError(t, err)
	})
}

func TestAutomationClientUpsert(t *testing.T) {
	jsonData := []byte(`{"id" : "91cc8988-2223-404a-a3f5-5f1a839ecd45", "data" : "some-data"}`)

	t.Run("Upsert - with invalid JSON payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Fail(t, "server was called but shouldn't")
		}))
		defer server.Close()
		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Upsert(context.TODO(), automation.Workflows, "id", []byte{})
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("Upsert - Create - with missing ID field", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Fail(t, "unexpected call to server")
		}))
		defer server.Close()

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Upsert(context.TODO(), automation.Workflows, "", jsonData)
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("Upsert - Create - OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut {
				assert.True(t, strings.HasSuffix(req.URL.Path, "some-monaco-generated-ID"))
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

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Upsert(context.TODO(), automation.Workflows, "some-monaco-generated-ID", jsonData)
		assert.NotNil(t, wf)
		assert.NoError(t, err)
	})

	t.Run("Upsert - API returns different ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut {
				assert.True(t, strings.HasSuffix(req.URL.Path, "some-monaco-generated-ID"))
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

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Upsert(context.TODO(), automation.Workflows, "some-monaco-generated-ID", jsonData)
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

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Upsert(context.TODO(), automation.Workflows, "some-monaco-generated-ID", jsonData)
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

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Upsert(context.TODO(), automation.Workflows, "some-monaco-generated-ID", jsonData)
		assert.Nil(t, wf)
		assert.Error(t, err)
	})

	t.Run("Upsert - Update - OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut {
				// check for absence of ID field
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

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Upsert(context.TODO(), automation.Workflows, "some-monaco-generated-ID", jsonData)
		assert.NotNil(t, wf)
		assert.NoError(t, err)
	})

	t.Run("Upsert - Update - First call with admin access fails - subsequent OK", func(t *testing.T) {
		noCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPut && noCalls == 0 {
				rw.WriteHeader(http.StatusForbidden)
				noCalls++
				return
			}
			if req.Method == http.MethodPut {
				// check for absence of ID field
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

		workflowClient := automation.NewClient(server.URL, server.Client())
		wf, err := workflowClient.Upsert(context.TODO(), automation.Workflows, "some-monaco-generated-ID", jsonData)
		assert.NotNil(t, wf)
		assert.NoError(t, err)
	})

}

func TestAutomationClientDelete(t *testing.T) {
	t.Run("Delete - OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete {
				assert.True(t, strings.HasSuffix(req.URL.Path, "some-monaco-generated-ID"))
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		c := automation.NewClient(server.URL, server.Client())
		err := c.Delete(automation.Workflows, "some-monaco-generated-ID")
		assert.NoError(t, err)
	})

	t.Run("Delete - workflow admin access fails - subsequent OK", func(t *testing.T) {
		noCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && noCalls == 0 {
				rw.WriteHeader(http.StatusForbidden)
				noCalls++
				return
			}

			if req.Method == http.MethodDelete {
				assert.True(t, strings.HasSuffix(req.URL.Path, "some-monaco-generated-ID"))
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		c := automation.NewClient(server.URL, server.Client())
		err := c.Delete(automation.Workflows, "some-monaco-generated-ID")
		assert.NoError(t, err)
	})

	t.Run("Delete - Without ID - Fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		c := automation.NewClient(server.URL, server.Client())
		err := c.Delete(automation.Workflows, "")
		assert.ErrorContains(t, err, "id must be non empty")
	})

	t.Run("Delete - Object Not Found no counted as Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete {
				assert.True(t, strings.HasSuffix(req.URL.Path, "some-monaco-generated-ID"))
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		c := automation.NewClient(server.URL, server.Client())
		err := c.Delete(automation.Workflows, "some-monaco-generated-ID")
		assert.NoError(t, err)
	})

	t.Run("Delete - Server Error - Fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete {
				assert.True(t, strings.HasSuffix(req.URL.Path, "some-monaco-generated-ID"))
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		c := automation.NewClient(server.URL, server.Client())
		err := c.Delete(automation.Workflows, "some-monaco-generated-ID")
		assert.ErrorContains(t, err, "unable to delete")
	})
}

func TestContext(t *testing.T) {
	ctx := context.TODO()
	takeCtx(context.WithValue(ctx, "environment", "my-env"))
	fmt.Println(ctx.Value("name"))
}

func takeCtx(ctx context.Context) {
	fmt.Println(ctx.Value("environment"))
}
