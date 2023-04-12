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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2/topologysort"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

var dashboardApi = api.API{ID: "dashboard", URLPath: "dashboard", DeprecatedBy: "dashboard-v2"}
var testApiMap = api.APIs{"dashboard": dashboardApi}

func TestDeploy(t *testing.T) {
	t.Run("", func(t *testing.T) {

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

		client := &dtclient.DummyClient{}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: "dashboard"},
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

		d := deployer{
			dtClient: client,
			apis:     testApiMap,
		}

		resolvedEntity, errors := d.deploy(&conf, newEntityMap(testApiMap))

		assert.Emptyf(t, errors, "errors: %v", errors)
		assert.Equal(t, name, resolvedEntity.EntityName, "%s == %s")
		assert.Equal(t, conf.Coordinate, resolvedEntity.Coordinate)
		assert.Equal(t, name, resolvedEntity.Properties[config.NameParameter])
		assert.Equal(t, owner, resolvedEntity.Properties[ownerParameterName])
		assert.Equal(t, timeout, resolvedEntity.Properties[timeoutParameterName])
		assert.Equal(t, false, resolvedEntity.Skip)
	})

	t.Run("TestDeploySetting", func(t *testing.T) { // TODO: this is test for config.SettingsType
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
			{
				Name: config.ScopeParameter,
				Parameter: &parameter.DummyParameter{
					Value: "something",
				},
			},
		}

		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().UpsertSettings(gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
			Id:   "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
			Name: "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
		}, nil)
		d := deployer{dtClient: c}

		conf := &config.Config{
			Type:       config.SettingsType{},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap(parameters),
		}

		_, errors := d.deploy(conf, newEntityMap(testApiMap))
		assert.Emptyf(t, errors, "errors: %v)", errors)
	})

	t.Run("TestDeploySettingShouldFailUpsert", func(t *testing.T) { // TODO: this is test for config.SettingsType
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
			{
				Name: config.ScopeParameter,
				Parameter: &parameter.DummyParameter{
					Value: "something",
				},
			},
		}

		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().UpsertSettings(gomock.Any()).Return(dtclient.DynatraceEntity{}, fmt.Errorf("upsert failed"))

		conf := &config.Config{
			Type:       config.SettingsType{},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap(parameters),
		}
		d := deployer{dtClient: c}

		_, errors := d.deploy(conf, newEntityMap(testApiMap))
		assert.NotEmpty(t, errors)
	})

	t.Run("TestDeployedSettingGetsNameFromConfig", func(t *testing.T) { // TODO: this is test for config.SettingsType
		cfgName := "THE CONFIG NAME"

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
			{
				Name: config.ScopeParameter,
				Parameter: &parameter.DummyParameter{
					Value: "something",
				},
			},
			{
				Name: config.NameParameter,
				Parameter: &parameter.DummyParameter{
					Value: cfgName,
				},
			},
		}

		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().UpsertSettings(gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
			Id:   "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
			Name: "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
		}, nil)
		d := deployer{dtClient: c}

		conf := &config.Config{
			Type:       config.SettingsType{},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap(parameters),
		}

		res, errors := d.deploy(conf, newEntityMap(testApiMap))
		assert.Equal(t, res.EntityName, cfgName, "expected resolved name to match configuration name")
		assert.Emptyf(t, errors, "errors: %v", errors)
	})

	t.Run("TestSettingsNameExtractionDoesNotFailIfCfgNameBecomesOptional", func(t *testing.T) {
		parametersWithoutName := []topologysort.ParameterWithName{
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
			{
				Name: config.ScopeParameter,
				Parameter: &parameter.DummyParameter{
					Value: "something",
				},
			},
		}

		objectId := "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0"

		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().UpsertSettings(gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
			Id:   objectId,
			Name: objectId,
		}, nil)
		d := deployer{dtClient: c}

		conf := &config.Config{
			Type:       config.SettingsType{},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap(parametersWithoutName),
		}

		res, errors := d.deploy(conf, newEntityMap(testApiMap))
		assert.Contains(t, res.EntityName, objectId, "expected resolved name to contain objectID if name is not configured")
		assert.Empty(t, errors, " errors: %v)", errors)
	})
}

func TestDeploySetting(t *testing.T) {
	t.Run("TestDeploySettingShouldFailCyclicParameterDependencies", func(t *testing.T) {
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

		client := &dtclient.DummyClient{}

		conf := &config.Config{
			Type:       config.ClassicApiType{},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap(parameters),
		}
		_, errors := deploySetting(client, nil, "", conf)
		assert.NotEmpty(t, errors)
	})

	t.Run("TestDeploySettingShouldFailRenderTemplate", func(t *testing.T) {
		client := &dtclient.DummyClient{}

		conf := &config.Config{
			Type:     config.ClassicApiType{},
			Template: generateFaultyTemplate(t),
		}

		_, errors := deploySetting(client, nil, "", conf)
		assert.NotEmpty(t, errors)
	})

	t.Run("happy path", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().UpsertSettings(gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
			Id:   "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
			Name: "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
		}, nil)
		d := deployer{dtClient: c}

		given := &config.Config{
			Type: config.SettingsType{SchemaId: "builtin:some-schema"},
			Coordinate: coordinate.Coordinate{
				Project:  "project",
				Type:     "bultin:some-schema",
				ConfigId: "some-settings-config",
			},
			Template: generateDummyTemplate(t),
			Parameters: toParameterMap([]topologysort.ParameterWithName{
				{
					Name: "name",
					Parameter: &parameter.DummyParameter{
						Value: "My Setting",
					},
				},
				{
					Name: config.ScopeParameter,
					Parameter: &parameter.DummyParameter{
						Value: "environment",
					},
				},
			}),
		}
		expected := &parameter.ResolvedEntity{
			EntityName: "My Setting",
			Coordinate: coordinate.Coordinate{
				Project:  "project",
				Type:     "bultin:some-schema",
				ConfigId: "some-settings-config",
			},
			Properties: parameter.Properties{
				"id":    "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
				"name":  "My Setting",
				"scope": "environment",
			},
		}

		got, errors := d.deploy(given, newEntityMap(testApiMap))

		assert.Equal(t, got, expected)
		assert.Emptyf(t, errors, "errors: %v)", errors)
	})
	type given struct {
		config           config.Config
		returnedEntityID string
	}

	t.Run("management zone settings get numeric ID", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().UpsertSettings(gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
			Id:   "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
			Name: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
		}, nil)

		given := &config.Config{
			Type: config.SettingsType{SchemaId: "builtin:management-zones"},
			Coordinate: coordinate.Coordinate{
				Project:  "project",
				Type:     "builtin:management-zones",
				ConfigId: "some-settings-config",
			},
			Template: generateDummyTemplate(t),
			Parameters: toParameterMap([]topologysort.ParameterWithName{
				{
					Name: "name",
					Parameter: &parameter.DummyParameter{
						Value: "My Setting",
					},
				},
				{
					Name: config.ScopeParameter,
					Parameter: &parameter.DummyParameter{
						Value: "environment",
					},
				},
			}),
		}
		expected := &parameter.ResolvedEntity{
			EntityName: "My Setting",
			Coordinate: coordinate.Coordinate{
				Project:  "project",
				Type:     "builtin:management-zones",
				ConfigId: "some-settings-config",
			},
			Properties: parameter.Properties{
				"id":    "-4292415658385853785",
				"name":  "My Setting",
				"scope": "environment",
			},
		}

		d := deployer{dtClient: c}
		got, errors := d.deploy(given, newEntityMap(testApiMap))
		assert.Equal(t, expected, got)
		assert.Emptyf(t, errors, "errors: %v)", errors)

	})

	t.Run("", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().UpsertSettings(gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
			Id:   "INVALID OBJECT ID",
			Name: "INVALID OBJECT ID",
		}, nil)

		given := &config.Config{
			Type: config.SettingsType{SchemaId: "builtin:management-zones"},
			Coordinate: coordinate.Coordinate{
				Project:  "project",
				Type:     "builtin:management-zones",
				ConfigId: "some-settings-config",
			},
			Template: generateDummyTemplate(t),
			Parameters: toParameterMap([]topologysort.ParameterWithName{
				{
					Name: "name",
					Parameter: &parameter.DummyParameter{
						Value: "My Setting",
					},
				},
				{
					Name: config.ScopeParameter,
					Parameter: &parameter.DummyParameter{
						Value: "environment",
					},
				},
			}),
		}

		d := deployer{dtClient: c}
		_, errors := d.deploy(given, newEntityMap(testApiMap))
		assert.NotEmptyf(t, errors, "errors: %v)", errors)

	})
}

func TestDeployConfig(t *testing.T) {
	t.Run("TestDeployConfigShouldFailOnAnAlreadyKnownEntityName", func(t *testing.T) {
		name := "test"
		parameters := []topologysort.ParameterWithName{
			{
				Name: config.NameParameter,
				Parameter: &parameter.DummyParameter{
					Value: name,
				},
			},
		}

		client := &dtclient.DummyClient{}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: "dashboard"},
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
		entityMap := newEntityMap(testApiMap)
		entityMap.put(parameter.ResolvedEntity{EntityName: name, Coordinate: coordinate.Coordinate{Type: "dashboard"}})
		_, errors := deployConfig(client, testApiMap, entityMap, nil, "", &conf)

		assert.NotEmpty(t, errors)
	})

	t.Run("TestDeployConfigShouldFailCyclicParameterDependencies", func(t *testing.T) {
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

		client := &dtclient.DummyClient{}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: "dashboard"},
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

		_, errors := deployConfig(client, testApiMap, newEntityMap(testApiMap), nil, "", &conf)
		assert.NotEmpty(t, errors)
	})

	t.Run("TestDeployConfigShouldFailOnMissingNameParameter", func(t *testing.T) {
		parameters := []topologysort.ParameterWithName{}

		client := &dtclient.DummyClient{}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: "dashboard"},
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

		_, errors := deployConfig(client, testApiMap, newEntityMap(testApiMap), nil, "", &conf)
		assert.NotEmpty(t, errors)
	})

	t.Run("TestDeployConfigShouldFailOnReferenceOnUnknownConfig", func(t *testing.T) {
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

		client := &dtclient.DummyClient{}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: "dashboard"},
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

		_, errors := deployConfig(client, testApiMap, newEntityMap(testApiMap), nil, "", &conf)
		assert.NotEmpty(t, errors)
	})

	t.Run("TestDeployConfigShouldFailOnReferenceOnSkipConfig", func(t *testing.T) {
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

		client := &dtclient.DummyClient{}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: "dashboard"},
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

		_, errors := deployConfig(client, testApiMap, newEntityMap(testApiMap), nil, "", &conf)
		assert.NotEmpty(t, errors)
	})
}

func TestDeployAll(t *testing.T) {
	t.Run("empty set of monaco config objects", func(t *testing.T) {
		d := deployer{}

		errs := d.DeployAll([]config.Config{})
		assert.Emptyf(t, errs, "there should be no errors (errors: %v)", errs)

		errs = d.DeployAll(nil)
		assert.Emptyf(t, errs, "there should be no errors (errors: %v)", errs)
	})

	t.Run("monaco config objet to skip", func(t *testing.T) { // TODO: move to TestDeploy
		d := deployer{}
		sortedConfigs := []config.Config{{Skip: true}}
		errors := d.DeployAll(sortedConfigs)
		assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
	})

	t.Run("monaco config objet is type of config.SettingsType", func(t *testing.T) { // TODO: move to TestDeploy
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().UpsertSettings(gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
			Id:   "42",
			Name: "Super Special Settings Object",
		}, nil)
		d := deployer{dtClient: c}

		sortedConfigs := []config.Config{
			{
				Template: generateDummyTemplate(t),
				Coordinate: coordinate.Coordinate{
					Project:  "some project",
					Type:     "schema",
					ConfigId: "some setting",
				},
				Type: config.SettingsType{
					SchemaId:      "schema",
					SchemaVersion: "schemaversion",
				},
				Parameters: config.Parameters{
					config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
				},
			},
		}

		errors := d.DeployAll(sortedConfigs)
		assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
	})

	t.Run("TestDeployConfigsTargetingClassicConfigUnique", func(t *testing.T) {
		theConfigName := "theConfigName"
		theApiName := "theApiName"

		theApi := api.API{ID: theApiName}

		client := dtclient.NewMockClient(gomock.NewController(t))
		client.EXPECT().UpsertConfigByName(gomock.Any(), theConfigName, gomock.Any()).Times(1)

		apis := api.APIs{theApiName: theApi}
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
				Type: config.ClassicApiType{
					Api: theApiName,
				},
			},
		}

		d := deployer{dtClient: client, apis: apis}

		errors := d.DeployAll(sortedConfigs)
		assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
	})

	t.Run("TestDeployConfigsTargetingClassicConfigNonUniqueWithExistingCfgsOfSameName", func(t *testing.T) {
		theConfigName := "theConfigName"
		theApiName := "theApiName"

		theApi := api.API{ID: theApiName, URLPath: "path", NonUniqueName: true}

		client := dtclient.NewMockClient(gomock.NewController(t))
		client.EXPECT().UpsertConfigByNonUniqueNameAndId(gomock.Any(), gomock.Any(), theConfigName, gomock.Any())

		apis := api.APIs{theApiName: theApi}
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
				Type: config.ClassicApiType{
					Api: theApiName,
				},
			},
		}

		d := deployer{
			dtClient: client,
			apis:     apis,
		}

		errors := d.DeployAll(sortedConfigs)
		assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
	})

}

func TestDeployConfigsNoApi(t *testing.T) { // TODO: this is test for config.ClassicApiType
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	client := dtclient.NewMockClient(gomock.NewController(t))

	apis := api.APIs{}
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
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
		{
			Parameters: toParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	t.Run("missing api - continue on error", func(t *testing.T) {
		d := deployer{
			dtClient:      client,
			apis:          apis,
			continueOnErr: true,
		}

		errors := d.DeployAll(sortedConfigs)
		assert.Equal(t, 2, len(errors), fmt.Sprintf("Expected 2 errors, but just got %d", len(errors)))
	})

	t.Run("missing api - stop on error", func(t *testing.T) {
		d := deployer{
			dtClient: client,
			apis:     apis,
		}

		errors := d.DeployAll(sortedConfigs)
		assert.Equal(t, 1, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})
	// test continue on error

}

func TestDeployConfigsWithDeploymentErrors(t *testing.T) { // TODO: this is test for config.ClassicApiType
	theApiName := "theApiName"
	theApi := api.API{ID: theApiName, URLPath: "path"}
	apis := api.APIs{theApiName: theApi}
	sortedConfigs := []config.Config{
		{
			Parameters: toParameterMap([]topologysort.ParameterWithName{}), // missing name parameter leads to deployment failure
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
		{
			Parameters: toParameterMap([]topologysort.ParameterWithName{}), // missing name parameter leads to deployment failure
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	t.Run("deployment error - stop on error", func(t *testing.T) {
		d := deployer{
			dtClient: &dtclient.DummyClient{},
			apis:     apis,
		}
		errors := d.DeployAll(sortedConfigs)
		assert.Equal(t, 1, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})

	t.Run("deployment error - stop on error", func(t *testing.T) {
		d := deployer{
			dtClient:      &dtclient.DummyClient{},
			apis:          apis,
			continueOnErr: true,
		}
		errors := d.DeployAll(sortedConfigs)
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

func generateDummyTemplate(t *testing.T) template.Template {
	newUUID, err := uuid.NewUUID()
	assert.NoError(t, err)
	templ := template.CreateTemplateFromString("deploy_test-"+newUUID.String(), "{}")
	return templ
}

func generateFaultyTemplate(t *testing.T) template.Template {
	newUUID, err := uuid.NewUUID()
	assert.NoError(t, err)
	templ := template.CreateTemplateFromString("deploy_test-"+newUUID.String(), "{")
	return templ
}
