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

package json

import (
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

const validJson1 = `{
	"key": "value"
}`

const validJson2 = `{
	"key": "value",
	"list": [
		{
			"foo": "bar",
			"boolean": true
		}
	]
}`

func TestJsonUnmarshallingWorks(t *testing.T) {
	err := ValidateJson(validJson1, Location{TemplateFilePath: "test.json"})
	require.NoError(t, err)

	err = ValidateJson(validJson2, Location{TemplateFilePath: "test2.json"})
	require.NoError(t, err)
}

const syntaxErrorMisplacedContext = `{
	"key": "value",
	sneakySyntaxError
}`

func TestJsonUnmarshallingWithMisplacedContentExpectedError(t *testing.T) {

	err := ValidateJson(syntaxErrorMisplacedContext, Location{TemplateFilePath: "test.json"})
	require.Error(t, err)

	jsonErr, ok := err.(JsonValidationError)
	require.Truef(t, ok, "err should be of type JsonValidationError, is: %T", err)

	require.Equal(t, "test.json", jsonErr.Location.TemplateFilePath)
	require.Equal(t, 3, jsonErr.LineNumber)
	require.Equal(t, 2, jsonErr.CharacterNumberInLine)
	require.Equal(t, "\tsneakySyntaxError", jsonErr.LineContent)
	require.Error(t, jsonErr.Err)
}

const syntaxErrorNoClosingBracket = `{
	"key": "value",
	"list": [
		{
			"foo": "bar"
	]
}`

func TestJsonUnmarshallingWithNoClosingBracketExpectedError(t *testing.T) {

	err := ValidateJson(syntaxErrorNoClosingBracket, Location{TemplateFilePath: "test.json"})
	require.Error(t, err)

	jsonErr, ok := err.(JsonValidationError)
	require.Truef(t, ok, "err should be of type JsonValidationError, is: %T", err)

	require.Equal(t, "test.json", jsonErr.Location.TemplateFilePath)
	require.Equal(t, 6, jsonErr.LineNumber)
	require.Equal(t, 2, jsonErr.CharacterNumberInLine)
	require.Equal(t, "\t]", jsonErr.LineContent)
	require.Error(t, jsonErr.Err)
}

const syntaxErrorNoComma = `{
	"key": "value",
	"list": [
		{
			"foo": "bar",
			"no": "comma"
			"boolean": true
		}
	]
}`

func TestJsonUnmarshallingNoCommaExpectedError(t *testing.T) {

	err := ValidateJson(syntaxErrorNoComma, Location{TemplateFilePath: "no-comma.json"})
	require.Error(t, err)

	jsonErr, ok := err.(JsonValidationError)
	require.Truef(t, ok, "err should be of type JsonValidationError, is: %T", err)

	require.Equal(t, "no-comma.json", jsonErr.Location.TemplateFilePath)
	require.Equal(t, 7, jsonErr.LineNumber)
	require.Equal(t, 4, jsonErr.CharacterNumberInLine)
	require.Equal(t, "\t\t\t\"boolean\": true", jsonErr.LineContent)
	require.Error(t, jsonErr.Err)
}

const syntaxErrorInFirstLine = `"key": "value",
"list": [
	{
		"foo": "bar",
		"no": "comma"
	}
]`

func TestJsonUnmarshallingNoOpeningParenthesisExpectedError(t *testing.T) {

	err := ValidateJson(syntaxErrorInFirstLine, Location{TemplateFilePath: "syntax-err.json"})
	require.Error(t, err)

	jsonErr, ok := err.(JsonValidationError)
	require.Truef(t, ok, "err should be of type JsonValidationError, is: %T", err)

	require.Equal(t, "syntax-err.json", jsonErr.Location.TemplateFilePath)
	require.Equal(t, 1, jsonErr.LineNumber)
	require.Equal(t, 6, jsonErr.CharacterNumberInLine)
	require.Equal(t, "\"key\": \"value\",", jsonErr.LineContent)
	require.Error(t, jsonErr.Err)
}

func TestMarshalIndent(t *testing.T) {
	tests := []struct {
		name       string
		jsonInput  []byte
		wantOutput []byte
	}{
		{
			name:       "Valid JSON input is indented",
			jsonInput:  []byte(`{"name": "Alice", "age": 30}`),
			wantOutput: []byte("{\n  \"name\": \"Alice\",\n  \"age\": 30\n}"),
		},
		{
			name:       "Invalid JSON input is returned as is",
			jsonInput:  []byte(`{s`),
			wantOutput: []byte(`{s`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutput := MarshalIndent(tt.jsonInput)

			if !reflect.DeepEqual(gotOutput, tt.wantOutput) {
				t.Errorf("MarshalIndent(%v) = %v, want %v", tt.jsonInput, gotOutput, tt.wantOutput)
			}
		})
	}
}
