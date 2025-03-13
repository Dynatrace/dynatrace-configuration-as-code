//go:build unit

/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package strings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
)

func TestCapitalizeFirstRuneInString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected string
	}{
		{
			name:     "first letter is capitalized",
			s:        "hello world",
			expected: "Hello world",
		},
		{
			name:     "no change if first letter is already capitalized",
			s:        "Hello world",
			expected: "Hello world",
		},
		{
			name:     "empty string returns empty string",
			s:        "",
			expected: "",
		},
		{
			name:     "utf8 works",
			s:        "世界",
			expected: "世界",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			assert.Equal(t, tt.expected, strings.CapitalizeFirstRuneInString(tt.s))
		})
	}
}
