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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/deployer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/writer"
	testrunner "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

const manyPermissionsMZonesManifest = "resources/many-permissions-mzones/manifest.yaml"

// must match the number of configs in resources/many-permissions-mzones/project/config.yaml
const manyPermissionsMZoneCount = 84

// only permissions known to be valid on management-zone scope
var managementZonePermissions = []string{
	"tenant-viewer",
	"tenant-logviewer",
	"tenant-manage-settings",
}

func manyPermissionsMZoneName(i int) string {
	return fmt.Sprintf("Many Permissions MZ %02d", i)
}

const manyPermissionsGroupID = "many-permissions-group"

// TestDeploy_ManyGroupPermissions verifies that groups with more than deployer.MaxPermissionsSize permissions
// are deployed by adding/removing single permissions instead of overwriting all of them in one call.
func TestDeploy_ManyGroupPermissions(t *testing.T) {
	require.Greater(t, manyPermissionsMZoneCount*len(managementZonePermissions), deployer.MaxPermissionsSize, "test setup must exceed the max permissions size")

	RunAccountTestCase(t, "resources/many-permissions", "manifest-account.yaml", "am-many-permissions", func(clients map[account.AccountInfo]*accounts.Client, o options) {
		accountUUID := o.accountUUID
		myGroup := "Many Permissions Group %RAND%"
		env1, ok := os.LookupEnv("ENVIRONMENT_ID_1")
		require.True(t, ok)
		env2, ok := os.LookupEnv("ENVIRONMENT_ID_2")
		require.True(t, ok)

		accInfo := account.AccountInfo{Name: o.accountName, AccountUUID: accountUUID}
		client := clients[accInfo]
		check := AccountResourceChecker{
			Client:      client,
			AccClient:   deployer.NewClient(accInfo, client),
			RandomizeFn: o.randomize,
		}

		mzoneIDs, mzoneName, cleanupMZones := deployManyPermissionsMZones(t, check, accountUUID, env1)
		defer cleanupMZones()

		cli, _ := runner.BuildCmd(o.fs)
		// deferred after the management zone cleanup, so the group referencing them is deleted first
		defer func() {
			t.Log("Deleting group")
			cli.SetArgs([]string{"account", "delete", "--manifest", "manifest-account.yaml", "--file", "delete.yaml", "--account", o.accountName})
			require.NoError(t, cli.Execute())
			check.GroupNotAvailable(t, accountUUID, myGroup)
		}()

		// STEP 1: deploy a group with a few permissions => permissions are overwritten in one call
		cli.SetArgs([]string{"account", "deploy", "-m", "manifest-account.yaml"})
		require.NoError(t, cli.Execute())

		check.PermissionBindingsCount(t, accountUUID, myGroup, 3)
		check.PermissionBinding(t, accountUUID, "account", accountUUID, "account-viewer", myGroup)
		check.PermissionBinding(t, accountUUID, "tenant", env2, "tenant-viewer", myGroup)
		check.PermissionBinding(t, accountUUID, "tenant", env2, "tenant-logviewer", myGroup)

		// STEP 2: add more than MaxPermissionsSize permissions and remove some of the previous ones
		replaceWithManyPermissions(t, o.fs, env1, mzoneName)
		require.NoError(t, cli.Execute())

		// remaining tenant-viewer on env2 + all management zone permissions
		check.PermissionBindingsCount(t, accountUUID, myGroup, 1+manyPermissionsMZoneCount*len(managementZonePermissions))
		check.PermissionBinding(t, accountUUID, "tenant", env2, "tenant-viewer", myGroup)
		check.PermissionNotBound(t, accountUUID, "account", accountUUID, "account-viewer", myGroup)
		check.PermissionNotBound(t, accountUUID, "tenant", env2, "tenant-logviewer", myGroup)
		for _, name := range []string{mzoneName(1), mzoneName(manyPermissionsMZoneCount)} {
			for _, p := range managementZonePermissions {
				check.PermissionBinding(t, accountUUID, "management-zone", fmt.Sprintf("%s:%s", env1, mzoneIDs[name]), p, myGroup)
			}
		}
	})
}

// deployManyPermissionsMZones deploys the test management zones with a unique suffix and waits until the account API knows them.
// It returns their IDs by name, a function returning the name of the i-th management zone, and a function deleting them.
func deployManyPermissionsMZones(t *testing.T, check AccountResourceChecker, accountUUID, environmentID string) (mzoneIDs map[string]string, mzoneName func(int) string, cleanup func()) {
	t.Helper()

	// CleanupIntegrationTest requires a fs that resolves absolute paths
	fs := afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())
	// monaco resolves the manifest to an absolute path, so the rewritten configs must be stored under it
	configFolder, err := filepath.Abs("resources/many-permissions-mzones/project")
	require.NoError(t, err)
	suffix := testrunner.AppendUniqueSuffixToIntegrationTestConfigs(t, fs, configFolder, "many-permissions")
	mzoneName = func(i int) string { return testrunner.AddSuffix(manyPermissionsMZoneName(i), suffix) }
	cleanup = func() {
		testrunner.CleanupIntegrationTest(t, fs, manyPermissionsMZonesManifest, "", "many-permissions")
	}

	// the caller can only defer cleanup after this function returns, so clean up here if deploying or waiting fails
	succeeded := false
	defer func() {
		if !succeeded {
			cleanup()
		}
	}()

	cli, _ := runner.BuildCmd(fs)
	cli.SetArgs([]string{"deploy", manyPermissionsMZonesManifest})
	require.NoError(t, cli.Execute())
	mzoneIDs = waitForManagementZones(t, check, accountUUID, environmentID, mzoneName)

	succeeded = true
	return mzoneIDs, mzoneName, cleanup
}

// replaceWithManyPermissions removes account-viewer and tenant-logviewer from the test group and adds all
// managementZonePermissions to every test management zone.
func replaceWithManyPermissions(t *testing.T, fs afero.Fs, environmentID string, mzoneName func(int) string) {
	t.Helper()

	resources, err := loader.Load(fs, "accounts")
	require.NoError(t, err)
	group := resources.Groups[manyPermissionsGroupID]
	group.Account.Permissions = slices.DeleteFunc(group.Account.Permissions, func(s string) bool { return s == "account-viewer" })
	group.Environment[0].Permissions = slices.DeleteFunc(group.Environment[0].Permissions, func(s string) bool { return s == "tenant-logviewer" })
	for i := 1; i <= manyPermissionsMZoneCount; i++ {
		group.ManagementZone = append(group.ManagementZone, account.ManagementZone{
			Environment:    environmentID,
			ManagementZone: mzoneName(i),
			Permissions:    managementZonePermissions,
		})
	}
	resources.Groups[manyPermissionsGroupID] = group

	require.NoError(t, writer.Write(writer.Context{Fs: fs, OutputFolder: ".", ProjectFolder: "accounts"}, *resources))
}

// waitForManagementZones waits until all test management zones of the given environment are known to the account API
// and returns their IDs by name.
func waitForManagementZones(t *testing.T, check AccountResourceChecker, accountUUID, environmentID string, mzoneName func(int) string) map[string]string {
	t.Helper()
	mzoneIDs := map[string]string{}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resources, _, err := check.Client.EnvironmentManagementAPI.GetEnvironmentResources(t.Context(), accountUUID).Execute()
		require.NoError(c, err)
		for _, mz := range resources.ManagementZoneResources {
			if mz.Parent == environmentID {
				mzoneIDs[mz.Name] = mz.Id
			}
		}
		for i := 1; i <= manyPermissionsMZoneCount; i++ {
			assert.Contains(c, mzoneIDs, mzoneName(i))
		}
	}, 5*time.Minute, 10*time.Second)
	return mzoneIDs
}
