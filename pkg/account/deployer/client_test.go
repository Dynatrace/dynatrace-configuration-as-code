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
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

func TestClient_UpsertUser_UserAlreadyExists(t *testing.T) {

	payload := `{
	   "uid": "3288032b-9bdc-4480-bb11-2ec0ad2610b2",
	   "email": "abcd@ef.com",
	   "emergencyContact": false,
	   "userStatus": "PENDING",
	   "groups": []
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
	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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
	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
	id, err := instance.upsertUser(context.TODO(), "abcd@ef.com")
	assert.Error(t, err)
	assert.Zero(t, id)
	assert.Equal(t, 2, server.Calls())
}

const testAccountPutGroupResponseBody = `{
	"uuid": "5d9ba2f2-a00c-433b-b5fa-589c5120244b",
	"name": "my-group",
	"description": "group-description",
	"federatedAttributeValues": [],
	"owner": {}
 }`

func TestClient_UpsertGroup_UpdateExistingLocalGroupWorks(t *testing.T) {
	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: makeTestAccountGetGroupsResponseBody("LOCAL"),
				}
			},
		},
		{
			PUT: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: testAccountPutGroupResponseBody,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
	id, err := instance.upsertGroup(context.TODO(), "", Group{Name: "my-group"})
	assert.NoError(t, err)
	assert.Equal(t, 2, server.Calls())
	assert.Equal(t, "5d9ba2f2-a00c-433b-b5fa-589c5120244b", id)
}

func TestClient_UpsertGroup_UpdateExistingSCIMGroupSkipped(t *testing.T) {
	t.Setenv(featureflags.SkipReadOnlyAccountGroupUpdates.EnvName(), "true") // turn on SkipReadOnlyAccountGroupUpdates for this test

	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: makeTestAccountGetGroupsResponseBody("SCIM"),
				}
			},
		},
		{
			PUT: func(t *testing.T, request *http.Request) testutils.Response {
				// this should not occur as SCIM groups should not be updated
				assert.FailNow(t, "Unexpected PUT request for SCIM account group")
				return testutils.Response{}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
	id, err := instance.upsertGroup(context.TODO(), "", Group{Name: "my-group"})
	assert.NoError(t, err)
	assert.Equal(t, 1, server.Calls())
	assert.Equal(t, "5d9ba2f2-a00c-433b-b5fa-589c5120244b", id)
}

func TestClient_UpsertGroup_UpdateExistingAllUsersGroupSkipped(t *testing.T) {
	t.Setenv(featureflags.SkipReadOnlyAccountGroupUpdates.EnvName(), "true") // turn on SkipReadOnlyAccountGroupUpdates for this test

	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: makeTestAccountGetGroupsResponseBody("ALL_USERS"),
				}
			},
		},
		{
			PUT: func(t *testing.T, request *http.Request) testutils.Response {
				// this should not occur as ALL_USERS groups should not be updated
				assert.FailNow(t, "Unexpected PUT request for ALL_USERS account group")
				return testutils.Response{}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
	id, err := instance.upsertGroup(context.TODO(), "", Group{Name: "my-group"})
	assert.NoError(t, err)
	assert.Equal(t, 1, server.Calls())
	assert.Equal(t, "5d9ba2f2-a00c-433b-b5fa-589c5120244b", id)
}

func makeTestAccountGetGroupsResponseBody(owner string) string {
	return `{
	"count": 1,
	"items": [
		 {
		 "uuid": "5d9ba2f2-a00c-433b-b5fa-589c5120244b",
		 "name": "my-group",
		 "createdAt": "2024-11-06T17:42:22Z",
		 "updatedAt": "2024-11-06T17:42:22Z",
		 "description": "group-description",
		 "federatedAttributeValues": [],
		 "owner": "` + owner + `"
		 }
	 ]
 }`
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
	   "createdAt": "2024-11-06T17:42:22Z",
	   "updatedAt": "2024-11-06T17:42:22Z",
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

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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
	 "createdAt": "2024-11-06T17:42:22Z",
	 "updatedAt": "2024-11-06T17:42:22Z",
	 "federatedAttributeValues": [],
	 "owner": "5d9ba2f2-a00c-433b-b5fa-589c5120244b"
   }
 ]`,
				}
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

	t.Run("Update Group Permissions - no group Id given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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
					assert.JSONEq(t, `[]`, string(b))
				},
			},
		}
		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updatePermissions(context.TODO(), "10bcc894-9b24-4b39-b26d-61622d4e163e", []accountmanagement.PermissionsDto{})
		assert.NoError(t, err)
		assert.Equal(t, 1, server.Calls())
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

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateAccountPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "unable to update policy binding between group with UUID 8b78ac8d-74fd-456f-bb19-13e078674745 and policies with UUIDs [155a39a5-159f-475e-b2ff-681dad70896e] (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Account Policy Bindings - No group id given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateAccountPolicyBindings(context.TODO(), "", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "group id must not be empty", err.Error())
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Account Policy Bindings - empty policy uuids given", func(t *testing.T) {
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
					require.JSONEq(t, `{"policyUuids":[]}`, string(body))
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateAccountPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{})
		assert.NoError(t, err)
		assert.Equal(t, 1, server.Calls())
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

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateEnvironmentPolicyBindings(context.TODO(), "env1234", "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "unable to update policy binding between group with UUID 8b78ac8d-74fd-456f-bb19-13e078674745 and policies with UUIDs [155a39a5-159f-475e-b2ff-681dad70896e] (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Environment Policy Bindings - No group id given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateEnvironmentPolicyBindings(context.TODO(), "env1234", "", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "group id must not be empty", err.Error())
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Environment Policy Bindings - empty policy uuids given", func(t *testing.T) {
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
					require.JSONEq(t, `{"policyUuids":[]}`, string(body))
				},
			},
		}
		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateEnvironmentPolicyBindings(context.TODO(), "env1234", "8b78ac8d-74fd-456f-bb19-13e078674745", []string{})
		assert.NoError(t, err)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Environment Policy Bindings - empty environment name given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateGroupBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.NoError(t, err)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Group Bindings - API call fails", func(t *testing.T) {
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

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateGroupBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "unable to add user 8b78ac8d-74fd-456f-bb19-13e078674745 to groups [155a39a5-159f-475e-b2ff-681dad70896e] (HTTP 500): {\"error\" : \"some-error\"}", err.Error())
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("Update Group Bindings - No group id given", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, nil)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateGroupBindings(context.TODO(), "", []string{"155a39a5-159f-475e-b2ff-681dad70896e"})
		assert.Error(t, err)
		assert.Equal(t, "user id must not be empty", err.Error())
		assert.Equal(t, 0, server.Calls())
	})

	t.Run("Update Group Bindings - empty policy uuids given", func(t *testing.T) {
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
					require.JSONEq(t, `[]`, string(body))
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.updateGroupBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745", []string{})
		assert.NoError(t, err)
		assert.Equal(t, 1, server.Calls())
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
					ResponseBody: `{"uuid": "256d42d9-5a75-49d8-94cf-673c45b9410d","name": "Monaco Test Policy", "tags": [], "description": "", "statementQuery":"", "statements": []}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/policies/256d42d9-5a75-49d8-94cf-673c45b9410d", request.URL.String())
				body, _ := io.ReadAll(request.Body)
				assert.JSONEq(t, `{
   "description": "Just a monaco test policy",
   "name": "Monaco Test Policy",
   "statementQuery": "ALLOW automation:workflows:read;"
 }`, string(body))

			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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
	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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
					ResponseBody: `{"uuid": "5bc7ce51-a41f-47f3-a0ca-207c899c7747","name": "Monaco Test Policy",  "description": "Just a monaco test policy", "tags": [], "statementQuery": "ALLOW automation:workflows:read;", "statements":[]}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/policies", request.URL.String())
				body, _ := io.ReadAll(request.Body)
				assert.JSONEq(t, `{
   "description": "Just a monaco test policy",
   "name": "Monaco Test Policy",
   "statementQuery": "ALLOW automation:workflows:read;"
 }`, string(body))
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()
	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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
	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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
	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
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
	instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
	policiesMap, err := instance.getGlobalPolicies(context.TODO())
	assert.NoError(t, err)
	assert.Len(t, policiesMap, 2)
	assert.Equal(t, policiesMap["Policy 1"], "8d68fb35-0fa9-499e-b924-55f1629dc71e")
	assert.Equal(t, policiesMap["Policy 2"], "a6f0bf51-dc92-4712-8fe7-73dfff2c3898")

}

func TestClient_DeleteAllEnvironmentPolicyBindings(t *testing.T) {

	t.Run("Delete all Policy Bindings - delete call fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"data":[{"id":"vsy13800","url":"https://vsy13800.dev.dynatracelabs.com","active":true,"name": "vsy13800"}]}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/env/v2/accounts/abcde/environments", request.URL.String())
				},
			},
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"policyUuids":["4136e779-3447-4d6f-8457-745dc23c00da","425179d0-791a-4aeb-8c87-c61207bfffd9"]}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/repo/environment/vsy13800/bindings/groups/8b78ac8d-74fd-456f-bb19-13e078674745", request.URL.String())
				},
			},
			{
				DELETE: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusInternalServerError,
						ResponseBody: `{ "error" : "something went wrong}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/repo/environment/vsy13800/bindings/4136e779-3447-4d6f-8457-745dc23c00da/8b78ac8d-74fd-456f-bb19-13e078674745?forceMultiple=true", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.deleteAllEnvironmentPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745")
		assert.Error(t, err)
		assert.Equal(t, 3, server.Calls())
	})

	t.Run("Delete all Policy Bindings - getting bindings call fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"data":[{"id":"vsy13800","url":"https://vsy13800.dev.dynatracelabs.com","active":true,"name": "vsy13800"}]}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/env/v2/accounts/abcde/environments", request.URL.String())
				},
			},
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusInternalServerError,
						ResponseBody: `{"error" : " something went wrong"}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/repo/environment/vsy13800/bindings/groups/8b78ac8d-74fd-456f-bb19-13e078674745", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.deleteAllEnvironmentPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745")
		assert.Error(t, err)
		assert.Equal(t, 2, server.Calls())
	})

	t.Run("Delete all Policy Bindings - getting environments fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusInternalServerError,
						ResponseBody: `{"data":[{"id":"vsy13800","url":"https://vsy13800.dev.dynatracelabs.com","active":true}]}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/env/v2/accounts/abcde/environments", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.deleteAllEnvironmentPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745")
		assert.Error(t, err)
		assert.Equal(t, 1, server.Calls())
	})
	t.Run("Delete all Policy Bindings - OK", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"data":[{"id":"vsy13800","url":"https://vsy13800.dev.dynatracelabs.com","active":true,"name": "vsy13800"}]}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/env/v2/accounts/abcde/environments", request.URL.String())
				},
			},
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"policyUuids":["4136e779-3447-4d6f-8457-745dc23c00da","425179d0-791a-4aeb-8c87-c61207bfffd9"]}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/repo/environment/vsy13800/bindings/groups/8b78ac8d-74fd-456f-bb19-13e078674745", request.URL.String())
				},
			},
			{
				DELETE: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"policyUuids":["4136e779-3447-4d6f-8457-745dc23c00da","425179d0-791a-4aeb-8c87-c61207bfffd9"]}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/repo/environment/vsy13800/bindings/4136e779-3447-4d6f-8457-745dc23c00da/8b78ac8d-74fd-456f-bb19-13e078674745?forceMultiple=true", request.URL.String())
				},
			},
			{
				DELETE: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"policyUuids":["4136e779-3447-4d6f-8457-745dc23c00da","425179d0-791a-4aeb-8c87-c61207bfffd9"]}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/repo/environment/vsy13800/bindings/425179d0-791a-4aeb-8c87-c61207bfffd9/8b78ac8d-74fd-456f-bb19-13e078674745?forceMultiple=true", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		err := instance.deleteAllEnvironmentPolicyBindings(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745")
		assert.NoError(t, err)
		assert.Equal(t, 4, server.Calls())
	})

}

func TestClient_getServiceUserEmailByUid(t *testing.T) {

	t.Run("success if found", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
  "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
  "name": "service-user",
  "surname": "string",
  "description": "string",
  "createdAt": "string",
  "groupUuid": "string"
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674744", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		email, err := instance.getServiceUserEmailByUid(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674744")
		assert.NoError(t, err)
		assert.Equal(t, "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com", email)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("returns ResourceNotFoundError if not found", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusNotFound,
						ResponseBody: `{
  "error": true,
  "payload": null,
  "message": "Requested service user not found"
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674744", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		_, err := instance.getServiceUserEmailByUid(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674744")
		rnfErr := ResourceNotFoundError{}
		assert.ErrorAs(t, err, &rnfErr)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("returns an error if failed", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusBadRequest,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674745", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		_, err := instance.getServiceUserEmailByUid(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745")
		assert.Error(t, err)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("returns an error if response empty", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674745", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		_, err := instance.getServiceUserEmailByUid(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674745")
		assert.ErrorContains(t, err, "the received data are empty")
		assert.Equal(t, 1, server.Calls())
	})
}

func TestClient_getServiceUserEmailByName(t *testing.T) {

	t.Run("success if found", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "results": [
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    }
  ],
  "totalCount": 1
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		email, err := instance.getServiceUserEmailByName(context.TODO(), "service-user")
		assert.NoError(t, err)
		assert.Equal(t, "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com", email)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("returns ResourceNotFoundError if not found", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "results": [
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674745",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674745@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    }
  ],
  "totalCount": 1
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		_, err := instance.getServiceUserEmailByName(context.TODO(), "another-service-user")
		rnfErr := &ResourceNotFoundError{}
		assert.ErrorAs(t, err, &rnfErr)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("returns an error if multiple are found", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "results": [
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    },
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    }
  ],
  "totalCount": 2
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		_, err := instance.getServiceUserEmailByName(context.TODO(), "service-user")
		assert.ErrorContains(t, err, "found multiple service users")
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("returns an error if list failed", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusBadRequest,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		_, err := instance.getServiceUserEmailByName(context.TODO(), "service-user")
		assert.Error(t, err)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("returns an error if response empty", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		_, err := instance.getServiceUserEmailByName(context.TODO(), "service-user")
		assert.ErrorContains(t, err, "the received data are empty")
		assert.Equal(t, 1, server.Calls())
	})
}

func TestClient_UpsertServiceUser(t *testing.T) {

	t.Run("upsert with ID and update works", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674744", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674744", ServiceUser{Name: "service-user", Description: accountmanagement.PtrString("A service user")})
		assert.NoError(t, err)
		assert.Equal(t, "8b78ac8d-74fd-456f-bb19-13e078674744", remoteId)
		assert.Equal(t, 1, server.Calls())
	})

	t.Run("upsert with ID returns error if update fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusBadRequest,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674744", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674744", ServiceUser{Name: "service-user", Description: accountmanagement.PtrString("A service user")})
		assert.Empty(t, remoteId)
		assert.ErrorContains(t, err, "failed to update service user")

		assert.Equal(t, 1, server.Calls())
	})

	t.Run("upsert with ID returns error if service user not found", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusNotFound,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674744", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "8b78ac8d-74fd-456f-bb19-13e078674744", ServiceUser{Name: "service-user", Description: accountmanagement.PtrString("A service user")})
		assert.Empty(t, remoteId)
		assert.ErrorContains(t, err, "not found")

		assert.Equal(t, 1, server.Calls())
	})

	t.Run("upsert with name succeeds", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "results": [
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    }
  ],
  "totalCount": 1
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674744", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "", ServiceUser{Name: "service-user", Description: accountmanagement.PtrString("A service user")})
		assert.Equal(t, "8b78ac8d-74fd-456f-bb19-13e078674744", remoteId)
		assert.NoError(t, err)

		assert.Equal(t, 2, server.Calls())
	})

	t.Run("upsert with name list fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusBadRequest,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "", ServiceUser{Name: "service-user", Description: accountmanagement.PtrString("A service user")})
		assert.Empty(t, remoteId)
		assert.ErrorContains(t, err, "failed to get service users")

		assert.Equal(t, 1, server.Calls())
	})

	t.Run("upsert with name fails if multiple are found", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "results": [
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    },
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    }
  ],
  "totalCount": 2
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "", ServiceUser{Name: "service-user", Description: accountmanagement.PtrString("A service user")})
		assert.Empty(t, remoteId)
		assert.ErrorContains(t, err, "found multiple service users")

		assert.Equal(t, 1, server.Calls())
	})

	t.Run("upsert with name creates if not found", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "results": [
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    }
  ],
  "totalCount": 1
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
			{
				POST: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "uid": "8b78ac8d-74fd-456f-bb19-13e078674745",
  "email": "8b78ac8d-74fd-456f-bb19-13e078674745@service.sso.dynatrace.com",
  "name": "another-service-user",
  "surname": "string",
  "description": "string",
  "createdAt": "string",
  "groupUuid": "string"
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "", ServiceUser{Name: "another-service-user", Description: accountmanagement.PtrString("Another service user")})
		assert.Equal(t, "8b78ac8d-74fd-456f-bb19-13e078674745", remoteId)
		assert.NoError(t, err)

		assert.Equal(t, 2, server.Calls())
	})

	t.Run("upsert with name errors if not found but create fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "results": [
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    }
  ],
  "totalCount": 1
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
			{
				POST: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusBadRequest,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "", ServiceUser{Name: "another-service-user", Description: accountmanagement.PtrString("Another service user")})
		assert.Empty(t, remoteId)
		assert.ErrorContains(t, err, "failed to create service user")

		assert.Equal(t, 2, server.Calls())
	})

	t.Run("upsert with name errors if update fails", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{
  "results": [
    {
      "uid": "8b78ac8d-74fd-456f-bb19-13e078674744",
      "email": "8b78ac8d-74fd-456f-bb19-13e078674744@service.sso.dynatrace.com",
      "name": "service-user",
      "surname": "string",
      "description": "string",
      "createdAt": "string"
    }
  ],
  "totalCount": 1
}`,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users?page=1&page-size=1000", request.URL.String())
				},
			},
			{
				PUT: func(t *testing.T, request *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusBadRequest,
					}
				},
				ValidateRequest: func(t *testing.T, request *http.Request) {
					assert.Equal(t, "/iam/v1/accounts/abcde/service-users/8b78ac8d-74fd-456f-bb19-13e078674744", request.URL.String())
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		instance := NewClient(account.AccountInfo{Name: "my-account", AccountUUID: "abcde"}, accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
		remoteId, err := instance.upsertServiceUser(context.TODO(), "", ServiceUser{Name: "service-user", Description: accountmanagement.PtrString("A service user")})
		assert.Empty(t, remoteId)
		assert.ErrorContains(t, err, "failed to update service user")

		assert.Equal(t, 2, server.Calls())
	})
}
