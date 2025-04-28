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
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

type entityLookup map[coordinate.Coordinate]entities.ResolvedEntity

func (e entityLookup) GetResolvedEntity(config coordinate.Coordinate) (entities.ResolvedEntity, bool) {
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
			Name: NameParameter,
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

	conf := Config{
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

	values, errs := conf.ResolveParameterValues(entityLookup{})

	assert.Empty(t, errs, "there should be no errors (errors: %s)", errs)
	assert.Equal(t, name, values[NameParameter])
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
			Name: NameParameter,
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

	conf := Config{
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

	_, errs := conf.ResolveParameterValues(entityLookup{})

	assert.NotEmpty(t, errs, "there should be errors (no errors: %d)", len(errs))
}

func TestConfigHasRefTo(t *testing.T) {

	parameters := []parameter.NamedParameter{
		{
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{Config: coordinate.Coordinate{Project: "p", Type: api.ManagementZone, ConfigId: "z"}},
					{Config: coordinate.Coordinate{Project: "p", Type: api.ApplicationWeb, ConfigId: "y"}},
				},
			},
		},
	}
	conf := Config{Parameters: toParameterMap(parameters)}
	assert.True(t, conf.HasRefTo(api.ManagementZone))
	assert.True(t, conf.HasRefTo(api.ApplicationWeb))
	assert.False(t, conf.HasRefTo(api.Dashboard))
}

func TestResolveParameterValuesShouldFailWhenReferencingSkippedConfig(t *testing.T) {
	referenceCoordinate := coordinate.Coordinate{
		Project:  "project1",
		Type:     "management-zone",
		ConfigId: "zone1",
	}

	parameters := []parameter.NamedParameter{
		{
			Name: NameParameter,
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

	conf := Config{
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

	lookup := entityLookup{
		referenceCoordinate: {
			EntityName: "zone1",
			Coordinate: referenceCoordinate,
			Properties: parameter.Properties{},
			Skip:       true,
		},
	}

	_, errs := conf.ResolveParameterValues(lookup)

	assert.NotEmpty(t, errs, "there should be errors (no errors: %d)", len(errs))
}

func TestResolveParameterValuesShouldFailWhenParameterResolveReturnsError(t *testing.T) {
	parameters := []parameter.NamedParameter{
		{
			Name: NameParameter,
			Parameter: &parameter.DummyParameter{
				Err: errors.New("error"),
			},
		},
	}

	conf := Config{
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

	_, errs := conf.ResolveParameterValues(entityLookup{})

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

	lookup := entityLookup{
		referencedConfigCoordinates: {
			EntityName: "zone1",
			Coordinate: referencedConfigCoordinates,
			Properties: parameter.Properties{
				"name": "test",
			},
			Skip: false,
		},
	}

	errs := validateParameterReferences(configCoordinates, "", "", lookup, "managementZoneName", param)

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

	errs := validateParameterReferences(configCoordinates, "", "", entityLookup{}, paramName, param)

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

	lookup := entityLookup{
		referencedConfigCoordinates: {
			EntityName: "zone1",
			Coordinate: referencedConfigCoordinates,
			Properties: parameter.Properties{},
			Skip:       true,
		},
	}

	errs := validateParameterReferences(configCoordinates, "", "", lookup, "managementZoneName", param)

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

	errs := validateParameterReferences(configCoordinates, "", "", entityLookup{}, "managementZoneName", param)

	assert.NotEmpty(t, errs, "should return errors (no errors: %d)", len(errs))
}

func TestConfig_Render_ErrorsOnInvalidJson(t *testing.T) {
	c := Config{
		Template:   generateFaultyTemplate(t),
		Coordinate: coordinate.Coordinate{},
		Type:       SettingsType{},
	}

	_, err := c.Render(nil)
	assert.ErrorAs(t, err, &configErrors.InvalidJsonError{})
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
	templ := template.NewInMemoryTemplate("deploy_test-"+newUUID.String(), "{}")
	return templ
}

func generateFaultyTemplate(t *testing.T) template.Template {
	newUUID, err := uuid.NewUUID()
	require.NoError(t, err)
	return template.NewInMemoryTemplate("deploy_test-"+newUUID.String(), "{")
}

func TestConfigMethodsAreNilSafe(t *testing.T) {

	t.Run("References", func(t *testing.T) {
		var c *Config
		c = nil
		assert.NotPanics(t, func() {
			_ = c.References()
		})
	})

	t.Run("Render", func(t *testing.T) {
		var c *Config
		c = nil
		assert.NotPanics(t, func() {
			_, _ = c.Render(nil)
		})
	})

	t.Run("Render - nil template", func(t *testing.T) {
		var c Config
		assert.NotPanics(t, func() {
			_, _ = c.Render(nil)
		})
	})

	t.Run("ResolveParameterValues", func(t *testing.T) {
		var c *Config
		c = nil
		assert.NotPanics(t, func() {
			_, _ = c.ResolveParameterValues(nil)
		})
	})
}
