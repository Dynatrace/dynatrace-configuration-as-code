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

package console

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"log"
)

var Instance = Logger{consoleLogger: log.Default()}

// Logger is a wrapper around the go default logger that just logs to the console
// Note, that this type of logger does not provide any special features but just forwards
// the log messages to the builtin log package of GO. It is not used in production code
type Logger struct {
	consoleLogger *log.Logger
}

func (l Logger) WithFields(fields ...field.Field) loggers.Logger {
	return l
}

func (l Logger) Info(msg string, args ...interface{}) {
	l.consoleLogger.Printf("INFO "+msg+"\n", args...)
}

func (l Logger) Error(msg string, args ...interface{}) {
	l.consoleLogger.Printf("ERROR "+msg+"\n", args...)
}

func (l Logger) Debug(msg string, args ...interface{}) {
	l.consoleLogger.Printf("DEBUG "+msg+"\n", args...)
}

func (l Logger) Warn(msg string, args ...interface{}) {
	l.consoleLogger.Printf("WARN "+msg+"\n", args...)
}

func (l Logger) Fatal(msg string, args ...interface{}) {
	l.consoleLogger.Fatalf("FATAL "+msg+"\n", args...)
}

func (l Logger) Level() loggers.LogLevel {
	return loggers.LevelDebug
}
