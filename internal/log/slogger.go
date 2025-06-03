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
package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// Slogger is a simple logger that wraps an `slog.Logger`.
type Slogger struct {
	Logger *slog.Logger
}

func (w *Slogger) Fatal(msg string, a ...any) {
	w.Logger.Error(fmt.Sprintf(msg, a...))
	os.Exit(1)
}

func (w *Slogger) FatalContext(ctx context.Context, msg string, a ...any) {
	w.Logger.ErrorContext(ctx, fmt.Sprintf(msg, a...))
	os.Exit(1)
}

func (w *Slogger) Error(msg string, a ...interface{}) {
	w.Logger.Error(fmt.Sprintf(msg, a...))
}

func (w *Slogger) ErrorContext(ctx context.Context, msg string, a ...interface{}) {
	w.Logger.ErrorContext(ctx, fmt.Sprintf(msg, a...))
}

func (w *Slogger) Warn(msg string, a ...interface{}) {
	w.Logger.Warn(fmt.Sprintf(msg, a...))
}

func (w *Slogger) WarnContext(ctx context.Context, msg string, a ...interface{}) {
	w.Logger.WarnContext(ctx, fmt.Sprintf(msg, a...))
}

func (w *Slogger) Info(msg string, a ...interface{}) {
	w.Logger.Info(fmt.Sprintf(msg, a...))
}

func (w *Slogger) InfoContext(ctx context.Context, msg string, a ...interface{}) {
	w.Logger.InfoContext(ctx, fmt.Sprintf(msg, a...))
}

func (w *Slogger) Debug(msg string, a ...interface{}) {
	w.Logger.Debug(fmt.Sprintf(msg, a...))
}

func (w *Slogger) DebugContext(ctx context.Context, msg string, a ...interface{}) {
	w.Logger.DebugContext(ctx, fmt.Sprintf(msg, a...))
}

func (w *Slogger) SLogger() *slog.Logger {
	return w.Logger
}

// WithFields adds additional [field.Field] for structured logs
// It accepts vararg fields and should not be called more than once per log call
func (w *Slogger) WithFields(fields ...field.Field) *Slogger {
	logger := w.Logger
	for _, f := range fields {
		logger = logger.With(f.Key, f.Value)
	}
	return &Slogger{Logger: logger}
}

// WithFields adds additional [field.Field] for structured logs
// It accepts vararg fields and should not be called more than once per log call
func WithFields(fields ...field.Field) *Slogger {
	return (&Slogger{Logger: slog.Default()}).WithFields(fields...)
}

// WithCtxFields creates a logger instance with preset structured logging [field.Field] based on the Context
// Coordinate (via [CtxKeyCoord]) and environment (via [CtxKeyEnv] [CtxValEnv]) information is added to logs from the Context
func WithCtxFields(ctx context.Context) *Slogger {
	f := make([]field.Field, 0, 2)
	if c, ok := ctx.Value(CtxKeyCoord{}).(coordinate.Coordinate); ok {
		f = append(f, field.Coordinate(c))
	}
	if e, ok := ctx.Value(CtxKeyEnv{}).(CtxValEnv); ok {
		f = append(f, field.Environment(e.Name, e.Group))
	}
	if a, ok := ctx.Value(CtxKeyAccount{}).(string); ok {
		f = append(f, field.F("account", a))
	}
	if c, ok := ctx.Value(CtxGraphComponentId{}).(CtxValGraphComponentId); ok {
		f = append(f, field.F("gid", c))
	}
	return WithFields(f...)
}
