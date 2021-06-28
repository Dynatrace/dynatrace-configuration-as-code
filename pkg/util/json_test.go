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
	"testing"

	"gotest.tools/assert"
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
	assert.NilError(t, err)

	err = ValidateJson(validJson2, Location{TemplateFilePath: "test2.json"})
	assert.NilError(t, err)
}

const syntaxErrorMisplacedContext = `{
	"key": "value",
	sneakySyntaxError
}`

func TestJsonUnmarshallingWithMisplacedContentExpectedError(t *testing.T) {

	err := ValidateJson(syntaxErrorMisplacedContext, Location{TemplateFilePath: "test.json"})
	assert.Check(t, err != nil)

	if jsonErr, ok := err.(*JsonValidationError); ok {
		assert.Equal(t, "test.json", jsonErr.Location.TemplateFilePath)
		assert.Equal(t, 3, jsonErr.LineNumber)
		assert.Equal(t, 2, jsonErr.CharacterNumberInLine)
		assert.Equal(t, "\tsneakySyntaxError", jsonErr.LineContent)
		assert.Check(t, jsonErr.Cause != nil)
	} else {
		t.Fail()
	}
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
	assert.Check(t, err != nil)

	if jsonErr, ok := err.(*JsonValidationError); ok {
		assert.Equal(t, "test.json", jsonErr.Location.TemplateFilePath)
		assert.Equal(t, 6, jsonErr.LineNumber)
		assert.Equal(t, 2, jsonErr.CharacterNumberInLine)
		assert.Equal(t, "\t]", jsonErr.LineContent)
		assert.Check(t, jsonErr.Cause != nil)
	} else {
		t.Fail()
	}
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
	assert.Check(t, err != nil)

	if jsonErr, ok := err.(*JsonValidationError); ok {
		assert.Equal(t, "no-comma.json", jsonErr.Location.TemplateFilePath)
		assert.Equal(t, 7, jsonErr.LineNumber)
		assert.Equal(t, 4, jsonErr.CharacterNumberInLine)
		assert.Equal(t, "\t\t\t\"boolean\": true", jsonErr.LineContent)
		assert.Check(t, jsonErr.Cause != nil)
	} else {
		t.Fail()
	}
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
	assert.Check(t, err != nil)

	if jsonErr, ok := err.(*JsonValidationError); ok {
		assert.Equal(t, "syntax-err.json", jsonErr.Location.TemplateFilePath)
		assert.Equal(t, 1, jsonErr.LineNumber)
		assert.Equal(t, 6, jsonErr.CharacterNumberInLine)
		assert.Equal(t, "\"key\": \"value\",", jsonErr.LineContent)
		assert.Check(t, jsonErr.Cause != nil)
	} else {
		t.Fail()
	}
}
