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

package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvePropValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		props    map[string]any
		expected any
		found    bool
	}{
		{
			name:     "Simple key found",
			key:      "first",
			props:    map[string]any{"first": 42},
			expected: 42,
			found:    true,
		},
		{
			name:     "Nested key found",
			key:      "first.second.third",
			props:    map[string]any{"first": map[string]any{"second": map[string]any{"third": 99}}},
			expected: 99,
			found:    true,
		},
		{
			name:     "Key not found",
			key:      "nonexistent.key",
			props:    map[string]any{"existing": 123},
			expected: nil,
			found:    false,
		},
		{
			name:     "Partial key not found",
			key:      "first.second.nonexistent",
			props:    map[string]any{"first": map[string]any{"second": 456}},
			expected: nil,
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := ResolvePropValue(tt.key, tt.props)
			assert.Equal(t, tt.expected, value)
			assert.Equal(t, tt.found, found)
		})
	}
}
