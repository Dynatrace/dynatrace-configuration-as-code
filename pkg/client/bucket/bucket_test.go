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
	"net/url"
	"strconv"
	"testing"
)

func TestGet(t *testing.T) {
	t.Run("successfully fetch a bucket", func(t *testing.T) {
		const payload = `{
  "bucketName": "bucket name",
  "table": "metrics",
  "displayName": "Default metrics (15 months)",
  "status": "active",
  "retentionDays": 462,
  "metricInterval": "PT1M",
  "version": 1
}`

		responses := serverResponses{
			http.MethodGet: {
				code:     http.StatusOK,
				response: payload,
			},
		}
		server := createServer(t, responses)
		defer server.Close()

		client := bucket.NewClient(server.URL, rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()))

		resp, err := client.Get(context.TODO(), "bucket name")
		assert.NoError(t, err)
		assert.Equal(t, resp.Status, "active")
		assert.Equal(t, resp.Version, 1)
		assert.Equal(t, resp.BucketName, "bucket name")
		assert.Equal(t, resp.Data, []byte(payload))
	})

	t.Run("correctly create the error in case of a server issue", func(t *testing.T) {
		responses := serverResponses{
			http.MethodGet: {
				code:     http.StatusNotFound,
				response: "my error",
			},
		}
		server := createServer(t, responses)
		defer server.Close()

		client := bucket.NewClient(server.URL, rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()))

		resp, err := client.Get(context.TODO(), "bucket name")
		assert.ErrorContains(t, err, "my error", strconv.Itoa(http.StatusNotFound))
		assert.Equal(t, resp, bucket.Response{})
	})
}

func TestUpsert(t *testing.T) {

	t.Run("update fails", func(t *testing.T) {
		responses := serverResponses{
			http.MethodPost: {
				code:     http.StatusBadRequest,
				response: "ERROR",
			},
			http.MethodGet: {
				code: http.StatusOK,
				response: `{
  "bucketName": "bucket name",
  "table": "metrics",
  "displayName": "Default metrics (15 months)",
  "status": "active",
  "retentionDays": 462,
  "metricInterval": "PT1M",
  "version": 1
}`,
			},
			http.MethodPut: {
				code:     http.StatusForbidden,
				response: "no write access message",
			},
		}
		server := createServer(t, responses)
		defer server.Close()

		client := bucket.NewClient(server.URL, rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()))

		data := []byte("{}")

		_, err := client.Upsert(context.TODO(), "bucket name", data)
		assert.ErrorContains(t, err, "no write access message", http.StatusForbidden)
	})

	t.Run("create new bucket - OK", func(t *testing.T) {
		responses := serverResponses{
			http.MethodPost: {
				code: http.StatusOK,
				response: `{
  "bucketName": "bucket name",
  "table": "metrics",
  "displayName": "Default metrics (15 months)",
  "status": "active",
  "retentionDays": 462,
  "metricInterval": "PT1M",
  "version": 1
}`,
				validate: func(req *http.Request) {
					data, err := io.ReadAll(req.Body)
					assert.NoError(t, err)

					m := map[string]any{}
					err = json.Unmarshal(data, &m)
					assert.NoError(t, err)

					assert.Equal(t, "bucket name", m["bucketName"])
				},
			},
		}
		server := createServer(t, responses)
		defer server.Close()

		client := bucket.NewClient(server.URL, rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()))
		data := []byte("{}")

		resp, err := client.Upsert(context.TODO(), "bucket name", data)
		assert.NoError(t, err)

		m := map[string]any{}
		err = json.Unmarshal(resp.Data, &m)
		assert.NoError(t, err)

		assert.Equal(t, "bucket name", m["bucketName"])
	})

	t.Run("update new bucket - OK", func(t *testing.T) {
		responses := serverResponses{
			http.MethodPost: {
				code:     http.StatusForbidden,
				response: "this is an error",
			},
			http.MethodGet: {
				code: http.StatusOK,
				response: `{
  "bucketName": "bucket name",
  "table": "metrics",
  "displayName": "Default metrics (15 months)",
  "status": "active",
  "retentionDays": 462,
  "metricInterval": "PT1M",
  "version": 1
}`,
				validate: func(req *http.Request) {
					assert.Contains(t, req.URL.String(), url.PathEscape("bucket name"))
				},
			},
			http.MethodPut: {
				code: http.StatusOK,
				response: `{
  "bucketName": "bucket name",
  "table": "metrics",
  "displayName": "Default metrics (15 months)",
  "status": "active",
  "retentionDays": 462,
  "metricInterval": "PT1M",
  "version": 1
}`,
				validate: func(req *http.Request) {
					data, err := io.ReadAll(req.Body)
					assert.NoError(t, err)

					m := map[string]any{}
					err = json.Unmarshal(data, &m)
					assert.NoError(t, err)

					assert.Equal(t, "bucket name", m["bucketName"])
				},
			},
		}
		server := createServer(t, responses)
		defer server.Close()

		client := bucket.NewClient(server.URL, rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()))
		data := []byte("{}")

		resp, err := client.Upsert(context.TODO(), "bucket name", data)
		assert.NoError(t, err)

		m := map[string]any{}
		err = json.Unmarshal(resp.Data, &m)
		assert.NoError(t, err)

		assert.Equal(t, "bucket name", m["bucketName"])
	})

}

type httpMethod = string
type serverResponses map[httpMethod]struct {
	code     int
	response string
	validate func(*http.Request)
}

func createServer(t *testing.T, arg serverResponses) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if res, found := arg[req.Method]; found {
			if res.validate != nil {
				res.validate(req)
			}
			rw.WriteHeader(res.code)
			rw.Write([]byte(res.response))
		} else {
			assert.Fail(t, "unexpected HTTP method call")
		}
	}))

}
