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
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/afero"
)

const envVarLogFormat = "MONACO_LOG_FORMAT"
const envVarLogTime = "MONACO_LOG_TIME"
const envVarLogSource = "MONACO_LOG_SOURCE"
const envVarLogColor = "MONACO_LOG_COLOR"

func PrepareLogging(ctx context.Context, fs afero.Fs, verbose bool, loggerSpy io.Writer, fileLogging bool) {
	var logFile afero.File
	var errorFile afero.File

	if fileLogging && fs != nil {
		lf, ef, err := prepareLogFiles(ctx, fs)
		if err != nil {
			Warn("Error preparing log files: %s", err.Error())
		}
		logFile = lf
		errorFile = ef
	}

	handlers := []slog.Handler{}
	if logFile != nil {
		handlers = append(handlers, getHandler(logFile, verbose, false))
	}

	if errorFile != nil {
		handlers = append(handlers, getHandler(errorFile, verbose, false))
	}

	if loggerSpy != nil {
		handlers = append(handlers, getHandler(loggerSpy, verbose, false))
	}

	otelHandler := initOpenTelemetryHandler()
	if otelHandler != nil {
		handlers = append(handlers, otelHandler)
	}

	consoleHandler := getHandler(os.Stderr, verbose, shouldAddColor())
	handlers = append(handlers, consoleHandler)

	var handler slog.Handler = NewTeeHandler(handlers...)
	if len(handlers) == 1 {
		handler = handlers[0]
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func getLevelFromVerbose(verbose bool) slog.Level {
	if verbose {
		return slog.LevelDebug
	}

	return slog.LevelInfo
}

func getHandler(w io.Writer, verbose bool, color bool) slog.Handler {
	if shouldUseJSON() {
		return slog.NewJSONHandler(w, &slog.HandlerOptions{
			AddSource:   shouldAddSource(),
			Level:       getLevelFromVerbose(verbose),
			ReplaceAttr: getReplaceAttrFunc(),
		})
	}

	if color {
		return tint.NewHandler(w, &tint.Options{
			AddSource:   shouldAddSource(),
			Level:       getLevelFromVerbose(verbose),
			ReplaceAttr: getReplaceAttrFunc(),
		})
	}

	return slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource:   shouldAddSource(),
		Level:       getLevelFromVerbose(verbose),
		ReplaceAttr: getReplaceAttrFunc(),
	})
}

func getReplaceAttrFunc() func(groups []string, a slog.Attr) slog.Attr {
	useUTC := shouldUseUTC()
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && useUTC {
			t := a.Value.Time()
			t = t.UTC()
			return slog.Attr{Key: slog.TimeKey, Value: slog.StringValue(t.Format(time.RFC3339))}
		}

		return a
	}
}

func shouldUseJSON() bool {
	v := os.Getenv(envVarLogFormat)
	return strings.ToLower(v) == "json"
}

func shouldUseUTC() bool {
	v := os.Getenv(envVarLogTime)
	return strings.ToLower(v) == "utc"
}

func shouldAddSource() bool {
	return getFeatureFlagValue(envVarLogSource, false)
}

func shouldAddColor() bool {
	return getFeatureFlagValue(envVarLogColor, false)
}

func getFeatureFlagValue(envName string, d bool) bool {
	if val, ok := os.LookupEnv(envName); ok {
		value, err := strconv.ParseBool(strings.ToLower(val))
		if err != nil {
			return d
		}
		return value
	}
	return d
}
