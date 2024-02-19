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

package v2

import (
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

var diffProjectDiffExtIDFolder = "test-resources/integration-different-projects-different-extid/"
var diffProjectDiffExtIDFolderManifest = diffProjectDiffExtIDFolder + "manifest.yaml"

// TestSettingsInDifferentProjectsGetDifferentExternalIDs tries to upload a project that contatins two projects with
// the exact same settings 2.0 object and verifies that deploying such a monaco configuration results in
// two different settings objects deployed on the environment
func TestSettingsInDifferentProjectsGetDifferentExternalIDs(t *testing.T) {

	RunIntegrationWithCleanup(t, diffProjectDiffExtIDFolder, diffProjectDiffExtIDFolderManifest, "", "DifferentProjectsGetDifferentExternalID", func(fs afero.Fs, _ TestContext) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", diffProjectDiffExtIDFolderManifest})
		err := cmd.Execute()

		assert.NoError(t, err)

		var manifestPath = diffProjectDiffExtIDFolderManifest
		loadedManifest := integrationtest.LoadManifest(t, fs, manifestPath, "")
		environment := loadedManifest.Environments["platform_env"]
		projects := integrationtest.LoadProjects(t, fs, manifestPath, loadedManifest)
		sortedConfigs, _ := sort.ConfigsPerEnvironment(projects, []string{"platform_env"})

		extIDProject1, _ := idutils.GenerateExternalID(sortedConfigs["platform_env"][0].Coordinate)
		extIDProject2, _ := idutils.GenerateExternalID(sortedConfigs["platform_env"][1].Coordinate)

		clientSet, err := dynatrace.CreateClients(environment.URL.Value, environment.Auth)
		assert.NoError(t, err)
		c := clientSet.Settings()
		settings, _ := c.ListSettings(context.TODO(), "builtin:anomaly-detection.metric-events", dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(object dtclient.DownloadSettingsObject) bool {
			return object.ExternalId == extIDProject1 || object.ExternalId == extIDProject2
		}})
		assert.Len(t, settings, 2)
	})
}
