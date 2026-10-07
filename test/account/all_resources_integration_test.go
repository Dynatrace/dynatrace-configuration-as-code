//go:build account_integration

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
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/deployer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/writer"
)

func TestDeployAndDelete_AllResources(t *testing.T) {
	createMZone(t)

	RunAccountTestCase(t, "resources/all-resources", "manifest-account.yaml", "am-all-resources", func(clients map[account.AccountInfo]*accounts.Client, o options) {

		accountName := o.accountName
		accountUUID := o.accountUUID
		myServiceUserName := "monaco service user %RAND%"
		myEmail := "monaco+%RAND%@dynatrace.com"
		myGroup := "My Group%RAND%"
		myBoundary := "My boundary %RAND%"
		myBoundary2 := "My boundary 2 %RAND%"
		myDefaultBoundaries := []string{myBoundary}
		myUpdatedBoundaries := []string{myBoundary, myBoundary2}
		mySAMLGroup := "My SAML Group%RAND%"
		myLocalGroup := "My LOCAL Group%RAND%"
		myPolicy := "My Policy%RAND%"
		myPolicy2 := "My Policy 2%RAND%"
		env1, ok := os.LookupEnv("ENVIRONMENT_ID_1")
		require.True(t, ok)
		env2, ok := os.LookupEnv("ENVIRONMENT_ID_2")
		require.True(t, ok)

		accInfo := account.AccountInfo{Name: accountName, AccountUUID: accountUUID}
		client := clients[accInfo]
		check := AccountResourceChecker{
			Client:      client,
			AccClient:   deployer.NewClient(accInfo, client),
			RandomizeFn: o.randomize,
		}

		// get current management zone id for later assertions
		mzones, _, err := check.Client.EnvironmentManagementAPI.GetEnvironmentResources(t.Context(), accountUUID).Execute()
		require.NoError(t, err)
		var mzoneID string
		for _, mz := range mzones.ManagementZoneResources {
			if mz.Name == "Management Zone 2000" && mz.Parent == env1 {
				mzoneID = mz.Id
				break
			}
		}
		require.NotZero(t, mzoneID, "Could not get exact management zone id for assertions")

		cli, _ := runner.BuildCmd(o.fs)

		defer func() {
			t.Log("Starting cleanup")
			// DELETE RESOURCES
			cli.SetArgs([]string{"account", "delete", "--manifest", "manifest-account.yaml", "--file", "delete.yaml", "--account", accountName})
			err = cli.Execute()
			require.NoError(t, err)

			// CHECK IF RESOURCES ARE DELETED
			check.UserNotAvailable(t, accountUUID, myEmail)
			check.ServiceUserNotAvailable(t, myServiceUserName)
			check.PolicyNotAvailable(t, "account", accountUUID, myPolicy)
			check.PolicyNotAvailable(t, "environment", env2, myPolicy2)
			check.GroupNotAvailable(t, accountUUID, myGroup)
		}()

		// DEPLOY RESOURCES
		cli.SetArgs([]string{"account", "deploy", "-m", "manifest-account.yaml"})
		err = cli.Execute()
		require.NoError(t, err)

		// CHECK IF RESOURCES ARE INDEED DEPLOYED
		check.UserAvailable(t, accountUUID, myEmail)
		check.ServiceUserAvailable(t, myServiceUserName)
		check.PolicyAvailable(t, "account", accountUUID, myPolicy)
		check.PolicyAvailable(t, "environment", env2, myPolicy2)
		check.GroupAvailable(t, myGroup)
		check.BoundaryAvailable(t, myBoundary)

		// Group created with federatedAttributeValues should be a group with SAML owner
		samlGroup := check.GetGroupByName(t, mySAMLGroup)
		require.EqualValues(t, "SAML", samlGroup.Owner)

		// Group created without federatedAttributeValues should be a group with LOCAL owner
		localGroup := check.GetGroupByName(t, myLocalGroup)
		require.EqualValues(t, "LOCAL", localGroup.Owner)

		check.PolicyBindingsCount(t, "environment", env2, myGroup, 2)
		check.EnvironmentPolicyBinding(t, accountUUID, myGroup, myPolicy2, env2, myDefaultBoundaries)
		check.EnvironmentPolicyBinding(t, accountUUID, myGroup, "Environment role - Replay session data without masking", env2, []string{})

		check.PolicyBindingsCount(t, "account", accountUUID, myGroup, 2)
		check.AccountPolicyBinding(t, accountUUID, myGroup, "Environment role - Access environment", []string{})
		check.AccountPolicyBinding(t, accountUUID, myGroup, myPolicy, myDefaultBoundaries)

		check.PermissionBindingsCount(t, accountUUID, myGroup, 6)
		check.PermissionBinding(t, accountUUID, "account", accountUUID, "account-viewer", myGroup)
		check.PermissionBinding(t, accountUUID, "tenant", env2, "tenant-viewer", myGroup)
		check.PermissionBinding(t, accountUUID, "tenant", env2, "tenant-logviewer", myGroup)
		check.PermissionBinding(t, accountUUID, "management-zone", fmt.Sprintf("%s:%s", env1, mzoneID), "tenant-viewer", myGroup)

		// REMOVE SOME BINDINGS
		resources, err := loader.Load(o.fs, "accounts")
		require.NoError(t, err)
		resources.Groups["my-group"].Environment[0].Policies = slices.DeleteFunc(resources.Groups["my-group"].Environment[0].Policies, func(pb account.PolicyBinding) bool {
			return pb.Policy.ID() == "Environment role - Replay session data without masking"
		})
		resources.Groups["my-group"].Environment[0].Permissions = slices.DeleteFunc(resources.Groups["my-group"].Environment[0].Permissions, func(s string) bool { return s == "tenant-logviewer" })

		resources.Groups["my-group"].Account.Policies = slices.DeleteFunc(resources.Groups["my-group"].Account.Policies, func(pb account.PolicyBinding) bool {

			return pb.Policy.ID() == "Environment role - Access environment"
		})
		resources.Groups["my-group"].Account.Permissions = slices.DeleteFunc(resources.Groups["my-group"].Account.Permissions, func(s string) bool {
			return s == "account-company-info"
		})
		resources.Groups["my-group"].ManagementZone[0].Permissions = slices.DeleteFunc(resources.Groups["my-group"].ManagementZone[0].Permissions, func(s string) bool {
			return s == "tenant-logviewer"
		})
		// UPDATE SOME BOUNDARIES
		newBoundaries := []account.Ref{
			account.Reference{Id: "my-boundary-2"},
			account.Reference{Id: "my-boundary"},
		}
		updateBoundary(t, resources.Groups["my-group"].Environment[0].Policies, "my-policy-2", newBoundaries)
		updateBoundary(t, resources.Groups["my-group"].Account.Policies, "my-policy", newBoundaries)

		// WRITE RESOURCES
		err = writer.Write(writer.Context{Fs: o.fs, OutputFolder: ".", ProjectFolder: "accounts"}, *resources)
		require.NoError(t, err)

		// DEPLOY
		err = cli.Execute()
		require.NoError(t, err)

		// CHECK BINDINGS ARE REMOVED
		check.PolicyBindingsCount(t, "environment", env2, myGroup, 1)
		check.PolicyBindingsCount(t, "account", accountUUID, myGroup, 1)
		check.PermissionBindingsCount(t, accountUUID, myGroup, 3)

		// CHECK BOUNDARY BINDING UPDATE
		check.AccountPolicyBinding(t, accountUUID, myGroup, myPolicy, myUpdatedBoundaries)
		check.EnvironmentPolicyBinding(t, accountUUID, myGroup, myPolicy2, env2, myUpdatedBoundaries)

		// DELETE ALL BINDINGS
		resources.Groups["my-group"].Environment[0].Policies = slices.DeleteFunc(resources.Groups["my-group"].Environment[0].Policies, func(pb account.PolicyBinding) bool { return true })
		resources.Groups["my-group"].Environment[0].Permissions = slices.DeleteFunc(resources.Groups["my-group"].Environment[0].Permissions, func(s string) bool { return true })
		resources.Groups["my-group"].Account.Policies = slices.DeleteFunc(resources.Groups["my-group"].Account.Policies, func(pb account.PolicyBinding) bool { return true })
		resources.Groups["my-group"].Account.Permissions = slices.DeleteFunc(resources.Groups["my-group"].Account.Permissions, func(s string) bool { return true })
		resources.Groups["my-group"].ManagementZone[0].Permissions = slices.DeleteFunc(resources.Groups["my-group"].ManagementZone[0].Permissions, func(s string) bool { return true })

		// WRITE RESOURCES
		err = writer.Write(writer.Context{Fs: o.fs, OutputFolder: ".", ProjectFolder: "accounts"}, *resources)
		require.NoError(t, err)

		// DEPLOY
		err = cli.Execute()
		require.NoError(t, err)

		check.PolicyBindingsCount(t, "environment", env2, myGroup, 0)
		check.PolicyBindingsCount(t, "account", accountUUID, myGroup, 0)
		check.PermissionBindingsCount(t, accountUUID, myGroup, 0)
	})
}

func updateBoundary(t *testing.T, policies []account.PolicyBinding, policyID string, newBoundaries []account.Ref) {
	idx := slices.IndexFunc(policies, func(pb account.PolicyBinding) bool {
		return pb.Policy.ID() == policyID
	})
	require.NotEqual(t, -1, idx)
	policies[idx].Boundaries = newBoundaries
}
