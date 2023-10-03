//go:build unit

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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"sync"
	"testing"
	"time"
)

func TestClone(t *testing.T) {

	encoder := fixedFieldsConsoleEncoder{
		concurrentMapObjectEncoder: &concurrentMapObjectEncoder{
			mu:  sync.RWMutex{},
			moe: zapcore.NewMapObjectEncoder(),
		},
	}
	encoder.moe.Fields = map[string]interface{}{"foo": "bar"}

	clone := encoder.Clone()
	assert.IsType(t, encoder, clone, "Clone should return a fixedFieldsConsoleEncoder")
	assert.Len(t, encoder.moe.Fields, 1)
	assert.Len(t, clone.(fixedFieldsConsoleEncoder).moe.Fields, 1)
}

func TestEncodeEntry_IgnoresFieldsGivenViaArgs(t *testing.T) {
	encoder := fixedFieldsConsoleEncoder{
		concurrentMapObjectEncoder: &concurrentMapObjectEncoder{
			mu:  sync.RWMutex{},
			moe: zapcore.NewMapObjectEncoder(),
		},
	}
	entry := zapcore.Entry{
		Time:    time.Date(2023, 7, 27, 12, 34, 56, 0, time.UTC),
		Level:   zapcore.InfoLevel,
		Message: "Test log message",
	}

	fields := []zapcore.Field{
		{Key: "coordinate", Type: zapcore.StringType, String: "coordinate"},
		{Key: "componentId", Type: zapcore.StringType, String: "12345"},
	}

	buffer, err := encoder.EncodeEntry(entry, fields)
	assert.NoError(t, err, "Error encoding entry")

	expectedOutput := "2023-07-27T12:34:56Z\tinfo\tTest log message\n"
	assert.Equal(t, expectedOutput, buffer.String(), "Unexpected encoded output")
}

func TestEncodeEntry_UsesFieldsFromObjectEncoder(t *testing.T) {
	objEnc := zapcore.NewMapObjectEncoder()
	objEnc.Fields = map[string]interface{}{"ignored": ":(", "coordinate": field.LogCoordinate{Reference: "a:b:c"}, "gid": 4}
	encoder := fixedFieldsConsoleEncoder{
		concurrentMapObjectEncoder: &concurrentMapObjectEncoder{
			mu:  sync.RWMutex{},
			moe: objEnc,
		},
	}
	entry := zapcore.Entry{
		Time:    time.Date(2023, 7, 27, 12, 34, 56, 0, time.UTC),
		Level:   zapcore.InfoLevel,
		Message: "Test log message",
	}

	buffer, err := encoder.EncodeEntry(entry, nil)
	assert.NoError(t, err, "Error encoding entry")

	expectedOutput := "2023-07-27T12:34:56Z\tinfo\t[coord=a:b:c][gid=4]\tTest log message\n"
	assert.Equal(t, expectedOutput, buffer.String(), "Unexpected encoded output")
}
