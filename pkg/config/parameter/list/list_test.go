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

package list

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

func TestParseListParameter(t *testing.T) {

	tests := []struct {
		name     string
		context  parameter.ParameterParserContext
		wantVals []value.ValueParameter
	}{
		{
			"simple values",
			parameter.ParameterParserContext{
				Value: map[string]interface{}{
					"values": []interface{}{"firstName", "lastName"},
				},
			},
			[]value.ValueParameter{{"firstName"}, {"lastName"}},
		},
		{
			"full values",
			parameter.ParameterParserContext{
				Value: map[string]interface{}{
					"values": []interface{}{
						map[interface{}]interface{}{
							"type":  "value",
							"value": "firstName",
						},
						map[interface{}]interface{}{
							"type":  "value",
							"value": "lastName",
						},
					},
				},
			},
			[]value.ValueParameter{{"firstName"}, {"lastName"}},
		},
		{
			"complex values",
			parameter.ParameterParserContext{
				Value: map[string]interface{}{
					"values": []interface{}{
						map[interface{}]interface{}{
							"type": "value",
							"value": map[interface{}]interface{}{
								"firstName": "John",
								"lastName":  "Dorian",
							},
						},
					},
				},
			},
			[]value.ValueParameter{{map[string]interface{}{
				"firstName": "John",
				"lastName":  "Dorian",
			}}},
		},
		{
			"empty values",
			parameter.ParameterParserContext{
				Value: map[string]interface{}{
					"values": []interface{}{},
				},
			},
			[]value.ValueParameter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param, err := parseListParameter(tt.context)
			require.NoError(t, err)

			listParam, ok := param.(*ListParameter)

			require.True(t, ok, "parsed parameter should be list parameter")
			require.Equal(t, "list", listParam.GetType())

			require.Equal(t, tt.wantVals, listParam.Values)
		})
	}
}

func TestParseListParameter_Error(t *testing.T) {

	tests := []struct {
		name          string
		context       parameter.ParameterParserContext
		expextedError string
	}{
		{
			"fails on missing values",
			parameter.ParameterParserContext{
				Value: map[string]interface{}{
					"just some random thing": struct{}{},
				},
			},
			"missing property `values`",
		},
		{
			"fails on non-list values",
			parameter.ParameterParserContext{
				Value: map[string]interface{}{
					"values": "this is a string, not a list",
				},
			},
			"malformed property `values`",
		},
		{
			"fails if list entries ar not ValueParameter",
			parameter.ParameterParserContext{
				Value: map[string]interface{}{
					"values": []interface{}{
						struct {
							This string
						}{
							This: "should not be parsable",
						},
					},
				},
			},
			"malformed list entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseListParameter(tt.context)

			require.ErrorContains(t, err, tt.expextedError)
		})
	}
}

func TestResolveValue(t *testing.T) {
	context := parameter.ResolveContext{}

	compoundParameter := New([]value.ValueParameter{{Value: "a"}, {Value: "b"}, {Value: "c"}})

	result, err := compoundParameter.ResolveValue(context)
	require.NoError(t, err)
	assert.Equal(t, `[ "a","b","c" ]`, strings.ToString(result))
}

func TestResolveSingleValue(t *testing.T) {
	context := parameter.ResolveContext{}

	compoundParameter := New([]value.ValueParameter{{Value: "a"}})

	result, err := compoundParameter.ResolveValue(context)
	require.NoError(t, err)
	assert.Equal(t, `[ "a" ]`, strings.ToString(result))
}

func TestResolveEmptyValue(t *testing.T) {
	context := parameter.ResolveContext{}

	compoundParameter := New([]value.ValueParameter{})

	result, err := compoundParameter.ResolveValue(context)
	require.NoError(t, err)
	assert.Equal(t, `[  ]`, strings.ToString(result))
}

func Test_writeListParameter(t *testing.T) {
	tests := []struct {
		name         string
		inputContext parameter.ParameterWriterContext
		want         map[string]interface{}
		wantErr      bool
	}{
		{
			"simple write",
			parameter.ParameterWriterContext{
				Parameter: &ListParameter{
					Values: []value.ValueParameter{{"one"}, {"two"}, {"three"}},
				},
			},
			map[string]interface{}{"values": []interface{}{"one", "two", "three"}},
			false,
		},
		{
			"complex write",
			parameter.ParameterWriterContext{
				Parameter: &ListParameter{
					Values: []value.ValueParameter{
						{
							map[interface{}]interface{}{
								"firstName": "John",
								"lastName":  "Dorian",
							},
						},
					},
				},
			},
			map[string]interface{}{
				"values": []interface{}{
					map[string]interface{}{
						"type": "value",
						"value": map[interface{}]interface{}{
							"firstName": "John",
							"lastName":  "Dorian",
						},
					},
				},
			},
			false,
		},
		{
			"does not fail on empty values",
			parameter.ParameterWriterContext{
				Parameter: &ListParameter{
					Values: []value.ValueParameter{},
				},
			},
			map[string]interface{}{"values": []interface{}{}},
			false,
		},
		{
			"returns error if parameter is not a list",
			parameter.ParameterWriterContext{
				Parameter: &value.ValueParameter{
					Value: "I'm not a list!",
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := writeListParameter(tt.inputContext)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeListParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("writeListParameter() got = %v, want %v", got, tt.want)
			}
		})
	}
}
