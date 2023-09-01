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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

// Logger wraps a zap baseLogger to perform logging
type Logger struct {
	logLevel   loggers.LogLevel
	baseLogger *zap.Logger
}

func (l *Logger) WithFields(fields ...field.Field) loggers.Logger {
	zFields := make([]zapcore.Field, 0, len(fields))
	for _, f := range fields {
		zFields = append(zFields, zap.Reflect(f.Key, f.Value))
	}

	return &Logger{
		baseLogger: l.baseLogger.With(zFields...),
		logLevel:   l.logLevel,
	}
}

// Info logs an info-level message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.baseLogger.Info(fmt.Sprintf(msg, args...))
}

// Error logs an error-level message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.baseLogger.Error(fmt.Sprintf(msg, args...))
}

// Debug logs a debug-level message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.baseLogger.Debug(fmt.Sprintf(msg, args...))
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	l.baseLogger.Warn(fmt.Sprintf(msg, args...))
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.baseLogger.Fatal(fmt.Sprintf(msg, args...))
}

func (l *Logger) Level() loggers.LogLevel {
	return l.logLevel
}

func (l Logger) GetLogr() logr.Logger {
	return zapr.NewLogger(l.baseLogger)
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
	logLevel := zap.NewAtomicLevelAt(levelMap[logOptions.LogLevel])

	var cores []zapcore.Core
	if logOptions.ConsoleLoggingJSON {
		consoleSyncer := zapcore.Lock(os.Stdout)
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), consoleSyncer, logLevel))
	} else {
		consoleSyncer := zapcore.Lock(os.Stderr)
		cores = append(cores, zapcore.NewCore(newFixedFieldsConsoleEncoder(), consoleSyncer, logLevel))
	}

	if logOptions.File != nil {
		debugLevel := zap.NewAtomicLevelAt(zapcore.DebugLevel) // always debug log to files
		fileSyncer := zapcore.Lock(zapcore.AddSync(logOptions.File))
		if logOptions.FileLoggingJSON {
			cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), fileSyncer, debugLevel))
		} else {
			cores = append(cores, zapcore.NewCore(newFixedFieldsConsoleEncoder(), fileSyncer, debugLevel))
		}
	}

	if logOptions.LogSpy != nil {
		spySyncer := zapcore.Lock(zapcore.AddSync(logOptions.LogSpy))
		if logOptions.ConsoleLoggingJSON {
			cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), spySyncer, logLevel))
		} else {
			cores = append(cores, zapcore.NewCore(newFixedFieldsConsoleEncoder(), spySyncer, logLevel))
		}

	}

	logger := zap.New(zapcore.NewTee(cores...))
	return &Logger{baseLogger: logger, logLevel: logOptions.LogLevel}, nil
}

var levelMap = map[loggers.LogLevel]zapcore.Level{
	loggers.LevelDebug: zapcore.DebugLevel,
	loggers.LevelInfo:  zapcore.InfoLevel,
	loggers.LevelWarn:  zapcore.WarnLevel,
	loggers.LevelError: zapcore.ErrorLevel,
	loggers.LevelFatal: zapcore.FatalLevel,
}
