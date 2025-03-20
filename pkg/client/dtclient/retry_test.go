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

package dtclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
)

func Test_sendWithsendWithRetryReturnsFirstSuccessfulResponse(t *testing.T) {
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, endpoint string, data io.Reader, options corerest.RequestOptions) (*http.Response, error) {
		if i < 3 {
			i++
			return nil, coreapi.APIError{StatusCode: 400}
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("Success")),
		}, nil
	})

	gotResp, err := SendWithRetry(t.Context(), mockCall, "some/path", corerest.RequestOptions{}, []byte("Success"), RetrySetting{MaxRetries: 5})
	require.NoError(t, err)
	assert.Equal(t, 200, gotResp.StatusCode)
	assert.Equal(t, "Success", string(gotResp.Data))
}

func Test_sendWithRetryFailsAfterDefinedTries(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data io.Reader, options corerest.RequestOptions) (*http.Response, error) {
		if i < maxRetries+1 {
			i++
			return nil, coreapi.APIError{StatusCode: 400}
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("Success")),
		}, nil
	})

	_, err := SendWithRetry(t.Context(), mockCall, "some/path", corerest.RequestOptions{}, []byte("body"), RetrySetting{MaxRetries: maxRetries})
	require.Error(t, err)
	assert.Equal(t, 2, i)
}

func Test_sendWithRetryReturnContainsOriginalApiError(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data io.Reader, options corerest.RequestOptions) (*http.Response, error) {
		if i < maxRetries+1 {
			i++
			return nil, fmt.Errorf("Something wrong")
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("Success")),
		}, nil
	})

	_, err := SendWithRetry(t.Context(), mockCall, "some/path", corerest.RequestOptions{}, []byte("body"), RetrySetting{MaxRetries: maxRetries})
	require.Error(t, err)
	assert.ErrorContains(t, err, "Something wrong")
}

func Test_sendWithRetryReturnsIfNotSuccess(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data io.Reader, options corerest.RequestOptions) (*http.Response, error) {
		if i < maxRetries+1 {
			i++
			return &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(strings.NewReader("{ err: 'failed to create thing'}")),
			}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("Success")),
		}, nil
	})

	_, err := SendWithRetry(t.Context(), mockCall, "some/path", corerest.RequestOptions{}, []byte("body"), RetrySetting{MaxRetries: maxRetries})
	apiError := coreapi.APIError{}
	require.ErrorAs(t, err, &apiError)
	assert.Equal(t, 400, apiError.StatusCode)
}

func Test_SendWithRetryWithInitialTry_RetryIgnoredIfForbidden(t *testing.T) {
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data io.Reader, options corerest.RequestOptions) (*http.Response, error) {
		i++
		return nil, coreapi.APIError{StatusCode: http.StatusForbidden}
	})

	_, err := SendWithRetryWithInitialTry(t.Context(), mockCall, "some/path", corerest.RequestOptions{}, []byte("body"), RetrySetting{MaxRetries: 10})
	require.Error(t, err)
	assert.Equal(t, 1, i)
}

func Test_GetWithRetry_RetryIgnoredIfForbidden(t *testing.T) {
	i := 0
	mux := http.NewServeMux()

	mux.HandleFunc("GET /some/path", func(w http.ResponseWriter, r *http.Request) {
		i++
		w.WriteHeader(http.StatusForbidden)
		_, err := w.Write([]byte("{}"))
		assert.NoError(t, err)
	})

	server := httptest.NewTLSServer(mux)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	restClient := corerest.NewClient(serverURL, server.Client())

	_, err = GetWithRetry(t.Context(), *restClient, "some/path", corerest.RequestOptions{}, RetrySetting{MaxRetries: 10})
	require.Error(t, err)
	assert.Equal(t, 1, i)
}
