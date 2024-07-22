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

package compound

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

func TestParseCompoundParameter(t *testing.T) {
	param, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format":     "{{ .firstName }} {{ .lastName }}",
			"references": []interface{}{"firstName", "lastName"},
		},
	})

	require.NoError(t, err)

	compoundParameter, ok := param.(*CompoundParameter)

	require.True(t, ok, "parsed parameter should be compound parameter")
	assert.Equal(t, "compound", compoundParameter.GetType())

	refs := compoundParameter.GetReferences()
	require.Len(t, refs, 2, "should be referencing 2 parameters")
	assert.Equal(t, "firstName", refs[0].Property)
	assert.Equal(t, "lastName", refs[1].Property)
}

func TestParseCompoundParameterComplexValue(t *testing.T) {
	param, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format":     "{{ .person.name }}: {{ .person.age }}",
			"references": []interface{}{"person"},
		},
	})

	require.NoError(t, err)

	compoundParameter, ok := param.(*CompoundParameter)
	require.True(t, ok, "parsed parameter should be compound parameter")

	refs := compoundParameter.GetReferences()
	require.Len(t, refs, 1, "should be referencing 1")
	assert.Equal(t, "person", refs[0].Property)
}

func TestParseCompoundParameterErrorOnMissingFormat(t *testing.T) {
	_, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"references": []interface{}{"firstName", "lastName"},
		},
	})

	require.Error(t, err, "expected an error parsing missing format")
}

func TestParseCompoundParameterErrorOnMissingReferences(t *testing.T) {
	_, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format": "{{ .firstName }} {{ .lastName }}",
		},
	})

	require.Error(t, err, "expected an error parsing missing references")
}

func TestParseCompoundParameterErrorOnWrongReferenceFormat(t *testing.T) {
	_, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format":     "{{ .firstName }} {{ .lastName }}",
			"references": []int{3, 4},
		}})

	require.Error(t, err, "expected an error parsing invalid references")
}

func TestParseCompoundParameterErrorOnWrongReferences(t *testing.T) {
	_, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format":     "{{ .firstName }} {{ .lastName }}",
			"references": []interface{}{[]interface{}{}},
		}})

	require.Error(t, err, "expected an error parsing invalid references")
}

func TestResolveValue(t *testing.T) {
	testFormat := "{{ .greeting }} {{ .entity }}!"
	context := parameter.ResolveContext{
		ResolvedParameterValues: parameter.Properties{
			"greeting": "Hello",
			"entity":   "World",
		},
	}
	compoundParameter, err := New("testName", testFormat, []parameter.ParameterReference{
		{Property: "greeting"},
		{Property: "entity"},
	})
	require.NoError(t, err)

	result, err := compoundParameter.ResolveValue(context)
	require.NoError(t, err)

	assert.Equal(t, "Hello World!", strings.ToString(result))
}

func TestResolveComplexValue(t *testing.T) {
	testFormat := "{{ .person.name }} is {{ .person.age }} years old"
	context := parameter.ResolveContext{
		ResolvedParameterValues: parameter.Properties{
			"person": map[string]interface{}{
				"age":  12,
				"name": "Hansi",
			},
		},
	}
	compoundParameter, err := New("testName", testFormat,
		[]parameter.ParameterReference{{Property: "person"}})
	require.NoError(t, err)

	result, err := compoundParameter.ResolveValue(context)
	require.NoError(t, err)

	assert.Equal(t, "Hansi is 12 years old", strings.ToString(result))
}

func TestResolveValueErrorOnUndefinedReference(t *testing.T) {
	testFormat := "{{ .firstName }} {{ .lastName }}"
	context := parameter.ResolveContext{
		ResolvedParameterValues: parameter.Properties{
			"person": map[string]interface{}{
				"age":  12,
				"name": "Hansi",
			},
		},
	}
	compoundParameter, err := New("testName", testFormat,
		[]parameter.ParameterReference{{Property: "firstName"}})
	require.NoError(t, err)

	_, err = compoundParameter.ResolveValue(context)

	require.Error(t, err, "expected an error resolving undefined references")
}

func TestWriteCompoundParameter(t *testing.T) {
	testFormat := "{{ .firstName }} {{ .lastName }}"
	testRef1 := "firstName"
	testRef2 := "lastName"
	testRefs := []parameter.ParameterReference{
		{Property: testRef1},
		{Property: testRef2},
	}
	compoundParameter, err := New("testName", testFormat, testRefs)
	require.NoError(t, err)

	context := parameter.ParameterWriterContext{Parameter: compoundParameter}

	result, err := writeCompoundParameter(context)
	require.NoError(t, err)

	require.Len(t, result, 2)

	format, ok := result["format"]
	require.True(t, ok, "should have parameter format")
	assert.Equal(t, testFormat, format)

	references, ok := result["references"]
	require.True(t, ok, "should have parameter references")

	referenceSlice, ok := references.([]interface{})
	require.True(t, ok, "references should be slice")

	require.Len(t, referenceSlice, 2)
	for i, testRef := range testRefs {
		assert.Equal(t, testRef.Property, referenceSlice[i])
	}
}

func TestWriteCompoundParameterErrorOnNonCompoundParameter(t *testing.T) {
	context := parameter.ParameterWriterContext{Parameter: &value.ValueParameter{}}

	_, err := writeCompoundParameter(context)
	require.Error(t, err, "expected an error writing wrong parameter type")
}

func TestWriteCompoundParameterErrorOnMissingFormat(t *testing.T) {
	compoundParameter, err := New("testName", "", nil)
	require.NoError(t, err)

	context := parameter.ParameterWriterContext{Parameter: compoundParameter}

	_, err = writeCompoundParameter(context)
	require.Error(t, err, "expected an error writing missing format")
}

func TestWriteCompoundParameterErrorOnMissingReferences(t *testing.T) {
	compoundParameter, err := New("testName", "testFormat", nil)
	require.NoError(t, err)

	context := parameter.ParameterWriterContext{Parameter: compoundParameter}

	_, err = writeCompoundParameter(context)
	require.Error(t, err, "expected an error writing missing references")
}

func TestCompoundParameter_Equal(t *testing.T) {
	c1, _ := New("testName", "testFormat", nil)
	c2, _ := New("testName", "testFormat", nil)
	c3, _ := New("testName", "testFormat_DIFF", nil)
	c4, _ := New("testName", "testFormat", []parameter.ParameterReference{{}})
	c5, _ := New("testName", "testFormat", []parameter.ParameterReference{{}})
	require.True(t, c1.Equal(c2))
	require.True(t, c2.Equal(c1))
	require.False(t, c1.Equal(c3))
	require.False(t, c3.Equal(c1))
	require.False(t, c1.Equal(c4))
	require.False(t, c4.Equal(c1))
	require.True(t, c5.Equal(c4))
	require.True(t, c4.Equal(c5))

}
