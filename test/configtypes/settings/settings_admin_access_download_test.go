//go:build integration

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

package settings

import (
	"fmt"
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

// TestAdminAccessDownload verifies the --admin-access download behavior for an owner-based
// OpenPipeline settings object.
//
// The object is created (and thereby owned) by the PLATFORM_TOKEN user and is private.
// A different user (PLATFORM_TOKEN_ADMIN, holding settings:objects:admin) then downloads it:
//   - without --admin-access the object of the other owner must NOT be found
//   - with --admin-access it must be found
func TestAdminAccessDownload(t *testing.T) {
	configFolder := "testdata/download-admin-access"
	manifestFile := path.Join(configFolder, "manifest.yaml")

	proj := "project"
	deployEnv := "platform_env"      // PLATFORM_TOKEN, owner of the object
	adminEnv := "platform_env_admin" // PLATFORM_TOKEN_ADMIN, different user with settings:objects:admin
	schema := "builtin:openpipeline.logs.pipelines"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifestFile),
			runner.WithSuffix("adminaccess"),
			runner.WithEnvironment(deployEnv), // clean up as the owner
		},
		func(fs afero.Fs, ctx runner.TestContext) {
			projectAndEnvName := proj + "_" + adminEnv // for manifest downloads: <download project>_<env>

			// create the private, owner-based object as the PLATFORM_TOKEN user
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --project=%s --verbose", manifestFile, deployEnv, proj))
			require.NoError(t, err, "deploy: did not expect error")

			// download as the admin user without admin access -> not found
			err = monaco.Run(t, fs, fmt.Sprintf("monaco download --manifest=%s --environment=%s --project=%s --output-folder=download-no-admin --verbose -s %s", manifestFile, adminEnv, proj, schema))
			require.NoError(t, err, "download without admin access: did not expect error")
			requireConfigNotDownloaded(t, fs, "download-no-admin", projectAndEnvName, schema, ctx.Suffix)

			// download as the admin user with admin access -> found
			err = monaco.Run(t, fs, fmt.Sprintf("monaco download --manifest=%s --environment=%s --project=%s --output-folder=download-admin --admin-access --verbose -s %s", manifestFile, adminEnv, proj, schema))
			require.NoError(t, err, "download with admin access: did not expect error")
			findDownloadedConfig(t, fs, "download-admin", projectAndEnvName, schema, ctx.Suffix)
		})
}
