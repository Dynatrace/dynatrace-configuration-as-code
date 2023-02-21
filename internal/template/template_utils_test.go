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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/regex"
	"gotest.tools/assert"
	"testing"
)

func Test_EscapeSpecialCharacters_EscapesNewline(t *testing.T) {

	p := map[string]interface{}{
		"string without newline": "just some string",
		"string with newline":    "some\nstring",
		"nested": map[string]interface{}{
			"nested without newline": "just some string",
			"nested with newline":    "some\nstring",
			"deepNested": map[string]interface{}{ // not yet used, but might be in the future
				"deepNested without newline": "just some string",
				"deepNested with newline":    "some\nstring",
			},
		},
		"nestedEnv": map[string]string{
			"nestedEnv without newline": "just some string",
			"nestedEnv with newline":    "some\nstring",
		},
	}

	result, err := EscapeSpecialCharacters(p)
	assert.NilError(t, err)

	expected := map[string]interface{}{
		`string without newline`: `just some string`,
		`string with newline`:    `some\nstring`,
		`nested`: map[string]interface{}{
			`nested without newline`: `just some string`,
			`nested with newline`:    `some\nstring`,
			`deepNested`: map[string]interface{}{ // not yet used, but might be in the future
				`deepNested without newline`: `just some string`,
				`deepNested with newline`:    `some\nstring`,
			},
		},
		`nestedEnv`: map[string]string{
			`nestedEnv without newline`: `just some string`,
			`nestedEnv with newline`:    `some\nstring`,
		},
	}

	assert.DeepEqual(t, expected, result)
}

func Test_EscapeSpecialCharacters_WithEmptyMap(t *testing.T) {

	empty := map[string]interface{}{}

	res, err := EscapeSpecialCharacters(empty)

	assert.NilError(t, err)
	assert.DeepEqual(t, res, empty)
}

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
	}
	for _, tt := range tests {
		t.Run(tt.inputString, func(t *testing.T) {
			got, err := escapeCharactersForJson(tt.inputString)
			if err != nil {
				t.Errorf("escapeCharactersForJson() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("escapeCharactersForJson() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_escapeNewlines(t *testing.T) {
	tests := []struct {
		inputString string
		want        string
	}{
		{
			`string with no quotes is unchanged`,
			`string with no quotes is unchanged`,
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
			`String with already escaped \n newline`,
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
	}
	for _, tt := range tests {
		t.Run(tt.inputString, func(t *testing.T) {
			got := escapeNewlines(tt.inputString)
			if got != tt.want {
				t.Errorf("escapeNewlines() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeNewlineCharacters(t *testing.T) {

	p := map[string]interface{}{
		"string without newline": "just some string",
		"string with newline":    "some\nstring",
		"nested": map[string]interface{}{
			"nested without newline": "just some string",
			"nested with newline":    "some\nstring",
			"deepNested": map[string]interface{}{ // not yet used, but might be in the future
				"deepNested without newline": "just some string",
				"deepNested with newline":    "some\nstring",
			},
		},
		"nestedEnv": map[string]string{
			"nestedEnv without newline": "just some string",
			"nestedEnv with newline":    "some\nstring",
		},
	}

	result, err := EscapeSpecialCharacters(p)
	assert.NilError(t, err)

	expected := map[string]interface{}{
		`string without newline`: `just some string`,
		`string with newline`:    `some\nstring`,
		`nested`: map[string]interface{}{
			`nested without newline`: `just some string`,
			`nested with newline`:    `some\nstring`,
			`deepNested`: map[string]interface{}{ // not yet used, but might be in the future
				`deepNested without newline`: `just some string`,
				`deepNested with newline`:    `some\nstring`,
			},
		},
		`nestedEnv`: map[string]string{
			`nestedEnv without newline`: `just some string`,
			`nestedEnv with newline`:    `some\nstring`,
		},
	}

	assert.DeepEqual(t, expected, result)
}

func TestEscapeNewlineCharactersWithEmptyMap(t *testing.T) {

	empty := map[string]interface{}{}

	res, err := EscapeSpecialCharacters(empty)

	assert.NilError(t, err)
	assert.DeepEqual(t, res, empty)
}

func Test_isListDefinition(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{
			"just a normal string",
			false,
		},
		{
			`"not a list! just broken",`,
			false,
		},
		{
			`"shortest list", "possible"`,
			true,
		},
		{
			`"slightly","longer","simple","list"`,
			true,
		},
		{
			`"slightly" , "longer",   "simple" , "list" , "with"    ,    "spaces"`,
			true,
		},
		{
			`"a", "list", "with", "trailing", "comma",`,
			true,
		},
		{
			`
                   "slightly",
                   "longer",
                   "simple",
                   "list",
                   "with"
                   ,"line breaks"

                  `,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := regex.IsListDefinition(tt.input); got != tt.want {
				t.Errorf("isListDefinition() = %v, want %v", got, tt.want)
			}
		})
	}
}
