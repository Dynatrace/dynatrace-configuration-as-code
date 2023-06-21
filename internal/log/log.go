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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers/console"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers/zap"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/spf13/afero"
	"golang.org/x/net/context"
	"io"
	"os"
	"path/filepath"
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
	return loggr.WithFields(f...)
}

var (
	std loggers.Logger = console.Instance
)

func PrepareLogging(fs afero.Fs, verbose *bool, loggerSpy io.Writer) {
	loglevel := loggers.LevelInfo
	if *verbose {
		loglevel = loggers.LevelDebug
	}

	logFile, err := prepareLogFile(fs)
	logFormat := loggers.ParseLogFormat(os.Getenv(loggers.EnvVarLogFormat))
	logTime := loggers.ParseLogTimeMode(os.Getenv(loggers.EnvVarLogTime))

	setDefaultLogger(loggers.LogOptions{
		File:               logFile,
		FileLoggingJSON:    logFormat == loggers.LogFormatJSON,
		ConsoleLoggingJSON: logFormat == loggers.LogFormatJSON,
		LogLevel:           loglevel,
		LogSpy:             loggerSpy,
		LogTimeMode:        logTime,
	})

	if err != nil {
		Warn(err.Error())
	}
}

func prepareLogFile(fs afero.Fs) (afero.File, error) {
	logDir := ".logs"
	timestamp := timeutils.TimeAnchor().Format("20060102-150405")
	if err := fs.MkdirAll(logDir, 0777); err != nil {
		return nil, fmt.Errorf("unable to prepare log directory %s: %w", logDir, err)

	}
	logFilePath := filepath.Join(logDir, timestamp+".log")
	logFile, err := fs.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare log file in %s directory: %w", logDir, err)
	}
	return logFile, nil

}

func setDefaultLogger(opts loggers.LogOptions) {
	logger, err := zap.New(opts)
	if err != nil {
		panic(err)
	}
	std = logger
}
