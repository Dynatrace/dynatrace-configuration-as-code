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
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"sync"
	"time"
)

var (
	_pool = buffer.NewPool()
	Get   = _pool.Get
)

func newFixedFieldsConsoleEncoder() zapcore.Encoder {
	return fixedFieldsConsoleEncoder{
		concurrentMapObjectEncoder: &concurrentMapObjectEncoder{
			mu:  sync.RWMutex{},
			moe: zapcore.NewMapObjectEncoder(),
		},
	}
}

// fixedFieldsConsoleEncoder is a custom console encoder that prints only prints the context
// fields with key "coordinate" and "componentId" (currently hard coded). Further,
// it takes care that the context is printed before that actual message and after the log level
type fixedFieldsConsoleEncoder struct {
	*concurrentMapObjectEncoder
}

func (e fixedFieldsConsoleEncoder) Clone() zapcore.Encoder {
	mapEnc := zapcore.NewMapObjectEncoder()
	for k, v := range e.Fields() {
		_ = mapEnc.AddReflected(k, v) // mapobj encoder does not return any err
	}
	cloneEncoder := fixedFieldsConsoleEncoder{
		concurrentMapObjectEncoder: &concurrentMapObjectEncoder{
			mu:  sync.RWMutex{},
			moe: mapEnc,
		},
	}
	return cloneEncoder
}

func (e fixedFieldsConsoleEncoder) EncodeEntry(entry zapcore.Entry, _ []zapcore.Field) (*buffer.Buffer, error) {
	line := Get()
	line.AppendString(entry.Time.Format(time.RFC3339))
	line.AppendString("\t")
	line.AppendString(entry.Level.String())
	line.AppendString("\t")

	additionalTab := false
	if f, ok := e.Fields()["coordinate"]; ok {
		additionalTab = true
		if logCoordinate, ook := f.(field.LogCoordinate); ook {
			line.AppendString(fmt.Sprintf("[%s=%v]", "coord", logCoordinate.Reference))
		}
	}

	if f, ok := e.Fields()["gid"]; ok {
		additionalTab = true
		line.AppendString(fmt.Sprintf("[%s=%v]", "gid", f))
	}

	if f, ok := e.Fields()["account"]; ok {
		additionalTab = true
		line.AppendString(fmt.Sprintf("[%s=%v]", "account", f))
	}

	if additionalTab {
		line.AppendString("\t")
	}
	line.AppendString(entry.Message)
	line.AppendString("\n")
	return line, nil
}
