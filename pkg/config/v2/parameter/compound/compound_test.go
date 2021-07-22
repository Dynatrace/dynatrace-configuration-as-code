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

// +build unit

package compound

import (
	"testing"
	"text/template"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"gotest.tools/assert"
)

func TestParseCompoundParameter(t *testing.T) {
	param, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format":     "{{ .firstName }} {{ .lastName }}",
			"references": []interface{}{"firstName", "lastName"},
		},
	})

	assert.NilError(t, err)

	compoundParameter, ok := param.(*CompoundParameter)
	assert.Assert(t, ok, "parsed parameter should be compound parameter")

	assert.Equal(t, len(compoundParameter.referencedParameters), 2, "should be referencing 2 parameters")
	assert.Equal(t, compoundParameter.referencedParameters[0].Property, "firstName")
	assert.Equal(t, compoundParameter.referencedParameters[1].Property, "lastName")
}

func TestParseCompoundParameterComplexValue(t *testing.T) {
	param, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format":     "{{ .person.name }}: {{ .person.age }}",
			"references": []interface{}{"person"},
		},
	})

	assert.NilError(t, err)

	compoundParameter, ok := param.(*CompoundParameter)
	assert.Assert(t, ok, "parsed parameter should be compound parameter")

	assert.Equal(t, len(compoundParameter.referencedParameters), 1, "should be referencing 1")
	assert.Equal(t, compoundParameter.referencedParameters[0].Property, "person")
}

func TestParseCompoundParameterErrorOnMissingFormat(t *testing.T) {
	_, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"references": []interface{}{"firstName", "lastName"},
		},
	})

	assert.ErrorContains(t, err, "missing property `format`")
}

func TestParseCompoundParameterErrorOnMissingReferences(t *testing.T) {
	_, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format": "{{ .firstName }} {{ .lastName }}",
		},
	})

	assert.ErrorContains(t, err, "missing property `references`")
}

func TestParseCompoundParameterErrorOnWrongReferenceFormat(t *testing.T) {
	_, err := parseCompoundParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"format":     "{{ .firstName }} {{ .lastName }}",
			"references": []int{3, 4},
		}})

	assert.ErrorContains(t, err, "malformed value `references`")
}

func TestResolveValue(t *testing.T) {
	testFormat, _ := template.New("").Option("missingkey=error").Parse("{{ .greeting }} {{ .entity }}!")
	context := parameter.ResolveContext{
		ResolvedParameterValues: parameter.Properties{
			"greeting": "Hello",
			"entity":   "World",
		},
	}
	compoundParameter := CompoundParameter{
		format: testFormat,
		referencedParameters: []parameter.ParameterReference{
			parameter.ParameterReference{Property: "greeting"},
			parameter.ParameterReference{Property: "entity"},
		},
	}

	result, err := compoundParameter.ResolveValue(context)
	assert.NilError(t, err)

	assert.Equal(t, "Hello World!", util.ToString(result))
}

func TestResolveComplexValue(t *testing.T) {
	testFormat, _ := template.New("").Option("missingkey=error").Parse("{{ .person.name }} is {{ .person.age }} years old")
	context := parameter.ResolveContext{
		ResolvedParameterValues: parameter.Properties{
			"person": map[string]interface{}{
				"age":  12,
				"name": "Hansi",
			},
		},
	}
	compoundParameter := CompoundParameter{
		format: testFormat,
		referencedParameters: []parameter.ParameterReference{
			parameter.ParameterReference{Property: "person"},
		},
	}

	result, err := compoundParameter.ResolveValue(context)
	assert.NilError(t, err)

	assert.Equal(t, "Hansi is 12 years old", util.ToString(result))
}

func TestResolveValueErrorOnUndefinedReference(t *testing.T) {
	testFormat, _ := template.New("").Option("missingkey=error").Parse("{{ .firstName }} {{ .lastName }}")
	context := parameter.ResolveContext{
		ResolvedParameterValues: parameter.Properties{
			"person": map[string]interface{}{
				"age":  12,
				"name": "Hansi",
			},
		},
	}
	compoundParameter := CompoundParameter{
		format: testFormat,
		referencedParameters: []parameter.ParameterReference{
			parameter.ParameterReference{Property: "firstName"},
		},
	}

	_, err := compoundParameter.ResolveValue(context)

	assert.ErrorContains(t, err, `map has no entry for key "lastName"`)
}

func TestWriteCompoundParameter(t *testing.T) {
	testFormat := "{{ .firstName }} {{ .lastName }}"
	testRef1 := "firstName"
	testRef2 := "lastName"
	compoundParameter := CompoundParameter{
		rawFormatString: testFormat,
		referencedParameters: []parameter.ParameterReference{
			parameter.ParameterReference{Property: testRef1},
			parameter.ParameterReference{Property: testRef2},
		},
	}

	context := parameter.ParameterWriterContext{Parameter: &compoundParameter}

	result, err := writeCompoundParameter(context)
	assert.NilError(t, err)

	assert.Equal(t, len(result), 2)

	format, ok := result["format"]
	assert.Assert(t, ok, "should have parameter format")
	assert.Equal(t, format, testFormat)

	references, ok := result["references"]
	assert.Assert(t, ok, "should have parameter references")

	referenceSlice, ok := references.([]interface{})
	assert.Assert(t, ok, "references should be slice")

	assert.Equal(t, len(referenceSlice), 2)
	for i, testRef := range []interface{}{testRef1, testRef2} {
		assert.Equal(t, referenceSlice[i], testRef)
	}
}

func TestWriteCompoundParameterErrorOnNonCompoundParameter(t *testing.T) {
	context := parameter.ParameterWriterContext{Parameter: &value.ValueParameter{}}

	_, err := writeCompoundParameter(context)
	assert.ErrorContains(t, err, "unexpected type. parameter is not of type `CompoundParameter`")
}

func TestWriteCompoundParameterErrorOnMissingFormat(t *testing.T) {
	compoundParameter := CompoundParameter{
		referencedParameters: []parameter.ParameterReference{
			parameter.ParameterReference{Property: "firstName"},
		},
	}
	context := parameter.ParameterWriterContext{Parameter: &compoundParameter}

	_, err := writeCompoundParameter(context)
	assert.ErrorContains(t, err, "missing property `format`")
}

func TestWriteCompoundParameterErrorOnMissingReferences(t *testing.T) {
	compoundParameter := CompoundParameter{
		rawFormatString: "{{ .firstName }} {{ .lastName }}",
	}
	context := parameter.ParameterWriterContext{Parameter: &compoundParameter}

	_, err := writeCompoundParameter(context)
	assert.ErrorContains(t, err, "missing property `references`")
}
