//go:build unit

/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

func TestSortParameters(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	ownerParameterName := "owner"
	timeoutParameterName := "timeout"

	parameters := Parameters{
		NameParameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: ownerParameterName,
				},
			},
		},
		ownerParameterName:   &parameter.DummyParameter{},
		timeoutParameterName: &parameter.DummyParameter{},
	}

	c := &Config{
		Environment: "dev",
		Coordinate:  configCoordinates,
		Parameters:  parameters,
	}

	sortedParams, errs := getSortedParameters(c)

	assert.Len(t, errs, 0, "expected zero errors when sorting")
	assert.Equal(t, len(sortedParams), len(parameters), "the same number of parameters should be sorted")

	indexName := indexOfParam(t, sortedParams, NameParameter)
	indexOwner := indexOfParam(t, sortedParams, ownerParameterName)

	assert.Greaterf(t, indexName, indexOwner, "parameter name (index %d) must be after parameter owner (%d)", indexName, indexOwner)
}

func TestSortParametersShouldFailOnCircularDependency(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	ownerParameterName := "owner"

	parameters := Parameters{
		NameParameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: ownerParameterName,
				},
			},
		},
		ownerParameterName: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: NameParameter,
				},
			},
		},
	}

	c := &Config{
		Environment: "dev",
		Coordinate:  configCoordinates,
		Parameters:  parameters,
	}

	_, errs := getSortedParameters(c)

	require.NotEmpty(t, errs, "should fail")

	for _, err := range errs {
		wantErr := &CircularDependencyParameterSortError{}
		assert.ErrorAs(t, err, &wantErr)
	}
}

func indexOfParam(t *testing.T, params []parameter.NamedParameter, name string) int {
	for i, p := range params {
		if p.Name == name {
			return i
		}
	}

	t.Fatalf("no parameter with name `%s` found", name)
	return -1
}

func TestIsReferencing(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := parameter.NamedParameter{
		Name: "name",
		Parameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{Config: referencingConfig, Property: referencingProperty},
			},
		},
	}

	referencedParameter := parameter.NamedParameter{
		Name:      referencingProperty,
		Parameter: &parameter.DummyParameter{},
	}

	result := parameterReference(param, referencingConfig, referencedParameter)

	assert.True(t, result, "should reference parameter")
}

func TestIsReferencingShouldReturnFalseForNotReferencing(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := parameter.NamedParameter{
		Name: "name",
		Parameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   referencingConfig,
					Property: referencingProperty,
				},
			},
		},
	}

	referencedParameter := parameter.NamedParameter{
		Name:      "name",
		Parameter: &parameter.DummyParameter{},
	}

	result := parameterReference(param, referencingConfig, referencedParameter)

	assert.False(t, result, "should not reference parameter")
}

func TestIsReferencingShouldReturnFalseForParameterWithoutReferences(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	param := parameter.NamedParameter{
		Name:      "name",
		Parameter: &parameter.DummyParameter{},
	}

	referencedParameter := parameter.NamedParameter{
		Name:      "name",
		Parameter: &parameter.DummyParameter{},
	}

	result := parameterReference(param, referencingConfig, referencedParameter)

	assert.False(t, result, "should not reference parameter")
}

func TestIsReferencingOneOfSeveral(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := parameter.NamedParameter{
		Name: "name",
		Parameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{Config: referencingConfig, Property: "not our param"},
				{Config: referencingConfig, Property: "also not our param"},
				{Config: referencingConfig, Property: referencingProperty},
				{Config: referencingConfig, Property: "really not our param"},
			},
		},
	}

	referencedParameter := parameter.NamedParameter{
		Name:      referencingProperty,
		Parameter: &parameter.DummyParameter{},
	}

	result := parameterReference(param, referencingConfig, referencedParameter)

	assert.True(t, result, "should reference parameter")
}

func TestIsReferencingOneOfSeveralMaps(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencingProperty := "map.is.found"

	param := parameter.NamedParameter{
		Name: "name",
		Parameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{Config: referencingConfig, Property: "map.not.found"},
				{Config: referencingConfig, Property: "map.not.found.either"},
				{Config: referencingConfig, Property: referencingProperty},
			},
		},
	}

	referencedParameter := parameter.NamedParameter{
		Name: "map",
		Parameter: &value.ValueParameter{
			Value: map[string]any{
				"is": map[string]any{
					"found": true,
				},
			},
		},
	}

	result := parameterReference(param, referencingConfig, referencedParameter)

	assert.True(t, result, "should reference parameter")
}

func TestSearchValueParameterForKey(t *testing.T) {
	testCases := []struct {
		name       string
		key        string
		paramName  string
		paramValue interface{}
		expected   bool
	}{
		{
			"simple key does exist",
			"key",
			"key",
			"value",
			true,
		},
		{
			"simple key does not exist",
			"key",
			"other",
			"value",
			false,
		},
		{
			"nested key does exist",
			"key.keyy",
			"key",
			map[string]any{"keyy": "value"},
			true,
		},
		{
			"more nested key does exist",
			"key.keyy.keyyy",
			"key",
			map[string]interface{}{"keyy": map[string]interface{}{"keyyy": "value"}},
			true,
		},
		{
			"more nested key does exist",
			"key.keyy.keyyy",
			"key",
			map[string]interface{}{"keyy": map[string]interface{}{"other": "value"}},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			param := &value.ValueParameter{
				Value: tc.paramValue,
			}
			result := searchValueParameterForKey(tc.key, tc.paramName, param)
			assert.Equal(t, tc.expected, result)
		})
	}
}
