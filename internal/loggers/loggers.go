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

package loggers

import (
	"github.com/spf13/afero"
	"io"
	"strings"
)

// Logger defines the interface for a logging implementation
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
}

const EnvVarLogFormat = "MONACO_LOG_FORMAT"
const EnvVarLogTime = "MONACO_LOG_TIME"

// LogOptions holds different options that can be passed to setup the logger to change its behavior
type LogOptions struct {
	// FileLogging specifies whether logs should be additionally written to a file
	// FileLoggingJSON specifies whether log lines to files should be JSON formatted
	FileLoggingJSON bool
	// ConsoleLoggingJSON specifies whether log lines written to the console should be JSON formatted
	ConsoleLoggingJSON bool
	// LogLevel specifies the log level to be used
	LogLevel LogLevel
	// LogSpy can be used as an additional log sink to capture the logs
	LogSpy io.Writer
	// LogTimeMode specifies which time mode shall be used when printing out logs
	LogTimeMode LogTimeMode

	File afero.File
}

type LogFormat int

const (
	LogFormatText LogFormat = iota
	LogFormatJSON
)

func ParseLogFormat(f string) LogFormat {
	if strings.ToLower(f) == "json" {
		return LogFormatJSON
	}
	return LogFormatText
}

type LogTimeMode int

const (
	LogTimeLocal LogTimeMode = iota
	LogTimeUTC
)

func ParseLogTimeMode(m string) LogTimeMode {
	if strings.ToLower(m) == "utc" {
		return LogTimeUTC
	}
	return LogTimeLocal
}

type LogLevel int

const (
	LevelInfo LogLevel = iota
	LevelDebug
	LevelError
	LevelWarn
	LevelFatal
)
