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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/support"
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
		{"token client set",
			server.URL,
			manifest.Auth{
				Token: &manifest.AuthSecret{
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
		{"oAuth and token client set",
			server.URL,
			manifest.Auth{
				Token: &manifest.AuthSecret{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateClientSet(context.TODO(), tt.url, tt.auth, ClientOptions{SupportArchive: support.SupportArchive})
			assert.NoError(t, err)
		})
	}
}
