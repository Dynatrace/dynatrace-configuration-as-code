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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
)

func TestAPIErrorsAreReported(t *testing.T) {
	configFolder := "test-resources/configs-with-invalid-payload/"
	manifest := configFolder + "manifest.yaml"

	RunIntegrationWithCleanup(t, configFolder, manifest, "", "InvalidJSON", func(fs afero.Fs, _ TestContext) {

		logOutput := strings.Builder{}
		cmd := runner.BuildCmdWithLogSpy(testutils.CreateTestFileSystem(), &logOutput)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest, "--continue-on-error"})
		err := cmd.Execute()

		assert.ErrorContains(t, err, "Deployment failed")
		assert.ErrorContains(t, err, "2 environment(s)")
		assert.ErrorContains(t, err, "classic_env")
		assert.ErrorContains(t, err, "platform_env")
		assert.ErrorContains(t, err, "2 deployment errors")

		runLog := strings.ToLower(logOutput.String())
		assert.Regexp(t, ".*?error.*?invalid-config-api-with-settings-payload.*?deployment failed - dynatrace api rejected http request.*?", runLog)
		assert.Regexp(t, ".*?error.*?tags.auto-tagging:invalid-setting-with-config-api-payload.*?deployment failed - dynatrace api rejected http request.*?", runLog)
		assert.Contains(t, runLog, "deployment failed for environment \\\"classic_env\\\"")
		assert.Contains(t, runLog, "deployment failed for environment \\\"platform_env\\\"")
	})
}
