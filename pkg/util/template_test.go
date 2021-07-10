//go:build unit
// +build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package util

import (
	"gotest.tools/assert"
	"testing"
)

const testMatrixTemplateWithEnvVar = "Follow the {{.color}} {{ .Env.ANIMAL }}"
const testMatrixTemplateWithProperty = "Follow the {{.color}} {{ .ANIMAL }}"

func TestGetStringWithEnvVar(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithEnvVar)
	assert.NilError(t, err)

	SetEnv(t, "ANIMAL", "cow")
	result, err := template.ExecuteTemplate(getTemplateTestProperties())
	UnsetEnv(t, "ANIMAL")

	assert.NilError(t, err)
	assert.Equal(t, "Follow the white cow", result)
}

func TestGetStringWithEnvVarLeadsToErrorIfEnvVarNotPresent(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithEnvVar)
	assert.NilError(t, err)

	UnsetEnv(t, "ANIMAL")
	_, err = template.ExecuteTemplate(getTemplateTestProperties())

	assert.ErrorContains(t, err, "map has no entry for key \"ANIMAL\"")
}

func TestGetStringLeadsToErrorIfPropertyNotPresent(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithEnvVar)
	assert.NilError(t, err)

	SetEnv(t, "ANIMAL", "cow")
	_, err = template.ExecuteTemplate(make(map[string]string)) // empty map
	UnsetEnv(t, "ANIMAL")

	assert.ErrorContains(t, err, "map has no entry for key \"color\"")
}

func TestGetStringWithEnvVarAndProperty(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithProperty)
	assert.NilError(t, err)

	SetEnv(t, "ANIMAL", "cow")
	result, err := template.ExecuteTemplate(getTemplateTestPropertiesClashingWithEnvVars())
	UnsetEnv(t, "ANIMAL")

	assert.NilError(t, err)
	assert.Equal(t, "Follow the white rabbit", result)
}

func TestGetStringWithEnvVarIncludingEqualSigns(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithEnvVar)
	assert.NilError(t, err)

	SetEnv(t, "ANIMAL", "cow=rabbit=chicken")
	result, err := template.ExecuteTemplate(getTemplateTestProperties())
	UnsetEnv(t, "ANIMAL")

	assert.NilError(t, err)
	assert.Equal(t, "Follow the white cow=rabbit=chicken", result)
}

func getTemplateTestProperties() map[string]string {

	m := make(map[string]string)

	m["color"] = "white"
	m["animalType"] = "rabbit"

	return m
}

func getTemplateTestPropertiesClashingWithEnvVars() map[string]string {

	m := make(map[string]string)

	m["color"] = "white"
	m["ANIMAL"] = "rabbit"

	return m
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

	result := EscapeNewlineCharacters(p)

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

	assert.DeepEqual(t, EscapeNewlineCharacters(empty), empty)
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
				t.Errorf("escapeCharactersForJson() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplatesWithSpecialCharactersProduceValidJson(t *testing.T) {
	tests := []struct {
		name           string
		templateString string
		properties     map[string]string
		want           string
	}{
		{
			"empty test should work",
			`{}`,
			map[string]string{},
			`{}`,
		},
		{
			"newlines are escaped",
			`{ "key": "{{ .value }}", "object": { "o_key": "{{ .object_value}}" } }`,
			map[string]string{
				"value":        "A string\nwith several lines\n\n - here's one\n\n - and another",
				"object_value": "and\none\nmore",
			},
			`{ "key": "A string\nwith several lines\n\n - here's one\n\n - and another", "object": { "o_key": "and\none\nmore" } }`,
		},
		{
			"regular slashes are not escaped",
			`{ "userAgent": "{{ .value }}" }`,
			map[string]string{
				"value": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36",
			},
			`{ "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36" }`,
		},
		{
			"a v1 list definition does not get quotes escaped",
			`{ "list": [ {{ .entries }} ] }`,
			map[string]string{
				"entries": `"element a", "element b", "element c"`,
			},
			`{ "list": [ "element a", "element b", "element c" ] }`,
		},
		{
			"a list definition can contain newlines",
			`{ "list": [ {{ .entries }} ] }`,
			map[string]string{
				"entries": `"element a",
"element b",
"element c"`,
			},
			"{ \"list\": [ \"element a\",\n\"element b\",\n\"element c\" ] }",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewTemplateFromString("template_test", tt.templateString)
			assert.NilError(t, err)

			result, err := template.ExecuteTemplate(tt.properties)
			assert.NilError(t, err)
			assert.Equal(t, result, tt.want)

			err = ValidateJson(result, "irrelevant filename")
			assert.NilError(t, err)
		})
	}
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
			if got := isListDefinition(tt.input); got != tt.want {
				t.Errorf("isListDefinition() = %v, want %v", got, tt.want)
			}
		})
	}
}
