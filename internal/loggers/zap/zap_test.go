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

package zap

import (
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/loggers"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	logger, err := New(loggers.LogOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestNewLoggerWithFileJSONEncoded(t *testing.T) {
	file, err := os.CreateTemp("", "logger-testfile_")
	defer file.Close()
	logger, err := New(loggers.LogOptions{File: file, FileLoggingJSON: true})
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	logger.Info("hello")

	content, _ := os.ReadFile(file.Name())
	var jsonData interface{}
	err = json.Unmarshal(content, &jsonData)
	require.NoError(t, err)
}

func TestNewLoggerWithFile(t *testing.T) {
	file, err := os.CreateTemp("", "logger-testfile_")
	defer file.Close()
	logger, err := New(loggers.LogOptions{File: file})
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	logger.Info("hello")

	content, _ := os.ReadFile(file.Name())
	assert.True(t, strings.HasSuffix(string(content), "info\thello\n"))
}
