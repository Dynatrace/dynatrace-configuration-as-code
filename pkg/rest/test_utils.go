//go:build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newDynatraceClientForTesting creates a new DynatraceClient for a given test-server
func newDynatraceClientForTesting(server *httptest.Server) DynatraceClient {
	return &dynatraceClientImpl{
		client:         server.Client(),
		environmentUrl: server.URL,
	}
}

// Creates a new test server and returns the created client & URL.
// The server is closed automatically upon exiting the testing environment
func newDynatraceTestServer(t *testing.T, callback func(res http.ResponseWriter, req *http.Request)) (*http.Client, string) {
	testServer := httptest.NewServer(http.HandlerFunc(callback))
	t.Cleanup(testServer.Close)

	return testServer.Client(), testServer.URL
}

// NewDynatraceTLSServerForTesting creates a new test server and returns it.
// The server is closed automatically upon exiting the testing environment
func NewDynatraceTLSServerForTesting(t *testing.T, callback func(res http.ResponseWriter, req *http.Request)) *httptest.Server {
	testServer := httptest.NewTLSServer(http.HandlerFunc(callback))
	t.Cleanup(testServer.Close)

	return testServer
}

func NewDynatraceClientForTesting(environmentUrl, token string, client *http.Client) (DynatraceClient, error) {
	return newDynatraceClient(environmentUrl, token, *client)
}
