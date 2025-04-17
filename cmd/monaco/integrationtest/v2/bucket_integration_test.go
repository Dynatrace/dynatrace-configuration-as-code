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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils/matcher"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

// Tests a dry run (validation)
func TestIntegrationBucketValidation(t *testing.T) {
	t.Setenv("UNIQUE_TEST_SUFFIX", "can-be-nonunique-for-validation")

	configFolder := "test-resources/integration-bucket/"

	t.Run("project is valid", func(t *testing.T) {
		manifest := configFolder + "manifest.yaml"
		err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --verbose --dry-run", manifest))
		assert.NoError(t, err)
	})

	t.Run("broken project is invalid", func(t *testing.T) {
		manifest := configFolder + "invalid-manifest.yaml"
		err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --verbose --dry-run", manifest))
		assert.Error(t, err)
	})
}

func TestIntegrationBucket(t *testing.T) {

	configFolder := "test-resources/integration-bucket/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""

	RunIntegrationWithCleanup(t, configFolder, manifest, specificEnvironment, "Buckets", func(fs afero.Fs, _ TestContext) {

		// Create the buckets
		err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=project --verbose", manifest))
		assert.NoError(t, err)

		// Update the buckets
		err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=project --verbose", manifest))
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{"project"}, "", true)
	})
}

func TestIntegrationComplexBucket(t *testing.T) {

	configFolder := "test-resources/integration-bucket/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""

	RunIntegrationWithCleanup(t, configFolder, manifest, specificEnvironment, "ComplexBuckets", func(fs afero.Fs, _ TestContext) {

		// Create the buckets
		err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=complex-bucket --verbose", manifest))
		assert.NoError(t, err)

		// Update the buckets
		err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=complex-bucket --verbose", manifest))
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{"complex-bucket"}, "", true)
	})
}

func TestUploadDownload(t *testing.T) {
	configFolder := "test-resources/integration-bucket/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""
	downloadFolder := "test-resources/download"

	RunIntegrationWithCleanup(t, configFolder, manifest, specificEnvironment, "buckets", func(fs afero.Fs, _ TestContext) {

		// Create the buckets
		err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=project --verbose", manifest))
		require.NoError(t, err)

		// Download the buckets
		err = monaco.Run(t, fs, fmt.Sprintf("monaco download --only-buckets --manifest=%s --project=project --verbose --output-folder=%s --environment=%s", manifest, downloadFolder, "platform_env"))
		require.NoError(t, err)

		downloadedManifestPath := filepath.Join(downloadFolder, "manifest.yaml")

		uploadedConfigs := loadConfigs(t, manifest, fs, []string{"project"})
		downloadedConfigs := loadConfigs(t, downloadedManifestPath, fs, []string{})

		assert.True(t, matcher.ConfigsMatch(t, uploadedConfigs, downloadedConfigs))
	})
}

func loadConfigs(t *testing.T, manifestPath string, fs afero.Fs, specificProjects []string) []config.Config {
	m, errs := loader.Load(&loader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Opts: loader.Options{
			DoNotResolveEnvVars:      true,
			RequireEnvironmentGroups: true,
		},
	})
	testutils.FailTestOnAnyError(t, errs, "error during manifest load")

	apis := api.NewAPIs().Filter(api.RemoveDisabled)
	loadedProjects, errs := project.LoadProjects(t.Context(), fs, project.ProjectLoaderContext{
		KnownApis:       apis.GetApiNameLookup(),
		WorkingDir:      filepath.Dir(manifestPath),
		Manifest:        m,
		ParametersSerde: config.DefaultParameterParsers,
	}, specificProjects)
	testutils.FailTestOnAnyError(t, errs, "error during projects load")

	configs := make([]config.Config, 0)

	for _, p := range loadedProjects {
		p.ForEveryConfigDo(func(c config.Config) {
			configs = append(configs, c)
		})
	}
	return configs
}
