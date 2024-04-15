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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestMonacoVersionLoggedOnStartup(t *testing.T) {
	fs := testutils.CreateTestFileSystem()
	logOutput := strings.Builder{}

	cmd := runner.BuildCliWithLogSpy(fs, &logOutput)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.NoError(t, err, "expect no error as usage to be displayed")

	runLog := logOutput.String()
	assert.Contains(t, runLog, "Monaco version")
}
