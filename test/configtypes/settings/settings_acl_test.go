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

package settings

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

func TestSettingsWithACL(t *testing.T) {
	configFolder := "testdata/settings-acl/"
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

		v2.Run(t, configFolder,
			v2.Options{
				v2.WithManifestPath(defaultManifest),
				v2.WithSuffix("settings-ACL"),
				v2.WithEnvironment(environment),
			},
			func(fs afero.Fs, testContext v2.TestContext) {
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
						ConfigId: "config-acl_" + testContext.Suffix,
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

func createSettingsClientPlatform(t *testing.T, env manifest.EnvironmentDefinition) client.SettingsClient {
	clientFactory := clients.Factory().
		WithOAuthCredentials(clientcredentials.Config{
			ClientID:     env.Auth.OAuth.ClientID.Value.Value(),
			ClientSecret: env.Auth.OAuth.ClientSecret.Value.Value(),
			TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
		}).
		WithPlatformURL(env.URL.Value)

	c, err := clientFactory.CreatePlatformClient(t.Context())
	require.NoError(t, err)

	dtClient, err := dtclient.NewPlatformSettingsClient(c)
	require.NoError(t, err)

	return dtClient
}
