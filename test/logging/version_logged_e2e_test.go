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

package logging

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
)

func TestMonacoVersionLogging(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		shouldLogVersion bool
	}{
		{
			name:             "With no args no version should be logged",
			args:             []string{},
			shouldLogVersion: false,
		},
		{
			name:             "Help should not log version",
			args:             []string{"help"},
			shouldLogVersion: false,
		},
		{
			name:             "Version should not log version",
			args:             []string{"version"},
			shouldLogVersion: false,
		},
		{
			name:             "Download should log version",
			args:             []string{"download", "--manifest", "non_existing_manifest.yaml", "--environment", "non_existing_env"},
			shouldLogVersion: true,
		},
		{
			name:             "Incomplete deploy should not log version",
			args:             []string{"deploy"},
			shouldLogVersion: false,
		},
		{
			name:             "Deploy should log version",
			args:             []string{"deploy", "non_existing_manifest.yaml"},
			shouldLogVersion: true,
		},
		{
			name:             "Incomplete account should not log version",
			args:             []string{"account"},
			shouldLogVersion: false,
		},
		{
			name:             "Account download should log version",
			args:             []string{"account", "download"},
			shouldLogVersion: true,
		},
		{
			name:             "Account deploy should log version",
			args:             []string{"account", "deploy"},
			shouldLogVersion: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fs := testutils.CreateTestFileSystem()
			logOutput := strings.Builder{}

			cmd := runner.BuildCmdWithLogSpy(fs, &logOutput)
			cmd.SetArgs(tt.args)
			_ = cmd.Execute()

			runLog := logOutput.String()
			const versionLogMessage = "Monaco version"
			if tt.shouldLogVersion {
				assert.Contains(t, runLog, versionLogMessage)
			} else {
				assert.NotContains(t, runLog, versionLogMessage)
			}
		})
	}
}
