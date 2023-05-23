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

package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEscapeJinjaTemplates(t *testing.T) {
	tc := []struct {
		expected, in string
	}{
		{
			in:       `Hello, {{planet}}!`,
			expected: "Hello, {{`{{`}}planet{{`}}`}}!",
		},
		{
			in:       `Hello , {{ calendar("abcde") }}`,
			expected: "Hello , {{`{{`}} calendar(\"abcde\") {{`}}`}}",
		},
		{
			in:       `no jinja`,
			expected: "no jinja",
		},
		{
			in:       `{{`,
			expected: "{{`{{`}}",
		},
		{
			in:       `{`,
			expected: "{",
		},
		{
			in:       `\{`,
			expected: "\\{",
		},
		{
			in:       `}}`,
			expected: "{{`}}`}}",
		},
		{
			in:       `}`,
			expected: "}"},
		{
			in:       `\}`,
			expected: "\\}",
		},
		{
			in:       `{{ }}`,
			expected: "{{`{{`}} {{`}}`}}",
		},
	}

	for _, tt := range tc {
		t.Run(tt.in, func(t *testing.T) {
			out := EscapeJinjaTemplates([]byte(tt.in))

			assert.Equal(t, tt.expected, string(out))
		})
	}
}
