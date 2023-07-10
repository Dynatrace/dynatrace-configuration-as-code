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
	"gotest.tools/assert"
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

	gotResp, err := SendWithRetry(context.TODO(), mockCall, "dont matter", "some/path", []byte("body"), RetrySetting{MaxRetries: 5})
	assert.NilError(t, err)
	assert.Equal(t, gotResp.StatusCode, 200)
	assert.Equal(t, string(gotResp.Body), "Success")
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

	_, err := SendWithRetry(context.TODO(), mockCall, "dont matter", "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	assert.Check(t, err != nil)
	assert.Equal(t, i, 2)
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

	_, err := SendWithRetry(context.TODO(), mockCall, "dont matter", "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	assert.Check(t, err != nil)
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

	_, err := SendWithRetry(context.TODO(), mockCall, "dont matter", "some/path", []byte("body"), RetrySetting{MaxRetries: maxRetries})
	assert.Check(t, err != nil)
	assert.ErrorContains(t, err, "400")
	assert.ErrorContains(t, err, "{ err: 'failed to create thing'}")
}
