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

package environment

import (
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"gotest.tools/assert"
)

func TestParseValueParameter(t *testing.T) {
	name := "test"

	param, err := parseEnvironmentValueParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"name": name,
		},
	})

	assert.NilError(t, err)

	envParameter, ok := param.(*EnvironmentVariableParameter)

	assert.Assert(t, ok, "parsed parameter should be environment parameter")
	assert.Equal(t, envParameter.GetType(), "environment")

	assert.Equal(t, name, envParameter.Name)
	assert.Assert(t, !envParameter.HasDefaultValue, "environment parameter should not have default")
}

func TestParseValueParameterWithDefault(t *testing.T) {
	name := "test"
	defaultValue := "this"

	param, err := parseEnvironmentValueParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"name":    name,
			"default": defaultValue,
		},
	})

	assert.NilError(t, err)

	envParameter, ok := param.(*EnvironmentVariableParameter)

	assert.Assert(t, ok, "parsed parameter should be environment parameter")
	assert.Equal(t, name, envParameter.Name)
	assert.Assert(t, envParameter.HasDefaultValue, "environment parameter should have default")
	assert.Equal(t, defaultValue, envParameter.DefaultValue)
}

func TestParseValueParameterMissingRequiredField(t *testing.T) {
	_, err := parseEnvironmentValueParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"wrong":   "value",
			"default": "value",
		},
	})

	assert.Assert(t, err != nil, "error should be present")
}

func TestGetReferences(t *testing.T) {
	fixture := New("test")

	references := fixture.GetReferences()

	assert.Equal(t, len(references), 0, "environment parameter should not have references")
}

func TestResolveValue(t *testing.T) {
	name := "test"
	value := "this is a test"

	t.Setenv(name, value)

	fixture := New(name)

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ParameterName: "test",
	})

	assert.NilError(t, err)
	assert.Equal(t, value, result)
}

func TestResolveValue_EscapesSpecialCharacters(t *testing.T) {
	name := "test"
	v := `this is a "test"`
	expected := `this is a \"test\"`

	t.Setenv(name, v)

	fixture := New(name)

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ParameterName: "test",
	})

	assert.NilError(t, err)
	assert.Equal(t, expected, result)
}

func TestResolveValueWithDefaultValue(t *testing.T) {
	name := "__not_set_test"
	defaultValue := "this is the default"

	fixture := NewWithDefault(name, defaultValue)

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ParameterName: name,
	})

	assert.NilError(t, err)
	assert.Equal(t, defaultValue, result)
}

func TestResolveValueErrorOnUnsetEnvVar(t *testing.T) {
	name := "__not_set_test"

	fixture := New(name)

	_, err := fixture.ResolveValue(parameter.ResolveContext{
		ParameterName: name,
	})

	assert.Assert(t, err != nil, "expected an error when resolving unset var without default")
}

func TestWriteEnvironmentValueParameter(t *testing.T) {
	name := "TEST"
	envParam := New(name)

	context := parameter.ParameterWriterContext{
		Parameter: envParam,
	}

	result, err := writeEnvironmentValueParameter(context)

	assert.NilError(t, err)
	assert.Equal(t, len(result), 1, "should have 1 property")

	resultEnv, ok := result["name"]
	assert.Assert(t, ok, "should have property `name`")
	assert.Equal(t, resultEnv, name)
}

func TestWriteEnvironmentValueParameterWithDefault(t *testing.T) {
	name := "TEST"
	defaultVal := "some default"
	envParam := NewWithDefault(name, defaultVal)

	context := parameter.ParameterWriterContext{
		Parameter: envParam,
	}

	result, err := writeEnvironmentValueParameter(context)

	assert.NilError(t, err)
	assert.Equal(t, len(result), 2, "should have 2 properties")

	resultDefault, ok := result["default"]
	assert.Assert(t, ok, "should have property `default`")
	assert.Equal(t, resultDefault, defaultVal)

	resultEnv, ok := result["name"]
	assert.Assert(t, ok, "should have property `name`")
	assert.Equal(t, resultEnv, name)
}

func TestWriteEnvironmentValueParameterErrorOnOtherParameterType(t *testing.T) {
	valueParam := value.ValueParameter{}

	context := parameter.ParameterWriterContext{
		Parameter: &valueParam,
	}

	_, err := writeEnvironmentValueParameter(context)

	assert.Assert(t, err != nil)
}
