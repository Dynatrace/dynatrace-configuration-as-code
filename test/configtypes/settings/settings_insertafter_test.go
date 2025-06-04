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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
)

// TestOrderedSettings tries to deploy two entity-scoped setting objects two times. The first time with "bbb" insert after "aaa", the second time with "aaa" insert after "bbb".
// After each of the two deployments the actual order is asserted.
func TestOrderedSettings(t *testing.T) {
	host := randomMeID("HOST")

	expectedExternalIdForAAA, err := idutils.GenerateExternalIDForSettingsObject(coordinate.Coordinate{Project: "project", ConfigId: "aaa", Type: "builtin:processavailability"})
	assert.NoError(t, err)
	expectedExternalIdForBBB, err := idutils.GenerateExternalIDForSettingsObject(coordinate.Coordinate{Project: "project", ConfigId: "bbb", Type: "builtin:processavailability"})
	assert.NoError(t, err)

	configFolder := "testdata/settings-ordered/order1"
	manifestPath := configFolder + "/manifest.yaml"

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithoutCleanup(),
			v2.WithMeId(host),
		},
		func(fs afero.Fs, tc v2.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=platform_env --project=project", manifestPath))
			require.NoError(t, err)

			integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project"}, "platform_env", true)

			loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, "platform_env")
			environment := loadedManifest.Environments.SelectedEnvironments["platform_env"]
			settingsClient := createSettingsClient(t, environment)

			results, err := settingsClient.List(t.Context(), "builtin:processavailability", dtclient.ListSettingsOptions{
				DiscardValue: true,
				Filter:       filterObjectsForScope(host),
			})
			require.NoError(t, err)
			require.Len(t, results, 2)
			assert.Equal(t, 0, findPositionWithExternalId(t, results, expectedExternalIdForAAA))
			assert.Equal(t, 1, findPositionWithExternalId(t, results, expectedExternalIdForBBB))
		})

	configFolder = "testdata/settings-ordered/order2"
	manifestPath = configFolder + "/manifest.yaml"

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithMeId(host),
		},
		func(fs afero.Fs, tc v2.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=platform_env --project=project", manifestPath))
			require.NoError(t, err)

			integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"project"}, "platform_env", true)

			loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, "platform_env")
			environment := loadedManifest.Environments.SelectedEnvironments["platform_env"]
			settingsClient := createSettingsClient(t, environment)

			results, err := settingsClient.List(t.Context(), "builtin:processavailability", dtclient.ListSettingsOptions{
				DiscardValue: true,
				Filter:       filterObjectsForScope(host),
			})
			require.NoError(t, err)
			require.Len(t, results, 2)
			assert.Equal(t, 0, findPositionWithExternalId(t, results, expectedExternalIdForBBB))
			assert.Equal(t, 1, findPositionWithExternalId(t, results, expectedExternalIdForAAA))
		})
}

// TestOrderedSettingsCrossProjects tries to deploy two setting objects A and B, while both are in different projects.
// After each of the two deployment the actual order is asserted.
func TestOrderedSettingsCrossProjects(t *testing.T) {
	const configFolder = "testdata/settings-ordered/cross-project-reference"
	const manifestPath = configFolder + "/manifest.yaml"

	const schema = "builtin:url-based-sampling"

	pgiMeId := randomMeID("PROCESS_GROUP_INSTANCE")

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithMeId(pgiMeId),
		},
		func(fs afero.Fs, tc v2.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=platform_env --project=source", manifestPath))
			require.NoError(t, err)
			integrationtest.AssertAllConfigsAvailability(t, fs, manifestPath, []string{"source"}, "platform_env", true)

			loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, "platform_env")
			environment := loadedManifest.Environments.SelectedEnvironments["platform_env"]
			settingsClient := createSettingsClient(t, environment)
			results, err := settingsClient.List(t.Context(), schema, dtclient.ListSettingsOptions{
				DiscardValue: true,
				Filter:       filterObjectsForScope(pgiMeId),
			})
			assert.NoError(t, err)

			assert.Len(t, results, 2)

			// target is first, as source 'insertsAfter' target
			targetConfigExternalId := settingsExternalIdForTest(t, coordinate.Coordinate{Project: "target", Type: schema, ConfigId: "target-id"}, tc)
			assert.NoError(t, err)
			assert.Equal(t, targetConfigExternalId, results[0].ExternalId)

			sourceConfigExternalId := settingsExternalIdForTest(t, coordinate.Coordinate{Project: "source", Type: schema, ConfigId: "source-id"}, tc)
			assert.NoError(t, err)
			assert.Equal(t, sourceConfigExternalId, results[1].ExternalId)
		})
}

func TestOrdered_InsertAtFrontWorksWithoutBeingSet(t *testing.T) {
	const configFolder = "testdata/settings-ordered/insert-position"

	const manifestFile = configFolder + "/manifest.yaml"

	const specificEnvironment = "platform"
	const project = "insert-after-not-set"
	const schema = "builtin:url-based-sampling"

	pgiMeId := randomMeID("PROCESS_GROUP_INSTANCE")

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithEnvironment(specificEnvironment),
			v2.WithMeId(pgiMeId),
		},
		func(fs afero.Fs, tc v2.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project %s --verbose", manifestFile, project))
			require.NoError(t, err)
			integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{project}, specificEnvironment, true)

			sClient := createSettingsClientFromManifest(t, fs, manifestFile, "platform")
			list, err := sClient.List(t.Context(), schema, dtclient.ListSettingsOptions{
				DiscardValue: true,
				Filter:       filterObjectsForScope(pgiMeId),
			})

			assert.Equal(t, 2, len(list), "Exactly two configs should be deployed")

			first := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "first"}, tc)
			second := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "second"}, tc)

			assert.Equal(t, 0, findPositionWithExternalId(t, list, first))
			assert.Equal(t, 1, findPositionWithExternalId(t, list, second))
		})
}

func TestOrdered_InsertAtFrontWorks(t *testing.T) {
	const configFolder = "testdata/settings-ordered/insert-position"

	const manifestFile = configFolder + "/manifest.yaml"

	const specificEnvironment = "platform"
	const project = "insert-after-set-to-front"
	const schema = "builtin:url-based-sampling"

	pgiMeId := randomMeID("PROCESS_GROUP_INSTANCE")

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithEnvironment(specificEnvironment),
			v2.WithMeId(pgiMeId),
		},
		func(fs afero.Fs, tc v2.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project %s --verbose", manifestFile, project))
			require.NoError(t, err)
			integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{project}, specificEnvironment, true)

			sClient := createSettingsClientFromManifest(t, fs, manifestFile, "platform")

			list, err := sClient.List(t.Context(), schema, dtclient.ListSettingsOptions{
				DiscardValue: true,
				Filter:       filterObjectsForScope(pgiMeId),
			})

			assert.Len(t, list, 3, "Exactly three configs should be deployed")

			first := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "first"}, tc)
			second := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "second"}, tc)
			third := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "third"}, tc)

			assert.Equal(t, 0, findPositionWithExternalId(t, list, second))
			assert.Equal(t, 1, findPositionWithExternalId(t, list, third))
			assert.Equal(t, 2, findPositionWithExternalId(t, list, first))
		})
}

func TestOrdered_InsertAtBackWorks(t *testing.T) {
	const configFolder = "testdata/settings-ordered/insert-position"

	const manifestFile = configFolder + "/manifest.yaml"

	const specificEnvironment = "platform"
	const project = "insert-after-set-to-back"
	const schema = "builtin:url-based-sampling"

	pgiMeId := randomMeID("PROCESS_GROUP_INSTANCE")

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithEnvironment(specificEnvironment),
			v2.WithMeId(pgiMeId),
		},
		func(fs afero.Fs, tc v2.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project %s --verbose", manifestFile, project))
			require.NoError(t, err)
			integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{project}, specificEnvironment, true)

			sClient := createSettingsClientFromManifest(t, fs, manifestFile, "platform")

			list, err := sClient.List(t.Context(), schema, dtclient.ListSettingsOptions{
				DiscardValue: true,
				Filter:       filterObjectsForScope(pgiMeId),
			})

			assert.Equal(t, 2, len(list), "Exactly two configs should be deployed")

			// Verify that last is actually the last object
			last := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "second"}, tc)
			assert.Equal(t, len(list)-1, findPositionWithExternalId(t, list, last))
		})
}

func TestOrdered_InsertAtFrontAndBackWorks(t *testing.T) {
	const configFolder = "testdata/settings-ordered/insert-position"

	const manifestFile = configFolder + "/manifest.yaml"

	const specificEnvironment = "platform"
	const project = "both-back-and-front-are-set-with-initial"
	const schema = "builtin:url-based-sampling"

	pgiMeId := randomMeID("PROCESS_GROUP_INSTANCE")

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithEnvironment(specificEnvironment),
			v2.WithMeId(pgiMeId),
		},
		func(fs afero.Fs, tc v2.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project %s --verbose", manifestFile, project))
			require.NoError(t, err)
			integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{project}, specificEnvironment, true)

			sClient := createSettingsClientFromManifest(t, fs, manifestFile, "platform")

			list, err := sClient.List(t.Context(), schema, dtclient.ListSettingsOptions{
				DiscardValue: true,
				Filter:       filterObjectsForScope(pgiMeId),
			})

			assert.Equal(t, 3, len(list), "Exactly three configs should be deployed")

			// Verify that last is actually the first object
			first := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "first"}, tc)
			assert.Equal(t, 0, findPositionWithExternalId(t, list, first))

			// Verify that last is actually the last object
			last := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "last"}, tc)
			assert.Equal(t, len(list)-1, findPositionWithExternalId(t, list, last))
		})
}

func TestOrdered_InsertAtFrontAndBackWorksDeployTwice(t *testing.T) {
	const configFolder = "testdata/settings-ordered/insert-position"

	const manifestFile = configFolder + "/manifest.yaml"

	const specificEnvironment = "platform"
	const project = "both-back-and-front-are-set-deploy-twice"
	const schema = "builtin:url-based-sampling"

	pgiMeId := randomMeID("PROCESS_GROUP_INSTANCE")

	v2.Run(t, configFolder,
		v2.Options{
			v2.WithEnvironment(specificEnvironment),
			v2.WithMeId(pgiMeId),
		},
		func(fs afero.Fs, tc v2.TestContext) {
			// first
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project %s --verbose", manifestFile, project))
			require.NoError(t, err)
			// second
			err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project %s --verbose", manifestFile, project))
			require.NoError(t, err)

			integrationtest.AssertAllConfigsAvailability(t, fs, manifestFile, []string{project}, specificEnvironment, true)

			sClient := createSettingsClientFromManifest(t, fs, manifestFile, "platform")

			list, err := sClient.List(t.Context(), schema, dtclient.ListSettingsOptions{
				DiscardValue: true,
				Filter:       filterObjectsForScope(pgiMeId),
			})

			assert.Equal(t, 2, len(list), "Exactly two configs should be deployed")

			// Verify that last is actually the first object
			first := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "first"}, tc)
			assert.Equal(t, 0, findPositionWithExternalId(t, list, first))

			// Verify that last is actually the last object
			last := settingsExternalIdForTest(t, coordinate.Coordinate{Project: project, Type: schema, ConfigId: "last"}, tc)
			assert.Equal(t, len(list)-1, findPositionWithExternalId(t, list, last))
		})
}

func filterObjectsForScope(pgiMeId string) func(object dtclient.DownloadSettingsObject) bool {
	return func(object dtclient.DownloadSettingsObject) bool {
		return object.Scope == pgiMeId
	}
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

func settingsExternalIdForTest(t *testing.T, originalCoordinate coordinate.Coordinate, testContext v2.TestContext) string {

	originalCoordinate.ConfigId += "_" + testContext.Suffix

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

	clientSet := integrationtest.CreateDynatraceClients(t, man.Environments.SelectedEnvironments[environment])
	return clientSet.SettingsClient
}

// getRandomMonitoredEntitySuffix gets a random 16 uppercase hexadecimal character string for use as a suffix for creating Dynatrace entity IDs, such as `HOST-...`.
func getRandomMonitoredEntitySuffix() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // will never return an error and always fill the array

	return strings.ToUpper(hex.EncodeToString(b))
}

func randomMeID(meType string) string {
	return meType + "-" + getRandomMonitoredEntitySuffix()
}
