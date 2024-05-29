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
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/spf13/afero"
)

func TestSupportArchiveIsCreatedAsExpected(t *testing.T) {
	configFolder := "test-resources/support-archive/"
	manifest := configFolder + "manifest.yaml"
	fixedTime := timeutils.TimeAnchor().Format(trafficlogs.TrafficLogFilePrefixFormat) // freeze time to ensure log files are created with expected names

	RunIntegrationWithCleanup(t, configFolder, manifest, "valid_env", "SupportArchive", func(fs afero.Fs, _ TestContext) {
		cleanupLogsDir(t)

		cmd := runner.BuildCmd(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			"--support-archive",
			"-e",
			"valid_env",
			manifest,
		})

		runner.RunCmd(fs, cmd)

		archive := "support-archive-" + fixedTime + ".zip"

		expectedFiles := []string{fixedTime + "-" + "req.log", fixedTime + "-" + "resp.log", fixedTime + ".log", fixedTime + "-errors.log", fixedTime + "-featureflag_state.log"}

		assertSupportArchive(t, fs, archive, expectedFiles)

		zipReader := readZipArchive(t, fs, archive)
		logFile, err := zipReader.Open(fixedTime + ".log")
		defer logFile.Close()
		assert.NoError(t, err)
		content, err := io.ReadAll(logFile)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "debug", "expected log file to contain debug log entries")
	})
}

// TestSupportArchiveIsCreatedInErrorCases is split from the success-case test as these test-cases will not create objects
// on the Dynatrace environment and do not need cleanup - or work successfully with the normal test runner which expects
// that it can load the manifest and connect to the environment
func TestSupportArchiveIsCreatedInErrorCases(t *testing.T) {
	configFolder := "test-resources/support-archive/"

	tests := []struct {
		name           string
		manifestFile   string
		environment    string
		expectAllFiles bool
	}{
		{
			"Full archive in case of HTTP auth errors",
			"manifest.yaml",
			"unauthorized_env", // has wrong Config API token
			true,
		},
		{
			"Partial archive in case of invalid URL",
			"manifest.yaml",
			"invalid_env", // has an invalid URL
			false,
		},
		{
			"Partial archive in case of invalid manifest",
			"invalid-manifest.yaml", // will fail on loading manifest, before any API calls are made
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupLogsDir(t)

			fixedTime := timeutils.TimeAnchor().Format(trafficlogs.TrafficLogFilePrefixFormat) // freeze time to ensure log files are created with expected names

			fs := testutils.CreateTestFileSystem()

			manifest := configFolder + tt.manifestFile

			cmd := runner.BuildCmd(fs)
			cmd.SetArgs([]string{
				"deploy",
				"--verbose",
				"--support-archive",
				"-e",
				tt.environment,
				manifest,
			})

			runner.RunCmd(fs, cmd)

			archive := "support-archive-" + fixedTime + ".zip"
			expectedFiles := []string{fixedTime + ".log", fixedTime + "-errors.log", fixedTime + "-featureflag_state.log"}
			if tt.expectAllFiles {
				expectedFiles = append(expectedFiles, fixedTime+"-"+"req.log", fixedTime+"-"+"resp.log")
			}

			assertSupportArchive(t, fs, archive, expectedFiles)
		})
	}
}

func assertSupportArchive(t *testing.T, fs afero.Fs, archive string, expectedFiles []string) {
	zipReader := readZipArchive(t, fs, archive)

	// Check that each expected file is present in the zip archive
	var foundFiles []string
	for _, file := range zipReader.File {
		foundFiles = append(foundFiles, file.Name)
	}

	assert.Len(t, foundFiles, len(expectedFiles), "expected archive to contain exactly %d files but got %d", len(expectedFiles), len(foundFiles))
	assert.ElementsMatchf(t, foundFiles, expectedFiles, "expected archive to contain all expected files %v", expectedFiles)
}

func readZipArchive(t *testing.T, fs afero.Fs, archive string) *zip.Reader {
	exists, err := afero.Exists(fs, archive)
	assert.NoError(t, err)
	assert.True(t, exists, "Expected support archive %s to exist, but it didn't", archive)

	// Read the created zip file
	zipFile, err := fs.Open(archive)
	assert.NoError(t, err, "Expected no error")
	defer zipFile.Close()

	// Extract the file names from the zip archive
	archiveData, err := io.ReadAll(zipFile)
	assert.NoError(t, err, "Expected no error")

	// Open the zip archive for reading
	zipReader, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
	assert.NoError(t, err, "Expected no error")

	return zipReader
}

// traffic logs always write to the OsFs so to ensure we tests start with a clean slate cleanupLogsDir removes the folder
func cleanupLogsDir(t *testing.T) {
	logPath, err := filepath.Abs(log.LogDirectory)
	assert.NoError(t, err)

	err = afero.NewOsFs().RemoveAll(logPath)
	assert.NoError(t, err)
}
