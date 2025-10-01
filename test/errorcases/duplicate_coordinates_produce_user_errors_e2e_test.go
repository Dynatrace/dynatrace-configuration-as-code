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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
)

func TestAllDuplicateErrorsAreReported(t *testing.T) {

	configFolder := "testdata/configs-with-duplicate-ids/"
	manifest := filepath.Join(configFolder, "manifest.yaml")

	logOutput := strings.Builder{}
	cmd, _ := runner.BuildCmdWithLogSpy(testutils.CreateTestFileSystem(), &logOutput)
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "failed to load projects")

	runLog := strings.ToLower(logOutput.String())
	assert.Contains(t, runLog, "duplicate")
	assert.Contains(t, runLog, "project:alerting-profile:profile")
}
