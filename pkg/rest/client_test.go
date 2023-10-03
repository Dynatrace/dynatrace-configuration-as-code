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

package rest

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_executeRequestReportsEOFConnectionErrors(t *testing.T) {
	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		server.CloseClientConnections() // cause connection reset on request
	})
	server.Start()
	defer server.Close()

	restClient := NewRestClient(server.Client(), nil, CreateRateLimitStrategy())

	_, err := restClient.Get(context.Background(), server.URL+"/some-url")

	assert.ErrorContains(t, err, "Unable to connect")
}
