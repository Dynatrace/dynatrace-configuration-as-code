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
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
)

const (
	LogDirectory                 = ".logs"
	LogFileTimestampPrefixFormat = "20060102-150405"

	// MONACO_LOG_FORMAT is an environment variable that specifies the format use when logging.
	// When set to "json", log entries are emitted as JSON lines. The plain text default logger is used in other cases.
	envVarLogFormat = "MONACO_LOG_FORMAT"

	// MONACO_LOG_TIME is an environment variable that specifies the time format used for timestamps when logging.
	// When set to "utc", timestamps are explicitly converted to UTC first.
	envVarLogTime = "MONACO_LOG_TIME"
)

// CtxKeyCoord context key used for contextual coordinate information
type CtxKeyCoord struct{}

// CtxKeyEnv context key used for contextual environment information
type CtxKeyEnv struct{}

// CtxValEnv context value used for contextual environment information
type CtxValEnv struct {
	Name  string
	Group string
}

// CtxGraphComponentId context key used for correlating logs that belong to deployment of a sub graph
type CtxGraphComponentId struct{}

// CtxKeyAccount context key used for contextual account information
type CtxKeyAccount struct{}

// CtxValGraphComponentId context value used for correlating logs that belong to deployment of a sub graph
type CtxValGraphComponentId int

func Fatal(msg string, a ...interface{}) {
	slog.Error(fmt.Sprintf(msg, a...))
	os.Exit(1)
}

func Error(msg string, a ...interface{}) {
	slog.Error(fmt.Sprintf(msg, a...))
}

func Warn(msg string, a ...interface{}) {
	slog.Warn(fmt.Sprintf(msg, a...))
}

func Info(msg string, a ...interface{}) {
	slog.Info(fmt.Sprintf(msg, a...))
}

func Debug(msg string, a ...interface{}) {
	slog.Debug(fmt.Sprintf(msg, a...))
}

// PrepareLogging sets up the default slog.Logger using the specified options.
func PrepareLogging(ctx context.Context, fs afero.Fs, verbose bool, loggerSpy io.Writer, fileLogging bool, enableMemstatLogging bool) {
	logger := slog.New(prepareHandler(ctx, fs, verbose, loggerSpy, fileLogging, enableMemstatLogging))
	slog.SetDefault(logger)
}

func prepareHandler(ctx context.Context, fs afero.Fs, verbose bool, loggerSpy io.Writer, fileLogging bool, enableMemstatLogging bool) slog.Handler {
	handlerOptions := getHandlerOptions(getLevelFromVerbose(verbose))
	handlers := []slog.Handler{getHandler(os.Stderr, handlerOptions)}

	if loggerSpy != nil {
		handlers = append(handlers, getHandler(loggerSpy, handlerOptions))
	}

	if fileLogging && fs != nil {
		logFile, errorFile, err := PrepareLogFiles(ctx, fs, enableMemstatLogging)
		if err != nil {
			Warn("Error preparing log files: %s", err)
		}

		if logFile != nil {
			handlers = append(handlers, getHandler(logFile, handlerOptions))
		}

		if errorFile != nil {
			handlers = append(handlers, getHandler(errorFile, getHandlerOptions(slog.LevelError)))
		}
	}

	if len(handlers) == 1 {
		return handlers[0]
	}

	return NewTeeHandler(handlers...)
}

func getLevelFromVerbose(verbose bool) slog.Level {
	if verbose {
		return slog.LevelDebug
	}

	return slog.LevelInfo
}

func getHandlerOptions(level slog.Leveler) *slog.HandlerOptions {
	return &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: getReplaceAttrFunc(),
	}
}

func getHandler(w io.Writer, options *slog.HandlerOptions) slog.Handler {
	if shouldUseJSON() {
		return slog.NewJSONHandler(w, options)
	}

	return slog.NewTextHandler(w, options)
}

func getReplaceAttrFunc() func(groups []string, a slog.Attr) slog.Attr {
	if shouldUseUTC() {
		return func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				return slog.Attr{Key: slog.TimeKey, Value: slog.TimeValue(t.UTC())}
			}
			return a
		}
	}

	return nil
}

func shouldUseJSON() bool {
	v := os.Getenv(envVarLogFormat)
	return strings.ToLower(v) == "json"
}

func shouldUseUTC() bool {
	v := os.Getenv(envVarLogTime)
	return strings.ToLower(v) == "utc"
}

// LogFilePath returns the path of a logfile for the current execution time - depending on when this function is called such a file may not yet exist
func LogFilePath() string {
	timestamp := timeutils.TimeAnchor().Format(LogFileTimestampPrefixFormat)
	return filepath.Join(LogDirectory, timestamp+".log")
}

// ErrorFilePath returns the path of an error logfile for the current execution time - depending on when this function is called such a file may not yet exist
func ErrorFilePath() string {
	timestamp := timeutils.TimeAnchor().Format(LogFileTimestampPrefixFormat)
	return filepath.Join(LogDirectory, timestamp+"-errors.log")
}

// MemStatFilePath returns the full path of an memory statistics log file for the current execution time - if no stats are written (yet) no file may exist at this path.
func MemStatFilePath() string {
	timestamp := timeutils.TimeAnchor().Format(LogFileTimestampPrefixFormat)
	return filepath.Join(LogDirectory, timestamp+"-memstat.log")
}

// prepareLogFiles tries to create a LogDirectory (if none exists) and a file each to write all logs and filtered error
// logs to. As errors in preparing log files are viewed as optional for the logger setup using this method, partial data
// may be returned in case of errors.
// If log directory or logFile creation fails, no log files are returned.
// If errLog creation fails, a valid logFile is still being returned with an error.
func PrepareLogFiles(ctx context.Context, fs afero.Fs, enableMemstatLogging bool) (logFile afero.File, errFile afero.File, err error) {
	if err := fs.MkdirAll(LogDirectory, 0777); err != nil {
		return nil, nil, fmt.Errorf("unable to prepare log directory %s: %w", LogDirectory, err)
	}

	logFilePath := LogFilePath()
	logFile, err = fs.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to prepare log file %s: %w", logFilePath, err)
	}

	errFilePath := ErrorFilePath()
	errFile, err = fs.OpenFile(errFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return logFile, nil, fmt.Errorf("unable to prepare error file %s: %w", errFilePath, err)
	}

	if enableMemstatLogging {
		go func() {
			err := createMemStatFile(ctx, fs, MemStatFilePath())
			if err != nil {
				Warn("Failed to start MemStat routine: %s", err)
			}
		}()
	}

	return logFile, errFile, nil
}
