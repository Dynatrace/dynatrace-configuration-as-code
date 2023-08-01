//go:build unit

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

package deploy

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"gotest.tools/assert"
	"testing"
)

func TestResolveParameterValues(t *testing.T) {
	name := "test"
	owner := "hansi"
	ownerParameterName := "owner"
	timeout := 5
	timeoutParameterName := "timeout"
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: name,
			},
		},
		{
			Name: ownerParameterName,
			Parameter: &parameter.DummyParameter{
				Value: owner,
			},
		},
		{
			Name: timeoutParameterName,
			Parameter: &parameter.DummyParameter{
				Value: timeout,
			},
		},
	}

	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}

	entities := newEntityMap(api.NewAPIs())

	values, errors := ResolveParameterValues(&conf, entities, parameters)

	assert.Assert(t, len(errors) == 0, "there should be no errors (errors: %s)", errors)
	assert.Equal(t, name, values[config.NameParameter])
	assert.Equal(t, owner, values[ownerParameterName])
	assert.Equal(t, timeout, values[timeoutParameterName])
}

func TestResolveParameterValuesShouldFailWhenReferencingNonExistingConfig(t *testing.T) {
	nonExistingConfig := coordinate.Coordinate{
		Project:  "non-existing",
		Type:     "management-zone",
		ConfigId: "zone1",
	}
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   nonExistingConfig,
						Property: "name",
					},
				},
			},
		},
	}

	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}

	entities := newEntityMap(api.NewAPIs())

	_, errors := ResolveParameterValues(&conf, entities, parameters)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestResolveParameterValuesShouldFailWhenReferencingSkippedConfig(t *testing.T) {
	referenceCoordinate := coordinate.Coordinate{
		Project:  "project1",
		Type:     "management-zone",
		ConfigId: "zone1",
	}

	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   referenceCoordinate,
						Property: "name",
					},
				},
			},
		},
	}

	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}

	entities := &entityMap{
		resolvedEntities: map[coordinate.Coordinate]ResolvedEntity{
			referenceCoordinate: {
				EntityName: "zone1",
				Coordinate: referenceCoordinate,
				Properties: parameter.Properties{},
				Skip:       true,
			},
		},
	}

	_, errors := ResolveParameterValues(&conf, entities, parameters)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestResolveParameterValuesShouldFailWhenParameterResolveReturnsError(t *testing.T) {
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Err: errors.New("error"),
			},
		},
	}

	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}

	entities := newEntityMap(api.NewAPIs())

	_, errors := ResolveParameterValues(&conf, entities, parameters)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestValidateParameterReferences(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencedConfigCoordinates := coordinate.Coordinate{
		Project:  "project2",
		Type:     "management-zone",
		ConfigId: "zone1",
	}

	param := &parameter.DummyParameter{
		Value: "test",
		References: []parameter.ParameterReference{
			{
				Config:   configCoordinates,
				Property: "name",
			},
			{
				Config:   referencedConfigCoordinates,
				Property: "name",
			},
		},
	}

	entities := &entityMap{
		resolvedEntities: map[coordinate.Coordinate]ResolvedEntity{
			referencedConfigCoordinates: {
				EntityName: "zone1",
				Coordinate: referencedConfigCoordinates,
				Properties: parameter.Properties{
					"name": "test",
				},
				Skip: false,
			},
		},
	}

	errors := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.Assert(t, len(errors) == 0, "should not return errors (no errors: %d)", len(errors))
}

func TestValidateParameterReferencesShouldFailWhenReferencingSelf(t *testing.T) {
	paramName := "name"

	configCoordinates := coordinate.Coordinate{
		Project:  "project1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	param := &parameter.DummyParameter{
		Value: "test",
		References: []parameter.ParameterReference{
			{
				Config:   configCoordinates,
				Property: paramName,
			},
		},
	}

	entities := newEntityMap(api.NewAPIs())

	errors := validateParameterReferences(configCoordinates, "", "", entities, paramName, param)

	assert.Assert(t, len(errors) > 0, "should not errors (no errors: %d)", len(errors))
}

func TestValidateParameterReferencesShouldFailWhenReferencingSkippedConfig(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencedConfigCoordinates := coordinate.Coordinate{
		Project:  "project2",
		Type:     "management-zone",
		ConfigId: "zone1",
	}

	param := &parameter.DummyParameter{
		Value: "test",
		References: []parameter.ParameterReference{
			{
				Config:   referencedConfigCoordinates,
				Property: "name",
			},
		},
	}

	entities := &entityMap{
		resolvedEntities: map[coordinate.Coordinate]ResolvedEntity{
			referencedConfigCoordinates: {
				EntityName: "zone1",
				Coordinate: referencedConfigCoordinates,
				Properties: parameter.Properties{},
				Skip:       true,
			},
		},
	}

	errors := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.Assert(t, len(errors) > 0, "should return errors (no errors: %d)", len(errors))
}

func TestValidateParameterReferencesShouldFailWhenReferencingUnknownConfig(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencedConfigCoordinates := coordinate.Coordinate{
		Project:  "project2",
		Type:     "management-zone",
		ConfigId: "zone1",
	}

	param := &parameter.DummyParameter{
		Value: "test",
		References: []parameter.ParameterReference{
			{
				Config:   referencedConfigCoordinates,
				Property: "name",
			},
		},
	}

	entities := newEntityMap(api.NewAPIs())

	errors := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.Assert(t, len(errors) > 0, "should return errors (no errors: %d)", len(errors))
}

func TestExtractConfigName(t *testing.T) {
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	name := "test"

	properties := parameter.Properties{
		config.NameParameter: name,
	}

	val, err := extractConfigName(&conf, properties)

	assert.NilError(t, err)
	assert.Equal(t, name, val)
}

func TestExtractConfigNameShouldFailOnMissingName(t *testing.T) {
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	properties := parameter.Properties{}

	_, err := extractConfigName(&conf, properties)

	assert.Assert(t, err != nil, "error should not be nil (error val: %s)", err)
}

func TestExtractConfigNameShouldFailOnNameWithNonStringType(t *testing.T) {
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	properties := parameter.Properties{
		config.NameParameter: 1,
	}

	_, err := extractConfigName(&conf, properties)

	assert.Assert(t, err != nil, "error should not be nil (error val: %s)", err)
}
