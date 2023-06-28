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

package useragent

import (
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomerAgentStringIsSet(t *testing.T) {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		got := req.Header.Get("User-Agent")
		assert.Equal(t, got, "test-user-agent")
		res.WriteHeader(200)
	}))
	defer func() { testServer.Close() }()
	client := http.Client{Transport: NewCustomUserAgentTransport(testServer.Client().Transport, "test-user-agent")}

	client.Get(testServer.URL)
}

type testTransportWrapper struct {
	base http.RoundTripper
}

func (t *testTransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("test-transport-key-1", "1")
	req.Header.Set("test-transport-key-2", "2")
	req.Header.Set("test-transport-key-3", "3")
	return t.base.RoundTrip(req)
}

func TestCustomerAgentStringIsSetIfUnderlyingClientAlreadyWrapsTransport(t *testing.T) {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		assert.Equal(t, req.Header.Get("User-Agent"), "test-user-agent") // from UA RoundTripper
		assert.Equal(t, req.Header.Get("test-transport-key-1"), "1")     // from wrapped RoundTripper
		assert.Equal(t, req.Header.Get("test-transport-key-2"), "2")     // from wrapped RoundTripper
		assert.Equal(t, req.Header.Get("test-transport-key-3"), "3")     // from wrapped RoundTripper
		res.WriteHeader(200)
	}))
	defer func() { testServer.Close() }()
	baseClient := testTransportWrapper{base: testServer.Client().Transport}
	client := http.Client{Transport: NewCustomUserAgentTransport(&baseClient, "test-user-agent")}

	client.Get(testServer.URL)
}
