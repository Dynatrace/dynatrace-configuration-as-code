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

// With adds additional [attribute.Attr] for structured logs
// It accepts vararg attributes and should not be called more than once per log call
func (w *Slogger) With(attributes ...any) *Slogger {
	return &Slogger{Logger: w.Logger.With(attributes...)}
}

// With adds additional [attribute.Attr] for structured logs
// It accepts vararg attributes and should not be called more than once per log call
func With(attributes ...any) *Slogger {
	return (&Slogger{Logger: slog.Default()}).With(attributes...)
}
