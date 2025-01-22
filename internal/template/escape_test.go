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

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeGoTemplating(t *testing.T) {
	tc := []struct {
		expected, in string
		name         string
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
		{
			in:       "fetch bizevents | FILTER `event.provider` == $MyVariable | FILTER like(event.type,\\\"platform.LoginEvent%\\\") | FIELDS CountryIso, Country | SUMMARIZE quantity = toDouble(count()), by:{{CountryIso, alias:countryIso}, {Country, alias:country}} | sort quantity desc",
			expected: "fetch bizevents | FILTER `event.provider` == $MyVariable | FILTER like(event.type,\\\"platform.LoginEvent%\\\") | FIELDS CountryIso, Country | SUMMARIZE quantity = toDouble(count()), by:{{`{{`}}CountryIso, alias:countryIso}, {Country, alias:country{{`}}`}} | sort quantity desc",
		},
	}

	for _, tt := range tc {
		t.Run(tt.in, func(t *testing.T) {
			out := UseGoTemplatesForDoubleCurlyBraces([]byte(tt.in))

			assert.Equal(t, tt.expected, string(out))
		})
	}
}
