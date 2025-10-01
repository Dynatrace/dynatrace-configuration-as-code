//go:build nightly

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

package pagination

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	assert2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/assert"
	runner2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestPaginationClassic(t *testing.T) {
	testPagination(t, "classic_env")
}

func TestPaginationPlatform(t *testing.T) {
	testPagination(t, "platform_env")
}

func testPagination(t *testing.T, specificEnvironment string) {

	configFolder := "testdata/pagination-test-configs/"
	manifestPath := configFolder + "manifest.yaml"

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
	assert.NoError(t, err)
	err = afero.WriteFile(fs, configYamlPath, []byte(configContent), 644)
	assert.NoError(t, err)

	runner2.Run(t, configFolder,
		runner2.Options{
			runner2.WithManifestPath(manifestPath),
			runner2.WithSuffix("Pagination"),
			runner2.WithEnvironment(specificEnvironment),
			runner2.WithFs(fs),
		},
		func(fs afero.Fs, _ runner2.TestContext) {

			// Create/POST all 550 Settings
			logOutput := strings.Builder{}
			cmd, _ := runner.BuildCmdWithLogSpy(fs, &logOutput)
			cmd.SetArgs([]string{"deploy", "--verbose", manifestPath, "--environment", specificEnvironment})
			err := cmd.Execute()
			assert.NoError(t, err)
			assert.Equal(t, strings.Count(logOutput.String(), "Created/Updated"), totalSettings)

			assert2.AssertAllConfigsAvailability(t, fs, manifestPath, []string{}, specificEnvironment, true)

			logOutput.Reset()

			// Update/PUT all 550 Settings - means that all previously created ones were found, and more than one 500 element page retrieved
			cmd, _ = runner.BuildCmdWithLogSpy(fs, &logOutput)
			cmd.SetArgs([]string{"deploy", "--verbose", manifestPath, "--environment", specificEnvironment})
			err = cmd.Execute()
			assert.NoError(t, err)
			assert.Equal(t, strings.Count(logOutput.String(), "Created/Updated"), totalSettings)

			assert2.AssertAllConfigsAvailability(t, fs, manifestPath, []string{}, specificEnvironment, true)
		})
}
