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

package reference

import (
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"gotest.tools/assert"
)

func TestParseReferenceParameter(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	property := "title"

	param, err := parseReferenceParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"project":    project,
			"configType": configType,
			"configId":   config,
			"property":   property,
		},
	})

	assert.NilError(t, err)

	refParam, ok := param.(*ReferenceParameter)

	assert.Assert(t, ok, "parsed parameter should reference parameter")
	assert.Equal(t, refParam.GetType(), "reference")

	assert.Equal(t, project, refParam.Config.Project)
	assert.Equal(t, configType, refParam.Config.Type)
	assert.Equal(t, config, refParam.Config.ConfigId)
	assert.Equal(t, property, refParam.Property)
}

func TestParseReferenceParameterShouldFillValuesFromCurrentConfigIfMissing(t *testing.T) {
	project := "projectA"
	configType := "dashboard"
	config := "super-important"
	property := "title"

	param, err := parseReferenceParameter(parameter.ParameterParserContext{
		Coordinate: coordinate.Coordinate{Project: project, Type: configType, ConfigId: config},
		Value: map[string]interface{}{
			"property": property,
		},
	})

	assert.NilError(t, err)

	refParam, ok := param.(*ReferenceParameter)

	assert.Assert(t, ok, "parsed parameter should be reference parameter")
	assert.Equal(t, project, refParam.Config.Project)
	assert.Equal(t, configType, refParam.Config.Type)
	assert.Equal(t, config, refParam.Config.ConfigId)
	assert.Equal(t, property, refParam.Property)
}

func TestParseReferenceParameterShouldFailIfPropertyIsMissing(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"

	_, err := parseReferenceParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"project":    project,
			"configType": configType,
			"config":     config,
		},
	})

	assert.Assert(t, err != nil, "should return error")
}

func TestParseReferenceParameterShouldFailIfProjectIsSetButApiIsNot(t *testing.T) {
	project := "projectB"
	config := "alerting"
	property := "title"

	_, err := parseReferenceParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"project":  project,
			"config":   config,
			"property": property,
		},
	})

	assert.Assert(t, err != nil, "should return error")
}

func TestParseReferenceParameterShouldFailIfProjectIsSetButApiAndConfigAreNot(t *testing.T) {
	project := "projectB"
	property := "title"

	_, err := parseReferenceParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"project":  project,
			"property": property,
		},
	})

	assert.Assert(t, err != nil, "should return error")
}

func TestParseReferenceParameterShouldFailIfProjectAndApiAreSetButConfigIsNot(t *testing.T) {
	project := "projectB"
	configType := "alerting"
	property := "title"

	_, err := parseReferenceParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"project":    project,
			"configType": configType,
			"property":   property,
		},
	})

	assert.Assert(t, err != nil, "should return an error")
}

func TestParseReferenceParameterShouldFailIfApiIsSetButConfigIsNot(t *testing.T) {
	configType := "alerting"
	property := "title"

	_, err := parseReferenceParameter(parameter.ParameterParserContext{
		Value: map[string]interface{}{
			"configType": configType,
			"property":   property,
		},
	})

	assert.Assert(t, err != nil, "should return error")
}

func TestGetReferences(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	property := "title"

	fixture := New(project, configType, config, property)

	refs := fixture.GetReferences()

	assert.Assert(t, len(refs) == 1, "reference parameter should return a single reference")

	ref := refs[0]

	assert.Equal(t, project, ref.Config.Project)
	assert.Equal(t, configType, ref.Config.Type)
	assert.Equal(t, config, ref.Config.ConfigId)
	assert.Equal(t, property, ref.Property)
}

func TestResolveValue(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	property := "title"
	propertyValue := "THIS IS THE TITLE"
	referenceCoordinate := coordinate.Coordinate{Project: project, Type: configType, ConfigId: config}

	fixture := New(project, configType, config, property)

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project:  "projectA",
			Type:     "dashboard",
			ConfigId: "super-important",
		},
		ResolvedEntities: map[coordinate.Coordinate]parameter.ResolvedEntity{
			referenceCoordinate: {
				Coordinate: referenceCoordinate,
				Properties: map[string]interface{}{
					property: propertyValue,
				},
			},
		},
	})

	assert.NilError(t, err)
	assert.Equal(t, propertyValue, result)
}

func TestResolveValueOnPropertyInSameConfig(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	property := "title"
	propertyValue := "THIS IS THE TITLE"
	referenceCoordinate := coordinate.Coordinate{Project: project, Type: configType, ConfigId: config}

	fixture := New(project, configType, config, property)

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: referenceCoordinate,
		ResolvedParameterValues: map[string]interface{}{
			property: propertyValue,
		},
	})

	assert.NilError(t, err)
	assert.Equal(t, propertyValue, result)
}

func TestResolveValuePropertyNotYetResolved(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	property := "title"

	fixture := New(project, configType, config, property)

	_, err := fixture.ResolveValue(parameter.ResolveContext{})

	assert.Assert(t, err != nil, "should return an error")
}

func TestResolveValueOwnPropertyNotYetResolved(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	property := "title"
	referenceCoordinate := coordinate.Coordinate{Project: project, Type: configType, ConfigId: config}

	fixture := New(project, configType, config, property)

	_, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: referenceCoordinate,
	})

	assert.Assert(t, err != nil, "should return an error")
}

func TestWriteReferenceParameter(t *testing.T) {
	refProject := "projectB"
	refType := "alerting-profile"
	refConfig := "alerting"
	refProperty := "title"
	refParam := New(refProject, refType, refConfig, refProperty)

	coord := coordinate.Coordinate{
		Project:  "projectA",
		Type:     "dashboard",
		ConfigId: "hansi",
	}

	context := parameter.ParameterWriterContext{Parameter: refParam, Coordinate: coord}

	result, err := writeReferenceParameter(context)
	assert.NilError(t, err)

	assert.Equal(t, len(result), 4)

	project, ok := result["project"]
	assert.Assert(t, ok, "should have parameter project")
	assert.Equal(t, project, refProject)

	api, ok := result["configType"]
	assert.Assert(t, ok, "should have parameter configType")
	assert.Equal(t, api, refType)

	config, ok := result["configId"]
	assert.Assert(t, ok, "should have parameter configId")
	assert.Equal(t, config, refConfig)

	property, ok := result["property"]
	assert.Assert(t, ok, "should have parameter property")
	assert.Equal(t, property, refProperty)

}

func TestWriteReferenceParameterOnMatchingProject(t *testing.T) {
	refProject := "projectA"
	refApi := "alerting-profile"
	refConfig := "alerting"
	refProperty := "title"
	refParam := New(refProject, refApi, refConfig, refProperty)

	coord := coordinate.Coordinate{
		Project:  refProject,
		Type:     "dashboard",
		ConfigId: "hansi",
	}

	context := parameter.ParameterWriterContext{Parameter: refParam, Coordinate: coord}

	result, err := writeReferenceParameter(context)
	assert.NilError(t, err)

	assert.Equal(t, len(result), 3)

	api, ok := result["configType"]
	assert.Assert(t, ok, "should have parameter configType")
	assert.Equal(t, api, refApi)

	config, ok := result["configId"]
	assert.Assert(t, ok, "should have parameter configId")
	assert.Equal(t, config, refConfig)

	property, ok := result["property"]
	assert.Assert(t, ok, "should have parameter property")
	assert.Equal(t, property, refProperty)

}

func TestWriteReferenceParameterOnMatchingApi(t *testing.T) {
	refProject := "projectA"
	refApi := "alerting-profile"
	refConfig := "alerting"
	refProperty := "title"
	refParam := New(refProject, refApi, refConfig, refProperty)

	coord := coordinate.Coordinate{
		Project:  refProject,
		Type:     refApi,
		ConfigId: "hansi",
	}

	context := parameter.ParameterWriterContext{Parameter: refParam, Coordinate: coord}

	result, err := writeReferenceParameter(context)
	assert.NilError(t, err)

	assert.Equal(t, len(result), 2)

	config, ok := result["configId"]
	assert.Assert(t, ok, "should have parameter configId")
	assert.Equal(t, config, refConfig)

	property, ok := result["property"]
	assert.Assert(t, ok, "should have parameter property")
	assert.Equal(t, property, refProperty)

}

func TestWriteReferenceParameterOnMatchingConfig(t *testing.T) {
	refProject := "projectA"
	refApi := "alerting-profile"
	refConfig := "alerting"
	refProperty := "title"
	refParam := New(refProject, refApi, refConfig, refProperty)

	coord := coordinate.Coordinate{
		Project:  refProject,
		Type:     refApi,
		ConfigId: refConfig,
	}

	context := parameter.ParameterWriterContext{Parameter: refParam, Coordinate: coord}

	result, err := writeReferenceParameter(context)
	assert.NilError(t, err)

	assert.Equal(t, len(result), 1)

	property, ok := result["property"]
	assert.Assert(t, ok, "should have parameter property")
	assert.Equal(t, property, refProperty)

}

func TestWriteCompoundParameterErrorOnNonCompoundParameter(t *testing.T) {
	context := parameter.ParameterWriterContext{Parameter: &value.ValueParameter{}}

	_, err := writeReferenceParameter(context)
	assert.Assert(t, err != nil, "expected an error writing wrong parameter type")
}
