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

func TestDeprecatedSettingsSchemasProduceWarnings(t *testing.T) {
	configFolder := "test-resources/deprecated-settings-schemas/"
	manifest := configFolder + "manifest.yaml"

	RunIntegrationWithCleanup(t, configFolder, manifest, "", "DeprecatedSchema", func(fs afero.Fs, _ TestContext) {

		logOutput := strings.Builder{}
		cmd := runner.BuildCmdWithLogSpy(testutils.CreateTestFileSystem(), &logOutput)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest})
		err := cmd.Execute()

		assert.NoError(t, err)

		runLog := strings.ToLower(logOutput.String())
		assert.Regexp(t, `.*?warn.*?schema \\"builtin:span-attribute\\" is deprecated.*?project:builtin:span-attribute:span-attr.*`, runLog)
		assert.Regexp(t, `.*?warn.*?schema \\"builtin:span-event-attribute\\" is deprecated.*?project:builtin:span-event-attribute:span-event.*`, runLog)
	})
}
