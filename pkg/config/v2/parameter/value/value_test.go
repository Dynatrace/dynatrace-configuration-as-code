// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build unit

package value

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/json"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"

	"gotest.tools/assert"
)

func TestParseValueParameter(t *testing.T) {
	value := "test"

	param, err := parseValueParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"value": value,
		},
	})

	assert.NilError(t, err)

	valueParam, ok := param.(*ValueParameter)
	assert.Assert(t, ok, "parsed parameter should be value parameter")
	assert.Equal(t, valueParam.GetType(), "value")

	assert.Equal(t, value, valueParam.Value)
}

func TestParseValueParameterMap(t *testing.T) {
	value := map[string]string{
		"foo":  "bar",
		"fizz": "buzz",
	}

	param, err := parseValueParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"value": value,
		},
	})

	assert.NilError(t, err)

	valueParam, ok := param.(*ValueParameter)
	assert.Assert(t, ok, "parsed parameter should be value parameter")

	result, ok := valueParam.Value.(map[string]string)
	assert.Assert(t, ok, "result should be of type map[string]string, is: %T", valueParam.Value)
	assert.Equal(t, len(result), 2)

	for key, val := range value {
		assert.Equal(t, result[key], val)
	}
}

func TestParseValueParameterMissingValueParameterShouldReturnError(t *testing.T) {
	value := "test"

	_, err := parseValueParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"title": value,
		},
	})

	assert.Assert(t, err != nil)
}

func TestGetReferencesShouldNotReturnAnything(t *testing.T) {
	fixture := New("test")

	refs := fixture.GetReferences()

	assert.Assert(t, len(refs) == 0)
}

func TestResolveValue(t *testing.T) {
	value := "test"
	fixture := New(value)

	result, err := fixture.ResolveValue(parameter.ResolveContext{})

	assert.NilError(t, err)
	assert.Equal(t, value, result)
}

func TestResolveValueMap(t *testing.T) {
	value := map[string]string{
		"foo":  "bar",
		"some": "thing",
	}
	fixture := New(value)

	result, err := fixture.ResolveValue(parameter.ResolveContext{})

	assert.NilError(t, err)

	resultMap, ok := result.(map[string]string)
	assert.Assert(t, ok, "result should be of type map[string]string, is: %T", result)
	assert.Equal(t, len(resultMap), 2)

	for key, val := range value {
		assert.Equal(t, resultMap[key], val)
	}
}

func TestWriteValueParameter(t *testing.T) {
	value := "something"
	param := New(value)

	context := parameter.ParameterWriterContext{
		Parameter: param,
	}

	result, err := writeValueParameter(context)

	assert.NilError(t, err)
	assert.Equal(t, len(result), 1, "should have 1 property")

	resultVal, ok := result["value"]
	assert.Assert(t, ok, "should have property `name`")
	assert.Equal(t, resultVal, value)
}

func TestValuesWithSpecialCharactersReturnContentValidForJson(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			"empty test should work",
			"",
		},
		{
			"newlines are escaped",
			"A string\nwith several lines\n\n - here's one\n\n - and another",
		},
		{
			"regular slashes are not escaped",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36",
		},
		{
			"a list string gets quotes escaped",
			`"element a", "element b", "element c"`,
		},
		{
			"a list string can still contain newlines",
			`"element a",
"element b",
"element c"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := New(tt.value)

			result, err := fixture.ResolveValue(parameter.ResolveContext{})

			assert.NilError(t, err)

			resString := result.(string)

			sampleJson := fmt.Sprintf(`{ "val": "%s"}`, resString)

			err = json.ValidateJson(sampleJson, json.Location{})
			assert.NilError(t, err)
		})
	}
}
