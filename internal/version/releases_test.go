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

package version

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testCase struct {
	name           string
	handler        http.HandlerFunc
	expectedResult Version
	expectedError  error
}

func TestGetLatestVersion(t *testing.T) {
	testCases := []testCase{
		{
			name: "Successful response",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := release{TagName: "v1.2.3"}
				jsonResponse, _ := json.Marshal(response)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(jsonResponse)
			}),
			expectedResult: Version{1, 2, 3},
			expectedError:  nil,
		},
		{
			name: "Invalid JSON response",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			}),
			expectedResult: UnknownVersion,
			expectedError:  errors.New("unable to parse response data: invalid character 'i' looking for beginning of value"),
		},
		{
			name: "HTTP error",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
			expectedResult: UnknownVersion,
			expectedError:  errors.New("failed to fetch release data. Status code: 404"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			client := &http.Client{}
			result, err := GetLatestVersion(context.TODO(), client, server.URL)

			if err != nil {
				if tc.expectedError == nil || err.Error() != tc.expectedError.Error() {
					t.Errorf("Expected error: %v, got: %v", tc.expectedError, err)
				}
			} else if result != tc.expectedResult {
				t.Errorf("Expected result: %v, got: %v", tc.expectedResult, result)
			}
		})
	}
}
