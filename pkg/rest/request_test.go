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
	"fmt"
	"gotest.tools/assert"
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

			if err := DeleteConfig(server.Client(), server.URL, "checked ID does not matter"); (err != nil) != tt.wantErr {
				t.Errorf("DeleteConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sendWithsendWithRetryReturnsFirstSuccessfulResponse(t *testing.T) {
	i := 0
	mockCall := SendingRequest(func(client *http.Client, url string, data []byte) (Response, error) {
		if i < 3 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	gotResp, err := SendWithRetry(nil, mockCall, "dont matter", "some/path", []byte("body"), RetrySetting{MaxRetries: 5})
	assert.NilError(t, err)
	assert.Equal(t, gotResp.StatusCode, 200)
	assert.Equal(t, string(gotResp.Body), "Success")
}

func Test_sendWithRetryFailsAfterDefinedTries(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendingRequest(func(client *http.Client, url string, data []byte) (Response, error) {
		if i < maxRetries+1 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	_, err := SendWithRetry(nil, mockCall, "dont matter", "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	assert.Check(t, err != nil)
	assert.Equal(t, i, 2)
}

func Test_sendWithRetryReturnContainsOriginalApiError(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendingRequest(func(client *http.Client, url string, data []byte) (Response, error) {
		if i < maxRetries+1 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	_, err := SendWithRetry(nil, mockCall, "dont matter", "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	assert.Check(t, err != nil)
	assert.ErrorContains(t, err, "Something wrong")
}

func Test_sendWithRetryReturnContainsHttpErrorIfNotSuccess(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendingRequest(func(client *http.Client, url string, data []byte) (Response, error) {
		if i < maxRetries+1 {
			i++
			return Response{
				StatusCode: 400,
				Body:       []byte("{ err: 'failed to create thing'}"),
			}, nil
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	_, err := SendWithRetry(nil, mockCall, "dont matter", "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	assert.Check(t, err != nil)
	assert.ErrorContains(t, err, "400")
	assert.ErrorContains(t, err, "{ err: 'failed to create thing'}")
}
