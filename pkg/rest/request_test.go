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
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_sendWithsendWithRetryReturnsFirstSuccessfulResponse(t *testing.T) {
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data []byte) (Response, error) {
		if i < 3 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	gotResp, err := SendWithRetry(context.TODO(), mockCall, "some/path", []byte("body"), RetrySetting{MaxRetries: 5})
	require.NoError(t, err)
	assert.Equal(t, 200, gotResp.StatusCode)
	assert.Equal(t, "Success", string(gotResp.Body))
}

func Test_sendWithRetryFailsAfterDefinedTries(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data []byte) (Response, error) {
		if i < maxRetries+1 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	_, err := SendWithRetry(context.TODO(), mockCall, "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	require.Error(t, err)
	assert.Equal(t, 2, i)
}

func Test_sendWithRetryReturnContainsOriginalApiError(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data []byte) (Response, error) {
		if i < maxRetries+1 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	_, err := SendWithRetry(context.TODO(), mockCall, "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	require.Error(t, err)
	assert.ErrorContains(t, err, "Something wrong")
}

func Test_sendWithRetryReturnContainsHttpErrorIfNotSuccess(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data []byte) (Response, error) {
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

	_, err := SendWithRetry(context.TODO(), mockCall, "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	require.Error(t, err)
	assert.ErrorContains(t, err, "400")
	assert.ErrorContains(t, err, "{ err: 'failed to create thing'}")
}
