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

package zap

import (
	"bytes"
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
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
	file, err := os.CreateTemp("", "baseLogger-testfile_")
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
	file, err := os.CreateTemp("", "baseLogger-testfile_")
	defer file.Close()
	logger, err := New(loggers.LogOptions{File: file})
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	logger.Info("hello")

	content, _ := os.ReadFile(file.Name())
	assert.True(t, strings.HasSuffix(string(content), "info\thello\n"))
}

func TestLoggerReturnsCustomLogLevell(t *testing.T) {
	logger, err := New(loggers.LogOptions{LogLevel: loggers.LevelDebug})
	assert.NoError(t, err)
	assert.Equal(t, loggers.LevelDebug, logger.Level())
}

func TestLoggerReturnsDefaultLogLevel(t *testing.T) {
	logger, err := New(loggers.LogOptions{})
	assert.NoError(t, err)
	assert.Equal(t, loggers.LevelInfo, logger.Level())
}

func TestWithFieldsSubsequently(t *testing.T) {
	logSpy := bytes.Buffer{}
	logger, _ := New(loggers.LogOptions{ConsoleLoggingJSON: true, LogSpy: &logSpy})

	// log with field - contains field
	logger.WithFields(field.F("City", "Monaco")).Info("Hi")
	var data map[string]interface{}
	json.Unmarshal(logSpy.Bytes(), &data)
	assert.Equal(t, "Monaco", data["City"])

	logSpy.Reset()

	//log without field - does not contain field
	logger.Info("Hi")
	var data2 map[string]interface{}
	json.Unmarshal(logSpy.Bytes(), &data2)
	assert.NotContains(t, data2, "City")
}

func TestAllMethodsHaveFields(t *testing.T) {
	logSpy := bytes.Buffer{}
	logger, _ := New(loggers.LogOptions{ConsoleLoggingJSON: true, LogSpy: &logSpy, LogLevel: loggers.LevelDebug})

	t.Run("Info log - has fields", func(t *testing.T) {
		defer logSpy.Reset()
		var data map[string]interface{}
		logger.WithFields(field.F("City", "Berlin")).Info("Hi")
		json.Unmarshal(logSpy.Bytes(), &data)
		assert.Equal(t, "Berlin", data["City"])
	})
	t.Run("Debug log - has fields", func(t *testing.T) {
		defer logSpy.Reset()
		var data map[string]interface{}
		logger.WithFields(field.F("City", "London")).Debug("Hi")
		json.Unmarshal(logSpy.Bytes(), &data)
		assert.Equal(t, "London", data["City"])
	})
	t.Run("Warn log - has fields", func(t *testing.T) {
		defer logSpy.Reset()
		var data map[string]interface{}
		logger.WithFields(field.F("City", "Vienna")).Warn("Hi")
		json.Unmarshal(logSpy.Bytes(), &data)
		assert.Equal(t, "Vienna", data["City"])
	})
	t.Run("Error log - has fields", func(t *testing.T) {
		defer logSpy.Reset()
		var data map[string]interface{}
		logger.WithFields(field.F("City", "Amsterdam")).Error("Hi")
		json.Unmarshal(logSpy.Bytes(), &data)
		assert.Equal(t, "Amsterdam", data["City"])
	})
}

func TestLogger_WhenUsingUnstructuredLogFormat_DoesNotPrintFieldsToConsole(t *testing.T) {
	logSpy := bytes.Buffer{}
	logger, _ := New(loggers.LogOptions{ConsoleLoggingJSON: false, LogSpy: &logSpy})
	logger.WithFields(field.F("my-key", "")).Info("hello")
	assert.NotContains(t, logSpy.String(), "my-key")
}

func TestLogger_WhenUsingUnstructuredLogFormat_DoesNotPrintFieldsToFile(t *testing.T) {
	file, _ := os.CreateTemp("", "baseLogger-testfile_")
	defer file.Close()
	logger, _ := New(loggers.LogOptions{File: file})

	logger.WithFields(field.F("my-key", "")).Info("hello")
	content, _ := os.ReadFile(file.Name())
	assert.NotContains(t, content, "my-key")
}
