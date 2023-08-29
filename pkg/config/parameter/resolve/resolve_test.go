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

package resolve

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

type entityLookup map[coordinate.Coordinate]config.ResolvedEntity

func (e entityLookup) GetResolvedEntity(config coordinate.Coordinate) (config.ResolvedEntity, bool) {
	ent, f := e[config]
	return ent, f
}

func (e entityLookup) GetResolvedProperty(coordinate coordinate.Coordinate, propertyName string) (any, bool) {
	if ent, f := e.GetResolvedEntity(coordinate); f {
		if prop, f := ent.Properties[propertyName]; f {
			return prop, true
		}
	}

	return nil, false
}

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

	entities := entityLookup{}

	values, errs := ParameterValues(&conf, entities)

	assert.Empty(t, errs, "there should be no errors (errors: %s)", errs)
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

	entities := entityLookup{}

	_, errs := ParameterValues(&conf, entities)

	assert.NotEmpty(t, errs, "there should be errors (no errors: %d)", len(errs))
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

	entities := entityLookup{
		referenceCoordinate: {
			EntityName: "zone1",
			Coordinate: referenceCoordinate,
			Properties: parameter.Properties{},
			Skip:       true,
		},
	}

	_, errs := ParameterValues(&conf, entities)

	assert.NotEmpty(t, errs, "there should be errors (no errors: %d)", len(errs))
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

	entities := entityLookup{}

	_, errs := ParameterValues(&conf, entities)

	assert.NotEmpty(t, errs, "there should be errors (no : %d)", len(errs))
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

	entities := entityLookup{
		referencedConfigCoordinates: {
			EntityName: "zone1",
			Coordinate: referencedConfigCoordinates,
			Properties: parameter.Properties{
				"name": "test",
			},
			Skip: false,
		},
	}

	errs := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.Empty(t, errs, "should not return errors (no errors: %d)", len(errs))
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

	entities := entityLookup{}

	errs := validateParameterReferences(configCoordinates, "", "", entities, paramName, param)

	assert.NotEmpty(t, errs, "should not errors (no errors: %d)", len(errs))
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

	entities := entityLookup{
		referencedConfigCoordinates: {
			EntityName: "zone1",
			Coordinate: referencedConfigCoordinates,
			Properties: parameter.Properties{},
			Skip:       true,
		},
	}

	errs := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.NotEmpty(t, errs, "should return errors (no errors: %d)", len(errs))
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

	entities := entityLookup{}

	errs := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.NotEmpty(t, errs, "should return errors (no errors: %d)", len(errs))
}

func toParameterMap(params []parameter.NamedParameter) map[string]parameter.Parameter {
	result := make(map[string]parameter.Parameter)

	for _, p := range params {
		result[p.Name] = p.Parameter
	}

	return result
}

func generateDummyTemplate(t *testing.T) template.Template {
	newUUID, err := uuid.NewUUID()
	assert.NoError(t, err)
	templ := template.CreateTemplateFromString("deploy_test-"+newUUID.String(), "{}")
	return templ
}