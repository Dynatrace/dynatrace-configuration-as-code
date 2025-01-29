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

	"github.com/spf13/afero"
)

const envVarLogFormat = "MONACO_LOG_FORMAT"
const envVarLogTime = "MONACO_LOG_TIME"
const envVarLogSource = "MONACO_LOG_SOURCE"
const envVarLogColor = "MONACO_LOG_COLOR"

//const envVarLogToFile = "MONACO_LOG_FILE_ENABLED"

var logFile afero.File
var errorFile afero.File

func PrepareLogging(ctx context.Context, fs afero.Fs, verbose bool, loggerSpy io.Writer, fileLogging bool) {
	if fileLogging && fs != nil {
		if logFile == nil {
			lf, ef, err := prepareLogFiles(ctx, fs)
			if err != nil {
				Warn("Error preparing log files: %s", err.Error())
			}
			logFile = lf
			errorFile = ef
		}
	}

	handlers := []slog.Handler{}
	if logFile != nil {
		handlers = append(handlers, getHandler(logFile, &slog.HandlerOptions{
			AddSource:   shouldAddSource(),
			Level:       getLevelFromVerbose(verbose),
			ReplaceAttr: getReplaceAttrFunc(),
		}))
	}

	if errorFile != nil {
		handlers = append(handlers, getHandler(errorFile, &slog.HandlerOptions{
			Level:       slog.LevelError,
			ReplaceAttr: getReplaceAttrFunc(),
		}))
	}

	if loggerSpy != nil {
		handlers = append(handlers, getHandler(loggerSpy, &slog.HandlerOptions{
			AddSource:   shouldAddSource(),
			Level:       getLevelFromVerbose(verbose),
			ReplaceAttr: getReplaceAttrFunc(),
		}))
	}

	otelHandler := initOpenTelemetryHandler()
	if otelHandler != nil {
		handlers = append(handlers, otelHandler)
	}

	consoleHandler := getHandler(os.Stderr, &slog.HandlerOptions{
		AddSource:   shouldAddSource(),
		Level:       getLevelFromVerbose(verbose),
		ReplaceAttr: getReplaceAttrFunc(),
	})

	if shouldAddColor() {
		consoleHandler = NewColorHandler(consoleHandler)
	}

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

func getHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if shouldUseJSON() {
		return slog.NewJSONHandler(w, opts)
	}

	return slog.NewTextHandler(w, opts)
}

func getReplaceAttrFunc() func(groups []string, a slog.Attr) slog.Attr {
	useUTC := shouldUseUTC()
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key != slog.TimeKey {
			return a
		}

		t := a.Value.Time()
		if useUTC {
			t = t.UTC()
		}

		a.Value = slog.StringValue(t.Format(time.RFC3339))
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
