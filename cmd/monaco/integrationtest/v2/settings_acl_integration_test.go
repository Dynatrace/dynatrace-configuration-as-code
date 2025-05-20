//go:build integration

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package v2

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

func TestSettingsWithACL(t *testing.T) {
	configFolder := "test-resources/settings-acl/"
	defaultManifest := configFolder + "acl-empty/manifest.yaml"
	environment := "platform_env"
	project := "project"
	schemaId := "app:my.dynatrace.github.connector:connection"
	settingsType := config.SettingsType{SchemaId: schemaId}

	t.Run("Updates correctly", func(t *testing.T) {
		t.Setenv(featureflags.AccessControlSettings.EnvName(), "true")
		updates := []struct {
			ManifestFolder string
			WantPermission []dtclient.TypePermissions
		}{
			{
				// no permission (delete)
				ManifestFolder: "acl-empty",
				WantPermission: []dtclient.TypePermissions{},
			},
			{
				// create permission
				ManifestFolder: "acl-read",
				WantPermission: []dtclient.TypePermissions{dtclient.Read},
			},
			{
				// update permission
				ManifestFolder: "acl-write",
				WantPermission: []dtclient.TypePermissions{dtclient.Read, dtclient.Write},
			},
			{
				// delete permission
				ManifestFolder: "acl-none",
				WantPermission: []dtclient.TypePermissions{},
			},
		}

		Run(t, configFolder,
			Options{
				WithManifestPath(defaultManifest),
				WithSuffix("settings-ACL"),
				WithEnvironment(environment),
			},
			func(fs afero.Fs, testContext TestContext) {
				for _, update := range updates {
					t.Logf("Update permission with '%s'", update.ManifestFolder)

					manifestPath := configFolder + update.ManifestFolder + "/manifest.yaml"
					err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=%s --verbose", manifestPath, project))
					require.NoError(t, err)

					loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, environment)
					environmentDefinition := loadedManifest.Environments.SelectedEnvironments[environment]
					client := createSettingsClientPlatform(t, environmentDefinition)

					coord := coordinate.Coordinate{
						Project:  project,
						Type:     schemaId,
						ConfigId: "config-acl_" + testContext.suffix,
					}
					objectId := integrationtest.AssertSetting(t, client, settingsType, environment, true, config.Config{
						Coordinate: coord,
					})
					integrationtest.AssertPermission(t, client, objectId, update.WantPermission)
				}
			})
	})

	t.Run("With a disabled FF the deploy should fail", func(t *testing.T) {
		t.Setenv(featureflags.AccessControlSettings.EnvName(), "false")
		manifestPath := configFolder + "acl-write/manifest.yaml"

		logOutput := strings.Builder{}
		cmd := runner.BuildCmdWithLogSpy(monaco.NewTestFs(), &logOutput)
		cmd.SetArgs([]string{"deploy", "--verbose", manifestPath, "--environment", environment})
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, logOutput.String(), "unknown settings configuration property 'permissions'")
	})
}
