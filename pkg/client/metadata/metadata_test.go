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

package metadata

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
)

func TestGetDynatraceClassicURL(t *testing.T) {

	t.Run("client GET error results in error", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{})
		defer server.Close()

		classicURL, err := GetDynatraceClassicURL(context.TODO(), *corerest.NewClient(server.URL(), server.FaultyClient()))
		assert.Empty(t, classicURL)
		assert.Error(t, err)
	})

	t.Run("server responds with code != 200 results in error", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusNotFound,
						ResponseBody: "{}",
					}
				},
			},
		})
		defer server.Close()

		classicURL, err := GetDynatraceClassicURL(context.TODO(), *corerest.NewClient(server.URL(), server.Client()))
		assert.Empty(t, classicURL)
		assert.Error(t, err)
	})

	t.Run("unauthorized response results in error", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusUnauthorized,
						ResponseBody: "{}",
					}
				},
			},
		})
		defer server.Close()

		classicURL, err := GetDynatraceClassicURL(context.TODO(), *corerest.NewClient(server.URL(), server.Client()))
		assert.Empty(t, classicURL)
		assert.Error(t, err)
	})

	t.Run("server response with invalid data", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: "}",
					}
				},
			},
		})
		defer server.Close()

		classicURL, err := GetDynatraceClassicURL(context.TODO(), *corerest.NewClient(server.URL(), server.Client()))
		assert.Empty(t, classicURL)
		assert.Error(t, err)
	})

	t.Run("server response with valid data", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"domain" : "https://classic.env.com"}`,
					}
				},
			},
		})
		defer server.Close()

		classicURL, err := GetDynatraceClassicURL(context.TODO(), *corerest.NewClient(server.URL(), server.Client()))
		assert.EqualValues(t, "https://classic.env.com", classicURL)
		assert.NoError(t, err)
	})
}
