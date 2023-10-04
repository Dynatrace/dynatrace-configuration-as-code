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

package secret

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSensitiveString_String(t *testing.T) {
	tests := []struct {
		name     string
		input    MaskedString
		expected string
	}{
		{
			name:     "Non-empty string should be masked",
			input:    MaskedString("password123"),
			expected: "****",
		},
		{
			name:     "Empty string should be masked",
			input:    MaskedString(""),
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			assert.Equal(t, tt.expected, result, "Expected and actual values should be equal")
		})
	}
}

func TestSensitiveString_Value(t *testing.T) {
	tests := []struct {
		name     string
		input    MaskedString
		expected string
	}{
		{
			name:     "Non-empty string should return actual value",
			input:    MaskedString("password123"),
			expected: "password123",
		},
		{
			name:     "Empty string should return empty value",
			input:    MaskedString(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Value()
			assert.Equal(t, tt.expected, result, "Expected and actual values should be equal")
		})
	}
}
