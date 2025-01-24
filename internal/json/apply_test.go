//go:build unit

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package json

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyToStringValues_Success(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		f              func(s string) string
		expectedResult string
	}{
		{
			name:           "boolean value is preserved",
			content:        "true",
			expectedResult: "true",
		},
		{
			name:           "boolean value is not replaced",
			content:        "true",
			f:              func(s string) string { return strings.ReplaceAll(s, "true", "true") },
			expectedResult: "true",
		},
		{
			name:           "float value is preserved",
			content:        "10",
			expectedResult: "10",
		},
		{
			name:           "float value is not replaced",
			content:        "10",
			f:              func(s string) string { return strings.ReplaceAll(s, "10", "10") },
			expectedResult: "10",
		},
		{
			name:           "string value is replaced if found",
			content:        "\"find\"",
			f:              func(s string) string { return strings.ReplaceAll(s, "find", "replace") },
			expectedResult: "\"replace\"",
		},
		{
			name:           "string value is not replaced if not found",
			content:        "\"nope\"",
			f:              func(s string) string { return strings.ReplaceAll(s, "find", "replace") },
			expectedResult: "\"nope\"",
		},
		{
			name:           "string value is replaced if found in object",
			content:        `{"key": "find"}`,
			f:              func(s string) string { return strings.ReplaceAll(s, "find", "replace") },
			expectedResult: `{"key":"replace"}`,
		},
		{
			name:           "string value is not replaced if not found in object",
			content:        `{"key": "nope"}`,
			f:              func(s string) string { return strings.ReplaceAll(s, "find", "replace") },
			expectedResult: `{"key":"nope"}`,
		},
		{
			name:           "string value is replaced if found in object in array",
			content:        `[{"key": "find"}]`,
			f:              func(s string) string { return strings.ReplaceAll(s, "find", "replace") },
			expectedResult: `[{"key":"replace"}]`,
		},
		{
			name:           "string value is not replaced if not found in object in array",
			content:        `[{"key": "nope"}]`,
			f:              func(s string) string { return strings.ReplaceAll(s, "find", "replace") },
			expectedResult: `[{"key":"nope"}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyToStringValues(tt.content, tt.f)
			assert.EqualValues(t, tt.expectedResult, result)
			assert.NoError(t, err)
		})
	}
}

func TestApplyToStringValues_Errors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		f       func(s string) string
	}{
		{
			name:    "empty string doesnt work",
			content: "",
		},
		{
			name:    "unquoted string produces error",
			content: "something",
		},
		{
			name:    "truncated json produces error",
			content: `{ "key": "value", `,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyToStringValues(tt.content, tt.f)
			assert.Empty(t, result)
			assert.Error(t, err)
		})
	}
}
