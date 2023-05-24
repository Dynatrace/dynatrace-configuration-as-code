//go:build unit

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

package log

import (
	"fmt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// CustomMemMapFs embeds afero.MemMapFs and overrides the MkdirAll method
type CustomMemMapFs struct {
	afero.MemMapFs
}

// MkdirAll overrides the default implementation of MkdirAll
func (fs *CustomMemMapFs) MkdirAll(path string, perm os.FileMode) error {
	if fs.DirExists(path) {
		return fmt.Errorf("directory already exists: %s", path)
	}

	return fs.MemMapFs.MkdirAll(path, perm)
}

// DirExists checks if a directory exists in the file system
func (fs *CustomMemMapFs) DirExists(path string) bool {
	fi, err := fs.Stat(path)
	if err != nil {
		return false
	}

	return fi.IsDir()
}

func TestPrepareLogFile_ReturnsErrIfParentDirectoryAlreadyExists(t *testing.T) {
	fs := &CustomMemMapFs{}
	fs.MkdirAll(".logs", 0777)
	file, err := prepareLogFile(fs)
	assert.Nil(t, file)
	assert.Error(t, err)
}

func TestPrepareLogFile_ReturnsErrIfParentDirIsReadOnly(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	file, err := prepareLogFile(fs)
	assert.Nil(t, file)
	assert.Error(t, err)
}
