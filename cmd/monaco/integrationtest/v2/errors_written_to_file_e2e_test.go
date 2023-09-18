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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestManifestErrorsAreWrittenToFile(t *testing.T) {
	manifest := filepath.Join("test-resources/invalid-manifests/", "manifest_non_existent_project.yaml")

	fs := testutils.CreateTestFileSystem()

	cmd := runner.BuildCli(fs)
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()
	assert.ErrorContains(t, err, "error while loading projects")

	expectedErrFile := log.ErrorFilePath()

	exists, err := afero.Exists(fs, expectedErrFile)
	assert.NoError(t, err)
	assert.True(t, exists, "expected file to exist %s", expectedErrFile)

	errorLog, err := afero.ReadFile(fs, expectedErrFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, errorLog)
	expectedErrorLog := "filepath `this_does_not_exist` does not exist"
	assert.Contains(t, string(errorLog), expectedErrorLog)

}

func TestConfigErrorsAreWrittenToFile(t *testing.T) {

	configFolder := "test-resources/configs-with-duplicate-ids/"
	manifest := filepath.Join(configFolder, "manifest.yaml")

	fs := testutils.CreateTestFileSystem()

	cmd := runner.BuildCli(fs)
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "error while loading projects")

	expectedErrFile := log.ErrorFilePath()

	exists, err := afero.Exists(fs, expectedErrFile)
	assert.NoError(t, err)
	assert.True(t, exists, "expected file to exist %s", expectedErrFile)

	errorLog, err := afero.ReadFile(fs, expectedErrFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, errorLog)
	assert.Contains(t, string(errorLog), "duplicate")
	assert.Contains(t, string(errorLog), "project:alerting-profile:profile")
}
