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
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/golang/mock/gomock"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/client"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

var dashboardApi = api.NewStandardApi("dashboard", "dashboard", false, "dashboard-v2", false)
var testApiMap = api.ApiMap{"dashboard": dashboardApi}

func TestDeployConfig(t *testing.T) {
	name := "test"
	owner := "hansi"
	ownerParameterName := "owner"
	timeout := 5
	timeoutParameterName := "timeout"
	parameters := []topologysort.ParameterWithName{
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

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	resolvedEntity, errors := deployConfig(client, testApiMap, NewEntityMap(testApiMap), &conf)

	assert.Assert(t, len(errors) == 0, "there should be no errors (no errors: %d, %s)", len(errors), errors)
	assert.Equal(t, name, resolvedEntity.EntityName, "%s == %s")
	assert.Equal(t, conf.Coordinate, resolvedEntity.Coordinate)
	assert.Equal(t, name, resolvedEntity.Properties[config.NameParameter])
	assert.Equal(t, owner, resolvedEntity.Properties[ownerParameterName])
	assert.Equal(t, timeout, resolvedEntity.Properties[timeoutParameterName])
	assert.Equal(t, false, resolvedEntity.Skip)
}

func TestDeploySettingShouldFailCyclicParameterDependencies(t *testing.T) {
	ownerParameterName := "owner"
	configCoordinates := coordinate.Coordinate{}

	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
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
			Parameter: &parameter.DummyParameter{
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
	//entities := make(map[coordinate.Coordinate]parameter.ResolvedEntity)

	conf := &config.Config{
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap(parameters),
	}
	_, errors := deploySetting(client, NewEntityMap(testApiMap), conf)
	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeploySettingShouldFailRenderTemplate(t *testing.T) {
	client := &client.DummyClient{}

	conf := &config.Config{
		Template: generateFaultyTemplate(t),
	}

	_, errors := deploySetting(client, NewEntityMap(testApiMap), conf)
	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeploySettingShouldFailUpsert(t *testing.T) {
	name := "test"
	owner := "hansi"
	ownerParameterName := "owner"
	parameters := []topologysort.ParameterWithName{
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
	}

	client := rest.NewMockSettingsClient(gomock.NewController(t))
	client.EXPECT().UpsertSettings(gomock.Any(), gomock.Any()).Return(api.DynatraceEntity{}, fmt.Errorf("upsert failed"))
	client.EXPECT().ListKnownSettings(gomock.Any()).Return(nil, nil)

	conf := &config.Config{
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap(parameters),
	}
	_, errors := deploySetting(client, NewEntityMap(testApiMap), conf)
	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeploySetting(t *testing.T) {
	parameters := []topologysort.ParameterWithName{
		{
			Name: "franz",
			Parameter: &parameter.DummyParameter{
				Value: "foo",
			},
		},
		{
			Name: "hansi",
			Parameter: &parameter.DummyParameter{
				Value: "bar",
			},
		},
	}

	client := &client.DummyClient{}

	conf := &config.Config{
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap(parameters),
	}
	_, errors := deploySetting(client, NewEntityMap(testApiMap), conf)
	assert.Assert(t, len(errors) == 0, "there should be no errors (no errors: %d, %s)", len(errors), errors)
}

func TestDeployConfigShouldFailOnAnAlreadyKnownEntityName(t *testing.T) {
	name := "test"
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: name,
			},
		},
	}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}
	entityMap := NewEntityMap(testApiMap)
	entityMap.PutResolved(coordinate.Coordinate{Type: "dashboard"}, parameter.ResolvedEntity{EntityName: name})
	_, errors := deployConfig(client, testApiMap, entityMap, &conf)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailCyclicParameterDependencies(t *testing.T) {
	ownerParameterName := "owner"
	configCoordinates := coordinate.Coordinate{
		Project:  "project1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
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
			Parameter: &parameter.DummyParameter{
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
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	_, errors := deployConfig(client, testApiMap, NewEntityMap(testApiMap), &conf)
	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailOnMissingNameParameter(t *testing.T) {
	parameters := []topologysort.ParameterWithName{}

	client := &client.DummyClient{}
	conf := config.Config{
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	_, errors := deployConfig(client, testApiMap, NewEntityMap(testApiMap), &conf)
	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailOnReferenceOnUnknownConfig(t *testing.T) {
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config: coordinate.Coordinate{
							Project:  "project2",
							Type:     "dashboard",
							ConfigId: "dashboard",
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
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	_, errors := deployConfig(client, testApiMap, NewEntityMap(testApiMap), &conf)
	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigShouldFailOnReferenceOnSkipConfig(t *testing.T) {
	referenceCoordinates := coordinate.Coordinate{
		Project:  "project2",
		Type:     "dashboard",
		ConfigId: "dashboard",
	}

	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
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
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		References:  toReferences(parameters),
		Skip:        false,
	}

	_, errors := deployConfig(client, testApiMap, NewEntityMap(testApiMap), &conf)
	assert.Assert(t, len(errors) > 0, "there should be errors (no errors: %d)", len(errors))
}

func TestDeployConfigsWithNoConfigs(t *testing.T) {
	client := &client.DummyClient{}
	var apis map[string]api.Api
	var sortedConfigs []config.Config

	errors := DeployConfigs(client, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Assert(t, len(errors) == 0, "there should be no errors (errors: %s)", errors)
}

func TestDeployConfigsWithOneConfigToSkip(t *testing.T) {
	client := &client.DummyClient{}
	var apis map[string]api.Api
	sortedConfigs := []config.Config{
		{Skip: true},
	}
	errors := DeployConfigs(client, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Assert(t, len(errors) == 0, "there should be no errors (errors: %s)", errors)
}

func TestDeployConfigsTargetingSettings(t *testing.T) {
	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	var apis map[string]api.Api
	sortedConfigs := []config.Config{
		{
			Template: generateDummyTemplate(t),
			Coordinate: coordinate.Coordinate{
				Project:  "some project",
				Type:     "schema",
				ConfigId: "some setting",
			},
			Type: config.Type{
				SchemaId:      "schema",
				SchemaVersion: "schemaversion",
				Scope:         "scope",
			},
		},
	}
	client.EXPECT().ListKnownSettings(gomock.Any()).Times(1)
	client.EXPECT().UpsertSettings(gomock.Any(), gomock.Any()).Times(1).Return(api.DynatraceEntity{
		Id:   "42",
		Name: "Super Special Settings Object",
	}, nil)
	errors := DeployConfigs(client, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Assert(t, len(errors) == 0, "there should be no errors (errors: %s)", errors)
}

func TestDeployConfigsTargetingClassicConfigUnique(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.NewMockApi(gomock.NewController(t))
	theApi.EXPECT().GetId().AnyTimes().Return(theApiName)
	theApi.EXPECT().IsDeprecatedApi().Return(false)
	theApi.EXPECT().IsNonUniqueNameApi().Return(false)

	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().UpsertByName(gomock.Any(), theConfigName, gomock.Any()).Times(1)

	apis := map[string]api.Api{theApiName: theApi}
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: toParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.Type{
				Api: theApiName,
			},
		},
	}

	errors := DeployConfigs(client, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Assert(t, len(errors) == 0, "there should be no errors (errors: %s)", errors)
}

func TestDeployConfigsTargetingClassicConfigNonUnique(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.NewMockApi(gomock.NewController(t))
	theApi.EXPECT().GetId().AnyTimes().Return(theApiName)
	theApi.EXPECT().IsDeprecatedApi().Return(false)
	theApi.EXPECT().IsNonUniqueNameApi().Return(true)

	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().UpsertByEntityId(gomock.Any(), gomock.Any(), theConfigName, gomock.Any())

	apis := map[string]api.Api{theApiName: theApi}
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: toParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.Type{
				Api: theApiName,
			},
		},
	}

	errors := DeployConfigs(client, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Assert(t, len(errors) == 0, "there should be no errors (errors: %s)", errors)
}

func TestDeployConfigsNoApi(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	client := rest.NewMockDynatraceClient(gomock.NewController(t))

	apis := map[string]api.Api{}
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: toParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.Type{
				Api: theApiName,
			},
		},
		{
			Parameters: toParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.Type{
				Api: theApiName,
			},
		},
	}

	t.Run("missing api - continue on error", func(t *testing.T) {
		errors := DeployConfigs(client, apis, sortedConfigs, DeployConfigsOptions{ContinueOnErr: true})
		assert.Equal(t, 2, len(errors), fmt.Sprintf("Expected 2 errors, but just got %d", len(errors)))
	})

	t.Run("missing api - stop on error", func(t *testing.T) {
		errors := DeployConfigs(client, apis, sortedConfigs, DeployConfigsOptions{})
		assert.Equal(t, 1, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})
	// test continue on error

}

func TestDeployConfigsWithDeploymentErrors(t *testing.T) {
	theApiName := "theApiName"
	theApi := api.NewMockApi(gomock.NewController(t))
	theApi.EXPECT().GetId().AnyTimes().Return(theApiName)

	apis := map[string]api.Api{theApiName: theApi}
	sortedConfigs := []config.Config{
		{
			Parameters: toParameterMap([]topologysort.ParameterWithName{}), // missing name parameter leads to deployment failure
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.Type{
				Api: theApiName,
			},
		},
		{
			Parameters: toParameterMap([]topologysort.ParameterWithName{}), // missing name parameter leads to deployment failure
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.Type{
				Api: theApiName,
			},
		},
	}

	t.Run("deployment error - stop on error", func(t *testing.T) {
		errors := DeployConfigs(&client.DummyClient{}, apis, sortedConfigs, DeployConfigsOptions{})
		assert.Equal(t, 1, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})

	t.Run("deployment error - stop on error", func(t *testing.T) {
		errors := DeployConfigs(&client.DummyClient{}, apis, sortedConfigs, DeployConfigsOptions{ContinueOnErr: true})
		assert.Equal(t, 2, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})

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
	templ := template.CreateTemplateFromString("deploy_test-"+uuid.String(), "{}")
	return templ
}

func generateFaultyTemplate(t *testing.T) template.Template {
	uuid, err := uuid.NewUUID()
	assert.NilError(t, err)
	templ := template.CreateTemplateFromString("deploy_test-"+uuid.String(), "{")
	return templ
}
