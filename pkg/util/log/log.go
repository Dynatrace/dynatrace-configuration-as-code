/**
 * @license
 * Copyright 2020 Dynatrace LLC
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
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	prefixFatal = "FATAL "
	prefixError = "ERROR "
	prefixWarn  = "WARN  "
	prefixInfo  = "INFO  "
	prefixDebug = "DEBUG "
)

type logLevel int

const (
	LevelFatal logLevel = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
)

func (l logLevel) prefix() string {
	switch l {
	case LevelFatal:
		return prefixFatal
	case LevelError:
		return prefixError
	case LevelWarn:
		return prefixWarn
	case LevelInfo:
		return prefixInfo
	case LevelDebug:
		return prefixDebug
	}
	return ""
}

type extendedLogger struct {
	consoleLogger    *log.Logger
	fileLogger       *log.Logger
	additionalLogger *log.Logger
	level            logLevel
}

// New creates a new extendedLogger, which contains two
// underlying loggers, a console logger and a file logger.
// The file logger will always print all logs, while the
// console logger will only print logs according to the
// level of this logger, e.g. at LevelInfo.
func New(consoleLogger, fileLogger *log.Logger, level logLevel) *extendedLogger {
	return &extendedLogger{
		consoleLogger: consoleLogger,
		fileLogger:    fileLogger,
		level:         level,
	}
}

// Fatal logs the message with the prefix FATAL.
func (l *extendedLogger) Fatal(msg string, a ...interface{}) {
	doLog(l, LevelFatal, msg, a...)
}

// Error logs the message with the prefix ERROR.
func (l *extendedLogger) Error(msg string, a ...interface{}) {
	doLog(l, LevelError, msg, a...)
}

// Warn logs the message with the prefix WARN.
func (l *extendedLogger) Warn(msg string, a ...interface{}) {
	doLog(l, LevelWarn, msg, a...)
}

// Info logs the message with the prefix INFO.
func (l *extendedLogger) Info(msg string, a ...interface{}) {
	doLog(l, LevelInfo, msg, a...)
}

// Debug logs the message with the prefix DEBUG.
func (l *extendedLogger) Debug(msg string, a ...interface{}) {
	doLog(l, LevelDebug, msg, a...)
}

// SetLevel sets the log level for this logger. This
// influences only the console logger, the file logger
// will always have the highest level regardless.
func (l *extendedLogger) SetLevel(level logLevel) {
	l.level = level
}

var defaultLogger = &extendedLogger{
	consoleLogger: log.Default(),
	level:         LevelInfo,
}

// Default returns the default logger which is used, when
// functions like Info() and Debug() are called. The
// console logger will be the same default logger from
// the standard library, the level will be INFO and
// there's no file logger set up. The function
// SetupLogging() can be used to create a file logger.
func Default() *extendedLogger {
	return defaultLogger
}

// SetupLogging is used to enable file logging, including
// Request and Response logs. If logging functions are
// called without setup, logging will only be done to
// stdout. Otherwise, the file logger will be set to a log
// file whose name will be the current timestamp, in the
// format of <YYYYMMDD-hhmmss>.log
func SetupLogging(optionalAddedLogger *log.Logger) error {
	if err := setupFileLogging(); err != nil {
		return nil
	}
	if err := setupRequestLog(); err != nil {
		return nil
	}
	defaultLogger.additionalLogger = optionalAddedLogger
	return setupResponseLog()
}

func setupFileLogging() error {
	timestamp := time.Now().Format("20060102-150405")
	if err := createDirIfNotExists(".logs"); err != nil {
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(".logs", timestamp+".log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("unable to create file logger: %v", err)
	}
	defaultLogger.fileLogger = log.New(logFile, "", log.LstdFlags)
	return nil
}

// Fatal logs the message with the prefix FATAL to the
// default logger (see Default)).
func Fatal(msg string, a ...interface{}) {
	defaultLogger.Fatal(msg, a...)
}

// Error logs the message with the prefix ERROR to the
// default logger (see Default)).
func Error(msg string, a ...interface{}) {
	defaultLogger.Error(msg, a...)
}

// Warn logs the message with the prefix WARN to the
// default logger (see Default()).
func Warn(msg string, a ...interface{}) {
	defaultLogger.Warn(msg, a...)
}

// Info logs the message with the prefix INFO to the
// default logger (see Default()).
func Info(msg string, a ...interface{}) {
	defaultLogger.Info(msg, a...)
}

// Debug logs the message with the prefix DEBUG to the
// default logger (see Default()).
func Debug(msg string, a ...interface{}) {
	defaultLogger.Debug(msg, a...)
}

func doLog(logger *extendedLogger, level logLevel, msg string, a ...interface{}) {
	msg = fmt.Sprintf(level.prefix()+msg, a...)
	if logger.level >= level && logger.consoleLogger != nil {
		logger.consoleLogger.Println(msg)
	}
	if logger.fileLogger != nil {
		logger.fileLogger.Println(msg)
	}
	if logger.additionalLogger != nil {
		logger.additionalLogger.Println(msg)
	}
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, 0777)
	}
	return nil
}
