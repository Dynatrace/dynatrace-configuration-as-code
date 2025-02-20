//go:build integration

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
)

// tests all configs for a single environment
func TestIntegrationSettings(t *testing.T) {

	configFolder := "test-resources/integration-settings/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""

	RunIntegrationWithCleanup(t, configFolder, manifest, specificEnvironment, "SettingsTwo", func(fs afero.Fs, _ TestContext) {
		// This causes Creation of all Settings
		err := monaco.RunWithFSf(fs, "monaco deploy %s --verbose", manifest)
		assert.NoError(t, err)
		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{}, specificEnvironment, true)

		// This causes an Update of all Settings
		err = monaco.RunWithFSf(fs, "monaco deploy %s --verbose", manifest)
		assert.NoError(t, err)
		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{}, specificEnvironment, true)
	})
}

// Tests a dry run (validation)
func TestIntegrationValidationSettings(t *testing.T) {

	t.Setenv("UNIQUE_TEST_SUFFIX", "can-be-nonunique-for-validation")

	configFolder := "test-resources/integration-settings/"
	manifest := configFolder + "manifest.yaml"

	err := monaco.Runf("monaco deploy %s --verbose --dry-run", manifest)
	assert.NoError(t, err)
}

// TestOldExternalIDGetsUpdated tests whether a settings object with an "old" external ID that was
// generated using only "schemaID" and "configID" gets recognized and updated to have the "new" external ID
// that is composed of "projectName", "schemaID" and "configID"
func TestOldExternalIDGetsUpdated(t *testing.T) {

	fs := testutils.CreateTestFileSystem()
	var manifestPath = "test-resources/integration-settings-old-new-external-id/manifest.yaml"

	env := "platform_env"

	loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, env)
	projects := integrationtest.LoadProjects(t, fs, manifestPath, loadedManifest)
	sortedConfigs, _ := sort.ConfigsPerEnvironment(projects, []string{env})
	environment := loadedManifest.Environments[env]
	configToDeploy := sortedConfigs[env][0]

	t.Cleanup(func() {
		integrationtest.CleanupIntegrationTest(t, fs, manifestPath, env, "")
	})

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

	err = monaco.RunWithFSf(fs, "monaco deploy %s --verbose", manifestPath)
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

	configFolder := "test-resources/settings-unique-properties"
	manifestPath := configFolder + "/manifest.yaml"

	RunIntegrationWithCleanup(t, configFolder, manifestPath, "", "SettingsUniqueProps", func(fs afero.Fs, _ TestContext) {
		// create with project1 values
		err := monaco.RunWithFSf(fs, "monaco deploy %s --environment=platform_env --project=project1", manifestPath)
		assert.NoError(t, err)

		// update based on unique properties with project2 values
		err = monaco.RunWithFSf(fs, "monaco deploy %s --environment=platform_env --project=project2", manifestPath)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project1"}, "platform_env", false) // updated to project2 externalIds
		integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project2"}, "platform_env", true)
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

	configFolder := "test-resources/settings-unique-properties"
	manifestPath := configFolder + "/manifest.yaml"

	RunIntegrationWithCleanup(t, configFolder, manifestPath, "", "SettingsUniqueProps", func(fs afero.Fs, _ TestContext) {
		// create with project3 values
		err := monaco.RunWithFSf(fs, "monaco deploy %s --environment=platform_env --project=project3", manifestPath)
		assert.NoError(t, err)

		// update based on unique properties with project4 values and extend by one config
		err = monaco.RunWithFSf(fs, "monaco deploy %s --environment=platform_env --project=project4", manifestPath)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project3"}, "platform_env", false) // updated to project3 externalId
		integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project4"}, "platform_env", true)  // 1 setting updated, 1 newly created
	})
}

// TestOrderedSettings tries to deploy two setting objects A and B two times. The first time with A insert after B, the second time with B insert after A.
// After each of the two deployment the actual order is asserted.
func TestOrderedSettings(t *testing.T) {
	configFolder := "test-resources/settings-ordered/order1"
	manifestPath := configFolder + "/manifest.yaml"

	RunIntegrationWithCleanup(t, configFolder, manifestPath, "", "SettingsOrdered", func(fs afero.Fs, _ TestContext) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --environment=platform_env --project=project", manifestPath)
		assert.NoError(t, err)
		integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project"}, "platform_env", true)

		loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, "platform_env")
		environment := loadedManifest.Environments["platform_env"]
		settingsClient := createSettingsClient(t, environment)
		results, err := settingsClient.List(t.Context(), "builtin:container.monitoring-rule", dtclient.ListSettingsOptions{})
		assert.NoError(t, err)

		assert.Len(t, results, 2)
		assert.Equal(t, coordc23, results[0].ExternalId)
		assert.Equal(t, coord704, results[1].ExternalId)
	})

	configFolder = "test-resources/settings-ordered/order2"
	manifestPath = configFolder + "/manifest.yaml"

	RunIntegrationWithCleanup(t, configFolder, manifestPath, "", "SettingsOrdered", func(fs afero.Fs, _ TestContext) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --environment=platform_env --project=project", manifestPath)
		assert.NoError(t, err)
		integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project"}, "platform_env", true)

		loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, "platform_env")
		environment := loadedManifest.Environments["platform_env"]
		settingsClient := createSettingsClient(t, environment)
		results, err := settingsClient.List(t.Context(), "builtin:container.monitoring-rule", dtclient.ListSettingsOptions{})
		assert.NoError(t, err)

		assert.Len(t, results, 2)
		assert.Equal(t, coord704, results[0].ExternalId)
		assert.Equal(t, coordc23, results[1].ExternalId)
	})

}

// TestOrderedSettingsCrossProjects tries to deploy two setting objects A and B, while both are in different projects.
// After each of the two deployment the actual order is asserted.
func TestOrderedSettingsCrossProjects(t *testing.T) {
	configFolder := "test-resources/settings-ordered/cross-project-reference"
	manifestPath := configFolder + "/manifest.yaml"

	RunIntegrationWithCleanup(t, configFolder, manifestPath, "", "SettingsOrdered", func(fs afero.Fs, _ TestContext) {
		err := monaco.RunWithFSf(fs, "monaco deploy %s --environment=platform_env --project=source", manifestPath)
		assert.NoError(t, err)
		integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"source"}, "platform_env", true)

		loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, "platform_env")
		environment := loadedManifest.Environments["platform_env"]
		settingsClient := createSettingsClient(t, environment)
		results, err := settingsClient.List(t.Context(), "builtin:container.monitoring-rule", dtclient.ListSettingsOptions{})
		assert.NoError(t, err)

		assert.Len(t, results, 2)

		// target is first, as source 'insertsAfter' target
		targetConfigExternalId, err := idutils.GenerateExternalIDForSettingsObject(coordinate.Coordinate{Project: "target", Type: "builtin:container.monitoring-rule", ConfigId: "target-id"})
		assert.NoError(t, err)
		assert.Equal(t, targetConfigExternalId, results[0].ExternalId)

		sourceConfigExternalId, err := idutils.GenerateExternalIDForSettingsObject(coordinate.Coordinate{Project: "source", Type: "builtin:container.monitoring-rule", ConfigId: "source-id"})
		assert.NoError(t, err)
		assert.Equal(t, sourceConfigExternalId, results[1].ExternalId)
	})

}

func createSettingsClient(t *testing.T, env manifest.EnvironmentDefinition, opts ...func(dynatraceClient *dtclient.SettingsClient)) client.SettingsClient {

	clientFactory := clients.Factory().
		WithOAuthCredentials(clientcredentials.Config{
			ClientID:     env.Auth.OAuth.ClientID.Value.Value(),
			ClientSecret: env.Auth.OAuth.ClientSecret.Value.Value(),
			TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
		}).
		//WithConcurrentRequestLimit(concurrentRequestLimit).

		WithPlatformURL(env.URL.Value)

	client, err := clientFactory.CreatePlatformClient(t.Context())
	require.NoError(t, err)

	classicURL, err := metadata.GetDynatraceClassicURL(t.Context(), *client)
	require.NoError(t, err)

	clientFactory = clientFactory.WithClassicURL(classicURL).WithAccessToken(env.Auth.Token.Value.Value())

	classicClient, err := clientFactory.CreateClassicClient()
	require.NoError(t, err)

	dtClient, err := dtclient.NewClassicSettingsClient(classicClient)
	require.NoError(t, err)

	for _, o := range opts {
		o(dtClient)
	}
	return dtClient
}

func TestOrdered_InsertAtFrontAndBackWorks(t *testing.T) {
	const configFolder = "test-resources/settings-ordered/insert-position"

	const manifestFile = configFolder + "/manifest.yaml"

	const specificEnvironment = "platform"
	const project = "both-back-and-front-are-set"
	const schema = "builtin:url-based-sampling"

	RunIntegrationWithCleanup(t, configFolder, manifestFile, specificEnvironment, "InsertAtBackWorks", func(fs afero.Fs, tc TestContext) {

		err := monaco.RunWithFSf(fs, "monaco deploy %s --project %s --verbose", manifestFile, project)
		assert.NoError(t, err)
		integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{project}, specificEnvironment, true)

		sClient := createSettingsClientFromManifest(t, fs, manifestFile, "platform")

		list, err := sClient.List(t.Context(), schema, dtclient.ListSettingsOptions{
			DiscardValue: true,
		})

		assert.GreaterOrEqual(t, len(list), 2, "At least the two configs in the test should be deployed")

		// Verify that last is actually the first object
		first := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "first"}, tc)
		assert.Equal(t, 0, findPositionWithExternalId(t, list, first))

		// Verify that last is actually the last object
		last := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "last"}, tc)
		assert.Equal(t, len(list)-1, findPositionWithExternalId(t, list, last))
	})
}

func findPositionWithExternalId(t *testing.T, objects []dtclient.DownloadSettingsObject, externalId string) int {
	t.Helper()

	for i := range objects {
		if objects[i].ExternalId == externalId {
			return i
		}
	}

	t.Errorf("Could not find position ob object with external id %s", externalId)
	return -1
}

func settingsExternalIdForTest(t *testing.T, originalCoordinate coordinate.Coordinate, testContext TestContext) string {

	originalCoordinate.ConfigId += "_" + testContext.suffix

	id, err := idutils.GenerateExternalIDForSettingsObject(originalCoordinate)
	require.NoError(t, err)

	return id
}

func createSettingsClientFromManifest(t *testing.T, fs afero.Fs, manifestPath string, environment string) client.SettingsClient {
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	assert.Empty(t, errs)

	clientSet := integrationtest.CreateDynatraceClients(t, man.Environments[environment])
	return clientSet.SettingsClient
}
