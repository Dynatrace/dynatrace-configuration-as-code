//go:build account_integration

/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/accounts"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/deployer"
)

type AccountResourceChecker struct {
	Client      *accounts.Client
	AccClient   *deployer.AccountManagementClient
	RandomizeFn func(string) string
}

func (a AccountResourceChecker) ServiceUserAvailable(t *testing.T, name string) {
	expectedName := a.randomize(name)
	_, err := a.AccClient.GetServiceUserByName(t.Context(), expectedName)
	require.NoError(t, err)
}

func (a AccountResourceChecker) ServiceUserNotAvailable(t *testing.T, name string) {
	expectedName := a.randomize(name)
	_, err := a.AccClient.GetServiceUserByName(t.Context(), expectedName)
	require.Error(t, err)
	expErr := &deployer.ResourceNotFoundError{}
	require.ErrorAs(t, err, &expErr)
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

func (a AccountResourceChecker) GetGroupByName(t *testing.T, groupName string) *accountmanagement.GetGroupDto {
	expectedGroupName := a.randomize(groupName)
	foundGroups, err := a.AccClient.GetGroupsByName(t.Context(), expectedGroupName)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(foundGroups), 1)
	return &foundGroups[0]
}

func (a AccountResourceChecker) GetBoundaryByName(t *testing.T, boundaryName string) *accountmanagement.PolicyBoundaryOverview {
	expectedBoundaryName := a.randomize(boundaryName)

	boundary, err := a.AccClient.GetBoundaryByName(t.Context(), expectedBoundaryName)
	require.NoError(t, err)
	return boundary
}

func (a AccountResourceChecker) GroupAvailable(t *testing.T, groupName string) {
	_ = a.GetGroupByName(t, groupName)
}

func (a AccountResourceChecker) BoundaryAvailable(t *testing.T, boundaryName string) {
	_ = a.GetBoundaryByName(t, boundaryName)
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

func (a AccountResourceChecker) EnvironmentPolicyBinding(t *testing.T, accountUUID, groupName, policyName, environmentID string, boundaries []string) {
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
	gid := a.GetGroupByName(t, expectedGroupName).GetUuid()

	envPolBindings, _, err := a.Client.PolicyManagementAPI.GetAllLevelPoliciesBindings(t.Context(), environmentID, "environment").Execute()
	require.NoError(t, err)
	require.NotNil(t, envPolBindings)

	a.AssertPolicyBinding(t, pid, gid, envPolBindings.PolicyBindings, boundaries)
}

func (a AccountResourceChecker) AssertPolicyBinding(t *testing.T, pid, gid string, bindings []accountmanagement.Binding, boundaries []string) {
	idx := slices.IndexFunc(bindings, func(el accountmanagement.Binding) bool {
		return el.PolicyUuid == pid && slices.Contains(el.Groups, gid)
	})
	require.NotEqual(t, -1, idx)
	assert.Len(t, bindings[idx].Boundaries, len(boundaries))
	boundaryIDs := make([]string, 0, len(boundaries))
	for _, boundary := range boundaries {
		boundaryIDs = append(boundaryIDs, a.GetBoundaryByName(t, a.randomize(boundary)).Uuid)
	}
	require.ElementsMatch(t, boundaryIDs, bindings[idx].Boundaries)
}

func (a AccountResourceChecker) PolicyBindingsCount(t *testing.T, levelType string, levelId string, groupName string, number int) {
	expectedGroupName := a.randomize(groupName)
	gid := a.GetGroupByName(t, expectedGroupName).GetUuid()

	envPolBindings, _, err := a.Client.PolicyManagementAPI.GetAllLevelPoliciesBindings(t.Context(), levelId, levelType).Execute()
	require.NoError(t, err)
	require.NotNil(t, envPolBindings)

	result := slices.DeleteFunc(envPolBindings.PolicyBindings, func(binding accountmanagement.Binding) bool {
		return !slices.Contains(binding.Groups, gid)
	})

	require.Equal(t, number, len(result))
}

func (a AccountResourceChecker) AccountPolicyBinding(t *testing.T, accountUUID, groupName, policyName string, boundaries []string) {
	expectedPolicyName := a.randomize(policyName)
	var pid string
	pid, found := getPolicyIdByName(t.Context(), a.Client, expectedPolicyName, "account", accountUUID)
	if !found {
		pid, found = getPolicyIdByName(t.Context(), a.Client, expectedPolicyName, "global", "global")
	}
	require.True(t, found)

	expectedGroupName := a.randomize(groupName)
	gid := a.GetGroupByName(t, expectedGroupName).GetUuid()

	envPolBindings, _, err := a.Client.PolicyManagementAPI.GetAllLevelPoliciesBindings(t.Context(), accountUUID, "account").Execute()
	require.NoError(t, err)
	require.NotNil(t, envPolBindings)
	a.AssertPolicyBinding(t, pid, gid, envPolBindings.PolicyBindings, boundaries)
}

func (a AccountResourceChecker) PermissionBinding(t *testing.T, accountUUID, scopeType, scope, permissionName, groupName string) {
	expectedGroupName := a.randomize(groupName)
	gid := a.GetGroupByName(t, expectedGroupName).GetUuid()

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

func (a AccountResourceChecker) PermissionNotBound(t *testing.T, accountUUID, scopeType, scope, permissionName, groupName string) {
	expectedGroupName := a.randomize(groupName)
	gid := a.GetGroupByName(t, expectedGroupName).GetUuid()

	permissions, _, err := a.Client.PermissionManagementAPI.GetGroupPermissions(t.Context(), accountUUID, gid).Execute()
	require.NoError(t, err)
	require.NotNil(t, permissions)
	assertElementNotInSlice(t, permissions.Permissions, func(el accountmanagement.PermissionsDto) bool {
		return el.PermissionName == permissionName && el.ScopeType == scopeType && el.Scope == scope
	})
}

func (a AccountResourceChecker) PermissionBindingsCount(t *testing.T, accountUUID, groupName string, count int) {
	expectedGroupName := a.randomize(groupName)
	gid := a.GetGroupByName(t, expectedGroupName).GetUuid()

	permissions, _, err := a.Client.PermissionManagementAPI.GetGroupPermissions(t.Context(), accountUUID, gid).Execute()
	require.NoError(t, err)
	require.NotNil(t, permissions)
	assert.Equal(t, count, len(permissions.Permissions))
}

func (a AccountResourceChecker) randomize(in string) string {
	return a.RandomizeFn(in)
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
