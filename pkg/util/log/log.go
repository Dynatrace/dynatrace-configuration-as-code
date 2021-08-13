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
	prefixInfo  = "INFO  "
	prefixWarn  = "WARN  "
	prefixError = "ERROR "
	prefixFatal = "FATAL "
	prefixDebug = "DEBUG "
)

var (
	// Verbose controls if logs at DEBUG level are printed to stdout.
	// Default: false
	Verbose bool = false

	consoleLogger *log.Logger = log.Default()
	fileLogger    *log.Logger
)

// SetupLogging is used to enable file logging, including
// Request and Response logs. If logging functions are
// called without setup, logging will only be done to
// stdout. Otherwise, the file logger will be set to a log
// file whose name will be the current timestamp, in the
// format of <YYYYMMDD-hhmmss>.log
func SetupLogging() error {
	if err := setupFileLogging(); err != nil {
		return nil
	}
	if err := setupRequestLog(); err != nil {
		return nil
	}
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
	fileLogger = log.New(logFile, "", log.LstdFlags)
	return nil
}

// Info logs the message with the prefix INFO.
func Info(msg string, a ...interface{}) {
	doLog(msg, prefixInfo, a...)
}

// Warn logs the message with the prefix WARN.
func Warn(msg string, a ...interface{}) {
	doLog(msg, prefixWarn, a...)
}

// Error logs the message with the prefix ERROR.
func Error(msg string, a ...interface{}) {
	doLog(msg, prefixError, a...)
}

// Fatal logs the message with the prefix FATAL.
func Fatal(msg string, a ...interface{}) {
	doLog(msg, prefixFatal, a...)
}

// Debug logs the message with the prefix DEBUG.
// If verbose is set to true, this behaves like any other
// logging level, otherwise only the file logger will be
// used and no prints to the default logger will be done.
func Debug(msg string, a ...interface{}) {
	if Verbose {
		doLog(msg, prefixDebug, a...)
		return
	}
	if fileLogger != nil {
		fileLogger.Printf(prefixDebug+msg, a...)
	}
}

func doLog(msg, prefix string, a ...interface{}) {
	msg = fmt.Sprintf(prefix+msg, a...)
	consoleLogger.Println(msg)
	if fileLogger != nil {
		fileLogger.Println(msg)
	}
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, 0777)
	}
	return nil
}
