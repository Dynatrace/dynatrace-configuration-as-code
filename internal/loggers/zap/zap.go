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

package zap

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/loggers"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

// Logger wraps a zap logger to perform logging
type Logger struct {
	logLevel loggers.LogLevel
	logger   *zap.Logger
}

// Info logs an info-level message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(msg, args...))
}

// Error logs an error-level message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(msg, args...))
}

// Debug logs a debug-level message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(msg, args...))
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(msg, args...))
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(msg, args...))
}

func (l *Logger) Level() loggers.LogLevel {
	return l.logLevel
}

func customTimeEncoder(mode loggers.LogTimeMode) func(time.Time, zapcore.PrimitiveArrayEncoder) {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		layout := time.RFC3339
		if mode == loggers.LogTimeUTC {
			enc.AppendString(t.UTC().Format(layout))
		} else {
			enc.AppendString(t.Format(layout))
		}
	}
}

func New(logOptions loggers.LogOptions) (*Logger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = customTimeEncoder(logOptions.LogTimeMode)

	var consoleEncoder zapcore.Encoder
	if logOptions.ConsoleLoggingJSON {
		consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		consoleEncoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	consoleSyncer := zapcore.Lock(os.Stderr)

	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(levelMap[logOptions.LogLevel])

	var cores []zapcore.Core

	cores = append(cores, zapcore.NewCore(consoleEncoder, consoleSyncer, atomicLevel))

	if logOptions.File != nil {
		var fileEncoder zapcore.Encoder
		if logOptions.FileLoggingJSON {
			fileEncoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			fileEncoder = zapcore.NewConsoleEncoder(encoderConfig)
		}

		fileSyncer := zapcore.AddSync(logOptions.File)
		cores = append(cores, zapcore.NewCore(fileEncoder, fileSyncer, atomicLevel))
	}

	if logOptions.LogSpy != nil {
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(logOptions.LogSpy), atomicLevel))
	}
	logger := zap.New(zapcore.NewTee(cores...))
	return &Logger{logger: logger, logLevel: logOptions.LogLevel}, nil
}

var levelMap = map[loggers.LogLevel]zapcore.Level{
	loggers.LevelDebug: zapcore.DebugLevel,
	loggers.LevelInfo:  zapcore.InfoLevel,
	loggers.LevelWarn:  zapcore.WarnLevel,
	loggers.LevelError: zapcore.ErrorLevel,
	loggers.LevelFatal: zapcore.FatalLevel,
}
