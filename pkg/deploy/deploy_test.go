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
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/topologysort"
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

		clientSet := DummyClientSet
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

		resolvedEntity, errors := deploy(context.TODO(), clientSet, testApiMap, newEntityMap(testApiMap), &conf)

		assert.Emptyf(t, errors, "errors: %v", errors)
		assert.Equal(t, name, resolvedEntity.EntityName, "%s == %s")
		assert.Equal(t, conf.Coordinate, resolvedEntity.Coordinate)
		assert.Equal(t, name, resolvedEntity.Properties[config.NameParameter])
		assert.Equal(t, owner, resolvedEntity.Properties[ownerParameterName])
		assert.Equal(t, timeout, resolvedEntity.Properties[timeoutParameterName])
		assert.Equal(t, false, resolvedEntity.Skip)
	})
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
		{
			Name: config.ScopeParameter,
			Parameter: &parameter.DummyParameter{
				Value: "something",
			},
		},
	}

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any()).Return(dtclient.DynatraceEntity{}, fmt.Errorf("upsert failed"))

	conf := &config.Config{
		Type:       config.SettingsType{},
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap(parameters),
	}

	_, errors := deploy(context.TODO(), ClientSet{Settings: c}, nil, newEntityMap(testApiMap), conf)
	assert.NotEmpty(t, errors)
}

func TestDeploySetting(t *testing.T) {
	type given struct {
		config           config.Config
		returnedEntityID string
	}

	tests := []struct {
		name    string
		given   given
		want    parameter.ResolvedEntity
		wantErr bool
	}{
		{
			name: "happy path",
			given: given{
				config: config.Config{
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
				},
				returnedEntityID: "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
			},
			want: parameter.ResolvedEntity{
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
			},
			wantErr: false,
		},
		{
			name: "management zone settings get numeric ID",
			given: given{
				config: config.Config{
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
				},
				returnedEntityID: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
			},
			want: parameter.ResolvedEntity{
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
			},
			wantErr: false,
		},
		{
			name: "returns error if MZ object ID can't be decoded",
			given: given{
				config: config.Config{
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
				},
				returnedEntityID: "INVALID OBJECT ID",
			},
			want:    parameter.ResolvedEntity{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := dtclient.NewMockClient(gomock.NewController(t))
			c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
				Id:   tt.given.returnedEntityID,
				Name: tt.given.returnedEntityID,
			}, nil)

			got, errors := deploy(context.TODO(), ClientSet{Settings: c}, nil, newEntityMap(testApiMap), &tt.given.config)
			if !tt.wantErr {
				assert.Equal(t, got, &tt.want)
				assert.Emptyf(t, errors, "errors: %v)", errors)
			} else {
				assert.NotEmptyf(t, errors, "errors: %v)", errors)
			}
		})
	}
}

func TestDeployedSettingGetsNameFromConfig(t *testing.T) {
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
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
		Name: "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
	}, nil)

	conf := &config.Config{
		Type:       config.SettingsType{},
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap(parameters),
	}
	res, errors := deploy(context.TODO(), ClientSet{Settings: c}, nil, newEntityMap(testApiMap), conf)
	assert.Equal(t, res.EntityName, cfgName, "expected resolved name to match configuration name")
	assert.Emptyf(t, errors, "errors: %v", errors)
}

func TestSettingsNameExtractionDoesNotFailIfCfgNameBecomesOptional(t *testing.T) {
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
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   objectId,
		Name: objectId,
	}, nil)

	conf := &config.Config{
		Type:       config.SettingsType{},
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap(parametersWithoutName),
	}
	res, errors := deploy(context.TODO(), ClientSet{Settings: c}, nil, newEntityMap(testApiMap), conf)
	assert.Contains(t, res.EntityName, objectId, "expected resolved name to contain objectID if name is not configured")
	assert.Empty(t, errors, " errors: %v)", errors)
}

func TestDeployConfigsWithNoConfigs(t *testing.T) {
	var apis api.APIs
	var sortedConfigs []config.Config

	errors := DeployConfigs(DummyClientSet, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsWithOneConfigToSkip(t *testing.T) {
	var apis api.APIs
	sortedConfigs := []config.Config{
		{Skip: true},
	}
	errors := DeployConfigs(DummyClientSet, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingSettings(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	var apis api.APIs
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
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   "42",
		Name: "Super Special Settings Object",
	}, nil)
	errors := DeployConfigs(ClientSet{Settings: c}, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingClassicConfigUnique(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.API{ID: theApiName, URLPath: "path"}

	client := dtclient.NewMockClient(gomock.NewController(t))
	client.EXPECT().UpsertConfigByName(gomock.Any(), gomock.Any(), theConfigName, gomock.Any()).Times(1)

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

	errors := DeployConfigs(ClientSet{Classic: client}, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingClassicConfigNonUniqueWithExistingCfgsOfSameName(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.API{ID: theApiName, URLPath: "path", NonUniqueName: true}

	client := dtclient.NewMockClient(gomock.NewController(t))
	client.EXPECT().UpsertConfigByNonUniqueNameAndId(gomock.Any(), gomock.Any(), gomock.Any(), theConfigName, gomock.Any())

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

	errors := DeployConfigs(ClientSet{Classic: client}, apis, sortedConfigs, DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsNoApi(t *testing.T) {
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
		errors := DeployConfigs(ClientSet{Classic: client}, apis, sortedConfigs, DeployConfigsOptions{ContinueOnErr: true})
		assert.Equal(t, 2, len(errors), fmt.Sprintf("Expected 2 errors, but just got %d", len(errors)))
	})

	t.Run("missing api - stop on error", func(t *testing.T) {
		errors := DeployConfigs(ClientSet{Classic: client}, apis, sortedConfigs, DeployConfigsOptions{})
		assert.Equal(t, 1, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})
	// test continue on error

}

func TestDeployConfigsWithDeploymentErrors(t *testing.T) {
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
		errors := DeployConfigs(DummyClientSet, apis, sortedConfigs, DeployConfigsOptions{})
		assert.Equal(t, 1, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})

	t.Run("deployment error - stop on error", func(t *testing.T) {
		errors := DeployConfigs(DummyClientSet, apis, sortedConfigs, DeployConfigsOptions{ContinueOnErr: true})
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
