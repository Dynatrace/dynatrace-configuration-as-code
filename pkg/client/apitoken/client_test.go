/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package apitoken_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/apitoken"
)

type Stub struct {
	post func() (*http.Response, error)
}

func (s Stub) POST(_ context.Context, _ string, _ io.Reader, _ corerest.RequestOptions) (*http.Response, error) {
	return s.post()
}

func TestGetTokenMetadata(t *testing.T) {
	t.Run("Returns token metadata", func(t *testing.T) {
		stub := Stub{func() (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(`
				{
				  "id": "abc-xy",
				  "name": "my-token",
				  "enabled": true,
				  "personalAccessToken": false,
				  "owner": "my-owner-email",
				  "creationDate": "2024-01-11T16:56:05.499Z",
				  "scopes": [
					"settings.read",
					"settings.write"
				  ]
				}`)),
			}, nil
		},
		}
		resp, err := apitoken.GetTokenMetadata(t.Context(), stub, "my-token")
		assert.NoError(t, err)
		assert.Equal(t, apitoken.Response{
			ID:                  "abc-xy",
			Name:                "my-token",
			Enabled:             true,
			PersonalAccessToken: false,
			Owner:               "my-owner-email",
			CreationDate:        "2024-01-11T16:56:05.499Z",
			Scopes:              []string{"settings.read", "settings.write"},
		}, resp)
	})

	t.Run("Errors if request errors", func(t *testing.T) {
		stub := Stub{func() (*http.Response, error) {
			return &http.Response{}, errors.New("client error")
		}}
		resp, err := apitoken.GetTokenMetadata(t.Context(), stub, "my-token")

		assert.Equal(t, apitoken.Response{}, resp)
		assert.ErrorContains(t, err, "client error")
	})

	t.Run("Errors if request returns a not successful status code", func(t *testing.T) {
		stub := Stub{func() (*http.Response, error) {
			return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("api error"))}, nil
		}}
		resp, err := apitoken.GetTokenMetadata(t.Context(), stub, "my-token")

		assert.Equal(t, apitoken.Response{}, resp)
		assert.ErrorContains(t, err, "api error")
	})

	t.Run("Errors if response is not JSON", func(t *testing.T) {
		stub := Stub{func() (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{"))}, nil
		}}
		resp, err := apitoken.GetTokenMetadata(t.Context(), stub, "my-token")

		assert.Equal(t, apitoken.Response{}, resp)
		assert.ErrorContains(t, err, "unmarshal")
	})
}
