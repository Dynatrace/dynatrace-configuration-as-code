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

package deploy_test

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestDeployConfigGraph_SingleConfig(t *testing.T) {
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

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					"dashboard": []config.Config{conf},
				},
			},
		},
	}

	dummyClient := dtclient.DummyClient{}
	clientSet := deploy.ClientSet{Classic: &dummyClient}

	c := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: clientSet,
	}

	errors := deploy.DeployConfigGraph(p, c, deploy.DeployConfigsOptions{})

	assert.Emptyf(t, errors, "errors: %v", errors)

	createdEntities := dummyClient.Entries[api.NewAPIs()["dashboard"]]
	assert.Len(t, createdEntities, 1)

	entity := createdEntities[0]
	assert.Equal(t, name, entity.Name)
}

func TestDeployConfigGraph_SettingShouldFailUpsert(t *testing.T) {
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
	c.EXPECT().UpsertSettings(gomock.Any(), gomock.Any()).Return(dtclient.DynatraceEntity{}, fmt.Errorf("upsert failed"))

	conf := config.Config{
		Type: config.SettingsType{
			SchemaId: "builtin:test",
		},
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap(parameters),
	}

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					"builtin:test": []config.Config{conf},
				},
			},
		},
	}

	clients := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: deploy.ClientSet{Settings: c},
	}

	errors := deploy.DeployConfigGraph(p, clients, deploy.DeployConfigsOptions{})
	assert.NotEmpty(t, errors)
}

func TestDeployConfigGraph_DoesNotFailOnEmptyConfigs(t *testing.T) {

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					"builtin:test": []config.Config{},
				},
			},
		},
	}

	c := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: deploy.DummyClientSet,
	}

	errors := deploy.DeployConfigGraph(p, c, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigGraph_DoesNotFailOnEmptyProject(t *testing.T) {

	var p []project.Project

	c := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: deploy.DummyClientSet,
	}

	errors := deploy.DeployConfigGraph(p, c, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigGraph_DoesNotFailNilProject(t *testing.T) {

	c := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: deploy.DummyClientSet,
	}

	errors := deploy.DeployConfigGraph(nil, c, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigGraph_DoesNotDeploySkippedConfig(t *testing.T) {
	configs := []config.Config{
		{Skip: true},
	}
	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					"dashboard": configs,
				},
			},
		},
	}

	dummyClient := dtclient.DummyClient{}
	clientSet := deploy.ClientSet{Classic: &dummyClient}

	c := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: clientSet,
	}

	errors := deploy.DeployConfigGraph(p, c, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
	assert.Len(t, dummyClient.Entries[api.NewAPIs()["dashboard"]], 0)
}

func TestDeployConfigGraph_DeploysSetting(t *testing.T) {
	c := dtclient.NewMockClient(gomock.NewController(t))

	configs := []config.Config{
		{
			Template: generateDummyTemplate(t),
			Coordinate: coordinate.Coordinate{
				Project:  "some project",
				Type:     "schema",
				ConfigId: "some setting",
			},
			Type: config.SettingsType{
				SchemaId:      "builtin:test",
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

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					"builtin:test": configs,
				},
			},
		},
	}

	clients := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: deploy.ClientSet{Settings: c},
	}

	errors := deploy.DeployConfigGraph(p, clients, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingClassicConfigUnique(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "management-zone"

	client := dtclient.NewMockClient(gomock.NewController(t))
	client.EXPECT().UpsertConfigByName(gomock.Any(), gomock.Any(), theConfigName, gomock.Any()).Times(1)

	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	configs := []config.Config{
		{
			Parameters: toParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					theApiName: configs,
				},
			},
		},
	}

	clients := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: deploy.ClientSet{Classic: client},
	}

	errors := deploy.DeployConfigGraph(p, clients, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingClassicConfigNonUniqueWithExistingCfgsOfSameName(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "alerting-profile"

	client := dtclient.NewMockClient(gomock.NewController(t))
	client.EXPECT().UpsertConfigByNonUniqueNameAndId(gomock.Any(), gomock.Any(), gomock.Any(), theConfigName, gomock.Any())

	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: theConfigName,
			},
		},
	}
	configs := []config.Config{
		{
			Parameters: toParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   generateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					theApiName: configs,
				},
			},
		},
	}

	clients := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: "env"}: deploy.ClientSet{Classic: client},
	}

	errors := deploy.DeployConfigGraph(p, clients, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsWithDeploymentErrors(t *testing.T) {
	theApiName := "management-zone"

	configs := []config.Config{
		{
			Parameters: toParameterMap([]parameter.NamedParameter{}), // missing name parameter leads to deployment failure
			Coordinate: coordinate.Coordinate{Type: theApiName, ConfigId: "config_1"},
			Template:   generateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
		{
			Parameters: toParameterMap([]parameter.NamedParameter{}), // missing name parameter leads to deployment failure
			Coordinate: coordinate.Coordinate{Type: theApiName, ConfigId: "config_2"},
			Template:   generateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApiName,
			},
		},
	}

	env := "test-environment"

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				env: project.ConfigsPerType{
					theApiName: configs,
				},
			},
		},
	}

	c := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: env}: deploy.DummyClientSet,
	}

	t.Run("deployment error - stop on error", func(t *testing.T) {
		envErrs := deploy.DeployConfigGraph(p, c, deploy.DeployConfigsOptions{})
		assert.Len(t, envErrs, 1)
		assert.Len(t, envErrs[env], 1, "Expected deployment to return after the first error")
	})

	t.Run("deployment error - continue on error", func(t *testing.T) {
		envErrs := deploy.DeployConfigGraph(p, c, deploy.DeployConfigsOptions{ContinueOnErr: true})
		assert.Len(t, envErrs, 1)
		assert.Len(t, envErrs[env], 2, "Expected deployment to continue after the first error and return errors for both invalid configs")
	})

}

func TestDeployConfigGraph_DoesNotDeployConfigsDependingOnSkippedConfigs(t *testing.T) {
	projectId := "project1"
	referencedProjectId := "project2"
	environmentName := "dev"

	dashboardApiId := "dashboard"
	dashboardConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "sample dashboard",
	}

	autoTagApiId := "auto-tag"
	autoTagConfigId := "tag"
	autoTagCoordinates := coordinate.Coordinate{
		Project:  referencedProjectId,
		Type:     autoTagApiId,
		ConfigId: autoTagConfigId,
	}

	referencedPropertyName := "tagId"

	individualConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "Random Dashboard",
	}

	projects := []project.Project{
		{
			Id: projectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					dashboardApiId: []config.Config{
						{
							Coordinate:  dashboardConfigCoordinate,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"autoTagId": &parameter.DummyParameter{
									References: []parameter.ParameterReference{
										{
											Config:   autoTagCoordinates,
											Property: referencedPropertyName,
										},
									},
								},
							},
						},
						{
							Coordinate:  individualConfigCoordinate,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"name": &parameter.DummyParameter{
									Value: "sample",
								},
								"dashboard": &parameter.DummyParameter{
									References: []parameter.ParameterReference{
										{
											Config:   dashboardConfigCoordinate,
											Property: "autoTagId",
										},
									},
								},
							},
						},
					},
				},
			},
			Dependencies: project.DependenciesPerEnvironment{
				environmentName: []string{
					referencedProjectId,
				},
			},
		},
		{
			Id: referencedProjectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					autoTagApiId: []config.Config{
						{
							Coordinate:  autoTagCoordinates,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								referencedPropertyName: &parameter.DummyParameter{
									Value: "10",
								},
							},
							Skip: true,
						},
					},
				},
			},
		},
	}

	environments := []string{
		environmentName,
	}

	graphs := graph.New(projects, environments)
	components, err := graphs.GetIndependentlySortedConfigs(environmentName)
	assert.NoError(t, err)
	assert.Len(t, components, 1)

	dummyClient := dtclient.DummyClient{}
	clientSet := deploy.ClientSet{
		Classic:  &dummyClient,
		Settings: &dummyClient,
	}

	clients := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: environmentName}: clientSet,
	}

	errs := deploy.DeployConfigGraph(projects, clients, deploy.DeployConfigsOptions{})
	assert.Len(t, errs, 0)
	assert.Len(t, dummyClient.Entries, 0)
}

func TestDeployConfigGraph_DeploysIndependentConfigurations(t *testing.T) {
	projectId := "project1"
	referencedProjectId := "project2"
	environmentName := "dev"

	dashboardApiId := "dashboard"
	dashboardConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "sample dashboard",
	}

	autoTagApiId := "auto-tag"
	autoTagConfigId := "tag"
	autoTagCoordinates := coordinate.Coordinate{
		Project:  referencedProjectId,
		Type:     autoTagApiId,
		ConfigId: autoTagConfigId,
	}

	referencedPropertyName := "tagId"

	individualConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "Random Dashboard",
	}
	individualConfigName := "Random Dashboard"

	projects := []project.Project{
		{
			Id: projectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					dashboardApiId: []config.Config{
						{
							Coordinate:  dashboardConfigCoordinate,
							Type:        config.ClassicApiType{Api: dashboardApiId},
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"autoTagId": &parameter.DummyParameter{
									References: []parameter.ParameterReference{
										{
											Config:   autoTagCoordinates,
											Property: referencedPropertyName,
										},
									},
								},
							},
							Template: generateDummyTemplate(t),
						},
						{
							Coordinate:  individualConfigCoordinate,
							Type:        config.ClassicApiType{Api: dashboardApiId},
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"name": &parameter.DummyParameter{
									Value: individualConfigName,
								},
							},
							Template: generateDummyTemplate(t),
						},
					},
				},
			},
			Dependencies: project.DependenciesPerEnvironment{
				environmentName: []string{
					referencedProjectId,
				},
			},
		},
		{
			Id: referencedProjectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					autoTagApiId: []config.Config{
						{
							Coordinate:  autoTagCoordinates,
							Type:        config.ClassicApiType{Api: autoTagApiId},
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								referencedPropertyName: &parameter.DummyParameter{
									Value: "10",
								},
							},
							Template: generateDummyTemplate(t),
							Skip:     true,
						},
					},
				},
			},
		},
	}

	environments := []string{
		environmentName,
	}

	graphs := graph.New(projects, environments)
	components, err := graphs.GetIndependentlySortedConfigs(environmentName)
	assert.NoError(t, err)
	assert.Len(t, components, 2)

	dummyClient := dtclient.DummyClient{}
	clientSet := deploy.ClientSet{
		Classic:  &dummyClient,
		Settings: &dummyClient,
	}

	clients := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: environmentName}: clientSet,
	}

	errs := deploy.DeployConfigGraph(projects, clients, deploy.DeployConfigsOptions{})
	assert.Len(t, errs, 0)
	assert.Len(t, dummyClient.Entries, 1)
	dashboards := dummyClient.Entries[api.NewAPIs()["dashboard"]]
	assert.Len(t, dashboards, 1)

	assert.Equal(t, dashboards[0].Name, individualConfigName)
}

func TestDeployConfigGraph_DeploysIndependentConfigurations_IfContinuingAfterFailure(t *testing.T) {
	projectId := "project1"
	referencedProjectId := "project2"
	environmentName := "dev"

	dashboardApiId := "dashboard"
	dashboardConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "sample dashboard",
	}

	autoTagApiId := "auto-tag"
	autoTagConfigId := "tag"
	autoTagCoordinates := coordinate.Coordinate{
		Project:  referencedProjectId,
		Type:     autoTagApiId,
		ConfigId: autoTagConfigId,
	}

	referencedPropertyName := "tagId"

	individualConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "Random Dashboard",
	}
	individualConfigName := "Random Dashboard"

	projects := []project.Project{
		{
			Id: projectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					dashboardApiId: []config.Config{
						{
							Coordinate:  dashboardConfigCoordinate,
							Type:        config.ClassicApiType{Api: dashboardApiId},
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"autoTagId": &parameter.DummyParameter{
									References: []parameter.ParameterReference{
										{
											Config:   autoTagCoordinates,
											Property: referencedPropertyName,
										},
									},
								},
							},
							Template: generateDummyTemplate(t),
						},
						{
							Coordinate:  individualConfigCoordinate,
							Type:        config.ClassicApiType{Api: dashboardApiId},
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"name": &parameter.DummyParameter{
									Value: individualConfigName,
								},
							},
							Template: generateDummyTemplate(t),
						},
					},
				},
			},
			Dependencies: project.DependenciesPerEnvironment{
				environmentName: []string{
					referencedProjectId,
				},
			},
		},
		{
			Id: referencedProjectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					autoTagApiId: []config.Config{
						{
							Coordinate:  autoTagCoordinates,
							Type:        config.ClassicApiType{Api: autoTagApiId},
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								referencedPropertyName: &parameter.DummyParameter{
									Value: "10",
								},
							},
							Template: generateFaultyTemplate(t), // deploying this will fail, and should result in the dependent dashboard not being deployed either
						},
					},
				},
			},
		},
	}

	environments := []string{
		environmentName,
	}

	graphs := graph.New(projects, environments)
	components, err := graphs.GetIndependentlySortedConfigs(environmentName)
	assert.NoError(t, err)
	assert.Len(t, components, 2)

	dummyClient := dtclient.DummyClient{}
	clientSet := deploy.ClientSet{
		Classic:  &dummyClient,
		Settings: &dummyClient,
	}

	clients := deploy.EnvironmentClients{
		deploy.EnvironmentInfo{Name: environmentName}: clientSet,
	}

	errs := deploy.DeployConfigGraph(projects, clients, deploy.DeployConfigsOptions{ContinueOnErr: true})
	assert.Len(t, errs, 1)
	assert.Len(t, dummyClient.Entries, 1)
	dashboards := dummyClient.Entries[api.NewAPIs()["dashboard"]]
	assert.Len(t, dashboards, 1)

	assert.Equal(t, dashboards[0].Name, individualConfigName)
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

func generateFaultyTemplate(t *testing.T) template.Template {
	newUUID, err := uuid.NewUUID()
	assert.NoError(t, err)
	templ := template.CreateTemplateFromString("deploy_test-"+newUUID.String(), "{")
	return templ
}
