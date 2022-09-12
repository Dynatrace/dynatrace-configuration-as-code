//go:build integration
// +build integration

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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"path"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// tests conversion from v1 by converting v1 test-resources before deploying as v2
func TestV1ConfigurationCanBeConvertedAndDeployedAfterConversion(t *testing.T) {

	// QUICKFIX: ensure the conversion reads configs from disk rather than anything cached from prev runs
	template.InitTemplateCache()

	configV1Folder := "../v1/test-resources/integration-all-configs/"
	env := path.Join(configV1Folder, "environments.yaml")

	convertedConfigV2Folder, err := filepath.Abs("./test-resources/converted-v1-integration-all-configs")
	assert.NilError(t, err)

	fs := util.CreateTestFileSystem()

	cmd := runner.BuildCli(fs)
	cmd.SetArgs(
		[]string{
			"convert",
			env,
			configV1Folder,
			"--output-folder", convertedConfigV2Folder,
		},
	)
	err = cmd.Execute()

	assert.NilError(t, err)

	_, err = fs.Stat(convertedConfigV2Folder)
	assert.NilError(t, err, "Expected converted config folder %s to exist", convertedConfigV2Folder)

	manifest := path.Join(convertedConfigV2Folder, "manifest.yaml")

	RunIntegrationWithCleanupOnGivenFs(t, fs, convertedConfigV2Folder, manifest, "", "AllConfigs", func(fs afero.Fs) {

		// This causes a POST for all configs:
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest})
		err := cmd.Execute()

		assert.NilError(t, err)
	})
}
