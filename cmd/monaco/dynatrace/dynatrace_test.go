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

package dynatrace

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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

var accessTokenPayload = []byte(`
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
}`)

func getClassicEnvPayload(host string) []byte {
	return []byte(fmt.Sprintf(`{"domain": "http://%s"}`, host))
}

func TestVerifyEnvironmentsAuthentication_OneOfManyFails(t *testing.T) {
	envCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/apiTokens/lookup", func(rw http.ResponseWriter, req *http.Request) {
		envCount++
		if envCount > 1 {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write(accessTokenPayload)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	err := VerifyEnvironmentsAuthentication(context.TODO(), manifest.EnvironmentDefinitionsByName{
		"env": manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{
				AccessToken: &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "some token"},
			},
		},
		"env2": manifest.EnvironmentDefinition{
			Name: "env2",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{
				AccessToken: &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "some token"},
			},
		},
	})
	assert.Error(t, err)
	assert.Equal(t, 2, envCount)
}

func TestVerifyEnvironmentsAuth(t *testing.T) {
	type args struct {
		envs manifest.EnvironmentDefinitionsByName
	}
	tests := []struct {
		name                 string
		args                 args
		classicEnvCheckFails bool
		handler              http.HandlerFunc
		wantErr              bool
	}{
		{
			name: "empty environment - passes",
			args: args{
				envs: manifest.EnvironmentDefinitionsByName{},
			},
			wantErr: false,
		},
		{
			name: "single environment without fields set - fails",
			args: args{
				envs: manifest.EnvironmentDefinitionsByName{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifyEnvironmentsAuthentication(context.TODO(), tt.args.envs); tt.wantErr && err == nil {
				t.Errorf("VerifyEnvironmentsAuthentication() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("Call classic endpoint - ok", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "Api-Token some token", req.Header.Get("Authorization"))
			rw.WriteHeader(200)
			_, _ = rw.Write(getClassicEnvPayload(req.Host))
		}))
		defer server.Close()

		err := VerifyEnvironmentsAuthentication(context.TODO(), manifest.EnvironmentDefinitionsByName{
			"env": manifest.EnvironmentDefinition{
				Name: "env",
				URL: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Name:  "URL",
					Value: server.URL,
				},
				Auth: manifest.Auth{AccessToken: &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "some token"}},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("Call Platform endpoint using OAuth - ok", func(t *testing.T) {
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
			_, _ = rw.Write(getClassicEnvPayload(req.Host))
		}))
		defer server.Close()

		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{
				OAuth: &manifest.OAuth{
					TokenEndpoint: &manifest.URLDefinition{
						Value: server.URL + "/sso",
					},
				},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("Call Platform endpoint using platform token - ok", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(200)
			assert.Equal(t, "Bearer some token", req.Header.Get("Authorization"))
			_, _ = rw.Write(getClassicEnvPayload(req.Host))
		}))
		defer server.Close()

		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{PlatformToken: &manifest.AuthSecret{Name: "PLATFORM_TOKEN", Value: "some token"}},
		})
		assert.NoError(t, err)
	})

	t.Run("classic endpoint not available ", func(t *testing.T) {
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

			rw.WriteHeader(404)
			_, _ = rw.Write(getClassicEnvPayload(req.Host))
		}))
		defer server.Close()

		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env1",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL + "/WRONG_URL",
			},
		})
		assert.Error(t, err)

		err = VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env2",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL + "/WRONG_URL",
			},
			Auth: manifest.Auth{
				OAuth: &manifest.OAuth{
					TokenEndpoint: &manifest.URLDefinition{
						Value: server.URL + "/sso",
					},
				},
			},
		})
		assert.Error(t, err)
	})

	t.Run("Fails if neither OAuth, nor platform token, nor access token is provided", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatal("Should not be called")
		}))
		defer server.Close()
		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{},
		})
		assert.Error(t, err)
	})

	t.Run("Fails if token is invalid", func(t *testing.T) {
		apiCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			apiCalls++
			rw.WriteHeader(400)
		}))
		defer server.Close()
		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{
				AccessToken: &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "some token"},
			},
		})
		assert.Error(t, err)
		assert.Equal(t, 1, apiCalls)
	})

	t.Run("Fails if classic client creation failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatal("Should not be called")
		}))
		defer server.Close()
		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: "",
			},
			Auth: manifest.Auth{
				AccessToken: &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "some token"},
			},
		})
		assert.Error(t, err)
	})

	t.Run("Fails if platform client creation failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatal("Should not be called")
		}))
		defer server.Close()
		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: "",
			},
			Auth: manifest.Auth{
				OAuth: &manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "OAUTH_ID",
						Value: "123",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "OAUTH_SECRET",
						Value: "xyz",
					},
					TokenEndpoint: &manifest.URLDefinition{
						Value: server.URL + "/sso",
					},
				},
			},
		})
		assert.Error(t, err)
	})

	t.Run("OAuth and token is validated", func(t *testing.T) {
		classicCalls := 0
		platformCalls := 0
		mux := http.NewServeMux()
		mux.HandleFunc("/sso", func(rw http.ResponseWriter, req *http.Request) {
			token := &oauth2.Token{
				AccessToken: "test-access-token",
				TokenType:   "Bearer",
				Expiry:      time.Now().Add(time.Hour),
			}

			rw.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode(token)
		})
		mux.HandleFunc(metadata.ClassicEnvironmentDomainPath, func(rw http.ResponseWriter, req *http.Request) {
			platformCalls++
			assert.Equal(t, "Bearer test-access-token", req.Header.Get("Authorization"))
			rw.WriteHeader(200)
			_, _ = rw.Write(getClassicEnvPayload(req.Host))
		})
		mux.HandleFunc("/api/v2/apiTokens/lookup", func(rw http.ResponseWriter, req *http.Request) {
			classicCalls++
			assert.Equal(t, "Api-Token some token", req.Header.Get("Authorization"))
			rw.WriteHeader(200)
			_, _ = rw.Write(accessTokenPayload)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{
				AccessToken: &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "some token"},
				OAuth: &manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "OAUTH_ID",
						Value: "123",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "OAUTH_SECRET",
						Value: "xyz",
					},
					TokenEndpoint: &manifest.URLDefinition{
						Value: server.URL + "/sso",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, classicCalls)
		assert.Equal(t, 1, platformCalls)
	})

	t.Run("Platform token and access token are validated", func(t *testing.T) {
		classicCalls := 0
		platformCalls := 0

		mux := http.NewServeMux()
		mux.HandleFunc(metadata.ClassicEnvironmentDomainPath, func(rw http.ResponseWriter, req *http.Request) {
			platformCalls++
			assert.Equal(t, "Bearer platform token", req.Header.Get("Authorization"))
			rw.WriteHeader(200)
			_, _ = rw.Write(getClassicEnvPayload(req.Host))
		})
		mux.HandleFunc("/api/v2/apiTokens/lookup", func(rw http.ResponseWriter, req *http.Request) {
			classicCalls++
			assert.Equal(t, "Api-Token api token", req.Header.Get("Authorization"))
			rw.WriteHeader(200)
			_, _ = rw.Write(accessTokenPayload)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{
				AccessToken:   &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "api token"},
				PlatformToken: &manifest.AuthSecret{Name: "DT_PLATFORM_TOKEN", Value: "platform token"},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, classicCalls)
		assert.Equal(t, 1, platformCalls)
	})

	t.Run("OAuth is validated and errors even if access token is given and valid", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/sso", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(400)
		})
		mux.HandleFunc("/api/v2/apiTokens/lookup", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(200)
			_, _ = rw.Write(accessTokenPayload)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		err := VerifyEnvironmentAuthentication(context.TODO(), manifest.EnvironmentDefinition{
			Name: "env",
			URL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Name:  "URL",
				Value: server.URL,
			},
			Auth: manifest.Auth{
				AccessToken: &manifest.AuthSecret{Name: "DT_API_TOKEN", Value: "some token"},
				OAuth: &manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "OAUTH_ID",
						Value: "123",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "OAUTH_SECRET",
						Value: "xyz",
					},
					TokenEndpoint: &manifest.URLDefinition{
						Value: server.URL + "/sso",
					},
				},
			},
		})
		assert.Error(t, err)
	})
}
