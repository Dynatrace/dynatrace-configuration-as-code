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

package environment

import (
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/envvars"
	"gotest.tools/assert"
)

func TestParseValueParameter(t *testing.T) {
	name := "test"

	param, err := parseEnvironmentValueParameter(parameter.ParameterParserContext{
		Coordinate: coordinate.Coordinate{
			Project: "projectA",
			Api:     "dashboard",
			Config:  "super-important",
		},
		ParameterName: "title",
		Value: map[string]interface{}{
			"name": name,
		},
	})

	assert.NilError(t, err)

	envParameter, ok := param.(*EnvironmentVariableParameter)

	assert.Assert(t, ok, "parsed parameter is environment parameter")
	assert.Equal(t, name, envParameter.Name)
	assert.Assert(t, !envParameter.HasDefaultValue, "environment parameter should not have default")
}

func TestParseValueParameterWithDefault(t *testing.T) {
	name := "test"
	defaultValue := "this"

	param, err := parseEnvironmentValueParameter(parameter.ParameterParserContext{
		Coordinate: coordinate.Coordinate{
			Project: "projectA",
			Api:     "dashboard",
			Config:  "super-important",
		},
		ParameterName: "title",
		Value: map[string]interface{}{
			"name":    name,
			"default": defaultValue,
		},
	})

	assert.NilError(t, err)

	envParameter, ok := param.(*EnvironmentVariableParameter)

	assert.Assert(t, ok, "parsed parameter is environment parameter")
	assert.Equal(t, name, envParameter.Name)
	assert.Assert(t, envParameter.HasDefaultValue, "environment parameter should have default")
	assert.Equal(t, defaultValue, envParameter.DefaultValue)
}

func TestParseValueParameterMissingRequiredField(t *testing.T) {
	_, err := parseEnvironmentValueParameter(parameter.ParameterParserContext{
		Coordinate: coordinate.Coordinate{
			Project: "projectA",
			Api:     "dashboard",
			Config:  "super-important",
		},
		ParameterName: "title",
		Value: map[string]interface{}{
			"wrong":   "value",
			"default": "value",
		},
	})

	assert.Assert(t, err != nil, "error should be present")
}

func TestGetReferences(t *testing.T) {
	fixture := EnvironmentVariableParameter{
		Name:            "test",
		HasDefaultValue: false,
	}

	assert.Assert(t, len(fixture.GetReferences()) == 0, "environment parameter should not have references")
}

func TestResolveValue(t *testing.T) {
	name := "test"
	value := "this is a test"

	envvars.InstallFakeEnvironment(map[string]string{
		name: value,
	})

	defer envvars.InstallOsBased()

	fixture := EnvironmentVariableParameter{
		Name:            name,
		HasDefaultValue: false,
	}

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project: "projectA",
			Api:     "dashboard",
			Config:  "super-important",
		},

		ParameterName: "test",
	})

	assert.NilError(t, err)
	assert.Equal(t, value, result)
}

func TestResolveValueWithDefaultValue(t *testing.T) {
	name := "test"
	defaultValue := "this is the default"

	envvars.InstallFakeEnvironment(map[string]string{})

	defer envvars.InstallOsBased()

	fixture := EnvironmentVariableParameter{
		Name:            name,
		HasDefaultValue: true,
		DefaultValue:    defaultValue,
	}

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project: "projectA",
			Api:     "dashboard",
			Config:  "super-important",
		},

		ParameterName: "test",
	})

	assert.NilError(t, err)
	assert.Equal(t, defaultValue, result)
}
