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

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

func TestCreateClientSet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "sso") {
			token := &oauth2.Token{
				AccessToken: "test-access-token",
				TokenType:   "Bearer",
				Expiry:      time.Now().Add(time.Hour),
			}

			rw.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode(token)
			return
		}

		rw.WriteHeader(200)
		output := fmt.Sprintf(`{"version" : "0.59.3.20231603", "domain": "http://%s/api/test", "endpoint": "http://%s/api/test"}`, req.Host, req.Host)
		_, _ = rw.Write([]byte(output))
	}))
	defer server.Close()

	tests := []struct {
		name string
		url  string
		auth manifest.Auth
	}{
		{"API token client set",
			server.URL,
			manifest.Auth{
				ApiToken: &manifest.AuthSecret{
					Name:  "token-env-var",
					Value: "mock token",
				},
			},
		},
		{"oAuth client set",
			server.URL,
			manifest.Auth{
				OAuth: &manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "client-id",
						Value: "resolved-client-id",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "client-secret",
						Value: "resolved-client-secret",
					},
					TokenEndpoint: &manifest.URLDefinition{
						Value: server.URL + "/sso",
					},
				},
			},
		},
		{"Platform token client set",
			server.URL,
			manifest.Auth{
				PlatformToken: &manifest.AuthSecret{
					Name:  "platform-token-env-var",
					Value: "platform mock token",
				},
			},
		},
		{"oAuth and API token client set",
			server.URL,
			manifest.Auth{
				ApiToken: &manifest.AuthSecret{
					Name:  "token-env-var",
					Value: "mock token",
				},
				OAuth: &manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "client-id",
						Value: "resolved-client-id",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "client-secret",
						Value: "resolved-client-secret",
					},
					TokenEndpoint: &manifest.URLDefinition{
						Value: server.URL + "/sso",
					},
				},
			},
		},
		{"Platform token and API token client set",
			server.URL,
			manifest.Auth{
				ApiToken: &manifest.AuthSecret{
					Name:  "token-env-var",
					Value: "mock token",
				},
				PlatformToken: &manifest.AuthSecret{
					Name:  "platform-token-env-var",
					Value: "platform mock token",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientSet, err := CreateClientSet(t.Context(), tt.url, tt.auth)
			assert.NoError(t, err)
			assert.NotNil(t, clientSet)
		})
	}
}

func TestCreateClientSetWithAdditionalHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		require.Equal(t, "Some-Value", req.Header.Get("Some-Header"))
		rw.WriteHeader(404)
	}))
	defer server.Close()

	t.Setenv(environment.AdditionalHTTPHeaders, "Some-Header: Some-Value")
	clientSet, _ := CreateClientSet(t.Context(), server.URL, manifest.Auth{
		ApiToken: &manifest.AuthSecret{
			Name:  "token-env-var",
			Value: "mock token",
		},
	})
	assert.NotNil(t, clientSet)

	var apiErr api.APIError
	_, err := clientSet.SettingsClient.Get(t.Context(), "")
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 404, apiErr.StatusCode)
}

func TestCreateClientSetWithPlatformToken_ClientsUsePlatformToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "sso") {
			t.Fatal("Should not be called")
		}

		require.Equal(t, "Bearer my-platform-token", req.Header.Get("Authorization"))
		if req.URL.Path == "/platform/classic/environment-api/v2/settings/objects/" {
			rw.WriteHeader(404)
			return
		}

		rw.WriteHeader(200)
		output := fmt.Sprintf(`{"version" : "0.59.3.20231603", "domain": "http://%s/api/test", "endpoint": "http://%s/api/test"}`, req.Host, req.Host)
		_, _ = rw.Write([]byte(output))
	}))
	defer server.Close()
	clientSet, _ := CreateClientSet(t.Context(), server.URL, manifest.Auth{
		PlatformToken: &manifest.AuthSecret{
			Name:  "token-env-var",
			Value: "my-platform-token",
		},
	})
	assert.NotNil(t, clientSet)

	var apiErr api.APIError
	_, err := clientSet.SettingsClient.Get(t.Context(), "")
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 404, apiErr.StatusCode)
}

func TestCreateClientSetWithPlatformTokenAndOAuth_ClientsUsePlatformToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "sso") {
			t.Fatal("Should not be called")
		}

		require.Equal(t, "Bearer my-platform-token", req.Header.Get("Authorization"))
		if req.URL.Path == "/platform/classic/environment-api/v2/settings/objects/" {
			rw.WriteHeader(404)
			return
		}

		rw.WriteHeader(200)
		output := fmt.Sprintf(`{"version" : "0.59.3.20231603", "domain": "http://%s/api/test", "endpoint": "http://%s/api/test"}`, req.Host, req.Host)
		_, _ = rw.Write([]byte(output))

	}))
	defer server.Close()

	clientSet, _ := CreateClientSet(t.Context(), server.URL, manifest.Auth{
		PlatformToken: &manifest.AuthSecret{
			Name:  "token-env-var",
			Value: "my-platform-token",
		},
		OAuth: &manifest.OAuth{
			ClientID: manifest.AuthSecret{
				Name:  "client-id",
				Value: "resolved-client-id",
			},
			ClientSecret: manifest.AuthSecret{
				Name:  "client-secret",
				Value: "resolved-client-secret",
			},
			TokenEndpoint: &manifest.URLDefinition{
				Value: server.URL + "/sso",
			},
		},
	})
	assert.NotNil(t, clientSet)

	var apiErr api.APIError
	_, err := clientSet.SettingsClient.Get(t.Context(), "")
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 404, apiErr.StatusCode)
}
