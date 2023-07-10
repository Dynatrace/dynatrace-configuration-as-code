/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package zip

import (
	"archive/zip"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

import (
	"bytes"
	"io/ioutil"
)

func TestCreate(t *testing.T) {
	fs := afero.NewMemMapFs()
	file, _ := fs.Create("file1.txt")
	file.Close()

	file, _ = fs.Create("file2.txt")
	file.Close()

	files := []string{"file1.txt", "file2.txt"}
	err := Create(fs, "test.zip", files, false)
	assert.NoError(t, err, "Expected no error")

	// Read the created zip file
	zipFile, err := fs.Open("test.zip")
	assert.NoError(t, err, "Expected no error")
	defer zipFile.Close()

	// Extract the file names from the zip archive
	archiveData, err := ioutil.ReadAll(zipFile)
	assert.NoError(t, err, "Expected no error")

	// Open the zip archive for reading
	zipReader, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
	assert.NoError(t, err, "Expected no error")

	// Check that each expected file is present in the zip archive
	foundFiles := make(map[string]bool)
	for _, file := range zipReader.File {
		foundFiles[file.Name] = true
	}

	for _, expectedFile := range files {
		assert.True(t, foundFiles[expectedFile], "Expected file '%s' in zip archive", expectedFile)
	}
}
