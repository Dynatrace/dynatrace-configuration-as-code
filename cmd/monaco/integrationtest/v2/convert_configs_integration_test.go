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
	"path"
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/afero"
)

func setupConvertedConfig(t *testing.T) (testFs afero.Fs, convertedFolder string) {
	configV1Folder := "../v1/test-resources/integration-all-configs/"
	env := path.Join(configV1Folder, "environments.yaml")

	convertedConfigV2Folder, err := filepath.Abs("./test-resources/converted-v1-integration-all-configs")
	assert.NoError(t, err)

	fs := testutils.CreateTestFileSystem()

	err = monaco.RunWithFSf(fs, "monaco convert %s %s --output-folder=%s ", env, configV1Folder, convertedConfigV2Folder)
	assert.NoError(t, err)

	return fs, convertedConfigV2Folder
}

func TestV1ConfigurationCanBeConverted(t *testing.T) {
	fs, convertedConfigV2Folder := setupConvertedConfig(t)

	assertExpectedPathExists(t, fs, convertedConfigV2Folder)
	assertExpectedPathExists(t, fs, path.Join(convertedConfigV2Folder, "manifest.yaml"))
	assertExpectedPathExists(t, fs, path.Join(convertedConfigV2Folder, "delete.yaml"))
	assertExpectedPathExists(t, fs, path.Join(convertedConfigV2Folder, "project/"))
	assertExpectedPathExists(t, fs, path.Join(convertedConfigV2Folder, "project/auto-tag/config.yaml")) // check one sample config
}

func assertExpectedPathExists(t *testing.T, fs afero.Fs, path string) {
	fileExists, _ := afero.Exists(fs, path)
	assert.True(t, fileExists, "Expected %s to exist", path)
}

// tests conversion from v1 by converting v1 test-resources before deploying as v2
func TestV1ConfigurationCanBeConvertedAndDeployedAfterConversion(t *testing.T) {

	fs, convertedConfigV2Folder := setupConvertedConfig(t)
	assertExpectedPathExists(t, fs, convertedConfigV2Folder)

	manifest := path.Join(convertedConfigV2Folder, "manifest.yaml")
	assertExpectedPathExists(t, fs, manifest)

	RunIntegrationWithCleanupOnGivenFs(t, fs, convertedConfigV2Folder, manifest, "", "AllConfigs", func(fs afero.Fs, _ TestContext) {
		// This causes a POST for all configs:
		err := monaco.RunWithFSf(fs, "monaco deploy %s --verbose", manifest)
		assert.NoError(t, err)
	})
}
