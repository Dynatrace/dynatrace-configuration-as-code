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
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/deployer"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"net/http"
	"slices"
	"testing"
)

func TestDeployAndDelete_AllResources(t *testing.T) {
	t.Setenv(featureflags.AccountManagement().EnvName(), "true")

	// Create a management zone, so that we can reliably refer to it
	cliDeployMZones := runner.BuildCli(afero.NewOsFs())
	cliDeployMZones.SetArgs([]string{"deploy", "testdata/all-resources/manifest-mzones.yaml"})

	err := cliDeployMZones.Execute()
	assert.NoError(t, err)

	RunAccountTestCase(t, "testdata/all-resources", "manifest-account.yaml", "am-all-resources", func(clients map[deployer.AccountInfo]*accounts.Client, o options) {

		accountName := "monaco-test-account"
		accountUUID := "17a8095e-a974-40a2-9049-8a5d683cdd0b"
		myEmail := "monaco+%RAND%@dynatrace.com"
		myGroup := "My Group%RAND%"
		myPolicy := "My Policy%RAND%"
		envVkb := "vkb66581"

		check := AccountResourceChecker{
			Client:      clients[deployer.AccountInfo{Name: accountName, AccountUUID: accountUUID}],
			RandomizeFn: o.randomize,
		}

		cli := runner.BuildCli(o.fs)

		// (0) DEPLOY RESOURCES
		cli.SetArgs([]string{"account", "deploy", "manifest-account.yaml"})
		err = cli.Execute()
		assert.NoError(t, err)

		// (1) CHECK IF RESOURCES ARE INDEED DEPLOYED
		check.UserAvailable(t, accountUUID, myEmail)
		check.AccountPolicyAvailable(t, accountUUID, myPolicy)
		check.GroupAvailable(t, accountUUID, myGroup)
		check.EnvironmentPolicyBinding(t, accountUUID, myGroup, myPolicy, envVkb)
		check.EnvironmentPolicyBinding(t, accountUUID, myGroup, "Environment role - Replay session data without masking", envVkb)
		check.AccountPolicyBinding(t, accountUUID, myGroup, "Environment role - Access environment")
		check.PermissionBinding(t, accountUUID, "account", accountUUID, "account-viewer", myGroup)
		check.PermissionBinding(t, accountUUID, "tenant", envVkb, "tenant-viewer", myGroup)
		check.PermissionBinding(t, accountUUID, "management-zone", "wbm16058:1939021364513288421", "tenant-viewer", myGroup)

		// (2) DELETE RESOURCES
		cli.SetArgs([]string{"account", "delete", "--manifest", "manifest-account.yaml", "--file", "accounts/delete.yaml", "--account", "monaco-test-account"})
		err = cli.Execute()
		assert.NoError(t, err)

		// (3) CHECK IF RESOURCES ARE INDEED DELETED
		check.UserNotAvailable(t, accountUUID, myEmail)
		check.AccountPolicyNotAvailable(t, accountUUID, myPolicy)
		check.GroupNotAvailable(t, accountUUID, myGroup)
	})
}

func getPolicyIdByName(cl *accounts.Client, name, level, levelId string) (string, bool) {
	all, _, _ := cl.PolicyManagementAPI.GetLevelPolicies(context.TODO(), level, levelId).Execute()

	p, found := getElementInSlice(all.Policies, func(el accountmanagement.PolicyDto) bool {
		return el.Name == name
	})

	if found && p != nil {
		return p.Uuid, found
	}
	return "", false
}

func getGroupIdByName(cl *accounts.Client, accountUUID, name string) (string, bool) {
	all, _, _ := cl.GroupManagementAPI.GetGroups(context.TODO(), accountUUID).Execute()
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

func (a AccountResourceChecker) UserAvailable(t *testing.T, accountUUID, email string) {
	expectedEmail := a.randomize(email)
	deployedUser, _, err := a.Client.UserManagementAPI.GetUserGroups(context.TODO(), accountUUID, expectedEmail).Execute()
	assert.NotNil(t, deployedUser)
	assert.NoError(t, err)
	assert.Equal(t, expectedEmail, deployedUser.Email)
}

func (a AccountResourceChecker) UserNotAvailable(t *testing.T, accountUUID, email string) {
	expectedEmail := a.randomize(email)
	_, res, _ := a.Client.UserManagementAPI.GetUserGroups(context.TODO(), accountUUID, expectedEmail).Execute()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func (a AccountResourceChecker) GroupAvailable(t *testing.T, accountUUID, groupName string) {
	expectedGroupName := a.randomize(groupName)
	all, _, err := a.Client.GroupManagementAPI.GetGroups(context.TODO(), accountUUID).Execute()
	assert.NotNil(t, all)
	assert.NoError(t, err)
	assertElementInSlice(t, all.GetItems(), func(el accountmanagement.GetGroupDto) bool { return el.Name == expectedGroupName })
}

func (a AccountResourceChecker) AccountPolicyAvailable(t *testing.T, accountUUID, policyName string) {
	expectedPolicyName := a.randomize(policyName)
	policies, _, err := a.Client.PolicyManagementAPI.GetLevelPolicies(context.TODO(), "account", accountUUID).Name(expectedPolicyName).Execute()
	assert.NotNil(t, policies)
	assert.NoError(t, err)
	_, found := getElementInSlice(policies.Policies, func(el accountmanagement.PolicyDto) bool { return el.Name == expectedPolicyName })
	assert.True(t, found)
}

func (a AccountResourceChecker) AccountPolicyNotAvailable(t *testing.T, accountUUID, policyName string) {
	expectedPolicyName := a.randomize(policyName)
	policies, _, err := a.Client.PolicyManagementAPI.GetLevelPolicies(context.TODO(), "account", accountUUID).Execute()
	assert.NotNil(t, policies)
	assert.NoError(t, err)
	assertElementNotInSlice(t, policies.Policies, func(el accountmanagement.PolicyDto) bool { return el.Name == expectedPolicyName })
}

func (a AccountResourceChecker) GroupNotAvailable(t *testing.T, accountUUID, groupName string) {
	expectedGroupName := a.randomize(groupName)
	groups, _, err := a.Client.GroupManagementAPI.GetGroups(context.TODO(), accountUUID).Execute()
	assert.NotNil(t, groups)
	assert.NoError(t, err)
	assertElementNotInSlice(t, groups.GetItems(), func(el accountmanagement.GetGroupDto) bool { return el.Name == expectedGroupName })
}

func (a AccountResourceChecker) EnvironmentPolicyBinding(t *testing.T, accountUUID, groupName, policyName, environmentName string) {
	expectedPolicyName := a.randomize(policyName)
	var pid string
	pid, found := getPolicyIdByName(a.Client, expectedPolicyName, "environment", environmentName)
	if !found {
		pid, found = getPolicyIdByName(a.Client, expectedPolicyName, "account", accountUUID)
	}
	if !found {
		pid, found = getPolicyIdByName(a.Client, expectedPolicyName, "global", "global")
	}
	assert.True(t, found)

	expectedGroupName := a.randomize(groupName)
	gid, found := getGroupIdByName(a.Client, accountUUID, expectedGroupName)
	assert.True(t, found)

	envPolBindings, _, err := a.Client.PolicyManagementAPI.GetAllLevelPoliciesBindings(context.TODO(), "environment", environmentName).Execute()
	assert.NoError(t, err)
	assertElementInSlice(t, envPolBindings.PolicyBindings, func(el accountmanagement.Binding) bool {
		return el.PolicyUuid == pid && slices.Contains(el.Groups, gid)
	})
}

func (a AccountResourceChecker) AccountPolicyBinding(t *testing.T, accountUUID, groupName, policyName string) {
	expectedPolicyName := a.randomize(policyName)
	var pid string
	pid, found := getPolicyIdByName(a.Client, expectedPolicyName, "account", accountUUID)
	if !found {
		pid, found = getPolicyIdByName(a.Client, expectedPolicyName, "global", "global")
	}
	assert.True(t, found)

	expectedGroupName := a.randomize(groupName)
	gid, found := getGroupIdByName(a.Client, accountUUID, expectedGroupName)
	assert.True(t, found)

	envPolBindings, _, err := a.Client.PolicyManagementAPI.GetAllLevelPoliciesBindings(context.TODO(), "account", accountUUID).Execute()
	assert.NoError(t, err)
	assertElementInSlice(t, envPolBindings.PolicyBindings, func(el accountmanagement.Binding) bool {
		return el.PolicyUuid == pid && slices.Contains(el.Groups, gid)
	})
}

func (a AccountResourceChecker) PermissionBinding(t *testing.T, accountUUID, scopeType, scope, permissionName, groupName string) {
	expectedGroupName := a.randomize(groupName)
	gid, found := getGroupIdByName(a.Client, accountUUID, expectedGroupName)
	assert.True(t, found)

	permissions, _, err := a.Client.PermissionManagementAPI.GetGroupPermissions(context.TODO(), accountUUID, gid).Execute()
	assert.NoError(t, err)
	assertElementInSlice(t, permissions.Permissions, func(el accountmanagement.PermissionsDto) bool {
		permissionFound := el.PermissionName == permissionName
		scopeTypeEqual := el.ScopeType == scopeType
		scopeEqual := el.Scope == scope
		return permissionFound && scopeTypeEqual && scopeEqual
	})
}

func (a AccountResourceChecker) randomize(in string) string {
	return a.RandomizeFn(in)
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
