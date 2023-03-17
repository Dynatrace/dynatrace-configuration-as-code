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
	fs := createTempTestingDir(t)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()

	assert.NotContains(t, logs, "failed to setup")
}

func TestSetupLogging_logsDirExists(t *testing.T) {
	fs := createTempTestingDir(t)
	mkdir(t, fs, logsDir, 0777)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()

	assert.NotContains(t, logs, "failed to setup")
}

func TestSetupLogging_logsReadonly(t *testing.T) {
	fs := createTempTestingDir(t)
	mkdir(t, fs, logsDir, 0444)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()
	fmt.Println(logs)

	assert.Contains(t, logs, "failed to setup monaco-logging")
}

func TestSetupLogging_parentReadonly(t *testing.T) {
	fs := createTempTestingDir(t)
	chmod(t, fs, ".", 0444)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()
	fmt.Println(logs)

	assert.Contains(t, logs, "failed to setup monaco-logging")
}

func TestSetupLogging_requestLogReadonly(t *testing.T) {
	t.Setenv(envKeyRequestLog, "requests.txt")

	fs := createTempTestingDir(t)
	touch(t, fs, "requests.txt", 0444)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()
	fmt.Println(logs)

	assert.Contains(t, logs, "failed to setup request-logging")
}

func TestSetupLogging_responseLogReadonly(t *testing.T) {
	t.Setenv(envKeyResponseLog, "response.txt")

	fs := createTempTestingDir(t)
	touch(t, fs, "response.txt", 0444)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()
	fmt.Println(logs)

	assert.Contains(t, logs, "failed to setup response-logging")
}

func TestSetupLogging_allErrors(t *testing.T) {
	t.Setenv(envKeyRequestLog, "request.txt")
	t.Setenv(envKeyResponseLog, "response.txt")

	fs := createTempTestingDir(t)
	touch(t, fs, "response.txt", 0444)
	touch(t, fs, "request.txt", 0444)
	mkdir(t, fs, logsDir, 0444)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()
	fmt.Println(logs)

	assert.Contains(t, logs, "failed to setup response-logging")
	assert.Contains(t, logs, "failed to setup request-logging")
	assert.Contains(t, logs, "failed to setup monaco-logging")
}

func TestSetupLogging_requestAndResponseFileExistsWithCorrectPermissions(t *testing.T) {
	t.Setenv(envKeyRequestLog, "request.txt")
	t.Setenv(envKeyResponseLog, "response.txt")

	fs := createTempTestingDir(t)
	touch(t, fs, "response.txt", 0644)
	touch(t, fs, "request.txt", 0644)

	capturedLogs := &strings.Builder{}
	logger := builtinLog.New(capturedLogs, "[TestSetupLogging]", builtinLog.LstdFlags)

	SetupLogging(fs, logger)

	logs := capturedLogs.String()
	fmt.Println(logs)

	assert.NotContains(t, logs, "WARN")
	assert.Contains(t, logs, "request log activated")
	assert.Contains(t, logs, "response log activated")
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
