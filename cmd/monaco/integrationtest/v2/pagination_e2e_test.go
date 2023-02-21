//go:build nightly

/**
 * @license
 * Copyright 2022 Dynatrace LLC
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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/testutils"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"path/filepath"
	"strings"
	"testing"
)

func TestPagination(t *testing.T) {

	configFolder := "test-resources/pagination-test-configs/"
	manifestPath := configFolder + "manifest.yaml"
	specificEnvironment := ""

	fs := testutils.CreateTestFileSystem()

	//create config yaml
	settingsPageSize := 500
	additionalSettingsOnNextPage := 50
	totalSettings := settingsPageSize + additionalSettingsOnNextPage

	configContent := "configs:\n"
	for i := 0; i < totalSettings; i++ {
		id := fmt.Sprintf("tag_%d", i)
		configContent += fmt.Sprintf("- id: %s\n  type:\n    settings:\n      schema: builtin:tags.auto-tagging\n      scope: environment\n  config:\n    name: %s\n    template: auto-tag-setting.json\n", id, id)
	}

	configYamlPath, err := filepath.Abs(filepath.Join(configFolder, "project", "config.yaml"))
	assert.NilError(t, err)
	err = afero.WriteFile(fs, configYamlPath, []byte(configContent), 644)
	assert.NilError(t, err)

	RunIntegrationWithCleanupOnGivenFs(t, fs, configFolder, manifestPath, specificEnvironment, "Pagination", func(fs afero.Fs) {

		// Create/POST all 550 Settings
		logOutput := strings.Builder{}
		cmd := runner.BuildCliWithCapturedLog(fs, &logOutput)
		cmd.SetArgs([]string{"deploy", "--verbose", manifestPath})
		err := cmd.Execute()
		assert.NilError(t, err)
		assert.Equal(t, strings.Count(logOutput.String(), "Created"), totalSettings)
		assert.Assert(t, !strings.Contains(logOutput.String(), "Updated"))

		logOutput.Reset()

		// Update/PUT all 550 Settings - means that all previously created ones were found, and more than one 500 element page retrieved
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifestPath})
		err = cmd.Execute()
		assert.NilError(t, err)
		assert.Assert(t, !strings.Contains(logOutput.String(), "Created"))
		assert.Equal(t, strings.Count(logOutput.String(), "Updated"), totalSettings)
	})
}
