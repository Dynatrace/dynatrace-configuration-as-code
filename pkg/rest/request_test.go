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

func Test_deleteConfig(t *testing.T) {
	tests := []struct {
		name            string
		givenStatusCode int
		wantErr         bool
	}{
		{
			"does not return error on http success",
			http.StatusOK,
			false,
		},
		{
			"does not return error on http accepted",
			http.StatusAccepted,
			false,
		},
		{
			"does not return error if delete API already returns not found",
			http.StatusNotFound,
			false,
		},
		{
			"returns error on bad request",
			http.StatusBadRequest,
			true,
		},
		{
			"returns error on unauthorized",
			http.StatusUnauthorized,
			true,
		},
		{
			"returns error on server error",
			http.StatusInternalServerError,
			true,
		},
		{
			"returns error on server unavailable",
			http.StatusServiceUnavailable,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(tt.givenStatusCode)
			}))
			defer server.Close()

			if err := deleteConfig(server.Client(), server.URL, "API TOKEN", "checked ID does not matter"); (err != nil) != tt.wantErr {
				t.Errorf("deleteConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
