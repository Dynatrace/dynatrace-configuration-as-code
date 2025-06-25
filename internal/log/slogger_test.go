//go:build unit

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

package log_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// TestSlogger tests that the slogger functions work as expected by wrapping slog.
func TestSlogger(t *testing.T) {
	options := slog.HandlerOptions{Level: slog.LevelDebug}
	t.Run("debug", func(t *testing.T) {
		handler := NewTestHandler(&options)

		logger := &log.Slogger{Logger: slog.New(handler)}
		logger.Debug("code %s reached", "here")

		assert.Contains(t, handler.Output.String(), "code here reached")
		assert.Contains(t, strings.ToLower(handler.Output.String()), "debug")
	})

	t.Run("debug with context", func(t *testing.T) {
		handler := NewTestHandler(&options)

		logger := &log.Slogger{Logger: slog.New(handler)}
		logger.DebugContext(t.Context(), "code %s reached", "here")

		assert.Contains(t, handler.Output.String(), "code here reached")
		assert.Contains(t, strings.ToLower(handler.Output.String()), "debug")
	})

	t.Run("info", func(t *testing.T) {
		handler := NewTestHandler(&options)

		logger := &log.Slogger{Logger: slog.New(handler)}
		logger.Info("code %s reached", "here")

		assert.Contains(t, handler.Output.String(), "code here reached")
		assert.Contains(t, strings.ToLower(handler.Output.String()), "info")
	})

	t.Run("info with context", func(t *testing.T) {
		handler := NewTestHandler(&options)

		logger := &log.Slogger{Logger: slog.New(handler)}
		logger.InfoContext(t.Context(), "code %s reached", "here")

		assert.Contains(t, handler.Output.String(), "code here reached")
		assert.Contains(t, strings.ToLower(handler.Output.String()), "info")
	})

	t.Run("warn", func(t *testing.T) {
		handler := NewTestHandler(&options)

		logger := &log.Slogger{Logger: slog.New(handler)}
		logger.Warn("code %s reached", "here")

		assert.Contains(t, handler.Output.String(), "code here reached")
		assert.Contains(t, strings.ToLower(handler.Output.String()), "warn")
	})

	t.Run("warn with context", func(t *testing.T) {
		handler := NewTestHandler(&options)

		logger := &log.Slogger{Logger: slog.New(handler)}
		logger.WarnContext(t.Context(), "code %s reached", "here")

		assert.Contains(t, handler.Output.String(), "code here reached")
		assert.Contains(t, strings.ToLower(handler.Output.String()), "warn")
	})

	t.Run("error", func(t *testing.T) {
		handler := NewTestHandler(&options)

		logger := &log.Slogger{Logger: slog.New(handler)}
		logger.Error("code %s reached", "here")

		assert.Contains(t, handler.Output.String(), "code here reached")
		assert.Contains(t, strings.ToLower(handler.Output.String()), "error")
	})

	t.Run("error with context", func(t *testing.T) {
		handler := NewTestHandler(&options)

		logger := &log.Slogger{Logger: slog.New(handler)}
		logger.ErrorContext(t.Context(), "code %s reached", "here")

		assert.Contains(t, handler.Output.String(), "code here reached")
		assert.Contains(t, strings.ToLower(handler.Output.String()), "error")
	})
}

// TestWith tests the creation of an Slogger with attributes that includes them in each log message.
func TestWith(t *testing.T) {
	logSpy := bytes.Buffer{}
	t.Setenv("MONACO_LOG_FORMAT", "json")
	log.PrepareLogging(t.Context(), afero.NewOsFs(), false, &logSpy, false, false)

	log.With(
		slog.Any("Title", "Captain"),
		slog.Any("Name", "Iglo"),
		log.CoordinateAttr(coordinate.Coordinate{Project: "p1", Type: "t1", ConfigId: "c1"}),
		log.EnvironmentAttr("env1", "group")).Info("Logging with %s", "attributes")

	var data map[string]interface{}
	err := json.Unmarshal(logSpy.Bytes(), &data)
	require.NoError(t, err)
	assert.Equal(t, "Logging with attributes", data["msg"])
	assert.Equal(t, "Captain", data["Title"])
	assert.Equal(t, "Iglo", data["Name"])
	assert.Equal(t, "p1", data["coordinate"].(map[string]interface{})["project"])
	assert.Equal(t, "t1", data["coordinate"].(map[string]interface{})["type"])
	assert.Equal(t, "c1", data["coordinate"].(map[string]interface{})["configId"])
	assert.Equal(t, "p1:t1:c1", data["coordinate"].(map[string]interface{})["reference"])
	assert.Equal(t, "env1", data["environment"].(map[string]interface{})["name"])
	assert.Equal(t, "group", data["environment"].(map[string]interface{})["group"])
}
