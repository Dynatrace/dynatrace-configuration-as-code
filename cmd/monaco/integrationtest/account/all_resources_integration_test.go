//go:build integration

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

package account

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/writer"
)

func TestDeployAndDelete_AllResources(t *testing.T) {
	createMZone(t)
	t.Setenv(featureflags.ServiceUsers.EnvName(), "true")

	RunAccountTestCase(t, "resources/all-resources", "manifest-account.yaml", "am-all-resources", func(clients map[account.AccountInfo]*accounts.Client, o options) {

		accountName := o.accountName
		accountUUID := o.accountUUID
		myServiceUserName := "monaco service user %RAND%"
		myEmail := "monaco+%RAND%@dynatrace.com"
		myGroup := "My Group%RAND%"
		mySAMLGroup := "My SAML Group%RAND%"
		myLocalGroup := "My LOCAL Group%RAND%"
		myPolicy := "My Policy%RAND%"
		myPolicy2 := "My Policy 2%RAND%"
		envVkb := "vkb66581"

		check := AccountResourceChecker{
			Client:      clients[account.AccountInfo{Name: accountName, AccountUUID: accountUUID}],
			RandomizeFn: o.randomize,
		}

		// get current management zone id for later assertions
		mzones, _, err := check.Client.EnvironmentManagementAPI.GetEnvironmentResources(t.Context(), accountUUID).Execute()
		require.NoError(t, err)
		var mzoneID string
		for _, mz := range mzones.ManagementZoneResources {
			if mz.Name == "Management Zone 2000" {
				mzoneID = mz.Id
				break
			}
		}
		require.NotZero(t, mzoneID, "Could not get exact management zone id for assertions")

		cli := runner.BuildCmd(o.fs)
		// DEPLOY RESOURCES
		cli.SetArgs([]string{"account", "deploy", "-m", "manifest-account.yaml"})
		err = cli.Execute()
		require.NoError(t, err)

		// CHECK IF RESOURCES ARE INDEED DEPLOYED
		check.UserAvailable(t, accountUUID, myEmail)
		check.ServiceUserAvailable(t, accountUUID, myServiceUserName)
		check.PolicyAvailable(t, "account", accountUUID, myPolicy)
		check.PolicyAvailable(t, "environment", envVkb, myPolicy2)
		check.GroupAvailable(t, accountUUID, myGroup)

		// Group created with federatedAttributeValues should be a group with SAML owner
		samlGroup := check.GetGroupByName(t, accountUUID, mySAMLGroup)
		require.EqualValues(t, "SAML", samlGroup.Owner)

		// Group created without federatedAttributeValues should be a group with LOCAL owner
		localGroup := check.GetGroupByName(t, accountUUID, myLocalGroup)
		require.EqualValues(t, "LOCAL", localGroup.Owner)

		check.PolicyBindingsCount(t, accountUUID, "environment", envVkb, myGroup, 2)
		check.EnvironmentPolicyBinding(t, accountUUID, myGroup, myPolicy2, envVkb)
		check.EnvironmentPolicyBinding(t, accountUUID, myGroup, "Environment role - Replay session data without masking", envVkb)

		check.PolicyBindingsCount(t, accountUUID, "account", accountUUID, myGroup, 2)
		check.AccountPolicyBinding(t, accountUUID, myGroup, "Environment role - Access environment")
		check.AccountPolicyBinding(t, accountUUID, myGroup, myPolicy)

		check.PermissionBindingsCount(t, accountUUID, myGroup, 6)
		check.PermissionBinding(t, accountUUID, "account", accountUUID, "account-viewer", myGroup)
		check.PermissionBinding(t, accountUUID, "tenant", envVkb, "tenant-viewer", myGroup)
		check.PermissionBinding(t, accountUUID, "tenant", envVkb, "tenant-logviewer", myGroup)
		check.PermissionBinding(t, accountUUID, "management-zone", "wbm16058:"+mzoneID, "tenant-viewer", myGroup)

		// REMOVE SOME BINDINGS
		resources, err := loader.Load(o.fs, "accounts")
		require.NoError(t, err)
		resources.Groups["my-group"].Environment[0].Policies = slices.DeleteFunc(resources.Groups["my-group"].Environment[0].Policies, func(ref account.Ref) bool {
			return ref.ID() == "Environment role - Replay session data without masking"
		})
		resources.Groups["my-group"].Environment[0].Permissions = slices.DeleteFunc(resources.Groups["my-group"].Environment[0].Permissions, func(s string) bool { return s == "tenant-logviewer" })

		resources.Groups["my-group"].Account.Policies = slices.DeleteFunc(resources.Groups["my-group"].Account.Policies, func(ref account.Ref) bool {
			return ref.ID() == "Environment role - Access environment"
		})
		resources.Groups["my-group"].Account.Permissions = slices.DeleteFunc(resources.Groups["my-group"].Account.Permissions, func(s string) bool {
			return s == "account-company-info"
		})
		resources.Groups["my-group"].ManagementZone[0].Permissions = slices.DeleteFunc(resources.Groups["my-group"].ManagementZone[0].Permissions, func(s string) bool {
			return s == "tenant-logviewer"
		})

		// WRITE RESOURCES
		err = writer.Write(writer.Context{Fs: o.fs, OutputFolder: ".", ProjectFolder: "accounts"}, *resources)
		require.NoError(t, err)

		// DEPLOY
		err = cli.Execute()
		require.NoError(t, err)

		// CHECK BINDINGS ARE REMOVED
		check.PolicyBindingsCount(t, accountUUID, "environment", envVkb, myGroup, 1)
		check.PolicyBindingsCount(t, accountUUID, "account", accountUUID, myGroup, 1)
		check.PermissionBindingsCount(t, accountUUID, myGroup, 3)

		// DELETE ALL BINDINGS
		resources.Groups["my-group"].Environment[0].Policies = slices.DeleteFunc(resources.Groups["my-group"].Environment[0].Policies, func(ref account.Ref) bool { return true })
		resources.Groups["my-group"].Environment[0].Permissions = slices.DeleteFunc(resources.Groups["my-group"].Environment[0].Permissions, func(s string) bool { return true })
		resources.Groups["my-group"].Account.Policies = slices.DeleteFunc(resources.Groups["my-group"].Account.Policies, func(ref account.Ref) bool { return true })
		resources.Groups["my-group"].Account.Permissions = slices.DeleteFunc(resources.Groups["my-group"].Account.Permissions, func(s string) bool { return true })
		resources.Groups["my-group"].ManagementZone[0].Permissions = slices.DeleteFunc(resources.Groups["my-group"].ManagementZone[0].Permissions, func(s string) bool { return true })

		// WRITE RESOURCES
		err = writer.Write(writer.Context{Fs: o.fs, OutputFolder: ".", ProjectFolder: "accounts"}, *resources)
		require.NoError(t, err)

		// DEPLOY
		err = cli.Execute()
		require.NoError(t, err)

		check.PolicyBindingsCount(t, accountUUID, "environment", envVkb, myGroup, 0)
		check.PolicyBindingsCount(t, accountUUID, "account", accountUUID, myGroup, 0)
		check.PermissionBindingsCount(t, accountUUID, myGroup, 0)

		// DELETE RESOURCES
		cli.SetArgs([]string{"account", "delete", "--manifest", "manifest-account.yaml", "--file", "delete.yaml", "--account", accountName})
		err = cli.Execute()
		require.NoError(t, err)

		// CHECK IF RESOURCES ARE DELETED
		check.UserNotAvailable(t, accountUUID, myEmail)
		check.ServiceUserNotAvailable(t, accountUUID, myServiceUserName)
		check.PolicyNotAvailable(t, "account", accountUUID, myPolicy)
		check.PolicyNotAvailable(t, "environment", envVkb, myPolicy2)
		check.GroupNotAvailable(t, accountUUID, myGroup)
	})
}

func getPolicyIdByName(ctx context.Context, cl *accounts.Client, name, level, levelId string) (string, bool) {
	all, _, _ := cl.PolicyManagementAPI.GetLevelPolicies(ctx, levelId, level).Execute()

	p, found := getElementInSlice(all.Policies, func(el accountmanagement.PolicyDto) bool {
		return el.Name == name
	})

	if found && p != nil {
		return p.Uuid, found
	}
	return "", false
}

func getGroupIdByName(ctx context.Context, cl *accounts.Client, accountUUID, name string) (string, bool) {
	all, _, _ := cl.GroupManagementAPI.GetGroups(ctx, accountUUID).Execute()
	p, found := getElementInSlice(all.GetItems(), func(el accountmanagement.GetGroupDto) bool {
		return el.Name == name
	})
	if found && p != nil {
		return p.GetUuid(), found
	}
	return "", false

}

type AccountResourceChecker struct {
	Client      *accounts.Client
	RandomizeFn func(string) string
}

func (a AccountResourceChecker) ServiceUserAvailable(t *testing.T, accountUUID, name string) {
	expectedName := a.randomize(name)
	allServiceUsers := a.getAllServiceUsers(t, accountUUID)
	assertElementInSlice(t, allServiceUsers, func(s accountmanagement.ExternalServiceUserDto) bool { return s.Name == expectedName })
}

func (a AccountResourceChecker) ServiceUserNotAvailable(t *testing.T, accountUUID, name string) {
	expectedName := a.randomize(name)
	allServiceUsers := a.getAllServiceUsers(t, accountUUID)
	assertElementNotInSlice(t, allServiceUsers, func(s accountmanagement.ExternalServiceUserDto) bool { return s.Name == expectedName })
}

func (a AccountResourceChecker) UserAvailable(t *testing.T, accountUUID, email string) {
	expectedEmail := a.randomize(email)
	deployedUser, _, err := a.Client.UserManagementAPI.GetUserGroups(t.Context(), accountUUID, expectedEmail).Execute()
	require.NotNil(t, deployedUser)
	require.NoError(t, err)
	assert.Equal(t, expectedEmail, deployedUser.Email)
}

func (a AccountResourceChecker) UserNotAvailable(t *testing.T, accountUUID, email string) {
	expectedEmail := a.randomize(email)
	_, res, _ := a.Client.UserManagementAPI.GetUserGroups(t.Context(), accountUUID, expectedEmail).Execute()
	require.NotNil(t, res)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func (a AccountResourceChecker) GetGroupByName(t *testing.T, accountUUID, groupName string) *accountmanagement.GetGroupDto {
	expectedGroupName := a.randomize(groupName)
	all, _, err := a.Client.GroupManagementAPI.GetGroups(t.Context(), accountUUID).Execute()
	require.NotNil(t, all)
	require.NoError(t, err)
	group, _ := assertElementInSlice(t, all.GetItems(), func(el accountmanagement.GetGroupDto) bool { return el.Name == expectedGroupName })
	return group
}

func (a AccountResourceChecker) GroupAvailable(t *testing.T, accountUUID, groupName string) {
	_ = a.GetGroupByName(t, accountUUID, groupName)
}

func (a AccountResourceChecker) PolicyAvailable(t *testing.T, levelType, levelId, policyName string) {
	expectedPolicyName := a.randomize(policyName)
	policies, _, err := a.Client.PolicyManagementAPI.GetLevelPolicies(t.Context(), levelId, levelType).Name(expectedPolicyName).Execute()
	require.NoError(t, err)
	require.NotNil(t, policies)
	_, found := getElementInSlice(policies.Policies, func(el accountmanagement.PolicyDto) bool { return el.Name == expectedPolicyName })
	require.True(t, found)
}

func (a AccountResourceChecker) PolicyNotAvailable(t *testing.T, levelType, levelId, policyName string) {
	expectedPolicyName := a.randomize(policyName)
	policies, _, err := a.Client.PolicyManagementAPI.GetLevelPolicies(t.Context(), levelId, levelType).Execute()
	require.NotNil(t, policies)
	require.NoError(t, err)
	assertElementNotInSlice(t, policies.Policies, func(el accountmanagement.PolicyDto) bool { return el.Name == expectedPolicyName })
}

func (a AccountResourceChecker) GroupNotAvailable(t *testing.T, accountUUID, groupName string) {
	expectedGroupName := a.randomize(groupName)
	groups, _, err := a.Client.GroupManagementAPI.GetGroups(t.Context(), accountUUID).Execute()
	require.NotNil(t, groups)
	require.NoError(t, err)
	assertElementNotInSlice(t, groups.GetItems(), func(el accountmanagement.GetGroupDto) bool { return el.Name == expectedGroupName })
}

func (a AccountResourceChecker) EnvironmentPolicyBinding(t *testing.T, accountUUID, groupName, policyName, environmentID string) {
	expectedPolicyName := a.randomize(policyName)
	var pid string
	pid, found := getPolicyIdByName(t.Context(), a.Client, expectedPolicyName, "environment", environmentID)
	if !found {
		pid, found = getPolicyIdByName(t.Context(), a.Client, expectedPolicyName, "account", accountUUID)
	}
	if !found {
		pid, found = getPolicyIdByName(t.Context(), a.Client, expectedPolicyName, "global", "global")
	}
	require.True(t, found)

	expectedGroupName := a.randomize(groupName)
	gid, found := getGroupIdByName(t.Context(), a.Client, accountUUID, expectedGroupName)
	require.True(t, found)

	envPolBindings, _, err := a.Client.PolicyManagementAPI.GetAllLevelPoliciesBindings(t.Context(), environmentID, "environment").Execute()
	require.NoError(t, err)
	require.NotNil(t, envPolBindings)
	assertElementInSlice(t, envPolBindings.PolicyBindings, func(el accountmanagement.Binding) bool {
		return el.PolicyUuid == pid && slices.Contains(el.Groups, gid)
	})
}

func (a AccountResourceChecker) PolicyBindingsCount(t *testing.T, accountUUID string, levelType string, levelId string, groupName string, number int) {
	expectedGroupName := a.randomize(groupName)
	gid, found := getGroupIdByName(t.Context(), a.Client, accountUUID, expectedGroupName)
	require.True(t, found)

	envPolBindings, _, err := a.Client.PolicyManagementAPI.GetAllLevelPoliciesBindings(t.Context(), levelId, levelType).Execute()
	require.NoError(t, err)
	require.NotNil(t, envPolBindings)

	result := slices.DeleteFunc(envPolBindings.PolicyBindings, func(binding accountmanagement.Binding) bool {
		return !slices.Contains(binding.Groups, gid)
	})

	require.Equal(t, number, len(result))
}

func (a AccountResourceChecker) AccountPolicyBinding(t *testing.T, accountUUID, groupName, policyName string) {
	expectedPolicyName := a.randomize(policyName)
	var pid string
	pid, found := getPolicyIdByName(t.Context(), a.Client, expectedPolicyName, "account", accountUUID)
	if !found {
		pid, found = getPolicyIdByName(t.Context(), a.Client, expectedPolicyName, "global", "global")
	}
	require.True(t, found)

	expectedGroupName := a.randomize(groupName)
	gid, found := getGroupIdByName(t.Context(), a.Client, accountUUID, expectedGroupName)
	require.True(t, found)

	envPolBindings, _, err := a.Client.PolicyManagementAPI.GetAllLevelPoliciesBindings(t.Context(), accountUUID, "account").Execute()
	require.NoError(t, err)
	require.NotNil(t, envPolBindings)
	assertElementInSlice(t, envPolBindings.PolicyBindings, func(el accountmanagement.Binding) bool {
		return el.PolicyUuid == pid && slices.Contains(el.Groups, gid)
	})
}

func (a AccountResourceChecker) PermissionBinding(t *testing.T, accountUUID, scopeType, scope, permissionName, groupName string) {
	expectedGroupName := a.randomize(groupName)
	gid, found := getGroupIdByName(t.Context(), a.Client, accountUUID, expectedGroupName)
	require.True(t, found)

	permissions, _, err := a.Client.PermissionManagementAPI.GetGroupPermissions(t.Context(), accountUUID, gid).Execute()
	require.NoError(t, err)
	require.NotNil(t, permissions)
	assertElementInSlice(t, permissions.Permissions, func(el accountmanagement.PermissionsDto) bool {
		permissionFound := el.PermissionName == permissionName
		scopeTypeEqual := el.ScopeType == scopeType
		scopeEqual := el.Scope == scope
		return permissionFound && scopeTypeEqual && scopeEqual
	})
}

func (a AccountResourceChecker) PermissionBindingsCount(t *testing.T, accountUUID, groupName string, count int) {
	expectedGroupName := a.randomize(groupName)
	gid, found := getGroupIdByName(t.Context(), a.Client, accountUUID, expectedGroupName)
	require.True(t, found)

	permissions, _, err := a.Client.PermissionManagementAPI.GetGroupPermissions(t.Context(), accountUUID, gid).Execute()
	require.NoError(t, err)
	require.NotNil(t, permissions)
	assert.Equal(t, count, len(permissions.Permissions))
}

func (a AccountResourceChecker) randomize(in string) string {
	return a.RandomizeFn(in)
}

func (a AccountResourceChecker) getAllServiceUsers(t *testing.T, accountUUID string) []accountmanagement.ExternalServiceUserDto {
	serviceUsers := []accountmanagement.ExternalServiceUserDto{}
	const pageSize = 1000
	page := (int32)(1)
	for {
		r := a.getServiceUsersPage(t, accountUUID, page, pageSize)
		serviceUsers = append(serviceUsers, r.Results...)
		if r.NextPageKey == nil {
			break
		}
		page++
	}
	return serviceUsers
}

func (a AccountResourceChecker) getServiceUsersPage(t *testing.T, accountUUID string, page int32, pageSize int32) *accountmanagement.ExternalServiceUsersPageDto {
	r, resp, err := a.Client.ServiceUserManagementAPI.GetServiceUsersFromAccount(t.Context(), accountUUID).Page(page).PageSize(pageSize).Execute()
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	return r
}

func assertElementNotInSlice[K any](t *testing.T, sl []K, check func(el K) bool) {
	_, found := getElementInSlice(sl, check)
	assert.False(t, found)
}

func assertElementInSlice[K any](t *testing.T, sl []K, check func(el K) bool) (*K, bool) {
	e, found := getElementInSlice(sl, check)
	assert.True(t, found)
	return e, found
}
func getElementInSlice[K any](sl []K, check func(el K) bool) (*K, bool) {
	for _, e := range sl {
		if check(e) {
			return &e, true
		}
	}
	return nil, false
}
