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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
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

	require.NoError(t, err)

	refParam, ok := param.(*ReferenceParameter)

	require.True(t, ok, "parsed parameter should reference parameter")
	assert.Equal(t, "reference", refParam.GetType())

	assert.Equal(t, refParam.Config.Project, project)
	assert.Equal(t, refParam.Config.Type, configType)
	assert.Equal(t, refParam.Config.ConfigId, config)
	assert.Equal(t, refParam.Property, property)
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

	require.NoError(t, err)

	refParam, ok := param.(*ReferenceParameter)

	require.True(t, ok, "parsed parameter should be reference parameter")
	assert.Equal(t, refParam.Config.Project, project)
	assert.Equal(t, refParam.Config.Type, configType)
	assert.Equal(t, refParam.Config.ConfigId, config)
	assert.Equal(t, refParam.Property, property)
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

	require.Error(t, err, "should return error")
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

	require.Error(t, err, "should return error")
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

	require.Error(t, err, "should return error")
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

	require.Error(t, err, "should return an error")
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

	require.Error(t, err, "should return error")
}

func TestGetReferences(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	property := "title"

	fixture := New(project, configType, config, property)

	refs := fixture.GetReferences()

	require.Len(t, refs, 1, "reference parameter should return a single reference")

	ref := refs[0]

	assert.Equal(t, project, ref.Config.Project)
	assert.Equal(t, configType, ref.Config.Type)
	assert.Equal(t, config, ref.Config.ConfigId)
	assert.Equal(t, property, ref.Property)
}

type testResolver struct {
	props map[coordinate.Coordinate]map[string]any
}

func (t testResolver) GetResolvedProperty(configCoordinate coordinate.Coordinate, propertyName string) (any, bool) {
	if e, f := t.props[configCoordinate]; f {
		if v, f := e[propertyName]; f {
			return v, true
		}
	}
	return nil, false
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
		PropertyResolver: testResolver{
			map[coordinate.Coordinate]map[string]any{
				referenceCoordinate: {
					property: propertyValue,
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, propertyValue, result)
}

func TestResolveComplexValueMap(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	referenceCoordinate := coordinate.Coordinate{Project: project, Type: configType, ConfigId: config}

	fixture := New(project, configType, config, "keys.key")

	entityMap := entities.New()
	entityMap.Put(entities.ResolvedEntity{
		Coordinate: referenceCoordinate,
		Properties: map[string]any{
			"keys": map[any]any{"key": "value"},
		},
	})

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project:  "projectA",
			Type:     "dashboard",
			ConfigId: "super-important",
		},
		PropertyResolver: entityMap,
	})

	require.NoError(t, err)
	assert.Equal(t, "value", result)
}

func TestResolveComplexValueMapInSameConfig(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"

	fixture := New(project, configType, config, "keys.key")

	entityMap := entities.New()

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project:  project,
			Type:     configType,
			ConfigId: config,
		},
		PropertyResolver: entityMap,
		ResolvedParameterValues: map[string]any{
			"keys": map[any]any{"key": "value"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "value", result)
}

func TestResolveComplexValueNestedMap(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	referenceCoordinate := coordinate.Coordinate{Project: project, Type: configType, ConfigId: config}

	fixture := New(project, configType, config, "keys.key.another.one.more")

	entityMap := entities.New()
	entityMap.Put(entities.ResolvedEntity{
		Coordinate: referenceCoordinate,
		Properties: map[string]any{
			"keys": map[any]any{
				"key": map[any]any{
					"another": map[any]any{
						"one": map[any]any{
							"more": "value",
						},
					},
				},
			},
		},
	})

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project:  "projectA",
			Type:     "dashboard",
			ConfigId: "super-important",
		},
		PropertyResolver: entityMap,
	})

	require.NoError(t, err)
	assert.Equal(t, "value", result)
}

func TestResolveComplexValueNestedMapInSameConfig(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"

	fixture := New(project, configType, config, "keys.key.another.one.more")

	entityMap := entities.New()

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project:  project,
			Type:     configType,
			ConfigId: config,
		},
		PropertyResolver: entityMap,
		ResolvedParameterValues: map[string]any{
			"keys": map[any]any{
				"key": map[any]any{
					"another": map[any]any{
						"one": map[any]any{
							"more": "value",
						},
					},
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "value", result)
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

	require.NoError(t, err)
	require.Equal(t, propertyValue, result)
}

func TestResolveValuePropertyNotYetResolved(t *testing.T) {
	project := "projectB"
	configType := "alerting-profile"
	config := "alerting"
	property := "title"

	fixture := New(project, configType, config, property)

	_, err := fixture.ResolveValue(parameter.ResolveContext{PropertyResolver: testResolver{}})

	require.Error(t, err, "should return an error")
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

	assert.Error(t, err, "should return an error")
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
	require.NoError(t, err)

	require.Len(t, result, 4)

	project, ok := result["project"]
	require.True(t, ok, "should have parameter project")
	assert.Equal(t, refProject, project)

	api, ok := result["configType"]
	require.True(t, ok, "should have parameter configType")
	assert.Equal(t, refType, api)

	config, ok := result["configId"]
	require.True(t, ok, "should have parameter configId")
	assert.Equal(t, refConfig, config)

	property, ok := result["property"]
	require.True(t, ok, "should have parameter property")
	require.Equal(t, property, refProperty)

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
	require.NoError(t, err)

	require.Len(t, result, 3)

	api, ok := result["configType"]
	require.True(t, ok, "should have parameter configType")
	assert.Equal(t, refApi, api)

	config, ok := result["configId"]
	require.True(t, ok, "should have parameter configId")
	assert.Equal(t, refConfig, config)

	property, ok := result["property"]
	require.True(t, ok, "should have parameter property")
	assert.Equal(t, refProperty, property)

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
	require.NoError(t, err)

	require.Len(t, result, 2)

	config, ok := result["configId"]
	require.True(t, ok, "should have parameter configId")
	assert.Equal(t, refConfig, config)

	property, ok := result["property"]
	require.True(t, ok, "should have parameter property")
	assert.Equal(t, refProperty, property)

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
	require.NoError(t, err)

	require.Len(t, result, 1)

	property, ok := result["property"]
	require.True(t, ok, "should have parameter property")
	assert.Equal(t, refProperty, property)

}

func TestWriteCompoundParameterErrorOnNonCompoundParameter(t *testing.T) {
	context := parameter.ParameterWriterContext{Parameter: &value.ValueParameter{}}

	_, err := writeReferenceParameter(context)
	require.Error(t, err, "expected an error writing wrong parameter type")
}
