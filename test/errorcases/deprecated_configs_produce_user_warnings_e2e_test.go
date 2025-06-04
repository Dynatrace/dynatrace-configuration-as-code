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

package errorcases

import (
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	runner2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestDeprecatedConfigsProduceWarnings(t *testing.T) {
	configFolder := "testdata/deprecated-configs/"
	manifest := configFolder + "manifest.yaml"

	runner2.Run(t, configFolder,
		runner2.Options{},
		func(fs afero.Fs, _ runner2.TestContext) {

			logOutput := strings.Builder{}
			cmd := runner.BuildCmdWithLogSpy(testutils.CreateTestFileSystem(), &logOutput)
			cmd.SetArgs([]string{"deploy", "--verbose", manifest})
			err := cmd.Execute()
			require.NoError(t, err)

			expectedRegexp := regexp.MustCompile(`API 'auto-tag' is deprecated. Please migrate to 'builtin:tags.auto-tagging'.`)
			found := expectedRegexp.FindAllString(logOutput.String(), -1)
			assert.Len(t, found, 1)
		})
}
