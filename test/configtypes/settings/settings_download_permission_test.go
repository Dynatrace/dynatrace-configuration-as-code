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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestPermissionDownload(t *testing.T) {
	configFolder := "testdata/download-acl"
	manifestFile := path.Join(configFolder, "manifest.yaml")

	proj := "project"
	env := "platform_env"
	appId := "app:my.dynatrace.github.connector:connection"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifestFile),
			runner.WithSuffix("permission"),
			runner.WithEnvironment(env),
		},
		func(fs afero.Fs, ctx runner.TestContext) {
			// create
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --project=%s --verbose", manifestFile, env, proj))
			require.NoError(t, err, "create: did not expect error")

			// download
			err = monaco.Run(t, fs, fmt.Sprintf("monaco download --manifest=%s --environment=%s --project=proj --output-folder=download --verbose -s %s", manifestFile, env, appId))
			require.NoError(t, err, "download: did not expect error")

			// find the downloaded config and check its permissions
			projectAndEnvName := "proj_" + env // for manifest downloads: <download project>_<env>
			cfg := findDownloadedConfig(t, fs, "download", projectAndEnvName, appId, ctx.Suffix)

			assert.Equal(t, *cfg.Type.(config.SettingsType).AllUserPermission, config.WritePermission)
		})
}
