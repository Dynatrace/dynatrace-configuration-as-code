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

package v2

import (
	"errors"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/client"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/test"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestResolveParameterValues(t *testing.T) {
	name := "test"
	owner := "hansi"
	ownerParameterName := "owner"
	timeout := 5
	timeoutParameterName := "timeout"
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				Value: name,
			},
		},
		{
			Name: ownerParameterName,
			Parameter: &test.DummyParameter{
				Value: owner,
			},
		},
		{
			Name: timeoutParameterName,
			Parameter: &test.DummyParameter{
				Value: timeout,
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}

	values, errors := resolveParameterValues(client, &conf, entities, parameters, false)

	assert.Assert(t, len(errors) == 0, "there should be no errors (errors: %s)", errors)
	assert.Equal(t, name, values[config.NameParameter])
	assert.Equal(t, owner, values[ownerParameterName])
	assert.Equal(t, timeout, values[timeoutParameterName])
}

func TestResolveParameterValuesShouldFailWhenReferencingNonExistingConfig(t *testing.T) {
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config: coordinate.Coordinate{
							Project: "non-existing",
							Api:     "management-zone",
							Config:  "zone1",
						},
						Property: "name",
					},
				},
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}

	_, errors := resolveParameterValues(client, &conf, entities, parameters, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestResolveParameterValuesShouldFailWhenReferencingSkippedConfig(t *testing.T) {
	referenceCoordinate := coordinate.Coordinate{
		Project: "project1",
		Api:     "management-zone",
		Config:  "zone1",
	}

	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   referenceCoordinate,
						Property: "name",
					},
				},
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{
		referenceCoordinate: {
			EntityName: "zone1",
			Coordinate: referenceCoordinate,
			Properties: parameter.Properties{},
			Skip:       true,
		},
	}

	_, errors := resolveParameterValues(client, &conf, entities, parameters, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestResolveParameterValuesShouldFailWhenParameterResolveReturnsError(t *testing.T) {
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				Err: errors.New("error"),
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}

	_, errors := resolveParameterValues(client, &conf, entities, parameters, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestValidateParameterReferences(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project: "project1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}

	referencedConfigCoordinates := coordinate.Coordinate{
		Project: "project2",
		Api:     "management-zone",
		Config:  "zone1",
	}

	param := &test.DummyParameter{
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

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{
		referencedConfigCoordinates: {
			EntityName: "zone1",
			Coordinate: referencedConfigCoordinates,
			Properties: parameter.Properties{
				"name": "test",
			},
			Skip: false,
		},
	}

	errors := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.Assert(t, len(errors) == 0, "should not return errors (no errors: %d)", len(errors))
}

func TestValidateParameterReferencesShouldFailWhenReferencingSelf(t *testing.T) {
	paramName := "name"

	configCoordinates := coordinate.Coordinate{
		Project: "project1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}

	param := &test.DummyParameter{
		Value: "test",
		References: []parameter.ParameterReference{
			{
				Config:   configCoordinates,
				Property: paramName,
			},
		},
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}

	errors := validateParameterReferences(configCoordinates, "", "", entities, paramName, param)

	assert.Assert(t, len(errors) > 0, "should not errors (no errors: %d)", len(errors))
}

func TestValidateParameterReferencesShouldFailWhenReferencingSkippedConfig(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project: "project1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}

	referencedConfigCoordinates := coordinate.Coordinate{
		Project: "project2",
		Api:     "management-zone",
		Config:  "zone1",
	}

	param := &test.DummyParameter{
		Value: "test",
		References: []parameter.ParameterReference{
			{
				Config:   referencedConfigCoordinates,
				Property: "name",
			},
		},
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{
		referencedConfigCoordinates: {
			EntityName: "zone1",
			Coordinate: referencedConfigCoordinates,
			Properties: parameter.Properties{},
			Skip:       true,
		},
	}

	errors := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.Assert(t, len(errors) > 0, "should return errors (no errors: %d)", len(errors))
}

func TestValidateParameterReferencesShouldFailWhenReferencingUnknownConfig(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project: "project1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}

	referencedConfigCoordinates := coordinate.Coordinate{
		Project: "project2",
		Api:     "management-zone",
		Config:  "zone1",
	}

	param := &test.DummyParameter{
		Value: "test",
		References: []parameter.ParameterReference{
			{
				Config:   referencedConfigCoordinates,
				Property: "name",
			},
		},
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}

	errors := validateParameterReferences(configCoordinates, "", "", entities, "managementZoneName", param)

	assert.Assert(t, len(errors) > 0, "should return errors (no errors: %d)", len(errors))
}

func TestExtractConfigName(t *testing.T) {
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		References:  []coordinate.Coordinate{},
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
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		References:  []coordinate.Coordinate{},
		Skip:        false,
	}

	properties := parameter.Properties{}

	_, err := extractConfigName(&conf, properties)

	assert.Assert(t, err != nil, "error should not be null (error val: %s)", err)
}

func TestExtractConfigNameShouldFailOnNameWithNonStringType(t *testing.T) {
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		References:  []coordinate.Coordinate{},
		Skip:        false,
	}

	properties := parameter.Properties{
		config.NameParameter: 1,
	}

	_, err := extractConfigName(&conf, properties)

	assert.Assert(t, err != nil, "error should not be null (error val: %s)", err)
}

func TestDeployConfig(t *testing.T) {
	name := "test"
	owner := "hansi"
	ownerParameterName := "owner"
	timeout := 5
	timeoutParameterName := "timeout"
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				Value: name,
			},
		},
		{
			Name: ownerParameterName,
			Parameter: &test.DummyParameter{
				Value: owner,
			},
		},
		{
			Name: timeoutParameterName,
			Parameter: &test.DummyParameter{
				Value: timeout,
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}
	apis := map[string]api.Api{
		"dashboard": api.NewStandardApi("dashboard", "dashboard"),
	}

	knownEntityNames := knownEntityMap{}

	resolvedEntity, errors := deployConfig(client, apis, entities, knownEntityNames, &conf, false)

	assert.Assert(t, len(errors) == 0, "there should be no errors (no errors: %d, %s)", len(errors), errors)
	assert.Equal(t, name, resolvedEntity.EntityName, "%s == %s")
	assert.Equal(t, conf.Coordinate, resolvedEntity.Coordinate)
	assert.Equal(t, name, resolvedEntity.Properties[config.NameParameter])
	assert.Equal(t, owner, resolvedEntity.Properties[ownerParameterName])
	assert.Equal(t, timeout, resolvedEntity.Properties[timeoutParameterName])
	assert.Equal(t, false, resolvedEntity.Skip)
}

func TestDeployConfigShouldFailOnAnAlreadyKnownEntityName(t *testing.T) {
	name := "test"
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				Value: name,
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}
	dashboardApiId := "dashboard"
	apis := map[string]api.Api{
		dashboardApiId: api.NewStandardApi(dashboardApiId, "dashboard"),
	}

	knownEntityNames := knownEntityMap{
		dashboardApiId: {
			name: struct{}{},
		},
	}

	_, errors := deployConfig(client, apis, entities, knownEntityNames, &conf, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailCyclicParameterDependencies(t *testing.T) {
	ownerParameterName := "owner"
	configCoordinates := coordinate.Coordinate{
		Project: "project1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}

	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   configCoordinates,
						Property: ownerParameterName,
					},
				},
			},
		},
		{
			Name: ownerParameterName,
			Parameter: &test.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   configCoordinates,
						Property: config.NameParameter,
					},
				},
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}
	dashboardApiId := "dashboard"
	apis := map[string]api.Api{
		dashboardApiId: api.NewStandardApi(dashboardApiId, "dashboard"),
	}

	knownEntityNames := knownEntityMap{}

	_, errors := deployConfig(client, apis, entities, knownEntityNames, &conf, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailOnMissingNameParameter(t *testing.T) {
	parameters := []topologysort.ParameterWithName{}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}
	dashboardApiId := "dashboard"
	apis := map[string]api.Api{
		dashboardApiId: api.NewStandardApi(dashboardApiId, "dashboard"),
	}

	knownEntityNames := knownEntityMap{}

	_, errors := deployConfig(client, apis, entities, knownEntityNames, &conf, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailOnReferenceOnUnknownConfig(t *testing.T) {
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config: coordinate.Coordinate{
							Project: "project2",
							Api:     "dashboard",
							Config:  "dashboard",
						},
						Property: "managementZoneId",
					},
				},
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}
	dashboardApiId := "dashboard"
	apis := map[string]api.Api{
		dashboardApiId: api.NewStandardApi(dashboardApiId, "dashboard"),
	}

	knownEntityNames := knownEntityMap{}

	_, errors := deployConfig(client, apis, entities, knownEntityNames, &conf, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailOnReferenceOnSkipConfig(t *testing.T) {
	referenceCoordinates := coordinate.Coordinate{
		Project: "project2",
		Api:     "dashboard",
		Config:  "dashboard",
	}

	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &test.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   referenceCoordinates,
						Property: "managementZoneId",
					},
				},
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{
		referenceCoordinates: {
			EntityName: referenceCoordinates.Config,
			Coordinate: referenceCoordinates,
			Properties: parameter.Properties{},
			Skip:       true,
		},
	}

	dashboardApiId := "dashboard"
	apis := map[string]api.Api{
		dashboardApiId: api.NewStandardApi(dashboardApiId, "dashboard"),
	}

	knownEntityNames := knownEntityMap{}

	_, errors := deployConfig(client, apis, entities, knownEntityNames, &conf, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailOnUnknownApi(t *testing.T) {
	parameters := []topologysort.ParameterWithName{}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	entities := map[coordinate.Coordinate]parameter.ResolvedEntity{}
	apis := map[string]api.Api{}

	knownEntityNames := knownEntityMap{}

	_, errors := deployConfig(client, apis, entities, knownEntityNames, &conf, false)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func toParameterMap(params []topologysort.ParameterWithName) map[string]parameter.Parameter {
	result := make(map[string]parameter.Parameter)

	for _, p := range params {
		result[p.Name] = p.Parameter
	}

	return result
}

func toReferences(params []topologysort.ParameterWithName) []coordinate.Coordinate {
	var result []coordinate.Coordinate

	for _, p := range params {
		refs := p.Parameter.GetReferences()

		if refs == nil {
			continue
		}

		for _, ref := range refs {
			result = append(result, ref.Config)
		}
	}

	return result
}

func generateDummyTemplate(t *testing.T) template.Template {
	uuid, err := uuid.NewUUID()

	assert.NilError(t, err)

	templ, err := template.LoadTemplateFromString(uuid.String(), "deploy_test-"+uuid.String(), "{}")

	assert.NilError(t, err)

	return templ
}
