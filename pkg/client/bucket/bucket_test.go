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

package bucket_test

import (
	"context"
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpsert(t *testing.T) {

	t.Run("error cases", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPost {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("bad request message"))
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		client := bucket.Client{
			Url:    server.URL,
			Client: rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()),
		}

		data := []byte("{}")

		_, err := client.Upsert(context.TODO(), "bucket name", data)
		assert.ErrorContains(t, err, "bad request message", http.StatusBadRequest)
	})

	t.Run("success cases", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPost {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(`{
  "bucketName": "bucket name",
  "table": "logs",
  "displayName": "Custom logs bucket",
  "retentionDays": 35
}`))
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		client := bucket.Client{
			Url:    server.URL,
			Client: rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()),
		}
		data := []byte("{}")

		resp, err := client.Upsert(context.TODO(), "bucket name", data)
		assert.NoError(t, err)

		m := map[string]any{}
		err = json.Unmarshal(resp.Data, &m)
		assert.NoError(t, err)

		assert.Equal(t, "bucket name", m["bucketName"])
	})

	t.Run("success case 2", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

			data, err := io.ReadAll(req.Body)
			assert.NoError(t, err)

			m := map[string]any{}
			err = json.Unmarshal(data, &m)
			assert.NoError(t, err)

			assert.Equal(t, "bucket name", m["bucketName"])

			if req.Method == http.MethodPost {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(`{
  "bucketName": "bucket name",
  "table": "logs",
  "displayName": "Custom logs bucket",
  "retentionDays": 35
}`))
				return
			}
			assert.Fail(t, "unexpected HTTP method call")
		}))
		defer server.Close()

		client := bucket.Client{
			Url:    server.URL,
			Client: rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()),
		}
		data := []byte("{}")

		resp, err := client.Upsert(context.TODO(), "bucket name", data)
		assert.NoError(t, err)

		m := map[string]any{}
		err = json.Unmarshal(resp.Data, &m)
		assert.NoError(t, err)

		assert.Equal(t, "bucket name", m["bucketName"])
	})

}
