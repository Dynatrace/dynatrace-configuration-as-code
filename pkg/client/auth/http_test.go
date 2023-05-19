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

package auth

import (
	"context"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDefaultTokenURL(t *testing.T) {

	defaultTokenPath := "/fake/sso/oauth2/token"
	defaultTokenURLCalled := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		println(req)
		rw.Header().Add("Content-Type", "application/json")
		if req.URL.Path == defaultTokenPath {
			defaultTokenURLCalled = true
			_, _ = rw.Write([]byte(`{ "access_token":"ABC", "token_type":"Bearer", "expires_in":3600, "refresh_token":"ABCD", "scope":"testing" }`))
		} else {
			_, _ = rw.Write([]byte(`{ "some":"reply" }`))
		}
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	defaultOAuthTokenURL = serverURL.JoinPath(defaultTokenPath).String()

	ctx := context.TODO()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, server.Client()) // ensure the oAuth client trusts the test server by passing its underlying client

	c := NewOAuthClient(ctx, OauthCredentials{
		ClientID:     "id",
		ClientSecret: "secret",
		TokenURL:     "", // no defined token URL should lead to default being used
	})

	_, err = c.Do(&http.Request{Method: http.MethodGet, URL: serverURL.JoinPath("/some/api/call")})
	assert.NoError(t, err)
	assert.True(t, defaultTokenURLCalled, "expected oAuth client to make an API call to the default URL")
}
func TestNonDefaultTokenURL(t *testing.T) {

	defaultTokenPath := "/fake/sso/oauth2/token"
	defaultTokenURLCalled := false

	specialTokenPath := "/magical/special/token"
	specialTokenURLCalled := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		println(req)
		rw.Header().Add("Content-Type", "application/json")
		if req.URL.Path == defaultTokenPath {
			defaultTokenURLCalled = true
			_, _ = rw.Write([]byte(`{ "access_token":"ABC", "token_type":"Bearer", "expires_in":3600, "refresh_token":"ABCD", "scope":"testing" }`))
		} else if req.URL.Path == specialTokenPath {
			specialTokenURLCalled = true
			_, _ = rw.Write([]byte(`{ "access_token":"ABC", "token_type":"Bearer", "expires_in":3600, "refresh_token":"ABCD", "scope":"testing" }`))
		} else {
			_, _ = rw.Write([]byte(`{ "some":"reply" }`))
		}
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	defaultOAuthTokenURL = serverURL.JoinPath(defaultTokenPath).String()

	ctx := context.TODO()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, server.Client()) // ensure the oAuth client trusts the test server by passing its underlying client

	c := NewOAuthClient(ctx, OauthCredentials{
		ClientID:     "id",
		ClientSecret: "secret",
		TokenURL:     serverURL.JoinPath(specialTokenPath).String(),
	})

	_, err = c.Do(&http.Request{Method: http.MethodGet, URL: serverURL.JoinPath("/some/api/call")})
	assert.NoError(t, err)
	assert.True(t, specialTokenURLCalled, "expected oAuth client to make an API call to the defined token URL")
	assert.True(t, defaultTokenURLCalled == false, "expected oAuth client to make NO API call to the default URL")
}
