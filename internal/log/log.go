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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/loggers/console"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/loggers/zap"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/timeutils"
	"github.com/spf13/afero"
	"io"
	"os"
	"path/filepath"
)

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
