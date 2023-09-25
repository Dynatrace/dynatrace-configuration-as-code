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

package sequential

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/testutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/net/context"
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

		clientSet := deploy.DummyClientSet
		conf := config.Config{
			Type:     config.ClassicApiType{Api: "dashboard"},
			Template: testutils.GenerateDummyTemplate(t),
			Coordinate: coordinate.Coordinate{
				Project:  "project1",
				Type:     "dashboard",
				ConfigId: "dashboard-1",
			},
			Environment: "development",
			Parameters:  testutils.ToParameterMap(parameters),
			Skip:        false,
		}

		resolvedEntity, errors := deployConfig(context.TODO(), clientSet, testApiMap, newEntityMapWithNames(), &conf)

		assert.Emptyf(t, errors, "errors: %v", errors)
		assert.Equal(t, name, resolvedEntity.EntityName)
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
			Name: config.ScopeParameter,
			Parameter: &parameter.DummyParameter{
				Value: "something",
			},
		},
	}

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any(), gomock.Any()).Return(dtclient.DynatraceEntity{}, fmt.Errorf("upsert failed"))

	conf := &config.Config{
		Type:       config.SettingsType{},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap(parameters),
	}

	_, errors := deployConfig(context.TODO(), deploy.ClientSet{Settings: c}, nil, newEntityMapWithNames(), conf)
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
		want    config.ResolvedEntity
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
					Template: testutils.GenerateDummyTemplate(t),
					Parameters: testutils.ToParameterMap([]parameter.NamedParameter{
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
			want: config.ResolvedEntity{
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
					Template: testutils.GenerateDummyTemplate(t),
					Parameters: testutils.ToParameterMap([]parameter.NamedParameter{
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
			want: config.ResolvedEntity{
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
					Template: testutils.GenerateDummyTemplate(t),
					Parameters: testutils.ToParameterMap([]parameter.NamedParameter{
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
			want:    config.ResolvedEntity{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := dtclient.NewMockClient(gomock.NewController(t))
			c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
				Id:   tt.given.returnedEntityID,
				Name: tt.given.returnedEntityID,
			}, nil)

			got, errors := deployConfig(context.TODO(), deploy.ClientSet{Settings: c}, nil, newEntityMapWithNames(), &tt.given.config)
			if !tt.wantErr {
				assert.Equal(t, got, tt.want)
				assert.Emptyf(t, errors, "errors: %v)", errors)
			} else {
				assert.NotEmptyf(t, errors, "errors: %v)", errors)
			}
		})
	}
}

func TestDeployedSettingGetsNameFromConfig(t *testing.T) {
	cfgName := "THE CONFIG NAME"

	parameters := []parameter.NamedParameter{
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
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
		Name: "vu9U3hXa3q0AAAABABlidWlsdGluOMmE1NGMxvu9U3hXa3q0",
	}, nil)

	conf := &config.Config{
		Type:       config.SettingsType{},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap(parameters),
	}
	res, errors := deployConfig(context.TODO(), deploy.ClientSet{Settings: c}, nil, newEntityMapWithNames(), conf)
	assert.Equal(t, res.EntityName, cfgName, "expected resolved name to match configuration name")
	assert.Emptyf(t, errors, "errors: %v", errors)
}

func TestSettingsNameExtractionDoesNotFailIfCfgNameBecomesOptional(t *testing.T) {
	parametersWithoutName := []parameter.NamedParameter{
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
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   objectId,
		Name: objectId,
	}, nil)

	conf := &config.Config{
		Type:       config.SettingsType{},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap(parametersWithoutName),
	}
	res, errors := deployConfig(context.TODO(), deploy.ClientSet{Settings: c}, nil, newEntityMapWithNames(), conf)
	assert.Contains(t, res.EntityName, objectId, "expected resolved name to contain objectID if name is not configured")
	assert.Empty(t, errors, " errors: %v)", errors)
}

func TestDeployConfigsWithNoConfigs(t *testing.T) {
	var apis api.APIs
	var sortedConfigs []config.Config

	errors := DeployConfigs(deploy.DummyClientSet, apis, sortedConfigs, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsWithOneConfigToSkip(t *testing.T) {
	var apis api.APIs
	sortedConfigs := []config.Config{
		{Skip: true},
	}
	errors := DeployConfigs(deploy.DummyClientSet, apis, sortedConfigs, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingSettings(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))
	var apis api.APIs
	sortedConfigs := []config.Config{
		{
			Template: testutils.GenerateDummyTemplate(t),
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
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   "42",
		Name: "Super Special Settings Object",
	}, nil)
	errors := DeployConfigs(deploy.ClientSet{Settings: c}, apis, sortedConfigs, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingClassicConfigUnique(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.API{ID: theApiName, URLPath: "path"}

	client := dtclient.NewMockClient(gomock.NewController(t))
	client.EXPECT().UpsertConfigByName(gomock.Any(), gomock.Any(), theConfigName, gomock.Any()).Times(1)

	apis := api.APIs{theApiName: theApi}
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	errors := DeployConfigs(deploy.ClientSet{Classic: client}, apis, sortedConfigs, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingClassicConfigNonUniqueWithExistingCfgsOfSameName(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.API{ID: theApiName, URLPath: "path", NonUniqueName: true}

	client := dtclient.NewMockClient(gomock.NewController(t))
	client.EXPECT().UpsertConfigByNonUniqueNameAndId(gomock.Any(), gomock.Any(), gomock.Any(), theConfigName, gomock.Any())

	apis := api.APIs{theApiName: theApi}
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	errors := DeployConfigs(deploy.ClientSet{Classic: client}, apis, sortedConfigs, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsNoApi(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	client := dtclient.NewMockClient(gomock.NewController(t))

	apis := api.APIs{}
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	t.Run("missing api - continue on error", func(t *testing.T) {
		errors := DeployConfigs(deploy.ClientSet{Classic: client}, apis, sortedConfigs, deploy.DeployConfigsOptions{ContinueOnErr: true})
		assert.Equal(t, 2, len(errors), fmt.Sprintf("Expected 2 errors, but just got %d", len(errors)))
	})

	t.Run("missing api - stop on error", func(t *testing.T) {
		errors := DeployConfigs(deploy.ClientSet{Classic: client}, apis, sortedConfigs, deploy.DeployConfigsOptions{})
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
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}), // missing name parameter leads to deployment failure
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
		{
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}), // missing name parameter leads to deployment failure
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	t.Run("deployment error - stop on error", func(t *testing.T) {
		errors := DeployConfigs(deploy.DummyClientSet, apis, sortedConfigs, deploy.DeployConfigsOptions{})
		assert.Equal(t, 1, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})

	t.Run("deployment error - stop on error", func(t *testing.T) {
		errors := DeployConfigs(deploy.DummyClientSet, apis, sortedConfigs, deploy.DeployConfigsOptions{ContinueOnErr: true})
		assert.Equal(t, 2, len(errors), fmt.Sprintf("Expected 1 error, but just got %d", len(errors)))
	})

}

func TestDeployConfigsWithDuplicateNameCausesError(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.API{ID: theApiName, URLPath: "path"}

	apis := api.APIs{theApiName: theApi}
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName, Project: "proj", ConfigId: "cfg_1"},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName, Project: "proj", ConfigId: "cfg_2"},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().UpsertConfigByName(gomock.Any(), gomock.Any(), theConfigName, gomock.Any()).Return(dtclient.DynatraceEntity{
		Id:   "42",
		Name: theConfigName,
	}, nil)

	errors := DeployConfigs(deploy.ClientSet{Classic: c}, apis, sortedConfigs, deploy.DeployConfigsOptions{})
	assert.NotEmpty(t, errors, "two configs using the same name should cause validation errors - but got none")
}

func TestDeploySkippedConfigsWithDuplicateNameNoError(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.API{ID: theApiName, URLPath: "path"}

	apis := api.APIs{theApiName: theApi}
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName, Project: "proj", ConfigId: "cfg_1"},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName, Project: "proj", ConfigId: "cfg_2"},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
			Skip: true,
		},
	}

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().UpsertConfigByName(gomock.Any(), gomock.Any(), theConfigName, gomock.Any()).Return(dtclient.DynatraceEntity{
		Id:   "42",
		Name: theConfigName,
	}, nil)

	errors := DeployConfigs(deploy.ClientSet{Classic: c}, apis, sortedConfigs, deploy.DeployConfigsOptions{})
	assert.Empty(t, errors, "skipped and deployed config having the same name should deploy without errors (errors: %v)", errors)
}

func TestDeployNonUniqueConfigsWithDuplicateNameNoError(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "theApiName"

	theApi := api.API{ID: theApiName, URLPath: "path", NonUniqueName: true}

	apis := api.APIs{theApiName: theApi}
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	sortedConfigs := []config.Config{
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName, Project: "proj", ConfigId: "cfg_1"},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
		{
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName, Project: "proj", ConfigId: "cfg_2"},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	c := dtclient.NewMockClient(gomock.NewController(t))
	c.EXPECT().UpsertConfigByNonUniqueNameAndId(gomock.Any(), gomock.Any(), gomock.Any(), theConfigName, gomock.Any()).Return(dtclient.DynatraceEntity{
		Id:   "42",
		Name: theConfigName,
	}, nil).Times(2)

	errors := DeployConfigs(deploy.ClientSet{Classic: c}, apis, sortedConfigs, deploy.DeployConfigsOptions{})
	assert.Empty(t, errors, "two non-unique-name configs with the same name should deploy without errors (errors: %v)", errors)
}
