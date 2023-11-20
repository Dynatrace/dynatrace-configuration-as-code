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

package deployer

import (
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestClient_UpsertUser_UserAlreadyExists(t *testing.T) {

	payload := `{
  "count": 1,
  "items": [
    {
      "uid": "3288032b-9bdc-4480-bb11-2ec0ad2610b2",
      "email": "abcd@ef.com",
      "emergencyContact": false,
      "userStatus": "PENDING",
      "type": "DEFAULT"
    }
  ]
}`

	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: payload,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertUser(context.TODO(), "abcd@ef.com")
	assert.NoError(t, err)
	assert.Equal(t, "abcd@ef.com", id)
}

func TestClient_UpsertUser_Get_Existing_Users_Fails(t *testing.T) {

	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusInternalServerError,
					ResponseBody: `{}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertUser(context.TODO(), "abcd@ef.com")
	assert.Error(t, err)
	assert.Zero(t, id)
}

func TestClient_UpsertUser_CreateNewUser(t *testing.T) {

	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusNotFound,
					ResponseBody: `{}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/accounts/abcde/users/abcd@ef.com", request.URL.String())
			},
		},
		{
			POST: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusCreated,
					ResponseBody: `{}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/accounts/abcde/users", request.URL.String())
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertUser(context.TODO(), "abcd@ef.com")
	assert.NoError(t, err)
	assert.Equal(t, "abcd@ef.com", id)
	assert.Equal(t, 2, server.Calls())
}

func TestClient_UpsertUser_CreatingNewFails(t *testing.T) {

	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusNotFound,
					ResponseBody: `{}`,
				}
			},
		},
		{
			POST: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusInternalServerError,
					ResponseBody: `{}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertUser(context.TODO(), "abcd@ef.com")
	assert.Error(t, err)
	assert.Zero(t, id)
	assert.Equal(t, 2, server.Calls())
}

func TestClient_UpsertGroup_Update_Existing(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{
  "count": 1,
  "items": [
    {
      "uuid": "5d9ba2f2-a00c-433b-b5fa-589c5120244b",
      "name": "my-group",
      "description": "group-description",
      "federatedAttributeValues": [],
      "owner": "LOCAL"
    }
  ]
}`,
				}
			},
		},
		{
			PUT: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{
  "uuid": "5d9ba2f2-a00c-433b-b5fa-589c5120244b",
  "name": "my-group",
  "description": "group-description",
  "federatedAttributeValues": [],
  "owner": {}
}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertGroup(context.TODO(), "", Group{Name: "my-group"})
	assert.NoError(t, err)
	assert.Equal(t, 2, server.Calls())
	assert.Equal(t, "5d9ba2f2-a00c-433b-b5fa-589c5120244b", id)
}
func TestClient_UpsertGroup_Update_Existing_Fails(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{
  "count": 1,
  "items": [
    {
      "uuid": "5d9ba2f2-a00c-433b-b5fa-589c5120244b",
      "name": "my-group",
      "description": "group-description",
      "federatedAttributeValues": [],
      "owner": "LOCAL"
    }
  ]
}`,
				}
			},
		},
		{
			PUT: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusInternalServerError,
					ResponseBody: `{}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertGroup(context.TODO(), "", Group{Name: "my-group"})
	assert.Error(t, err)
	assert.Equal(t, 2, server.Calls())
	assert.Zero(t, id)
}
func TestClient_UpsertGroup_Getting_Existing_Groups_Fail(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusInternalServerError,
					ResponseBody: `{}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertGroup(context.TODO(), "", Group{Name: "my-group"})
	assert.Error(t, err)
	assert.Equal(t, 1, server.Calls())
	assert.Zero(t, id)
}
func TestClient_UpsertGroup_Create_New(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{"count": 0,"items": []}`,
				}
			},
		},
		{
			POST: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `[
  {
    "uuid": "5d9ba2f2-a00c-433b-b5fa-589c5120244b",
    "name": "my-group",
    "description": "This is my group",
    "federatedAttributeValues": [],
    "owner": {}
  }
]`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertGroup(context.TODO(), "", Group{Name: "my-group"})
	assert.NoError(t, err)
	assert.Equal(t, 2, server.Calls())
	assert.Equal(t, "5d9ba2f2-a00c-433b-b5fa-589c5120244b", id)
}
func TestClient_UpsertGroup_Create_New_Fails(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{"count": 0,"items": []}`,
				}
			},
		},
		{
			POST: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusInternalServerError,
					ResponseBody: `{}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertGroup(context.TODO(), "", Group{Name: "my-group"})
	assert.Error(t, err)
	assert.Equal(t, 2, server.Calls())
	assert.Zero(t, id)
}

func TestClient_UpdateGroupPermissions(t *testing.T) {

	t.Run("Update Group Permissions - OK", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"count": 0,"items": []}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/groups/10bcc894-9b24-4b39-b26d-61622d4e163e/permissions", request.URL.String())
					b, _ := io.ReadAll(request.Body)
					assert.JSONEq(t, `[{"permissionName":"tenant-viewer","scope":"account","scopeType":"abcde"}]`, string(b))
				},
			},
		}
		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updatePermissions(context.TODO(), "10bcc894-9b24-4b39-b26d-61622d4e163e", []accountmanagement.PermissionsDto{{
			PermissionName: "tenant-viewer",
			Scope:          "account",
			ScopeType:      "abcde",
		},
		})
		assert.Equal(t, 1, server.Calls())
		assert.NoError(t, err)
	})

	t.Run("Update Group Permissions - API call fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusInternalServerError,
						ResponseBody: `{"error": "some-error"}`,
					}
				},
			},
		}
		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updatePermissions(context.TODO(), "10bcc894-9b24-4b39-b26d-61622d4e163e", []accountmanagement.PermissionsDto{{
			PermissionName: "tenant-viewer",
			Scope:          "account",
			ScopeType:      "abcde",
		},
		})
		assert.Equal(t, 1, server.Calls())
		assert.Error(t, err)
		assert.Equal(t, "unable to update permissions of group with UUID 10bcc894-9b24-4b39-b26d-61622d4e163e (HTTP 500): {\"error\": \"some-error\"}", err.Error())
	})

	t.Run("Update Group Permissions - Unsupported Permission", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updatePermissions(context.TODO(), "10bcc894-9b24-4b39-b26d-61622d4e163e", []accountmanagement.PermissionsDto{{
			PermissionName: "unsupported-permission",
			Scope:          "account",
			ScopeType:      "abcde",
		},
		})
		assert.Equal(t, 0, server.Calls())
		assert.Error(t, err)
		assert.Equal(t, "unsupported permission \"unsupported-permission\". Must be one of: [tenant-viewer]", err.Error())
	})

	t.Run("Update Group Permissions - no group Id given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updatePermissions(context.TODO(), "", []accountmanagement.PermissionsDto{{
			PermissionName: "perm1",
			Scope:          "account",
			ScopeType:      "abcde",
		},
		})
		assert.Error(t, err)
		assert.Equal(t, "group id must not be empty", err.Error())
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Group Permissions - no permissions given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updatePermissions(context.TODO(), "10bcc894-9b24-4b39-b26d-61622d4e163e", []accountmanagement.PermissionsDto{})
		assert.NoError(t, err)
		assert.Equal(t, 0, server.Calls())
	})

}

func TestClient_UpdatePolicyBindings(t *testing.T) {

	t.Run("Update Account Policy Bindings - OK", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/repo/account/abcde/bindings/groups/8b78ac8d-74fd-456f-bb19-13e078674745", request.URL.String())
					body, _ := io.ReadAll(request.Body)
					require.JSONEq(t, `{"policyUuids":["155a39a5-159f-475e-b2ff-681dad70896e"]}`, string(body))
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateAccountPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.NoError(t, err)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Account Policy Bindings - API call fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusInternalServerError,
						ResponseBody: `{"error" : "some-error"}`,
					}
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateAccountPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "unable to update policy binding between group with UUID 8b78ac8d-74fd-456f-bb19-13e078674745 and policies with UUIDs [155a39a5-159f-475e-b2ff-681dad70896e] (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Account Policy Bindings - No group id given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateAccountPolicyBindings(context.TODO(), "", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "group id must not be empty", err.Error())
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Account Policy Bindings - empty policy uuids given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateAccountPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{})
		assert.NoError(t, err)
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Environment Policy Bindings - OK", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/repo/environment/env1234/bindings/groups/8b78ac8d-74fd-456f-bb19-13e078674745", request.URL.String())
					body, _ := io.ReadAll(request.Body)
					require.JSONEq(t, `{"policyUuids":["155a39a5-159f-475e-b2ff-681dad70896e"]}`, string(body))
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateEnvironmentPolicyBindings(context.TODO(), "env1234", "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.NoError(t, err)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Environment Policy Bindings - API call fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusInternalServerError,
						ResponseBody: `{"error" : "some-error"}`,
					}
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateEnvironmentPolicyBindings(context.TODO(), "env1234", "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "unable to update policy binding between group with UUID 8b78ac8d-74fd-456f-bb19-13e078674745 and policies with UUIDs [155a39a5-159f-475e-b2ff-681dad70896e] (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Environment Policy Bindings - No group id given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateEnvironmentPolicyBindings(context.TODO(), "env1234", "", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "group id must not be empty", err.Error())
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Environment Policy Bindings - empty policy uuids given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateEnvironmentPolicyBindings(context.TODO(), "env1234", "8b78ac8d-74fd-456f-bb19-13e078674745", []string{})
		assert.NoError(t, err)
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Environment Policy Bindings - empty environment name given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateEnvironmentPolicyBindings(context.TODO(), "", "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "environment name must not be empty", err.Error())
		assert.Equal(t, 0, server.Calls())
	})
}

func TestClient_UpdateGroupBindings(t *testing.T) {

	t.Run("Update Group Bindings - OK", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/users/8b78ac8d-74fd-456f-bb19-13e078674745/groups", request.URL.String())
					body, _ := io.ReadAll(request.Body)
					require.JSONEq(t, `["155a39a5-159f-475e-b2ff-681dad70896e"]`, string(body))
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateGroupBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.NoError(t, err)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Grou pBindings - API call fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusInternalServerError,
						ResponseBody: `{"error" : "some-error"}`,
					}
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateGroupBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "unable to add user 8b78ac8d-74fd-456f-bb19-13e078674745 to groups [155a39a5-159f-475e-b2ff-681dad70896e] (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Group Bindings - No group id given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateGroupBindings(context.TODO(), "", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "user id must not be empty", err.Error())
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Group Bindings - empty policy uuids given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
		err := instance.updateGroupBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{})
		assert.NoError(t, err)
		assert.Equal(t, 0, server.Calls())
	})

}

func TestClient_UpsertPolicy_UpdateExisting(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{
  "policies": [
    {
      "uuid": "256d42d9-5a75-49d8-94cf-673c45b9410d",
      "name": "my-policy",
      "description": "This is my policy"
    }
  ]
}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/policies?name=Monaco+Test+Policy", request.URL.String())
			},
		},
		{
			PUT: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{"uuid": "256d42d9-5a75-49d8-94cf-673c45b9410d","name": "Monaco Test Policy"}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/policies/256d42d9-5a75-49d8-94cf-673c45b9410d", request.URL.String())
				body, _ := io.ReadAll(request.Body)
				assert.JSONEq(t, `{
  "description": "Just a monaco test policy",
  "name": "Monaco Test Policy",
  "statementQuery": "ALLOW automation:workflows:read;",
  "tags": null
}`, string(body))

			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertPolicy(context.TODO(), "account", "abcde", "", Policy{
		Name:           "Monaco Test Policy",
		Description:    "Just a monaco test policy",
		StatementQuery: "ALLOW automation:workflows:read;",
	})
	assert.NoError(t, err)
	assert.Equal(t, "256d42d9-5a75-49d8-94cf-673c45b9410d", id)
}

func TestClient_UpsertPolicy_UpdateExisting_UpdateFails(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{
  "policies": [
    {
      "uuid": "256d42d9-5a75-49d8-94cf-673c45b9410d",
      "name": "my-policy",
      "description": "This is my policy"
    }
  ]
}`,
				}
			},
		},
		{
			PUT: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusInternalServerError,
					ResponseBody: `{"error" : "some-error"}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertPolicy(context.TODO(), "account", "abcde", "", Policy{
		Name:           "Monaco Test Policy",
		Description:    "Just a monaco test policy",
		StatementQuery: "ALLOW automation:workflows:read;",
	})
	assert.Error(t, err)
	assert.Zero(t, id)
	assert.Equal(t, "unable to update policy with name: Monaco Test Policy (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
}

func TestClient_UpsertPolicy_CreateNew(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{"policies": []}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/policies?name=Monaco+Test+Policy", request.URL.String())
			},
		},
		{
			POST: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{"uuid": "5bc7ce51-a41f-47f3-a0ca-207c899c7747","name": "Monaco Test Policy"}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/policies", request.URL.String())
				body, _ := io.ReadAll(request.Body)
				assert.JSONEq(t, `{
  "description": "Just a monaco test policy",
  "name": "Monaco Test Policy",
  "statementQuery": "ALLOW automation:workflows:read;",
  "tags": null
}`, string(body))
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertPolicy(context.TODO(), "account", "abcde", "", Policy{
		Name:           "Monaco Test Policy",
		Description:    "Just a monaco test policy",
		StatementQuery: "ALLOW automation:workflows:read;",
	})
	assert.NoError(t, err)
	assert.Equal(t, "5bc7ce51-a41f-47f3-a0ca-207c899c7747", id)
}

func TestClient_UpsertPolicy_CreateNew_CreateFails(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{"policies": []}`,
				}
			},
		},
		{
			POST: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusInternalServerError,
					ResponseBody: `{"error" : "some-error"}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertPolicy(context.TODO(), "account", "abcde", "", Policy{
		Name:           "Monaco Test Policy",
		Description:    "Just a monaco test policy",
		StatementQuery: "ALLOW automation:workflows:read;",
	})
	assert.Error(t, err)
	assert.Zero(t, id)
	assert.Equal(t, "unable to create policy with name: Monaco Test Policy (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
}

func TestClient_UpsertPolicy_GetPoliciesFails(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusInternalServerError,
					ResponseBody: `{"error" : "some-error"}`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	id, err := instance.upsertPolicy(context.TODO(), "account", "abcde", "", Policy{
		Name:           "Monaco Test Policy",
		Description:    "Just a monaco test policy",
		StatementQuery: "ALLOW automation:workflows:read;",
	})
	assert.Error(t, err)
	assert.Zero(t, id)
	assert.Equal(t, "unable to get policy with name: Monaco Test Policy (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
}

func TestClient_GetGlobalPolicies(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{
  "policies": [
    {
      "uuid": "8d68fb35-0fa9-499e-b924-55f1629dc71e",
      "name": "Policy 1",
      "description": "I am policy 1"
    },
    {
      "uuid": "a6f0bf51-dc92-4712-8fe7-73dfff2c3898",
      "name": "Policy 2",
      "description": "I am policy 2"
    }
  ]
}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/global/global/policies", request.URL.String())
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())), []string{"tenant-viewer"})
	policiesMap, err := instance.getGlobalPolicies(context.TODO())
	assert.NoError(t, err)
	assert.Len(t, policiesMap, 2)
	assert.Equal(t, policiesMap["Policy 1"], "8d68fb35-0fa9-499e-b924-55f1629dc71e")
	assert.Equal(t, policiesMap["Policy 2"], "a6f0bf51-dc92-4712-8fe7-73dfff2c3898")

}
