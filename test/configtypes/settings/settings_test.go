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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	assert2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/assert"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

// tests all configs for a single environment
func TestIntegrationSettings(t *testing.T) {

	configFolder := "testdata/integration-settings/"
	manifest := configFolder + "manifest.yaml"

	runner.Run(t, configFolder,
		runner.Options{},
		func(fs afero.Fs, _ runner.TestContext) {
			// This causes Creation of all Settings
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
			assert.NoError(t, err)
			assert2.AssertAllConfigsAvailability(t, fs, manifest, []string{}, "", true)

			// This causes an Update of all Settings
			err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
			assert.NoError(t, err)
			assert2.AssertAllConfigsAvailability(t, fs, manifest, []string{}, "", true)
		})
}

// Tests a dry run (validation)
func TestIntegrationValidationSettings(t *testing.T) {

	t.Setenv("UNIQUE_TEST_SUFFIX", "can-be-nonunique-for-validation")

	configFolder := "testdata/integration-settings/"
	manifest := configFolder + "manifest.yaml"

	err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --verbose --dry-run", manifest))
	assert.NoError(t, err)
}

// TestOldExternalIDGetsUpdated tests whether a settings object with an "old" external ID that was
// generated using only "schemaID" and "configID" gets recognized and updated to have the "new" external ID
// that is composed of "projectName", "schemaID" and "configID"
func TestOldExternalIDGetsUpdated(t *testing.T) {

	fs := testutils.CreateTestFileSystem()
	var manifestPath = "testdata/integration-settings-old-new-external-id/manifest.yaml"

	env := "platform_env"

	loadedManifest := runner.LoadManifest(t, fs, manifestPath, env)
	projects := runner.LoadProjects(t, fs, manifestPath, loadedManifest)
	sortedConfigs, _ := graph.SortProjects(projects, []string{env})
	environment := loadedManifest.Environments.SelectedEnvironments[env]
	configToDeploy := sortedConfigs[env][0]

	defer func() {
		runner.CleanupIntegrationTest(t, fs, manifestPath, env, "")
	}()

	// first deploy with external id generate that does not consider the project name
	c := createSettingsClient(t, environment, dtclient.WithExternalIDGenerator(func(input coordinate.Coordinate) (string, error) {
		input.Project = ""
		id, _ := idutils.GenerateExternalIDForSettingsObject(input)
		return id, nil
	}))
	content, err := configToDeploy.Template.Content()
	assert.NoError(t, err)
	_, err = c.Upsert(t.Context(), dtclient.SettingsObject{
		Coordinate:     configToDeploy.Coordinate,
		SchemaId:       configToDeploy.Type.(config.SettingsType).SchemaId,
		SchemaVersion:  configToDeploy.Type.(config.SettingsType).SchemaVersion,
		Scope:          "environment",
		Content:        []byte(content),
		OriginObjectId: configToDeploy.OriginObjectId,
	}, dtclient.UpsertSettingsOptions{})
	assert.NoError(t, err)

	err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifestPath))
	assert.NoError(t, err)
	extID, _ := idutils.GenerateExternalIDForSettingsObject(sortedConfigs["platform_env"][0].Coordinate)

	// Check if settings 2.0 object with "new" external ID exists
	c = createSettingsClient(t, environment)
	settings, _ := c.List(t.Context(), "builtin:anomaly-detection.metric-events", dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(object dtclient.DownloadSettingsObject) bool {
		return object.ExternalId == extID
	}})
	assert.Len(t, settings, 1)

	// Check if no settings 2.0 object with "legacy" external ID exists
	coord := sortedConfigs["platform_env"][0].Coordinate
	coord.Project = ""
	legacyExtID, _ := idutils.GenerateExternalIDForSettingsObject(coord)
	settings, _ = c.List(t.Context(), "builtin:anomaly-detection.metric-events", dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(object dtclient.DownloadSettingsObject) bool {
		return object.ExternalId == legacyExtID
	}})
	assert.Len(t, settings, 0)

}

// TestDeploySettingsWithUniqueProperties asserts that settings with a schema that defines unique properties are updated based on those props.
// It deploys project1 and then project2 - both define the "same" Settings based on unique properties, but being in different projects,
// will get different monaco externalIds. The test then asserts that only the project2 externalIds can be found - monaco has updated the existing settings
// it found based on unique properties and attached new externalIds to them.
func TestDeploySettingsWithUniqueProperties(t *testing.T) {

	configFolder := "testdata/settings-unique-properties"
	manifestPath := configFolder + "/manifest.yaml"

	runner.Run(t, configFolder,
		runner.Options{},
		func(fs afero.Fs, _ runner.TestContext) {
			// create with project1 values
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=platform_env --project=project1", manifestPath))
			assert.NoError(t, err)

			// update based on unique properties with project2 values
			err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=platform_env --project=project2", manifestPath))
			assert.NoError(t, err)

			assert2.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project1"}, "platform_env", false) // updated to project2 externalIds
			assert2.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project2"}, "platform_env", true)
		})
}

// TestDeploySettingsWithUniqueProperties_ConsidersScopes is an extension of TestDeploySettingsWithUniqueProperties
// It uses project3 and project4 which both define settings in scope of certain hosts which match based on a unique property.
// project3 defines one setting in scope of a HOST-42...
// project4 defines a setting in scope of HOST-42... and one for HOST-21...
// Like TestDeploySettingsWithUniqueProperties the test asserts that only project4 settings can be found.
// In this case that means that the setting in scope of HOST-42 was updated and the setting for HOST-21 created, even though
// all three Settings share the same unique property (so this test also asserts that the scope is considered for finding
// settings by unique keys).
func TestDeploySettingsWithUniqueProperties_ConsidersScopes(t *testing.T) {

	configFolder := "testdata/settings-unique-properties"
	manifestPath := configFolder + "/manifest.yaml"

	runner.Run(t, configFolder,
		runner.Options{},
		func(fs afero.Fs, _ runner.TestContext) {
			// create with project3 values
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=platform_env --project=project3", manifestPath))
			assert.NoError(t, err)

			// update based on unique properties with project4 values and extend by one config
			err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=platform_env --project=project4", manifestPath))
			assert.NoError(t, err)

			assert2.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project3"}, "platform_env", false) // updated to project3 externalId
			assert2.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project4"}, "platform_env", true)  // 1 setting updated, 1 newly created
		})
}

func createSettingsClient(t *testing.T, env manifest.EnvironmentDefinition, opts ...func(dynatraceClient *dtclient.SettingsClient)) client.SettingsClient {
	clientFactory := clients.Factory().WithPlatformURL(env.URL.Value)

	if env.Auth.OAuth != nil {
		clientFactory = clientFactory.WithOAuthCredentials(clientcredentials.Config{
			ClientID:     env.Auth.OAuth.ClientID.Value.Value(),
			ClientSecret: env.Auth.OAuth.ClientSecret.Value.Value(),
			TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
		})
	}
	if env.Auth.PlatformToken != nil {
		clientFactory = clientFactory.WithPlatformToken(env.Auth.PlatformToken.Value.Value())
	}

	client, err := clientFactory.CreatePlatformClient(t.Context())
	require.NoError(t, err)

	classicURL, err := metadata.GetDynatraceClassicURL(t.Context(), *client)
	require.NoError(t, err)

	clientFactory = clientFactory.WithClassicURL(classicURL).WithAccessToken(env.Auth.ApiToken.Value.Value())

	classicClient, err := clientFactory.CreateClassicClient()
	require.NoError(t, err)

	dtClient, err := dtclient.NewClassicSettingsClient(classicClient)
	require.NoError(t, err)

	for _, o := range opts {
		o(dtClient)
	}
	return dtClient
}
