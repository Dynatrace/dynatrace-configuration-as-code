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
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
)

// TestTeeHandler_Enabled tests that enabled is reported correctly for a TeeHandler constructed with two handlers.
func TestTeeHandler_Enabled(t *testing.T) {
	t.Run("all enabled if both have level debug", func(t *testing.T) {
		testingHandler := log.NewTeeHandler(
			NewTestHandler(&slog.HandlerOptions{Level: slog.LevelDebug}),
			NewTestHandler(&slog.HandlerOptions{Level: slog.LevelDebug}),
		)

		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelDebug))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelInfo))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelWarn))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelError))
	})

	t.Run("all enabled if one has level debug, other info", func(t *testing.T) {
		testingHandler := log.NewTeeHandler(
			NewTestHandler(&slog.HandlerOptions{Level: slog.LevelDebug}),
			NewTestHandler(&slog.HandlerOptions{Level: slog.LevelInfo}),
		)

		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelDebug))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelInfo))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelWarn))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelError))
	})

	t.Run("all enabled if both have level info", func(t *testing.T) {
		testingHandler := log.NewTeeHandler(
			NewTestHandler(&slog.HandlerOptions{Level: slog.LevelInfo}),
			NewTestHandler(&slog.HandlerOptions{Level: slog.LevelInfo}),
		)

		assert.False(t, testingHandler.Enabled(context.TODO(), slog.LevelDebug))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelInfo))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelWarn))
		assert.True(t, testingHandler.Enabled(context.TODO(), slog.LevelError))
	})
}

// TestTeeHandler_Handle tests records are handled correctly.
func TestTeeHandler_Handle(t *testing.T) {
	t.Run("without attributes", func(t *testing.T) {
		handler1 := NewTestHandler(&slog.HandlerOptions{})
		handler2 := NewTestHandler(&slog.HandlerOptions{})

		testingHandler := log.NewTeeHandler(handler1, handler2)

		err := testingHandler.Handle(nil, slog.Record{
			Level:   slog.LevelWarn,
			Message: "test",
		})
		require.NoError(t, err)

		assert.Contains(t, handler1.Output.String(), "level=WARN msg=test")
		assert.Contains(t, handler2.Output.String(), "level=WARN msg=test")
	})

	t.Run("with attributes", func(t *testing.T) {
		handler1 := NewTestHandler(&slog.HandlerOptions{})
		handler2 := NewTestHandler(&slog.HandlerOptions{})

		testingHandler := log.NewTeeHandler(handler1, handler2)

		r := slog.Record{
			Level:   slog.LevelWarn,
			Message: "test",
		}
		r.AddAttrs(slog.String("key", "value"))
		err := testingHandler.Handle(nil, r)
		require.NoError(t, err)

		assert.Contains(t, handler1.Output.String(), "level=WARN msg=test key=value")
		assert.Contains(t, handler2.Output.String(), "level=WARN msg=test key=value")
	})

}

// TestTeeHandler_WithAttrs tests that a TeeHandler is returned that applies the given attributes to each handled record.
func TestTeeHandler_WithAttrs(t *testing.T) {
	handler1 := NewTestHandler(&slog.HandlerOptions{})
	handler2 := NewTestHandler(&slog.HandlerOptions{})

	testingHandler := log.NewTeeHandler(handler1, handler2).WithAttrs([]slog.Attr{slog.String("key", "value")})

	err := testingHandler.Handle(nil, slog.Record{
		Level:   slog.LevelWarn,
		Message: "test",
	})
	require.NoError(t, err)

	assert.Contains(t, handler1.Output.String(), "level=WARN msg=test key=value")
	assert.Contains(t, handler2.Output.String(), "level=WARN msg=test key=value")
}

// TestTeeHandler_WithGroup tests that a TeeHandler is returned which applies the given group to handled records.
func TestTeeHandler_WithGroup(t *testing.T) {
	handler1 := NewTestHandler(&slog.HandlerOptions{})
	handler2 := NewTestHandler(&slog.HandlerOptions{})

	testingHandler := log.NewTeeHandler(handler1, handler2).WithGroup("group1").WithAttrs([]slog.Attr{slog.String("key", "value")})

	err := testingHandler.Handle(nil, slog.Record{
		Level:   slog.LevelWarn,
		Message: "test",
	})
	require.NoError(t, err)

	assert.Contains(t, handler1.Output.String(), "level=WARN msg=test group1.key=value")
	assert.Contains(t, handler2.Output.String(), "level=WARN msg=test group1.key=value")
}

// testHandler is a TextHandler that writes to a strings.Builder for easy testing.
type testHandler struct {
	*slog.TextHandler
	Output *strings.Builder
}

func NewTestHandler(options *slog.HandlerOptions) *testHandler {
	output := &strings.Builder{}
	return &testHandler{
		TextHandler: slog.NewTextHandler(output, options),
		Output:      output,
	}
}
