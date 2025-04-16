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
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers/console"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers/zap"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

const (
	LogDirectory                 = ".logs"
	LogFileTimestampPrefixFormat = "20060102-150405"
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

var (
	_ loggers.Logger = (*zap.Logger)(nil)
	_ loggers.Logger = (*console.Logger)(nil)
)

func Fatal(msg string, a ...interface{}) {
	std.Fatal(msg, a...)
}

func Error(msg string, a ...interface{}) {
	std.Error(msg, a...)
}

func Warn(msg string, a ...interface{}) {
	std.Warn(msg, a...)
}

func Info(msg string, a ...interface{}) {
	std.Info(msg, a...)
}

func Debug(msg string, a ...interface{}) {
	std.Debug(msg, a...)
}

func Level() loggers.LogLevel {
	return std.Level()
}

// WithFields adds additional [field.Field] for structured logs
// It accepts vararg fields and should not be called more than once per log call
func WithFields(fields ...field.Field) loggers.Logger {
	return std.WithFields(fields...)
}

// WithCtxFields creates a logger instance with preset structured logging [field.Field] based on the Context
// Coordinate (via [CtxKeyCoord]) and environment (via [CtxKeyEnv] [CtxValEnv]) information is added to logs from the Context
func WithCtxFields(ctx context.Context) loggers.Logger {
	loggr := std
	f := make([]field.Field, 0, 2)
	if c, ok := ctx.Value(CtxKeyCoord{}).(coordinate.Coordinate); ok {
		f = append(f, field.Coordinate(c))
	}
	if e, ok := ctx.Value(CtxKeyEnv{}).(CtxValEnv); ok {
		f = append(f, field.Environment(e.Name, e.Group))
	}

	if a := ctx.Value(CtxKeyAccount{}); a != nil {
		f = append(f, field.F("account", a))
	}

	if c, ok := ctx.Value(CtxGraphComponentId{}).(CtxValGraphComponentId); ok {
		f = append(f, field.F("gid", c))
	}
	return loggr.WithFields(f...)
}

var (
	std loggers.Logger = console.Instance
)

func PrepareLogging(ctx context.Context, fs afero.Fs, verbose bool, loggerSpy io.Writer, fileLogging bool, enableMemstatLogging bool) {
	loglevel := loggers.LevelInfo
	if verbose {
		loglevel = loggers.LevelDebug
	}

	var logFile, errFile afero.File
	var err error
	if fileLogging && fs != nil {
		logFile, errFile, err = prepareLogFiles(ctx, fs, enableMemstatLogging)
	}

	logFormat := loggers.ParseLogFormat(os.Getenv(loggers.EnvVarLogFormat))
	logTime := loggers.ParseLogTimeMode(os.Getenv(loggers.EnvVarLogTime))

	setDefaultLogger(loggers.LogOptions{
		File:        logFile,
		ErrorFile:   errFile,
		JSONLogging: logFormat == loggers.LogFormatJSON,
		LogLevel:    loglevel,
		LogSpy:      loggerSpy,
		LogTimeMode: logTime,
	})

	if err != nil {
		Warn(err.Error())
	}
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
func prepareLogFiles(ctx context.Context, fs afero.Fs, enableMemstatLogging bool) (logFile afero.File, errFile afero.File, err error) {
	if err := fs.MkdirAll(LogDirectory, 0777); err != nil {
		return nil, nil, fmt.Errorf("unable to prepare log directory %s: %w", LogDirectory, err)
	}

	logFilePath := LogFilePath()
	logFile, err = fs.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to prepare log file in %s directory: %w", LogDirectory, err)
	}

	errFilePath := ErrorFilePath()
	errFile, err = fs.OpenFile(errFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return logFile, nil, fmt.Errorf("unable to prepare error file in %s directory: %w", LogDirectory, err)
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

func setDefaultLogger(opts loggers.LogOptions) {
	logger, err := zap.New(opts)
	if err != nil {
		panic(err)
	}
	std = logger
}
