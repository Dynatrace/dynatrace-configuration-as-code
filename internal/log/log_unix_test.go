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

package log

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	builtinLog "log"
	"strings"
	"testing"
)

func TestSetupLogging_logsReadonly(t *testing.T) {
	defer func() {
		errs := closeLoggingFiles()
		assert.Empty(t, errs)
	}()
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
	defer func() {
		errs := closeLoggingFiles()
		assert.Empty(t, errs)
	}()
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
	defer func() {
		errs := closeLoggingFiles()
		assert.Empty(t, errs)
	}()
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
