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
	builtinLog "log"
	"os"
	"strings"
	"testing"
)

func TestSetupLogging_noError(t *testing.T) {
	defer func() {
		errs := closeLoggingFiles()
		assert.Empty(t, errs)
	}()
	fs := createTempTestingDir(t)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()

	assert.NotContains(t, logs, "failed to setup")
}

func TestSetupLogging_logsDirExists(t *testing.T) {
	defer func() {
		errs := closeLoggingFiles()
		assert.Empty(t, errs)
	}()
	fs := createTempTestingDir(t)
	mkdir(t, fs, logsDir, 0777)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()

	assert.NotContains(t, logs, "failed to setup")
}

func createTempTestingDir(t *testing.T) afero.Fs {
	return afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
}

func mkdir(t *testing.T, fs afero.Fs, path string, perm os.FileMode) {
	if err := fs.Mkdir(path, 0777); err != nil {
		t.Error(err)
	}

	chmod(t, fs, path, perm)
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

func Test_extendedLogger_DebugEnabled(t *testing.T) {

	tests := []struct {
		given logLevel
		want  bool
	}{
		{
			given: LevelInfo,
			want:  false,
		},
		{
			given: LevelWarn,
			want:  false,
		},
		{
			given: LevelError,
			want:  false,
		},
		{
			given: LevelFatal,
			want:  false,
		},
		{
			given: LevelDebug,
			want:  true,
		},
		{
			given: LevelDebug + 1, // imaginary added level (e.g. 'TRACE')
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("logLevel(%v)->DebugEnabled==%v", tt.given.prefix(), tt.want), func(t *testing.T) {
			Default().SetLevel(tt.given)
			assert.Equalf(t, tt.want, DebugEnabled(), "DebugEnabled()")
		})
	}
}
