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

package testutils

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertSupportArchive(t *testing.T, fs afero.Fs, archive string, expectedFiles []string) {
	t.Helper()
	zipReader := ReadZipArchive(t, fs, archive)

	// Check that each expected file is present in the zip archive
	var foundFiles []string
	for _, file := range zipReader.File {
		foundFiles = append(foundFiles, file.Name)
	}

	assert.Len(t, foundFiles, len(expectedFiles), "expected archive to contain exactly %d files but got %d", len(expectedFiles), len(foundFiles))
	assert.ElementsMatchf(t, foundFiles, expectedFiles, "expected archive to contain all expected files %v", expectedFiles)
}

func ReadZipArchive(t *testing.T, fs afero.Fs, archive string) *zip.Reader {
	t.Helper()
	exists, err := afero.Exists(fs, archive)
	require.NoError(t, err)
	assert.True(t, exists, "Expected support archive %s to exist, but it didn't", archive)

	// Read the created zip file
	zipFile, err := fs.Open(archive)
	require.NoError(t, err, "Expected no error")
	defer zipFile.Close()

	// Extract the file names from the zip archive
	archiveData, err := io.ReadAll(zipFile)
	require.NoError(t, err, "Expected no error")

	// Open the zip archive for reading
	zipReader, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
	require.NoError(t, err, "Expected no error")

	return zipReader
}
