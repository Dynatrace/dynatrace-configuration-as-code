//go:build unit && unix

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

package trafficlogs

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestSetupLogging_RequestAndResponseLogReadonly(t *testing.T) {
	defer func() {
		errs := closeLoggingFiles()
		assert.Empty(t, errs)
	}()
	t.Setenv(envKeyRequestLog, "requests.txt")
	t.Setenv(envKeyResponseLog, "responses.txt")

	fs := createTempTestingDir(t)
	touch(t, fs, "requests.txt", 0444)
	touch(t, fs, "responses.txt", 0444)

	err := setupRequestLog(fs)
	assert.Error(t, err)
	err = setupResponseLog(fs)
	assert.Error(t, err)
}

func createTempTestingDir(t *testing.T) afero.Fs {
	return afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
}

func chmod(t *testing.T, fs afero.Fs, path string, perm os.FileMode) {
	if err := fs.Chmod(path, perm); err != nil {
		t.Error(err)
	}
}

func touch(t *testing.T, fs afero.Fs, path string, perm os.FileMode) {
	file, err := fs.Create(path)
	if err != nil {
		t.Error(err)
		return
	}

	if err := file.Close(); err != nil {
		t.Error(err)
		return
	}

	chmod(t, fs, path, perm)
}
