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
// +build unit

package list

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"gotest.tools/assert"
)

func TestParseListParameter(t *testing.T) {
	param, err := parseListParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"values": []interface{}{"firstName", "lastName"},
		},
	})

	assert.NilError(t, err)

	listParam, ok := param.(*ListParameter)

	assert.Assert(t, ok, "parsed parameter should be list parameter")
	assert.Equal(t, listParam.GetType(), "list")

	vals := listParam.Values
	assert.Equal(t, len(vals), 2, "should have loaded two values")
	assert.Equal(t, vals[0], "firstName")
	assert.Equal(t, vals[1], "lastName")
}

func TestParseListParameter_ErrorOnMissingValues(t *testing.T) {
	_, err := parseListParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"just some random thing": struct{}{},
		},
	})

	assert.ErrorContains(t, err, "missing property `values`")
}

func TestParseListParameter_ErrorIfValuesAreNoList(t *testing.T) {
	_, err := parseListParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"values": "this is a string, not a list",
		},
	})

	assert.ErrorContains(t, err, "malformed property `values`")
}

//	func TestParseCompoundParameterComplexValue(t *testing.T) {
//		param, err := parseCompoundParameter(parameter.ParameterParserContext{
//			Value: map[string]interface{}{
//				"format":     "{{ .person.name }}: {{ .person.age }}",
//				"references": []interface{}{"person"},
//			},
//		})
//
//		assert.NilError(t, err)
//
//		compoundParameter, ok := param.(*CompoundParameter)
//		assert.Assert(t, ok, "parsed parameter should be compound parameter")
//
//		refs := compoundParameter.GetReferences()
//		assert.Equal(t, len(refs), 1, "should be referencing 1")
//		assert.Equal(t, refs[0].Property, "person")
//	}
//
//	func TestParseCompoundParameterErrorOnMissingFormat(t *testing.T) {
//		_, err := parseCompoundParameter(parameter.ParameterParserContext{
//			Value: map[string]interface{}{
//				"references": []interface{}{"firstName", "lastName"},
//			},
//		})
//
//		assert.Assert(t, err != nil, "expected an error parsing missing format")
//	}
//
//	func TestParseCompoundParameterErrorOnMissingReferences(t *testing.T) {
//		_, err := parseCompoundParameter(parameter.ParameterParserContext{
//			Value: map[string]interface{}{
//				"format": "{{ .firstName }} {{ .lastName }}",
//			},
//		})
//
//		assert.Assert(t, err != nil, "expected an error parsing missing references")
//	}
//
//	func TestParseCompoundParameterErrorOnWrongReferenceFormat(t *testing.T) {
//		_, err := parseCompoundParameter(parameter.ParameterParserContext{
//			Value: map[string]interface{}{
//				"format":     "{{ .firstName }} {{ .lastName }}",
//				"references": []int{3, 4},
//			}})
//
//		assert.Assert(t, err != nil, "expected an error parsing invalid references")
//	}
//
//	func TestParseCompoundParameterErrorOnWrongReferences(t *testing.T) {
//		_, err := parseCompoundParameter(parameter.ParameterParserContext{
//			Value: map[string]interface{}{
//				"format":     "{{ .firstName }} {{ .lastName }}",
//				"references": []interface{}{[]interface{}{}},
//			}})
//
//		assert.Assert(t, err != nil, "expected an error parsing invalid references")
//	}
func TestResolveValue(t *testing.T) {
	context := parameter.ResolveContext{}

	compoundParameter := New([]string{"a", "b", "c"})

	result, err := compoundParameter.ResolveValue(context)
	assert.NilError(t, err)

	assert.Equal(t, `[ "a","b","c" ]`, util.ToString(result))
}

func TestResolveSingleValue(t *testing.T) {
	context := parameter.ResolveContext{}

	compoundParameter := New([]string{"a"})

	result, err := compoundParameter.ResolveValue(context)
	assert.NilError(t, err)

	assert.Equal(t, `[ "a" ]`, util.ToString(result))
}

func TestResolveEmptyValue(t *testing.T) {
	context := parameter.ResolveContext{}

	compoundParameter := New([]string{})

	result, err := compoundParameter.ResolveValue(context)
	assert.NilError(t, err)

	assert.Equal(t, `[  ]`, util.ToString(result))
}

//
//func TestWriteCompoundParameter(t *testing.T) {
//	testFormat := "{{ .firstName }} {{ .lastName }}"
//	testRef1 := "firstName"
//	testRef2 := "lastName"
//	testRefs := []parameter.ParameterReference{
//		{Property: testRef1},
//		{Property: testRef2},
//	}
//	compoundParameter, err := New("testName", testFormat, testRefs)
//	assert.NilError(t, err)
//
//	context := parameter.ParameterWriterContext{Parameter: compoundParameter}
//
//	result, err := writeCompoundParameter(context)
//	assert.NilError(t, err)
//
//	assert.Equal(t, len(result), 2)
//
//	format, ok := result["format"]
//	assert.Assert(t, ok, "should have parameter format")
//	assert.Equal(t, format, testFormat)
//
//	references, ok := result["references"]
//	assert.Assert(t, ok, "should have parameter references")
//
//	referenceSlice, ok := references.([]interface{})
//	assert.Assert(t, ok, "references should be slice")
//
//	assert.Equal(t, len(referenceSlice), 2)
//	for i, testRef := range testRefs {
//		assert.Equal(t, referenceSlice[i], testRef.Property)
//	}
//}
//
//func TestWriteCompoundParameterErrorOnNonCompoundParameter(t *testing.T) {
//	context := parameter.ParameterWriterContext{Parameter: &value.ValueParameter{}}
//
//	_, err := writeCompoundParameter(context)
//	assert.Assert(t, err != nil, "expected an error writing wrong parameter type")
//}
//
//func TestWriteCompoundParameterErrorOnMissingFormat(t *testing.T) {
//	compoundParameter, err := New("testName", "", nil)
//	assert.NilError(t, err)
//
//	context := parameter.ParameterWriterContext{Parameter: compoundParameter}
//
//	_, err = writeCompoundParameter(context)
//	assert.Assert(t, err != nil, "expected an error writing missing format")
//}
//
//func TestWriteCompoundParameterErrorOnMissingReferences(t *testing.T) {
//	compoundParameter, err := New("testName", "testFormat", nil)
//	assert.NilError(t, err)
//
//	context := parameter.ParameterWriterContext{Parameter: compoundParameter}
//
//	_, err = writeCompoundParameter(context)
//	assert.Assert(t, err != nil, "expected an error writing missing references")
//}
