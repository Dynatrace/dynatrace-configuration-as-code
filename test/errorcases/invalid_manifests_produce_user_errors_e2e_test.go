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
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"

	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidManifest_ReportsError(t *testing.T) {
	tests := []struct {
		name             string
		manifestFileName string
		expectedErrorLog string
	}{
		{
			"version missing",
			"manifest_missing_version.yaml",
			"`manifestVersion` missing",
		},
		{
			"unsupported old manifest version",
			"manifest_too_low_version.yaml",
			"`manifestVersion` 0.0 is no longer supported",
		},
		{
			"unsupported new manifest version",
			"manifest_too_high_version.yaml",
			"`manifestVersion` 999999999.999999999 is not supported",
		},
		{
			"environments missing",
			"manifest_missing_envs.yaml",
			"'environmentGroups' are required, but not defined",
		},
		{
			"Both OAuth and platform token defined",
			"manifest_oauth_and_platform_token.yaml",
			"OAuth credentials and a platform token can't be used at the same time",
		},
	}

	t.Setenv(featureflags.PlatformToken.EnvName(), "true")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := filepath.Join("testdata/invalid-manifests/", tt.manifestFileName)

			logOutput := strings.Builder{}
			cmd := runner.BuildCmdWithLogSpy(testutils.CreateTestFileSystem(), &logOutput)
			cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
			err := cmd.Execute()

			assert.ErrorContains(t, err, "error while loading manifest")

			runLog := strings.ToLower(logOutput.String())
			lowerCaseExpectedErrorLog := strings.ToLower(tt.expectedErrorLog)
			assert.True(t, strings.Contains(runLog, lowerCaseExpectedErrorLog), "Expected command output to contain: %s", tt.expectedErrorLog)
		})
	}
}

func TestNonExistentProjectInManifestReturnsError(t *testing.T) {
	manifest := "testdata/invalid-manifests/manifest_non_existent_project.yaml"

	logOutput := strings.Builder{}
	cmd := runner.BuildCmdWithLogSpy(testutils.CreateTestFileSystem(), &logOutput)
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "failed to load projects")

	runLog := strings.ToLower(logOutput.String())
	expectedErrorLog := "filepath `this_does_not_exist` does not exist"
	assert.True(t, strings.Contains(runLog, expectedErrorLog), "Expected command output to contain: %s", expectedErrorLog)
}
