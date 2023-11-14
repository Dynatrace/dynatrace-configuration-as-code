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

package delete_test

import (
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/delete"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAccountAPIClient_DeleteGroup(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/accounts/1234/groups") {
				t.Fatalf("expected API call to '/iam/v1/accounts/1234/groups' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/accounts/1234/groups":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
 "count": 2,
 "items": [
   {
     "uuid": "5678",
     "name": "test-group",
     "description": "THIS SHOULD BE FOUND AND DELETED",
     "federatedAttributeValues": [],
     "owner": "LOCAL",
     "createdAt": "2023-11-14T00:00:00",
     "updatedAt": "2023-11-14T00:00:00"
   },
   {
     "uuid": "8765",
     "name": "another-group",
     "description": "THIS IS SOMETHING ELSE",
     "federatedAttributeValues": [ "string" ],
     "owner": "LOCAL",
     "createdAt": "2023-11-14T00:00:00",
     "updatedAt": "2023-11-14T00:00:00"
   }
 ]
}`))
			case "/iam/v1/accounts/1234/groups/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(200)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteGroup(context.Background(), "test-group")
		assert.NoError(t, err)
	})
	t.Run("does nothing if name is not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/accounts/1234/groups") {
				t.Fatalf("expected API call to '/iam/v1/accounts/1234/groups' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/accounts/1234/groups":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
 "count": 1,
 "items": [
   {
     "uuid": "8765",
     "name": "another-group",
     "description": "THIS IS SOMETHING ELSE",
     "federatedAttributeValues": [ "string" ],
     "owner": "LOCAL",
     "createdAt": "2023-11-14T00:00:00",
     "updatedAt": "2023-11-14T00:00:00"
   }
 ]
}`))
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteGroup(context.Background(), "test-group")
		assert.ErrorIs(t, err, delete.NotFoundErr)
	})
	t.Run("returns NotFoundError if delete result is a 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/accounts/1234/groups") {
				t.Fatalf("expected API call to '/iam/v1/accounts/1234/groups' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/accounts/1234/groups":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
 "count": 2,
 "items": [
   {
     "uuid": "5678",
     "name": "test-group",
     "description": "THIS SHOULD BE FOUND AND DELETED",
     "federatedAttributeValues": [],
     "owner": "LOCAL",
     "createdAt": "2023-11-14T00:00:00",
     "updatedAt": "2023-11-14T00:00:00"
   },
   {
     "uuid": "8765",
     "name": "another-group",
     "description": "THIS IS SOMETHING ELSE",
     "federatedAttributeValues": [ "string" ],
     "owner": "LOCAL",
     "createdAt": "2023-11-14T00:00:00",
     "updatedAt": "2023-11-14T00:00:00"
   }
 ]
}`))
			case "/iam/v1/accounts/1234/groups/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(404)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteGroup(context.Background(), "test-group")
		assert.ErrorIs(t, err, delete.NotFoundErr)
	})
	t.Run("returns an error if finding ID failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/accounts/1234/groups") {
				t.Fatalf("expected API call to '/iam/v1/accounts/1234/groups' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/accounts/1234/groups":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.WriteHeader(400)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteGroup(context.Background(), "test-group")
		assert.Error(t, err)
	})
	t.Run("returns an error if delete failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/accounts/1234/groups") {
				t.Fatalf("expected API call to '/iam/v1/accounts/1234/groups' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/accounts/1234/groups":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
 "count": 2,
 "items": [
   {
     "uuid": "5678",
     "name": "test-group",
     "description": "THIS SHOULD BE FOUND AND DELETED",
     "federatedAttributeValues": [],
     "owner": "LOCAL",
     "createdAt": "2023-11-14T00:00:00",
     "updatedAt": "2023-11-14T00:00:00"
   },
   {
     "uuid": "8765",
     "name": "another-group",
     "description": "THIS IS SOMETHING ELSE",
     "federatedAttributeValues": [ "string" ],
     "owner": "LOCAL",
     "createdAt": "2023-11-14T00:00:00",
     "updatedAt": "2023-11-14T00:00:00"
   }
 ]
}`))
			case "/iam/v1/accounts/1234/groups/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(400)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteGroup(context.Background(), "test-group")
		assert.Error(t, err)
	})
}

func TestAccountAPIClient_DeleteAccountPolicy(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/account/1234/policies") {
				t.Fatalf("expected API call to '/iam/v1/repo/account/1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/account/1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
  "policies": [
    {
      "uuid": "5678",
      "name": "test-policy",
      "description": "THE POLICY TO DELETE"
    },
    {
      "uuid": "8765",
      "name": "another-policy",
      "description": "SOME OTHER THING"
    }
  ]
}`))
			case "/iam/v1/repo/account/1234/policies/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				assert.True(t, req.URL.Query().Has("force"))
				rw.WriteHeader(200)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteAccountPolicy(context.Background(), "test-policy")
		assert.NoError(t, err)
	})
	t.Run("does nothing if name is not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/account/1234/policies") {
				t.Fatalf("expected API call to '/iam/v1/repo/account/1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/account/1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
  "policies": [
    {
      "uuid": "8765",
      "name": "another-policy",
      "description": "SOME OTHER THING"
    }
  ]
}`))
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteAccountPolicy(context.Background(), "test-policy")
		assert.ErrorIs(t, err, delete.NotFoundErr)
	})
	t.Run("returns NotFoundError if delete result is a 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/account/1234/policies") {
				t.Fatalf("expected API call to '/iam/v1/repo/account/1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/account/1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
  "policies": [
    {
      "uuid": "5678",
      "name": "test-policy",
      "description": "THE POLICY TO DELETE"
    },
    {
      "uuid": "8765",
      "name": "another-policy",
      "description": "SOME OTHER THING"
    }
  ]
}`))
			case "/iam/v1/repo/account/1234/policies/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(404)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteAccountPolicy(context.Background(), "test-policy")
		assert.ErrorIs(t, err, delete.NotFoundErr)
	})
	t.Run("returns an error if finding ID failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/account/1234/policies") {
				t.Fatalf("expected API call to '/iam/v1/repo/account/1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/account/1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.WriteHeader(400)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteAccountPolicy(context.Background(), "test-policy")
		assert.Error(t, err)
	})
	t.Run("returns an error if delete failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/account/1234/policies") {
				t.Fatalf("expected API call to /iam/v1/repo/account/1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/account/1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
  "policies": [
    {
      "uuid": "5678",
      "name": "test-policy",
      "description": "THE POLICY TO DELETE"
    },
    {
      "uuid": "8765",
      "name": "another-policy",
      "description": "SOME OTHER THING"
    }
  ]
}`))
			case "/iam/v1/repo/account/1234/policies/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(400)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteAccountPolicy(context.Background(), "test-policy")
		assert.Error(t, err)
	})
}

func TestAccountAPIClient_DeleteEnvironmentPolicy(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/environment/abc1234/policies") {
				t.Fatalf("expected API call to '/iam/v1/repo/environment/abc1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/environment/abc1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
  "policies": [
    {
      "uuid": "5678",
      "name": "test-policy",
      "description": "THE POLICY TO DELETE"
    },
    {
      "uuid": "8765",
      "name": "another-policy",
      "description": "SOME OTHER THING"
    }
  ]
}`))
			case "/iam/v1/repo/environment/abc1234/policies/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				assert.True(t, req.URL.Query().Has("force"))
				rw.WriteHeader(200)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteEnvironmentPolicy(context.Background(), "abc1234", "test-policy")
		assert.NoError(t, err)
	})
	t.Run("does nothing if name is not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/environment/abc1234/policies") {
				t.Fatalf("expected API call to '/iam/v1/repo/environment/abc1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/environment/abc1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
  "policies": [
    {
      "uuid": "8765",
      "name": "another-policy",
      "description": "SOME OTHER THING"
    }
  ]
}`))
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteEnvironmentPolicy(context.Background(), "abc1234", "test-policy")
		assert.ErrorIs(t, err, delete.NotFoundErr)
	})
	t.Run("returns NotFoundError if delete result is a 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/environment/abc1234/policies") {
				t.Fatalf("expected API call to '/iam/v1/repo/environment/abc1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/environment/abc1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
  "policies": [
    {
      "uuid": "5678",
      "name": "test-policy",
      "description": "THE POLICY TO DELETE"
    },
    {
      "uuid": "8765",
      "name": "another-policy",
      "description": "SOME OTHER THING"
    }
  ]
}`))
			case "/iam/v1/repo/environment/abc1234/policies/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(404)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteEnvironmentPolicy(context.Background(), "abc1234", "test-policy")
		assert.ErrorIs(t, err, delete.NotFoundErr)
	})
	t.Run("returns an error if finding ID failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/environment/abc1234/policies") {
				t.Fatalf("expected API call to '/iam/v1/repo/environment/abc1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/environment/abc1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.WriteHeader(400)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteEnvironmentPolicy(context.Background(), "abc1234", "test-policy")
		assert.Error(t, err)
	})
	t.Run("returns an error if delete failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/repo/environment/abc1234/policies") {
				t.Fatalf("expected API call to /iam/v1/repo/environment/abc1234/policies' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/repo/environment/abc1234/policies":
				assert.Equal(t, http.MethodGet, req.Method)
				rw.Header().Set("Content-Type", "application/json")
				_, _ = rw.Write([]byte(`{
  "policies": [
    {
      "uuid": "5678",
      "name": "test-policy",
      "description": "THE POLICY TO DELETE"
    },
    {
      "uuid": "8765",
      "name": "another-policy",
      "description": "SOME OTHER THING"
    }
  ]
}`))
			case "/iam/v1/repo/environment/abc1234/policies/5678":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(400)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteEnvironmentPolicy(context.Background(), "abc1234", "test-policy")
		assert.Error(t, err)
	})
}

func TestAccountAPIClient_DeleteUser(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/accounts/1234/users") {
				t.Fatalf("expected API call to '/iam/v1/accounts/1234/users' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/accounts/1234/users/user@test.com":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(200)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteUser(context.Background(), "user@test.com")
		assert.NoError(t, err)
	})
	t.Run("returns NotFoundError if delete result is a 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/accounts/1234/users") {
				t.Fatalf("expected API call to '/iam/v1/accounts/1234/users' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/accounts/1234/users/user@test.com":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(404)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteUser(context.Background(), "user@test.com")
		assert.ErrorIs(t, err, delete.NotFoundErr)
	})
	t.Run("returns an error if delete failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/iam/v1/accounts/1234/users") {
				t.Fatalf("expected API call to '/iam/v1/accounts/1234/users' but got %q", req.URL.Path)
			}

			switch req.URL.Path {
			case "/iam/v1/accounts/1234/users/user@test.com":
				assert.Equal(t, http.MethodDelete, req.Method)
				rw.WriteHeader(400)
			default:
				t.Fatalf("Unexpected API call to %s", req.URL.Path)
			}
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		restClient := rest.NewClient(serverURL, server.Client())
		accountClient := delete.NewAccountAPIClient("1234", accounts.NewClient(restClient))

		err = accountClient.DeleteUser(context.Background(), "user@test.com")
		assert.Error(t, err)
	})
}
