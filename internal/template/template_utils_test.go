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

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_escapeCharactersForJson(t *testing.T) {
	tests := []struct {
		inputString string
		want        string
	}{
		{
			`string with no quotes is unchanged`,
			`string with no quotes is unchanged`,
		},
		{
			`string """ with "double quotes" results in quotes being "escaped"`,
			`string \"\"\" with \"double quotes\" results in quotes being \"escaped\"`,
		},
		{
			`string with 'single quotes' quotes is unchanged`,
			`string with 'single quotes' quotes is unchanged`,
		},
		{
			"\nString with multiple \n new\nlines on many positions\n\n",
			`\nString with multiple \n new\nlines on many positions\n\n`,
		},
		{
			"String with already escaped \\n newline",
			`String with already escaped \\n newline`,
		},
		{
			"String with one windows\r\nnewline",
			`String with one windows\r\nnewline`,
		},
		{
			"String with one\nnewline",
			`String with one\nnewline`,
		},
		{
			"String with one\nnewline",
			`String with one\nnewline`,
		},
		{
			"String without newline",
			`String without newline`,
		},
		{
			"String { containing {{ some {json : like text } stays as is",
			`String { containing {{ some {json : like text } stays as is`,
		},
		{
			"String containing <, >, and & must not be escaped",
			"String containing <, >, and & must not be escaped",
		},
		{
			"Real world example: [8/5] Disk space available < 15% (/media/datastore)",
			"Real world example: [8/5] Disk space available < 15% (/media/datastore)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.inputString, func(t *testing.T) {
			got, err := escapeCharactersForJson(tt.inputString)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
