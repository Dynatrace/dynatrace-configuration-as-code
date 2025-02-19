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

package deploy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

var dashboardApi = api.API{ID: "dashboard", URLPath: "dashboard", DeprecatedBy: "dashboard-v2"}
var testApiMap = api.APIs{"dashboard": dashboardApi}

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

	dryRunConfigClient := dtclient.DryRunConfigClient{}
	clientSet := &client.ClientSet{ConfigClient: &dryRunConfigClient, SettingsClient: &dtclient.DryRunSettingsClient{}}

	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: clientSet,
	}

	errors := deploy.Deploy(context.TODO(), p, c, deploy.DeployConfigsOptions{})

	assert.Emptyf(t, errors, "errors: %v", errors)

	createdEntities, found := dryRunConfigClient.GetEntries(api.NewAPIs()["dashboard"])
	assert.True(t, found, "expected entries for dashboard API to exist in dry-run client after deployment")
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

	c := client.NewMockSettingsClient(gomock.NewController(t))
	c.EXPECT().Cache(gomock.Any(), gomock.Eq("builtin:test")).Times(1)
	c.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Return(dtclient.DynatraceEntity{}, fmt.Errorf("upsert failed"))

	conf := config.Config{
		Type: config.SettingsType{
			SchemaId: "builtin:test",
		},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap(parameters),
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

	clients := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &client.ClientSet{SettingsClient: c},
	}

	errors := deploy.Deploy(context.TODO(), p, clients, deploy.DeployConfigsOptions{})
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

	clientSet := client.ClientSet{ConfigClient: &dtclient.DryRunConfigClient{}, SettingsClient: &dtclient.DryRunSettingsClient{}}

	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &clientSet,
	}

	errors := deploy.Deploy(context.TODO(), p, c, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigGraph_DoesNotFailOnEmptyProject(t *testing.T) {

	var p []project.Project

	clientSet := client.ClientSet{ConfigClient: &dtclient.DryRunConfigClient{}, SettingsClient: &dtclient.DryRunSettingsClient{}}

	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &clientSet,
	}

	errors := deploy.Deploy(context.TODO(), p, c, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigGraph_DoesNotFailNilProject(t *testing.T) {
	clientSet := client.ClientSet{}
	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &clientSet,
	}

	errors := deploy.Deploy(context.TODO(), nil, c, deploy.DeployConfigsOptions{})
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

	dryRunConfigClient := dtclient.DryRunConfigClient{}
	clientSet := client.ClientSet{ConfigClient: &dryRunConfigClient, SettingsClient: &dtclient.DryRunSettingsClient{}}

	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &clientSet,
	}

	errors := deploy.Deploy(context.TODO(), p, c, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
	createdEntities, found := dryRunConfigClient.GetEntries(api.NewAPIs()["dashboard"])
	assert.False(t, found, "expected NO entries for dashboard API to exist")
	assert.Len(t, createdEntities, 0)
}

func TestDeployConfigGraph_DeploysSetting(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))

	configs := []config.Config{
		{
			Template: testutils.GenerateDummyTemplate(t),
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
	c.EXPECT().Cache(gomock.Any(), gomock.Eq("builtin:test")).Times(1)
	c.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
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

	clientSet := client.ClientSet{SettingsClient: c}

	clients := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &clientSet,
	}

	errors := deploy.Deploy(context.TODO(), p, clients, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingClassicConfigUnique(t *testing.T) {
	theConfigName := "theConfigName"
	theApi := api.NewAPIs()["management-zone"]

	cl := client.NewMockConfigClient(gomock.NewController(t))
	cl.EXPECT().Cache(gomock.Any(), gomock.Eq(theApi)).Times(1)
	cl.EXPECT().UpsertByName(gomock.Any(), gomock.Any(), theConfigName, gomock.Any()).Times(1)

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
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApi.ID},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApi.ID,
			},
		},
	}

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					theApi.ID: configs,
				},
			},
		},
	}

	clientSet := client.ClientSet{ConfigClient: cl}
	clients := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &clientSet,
	}

	errors := deploy.Deploy(context.TODO(), p, clients, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsTargetingClassicConfigNonUniqueWithExistingCfgsOfSameName(t *testing.T) {
	theConfigName := "theConfigName"
	theApiName := "alerting-profile"

	cl := client.NewMockConfigClient(gomock.NewController(t))
	cl.EXPECT().Cache(gomock.Any(), gomock.Eq(api.NewAPIs()[theApiName])).Times(1)
	cl.EXPECT().UpsertByNonUniqueNameAndId(gomock.Any(), gomock.Any(), gomock.Any(), theConfigName, gomock.Any(), false)

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
			Parameters: testutils.ToParameterMap(parameters),
			Coordinate: coordinate.Coordinate{Type: theApiName},
			Template:   testutils.GenerateDummyTemplate(t),
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

	clientSet := client.ClientSet{ConfigClient: cl}
	clients := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &clientSet,
	}

	errors := deploy.Deploy(context.TODO(), p, clients, deploy.DeployConfigsOptions{})
	assert.Emptyf(t, errors, "there should be no errors (errors: %v)", errors)
}

func TestDeployConfigsWithDeploymentErrors(t *testing.T) {
	theApi := api.NewAPIs()["management-zone"]
	env := "test-environment"

	configs := []config.Config{
		{
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{
				{Name: "name", Parameter: value.New("something")},
				{Name: "invalid-ref", Parameter: reference.New("proj", "non-existing-type", "id", "prop")}, // non-existing reference leads to deployment failure
			}),
			Coordinate: coordinate.Coordinate{Type: theApi.ID, ConfigId: "config_1"},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApi.ID,
			},
			Environment: env,
		},
		{
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{
				{Name: "name", Parameter: value.New("something else")},
				{Name: "invalid-ref", Parameter: reference.New("proj", "non-existing-type", "id", "prop")}, // non-existing reference leads to deployment failure
			}),
			Coordinate: coordinate.Coordinate{Type: theApi.ID, ConfigId: "config_2"},
			Template:   testutils.GenerateDummyTemplate(t),
			Type: config.ClassicApiType{
				Api: theApi.ID,
			},
			Environment: env,
		},
	}

	p := []project.Project{
		{
			Id: "proj",
			Configs: project.ConfigsPerTypePerEnvironments{
				env: project.ConfigsPerType{
					theApi.ID: configs,
				},
			},
		},
	}

	clientSet := client.ClientSet{ConfigClient: &dtclient.DryRunConfigClient{}, SettingsClient: &dtclient.SettingsClient{}}

	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: env}: &clientSet,
	}

	t.Run("deployment error - always continues on error", func(t *testing.T) {

		err := deploy.Deploy(context.TODO(), p, c, deploy.DeployConfigsOptions{}) // continues even without option set
		assert.Error(t, err)

		envErrs := make(errors.EnvironmentDeploymentErrors)
		assert.ErrorAs(t, err, &envErrs)
		assert.Len(t, envErrs, 1)
		assert.Len(t, envErrs[env], 1)
		var depErr errors.DeploymentErrors
		assert.ErrorAs(t, envErrs[env][0], &depErr)
		assert.Equal(t, 2, depErr.ErrorCount, "Expected deployment to continue after the first error and count errors for both invalid configs")
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
							Type:        config.ClassicApiType{Api: "dashboard"},
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
							Type:        config.ClassicApiType{Api: "dashboard"},
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
							Type:        config.ClassicApiType{Api: "auto-tag"},
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

	dryRunConfigClient := dtclient.DryRunConfigClient{}
	clientSet := client.ClientSet{ConfigClient: &dryRunConfigClient, SettingsClient: &dtclient.DryRunSettingsClient{}}

	clients := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: environmentName}: &clientSet,
	}

	errs := deploy.Deploy(context.TODO(), projects, clients, deploy.DeployConfigsOptions{})
	assert.NoError(t, errs)
	assert.Zero(t, dryRunConfigClient.CreatedObjects())
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
							Template: testutils.GenerateDummyTemplate(t),
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
							Template: testutils.GenerateDummyTemplate(t),
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
							Template: testutils.GenerateDummyTemplate(t),
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

	dryRunConfigClient := dtclient.DryRunConfigClient{}
	clientSet := client.ClientSet{ConfigClient: &dryRunConfigClient, SettingsClient: &dtclient.DryRunSettingsClient{}}
	clients := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: environmentName}: &clientSet,
	}

	errs := deploy.Deploy(context.TODO(), projects, clients, deploy.DeployConfigsOptions{})
	assert.NoError(t, errs)

	dashboards, found := dryRunConfigClient.GetEntries(api.NewAPIs()["dashboard"])
	assert.True(t, found, "expected entries for dashboard API to exist in dry-run client after deployment")
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
							Template: testutils.GenerateDummyTemplate(t),
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
							Template: testutils.GenerateDummyTemplate(t),
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
							Template: testutils.GenerateFaultyTemplate(t), // deploying this will fail, and should result in the dependent dashboard not being deployed either
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

	dryRunConfigClient := dtclient.DryRunConfigClient{}
	clientSet := client.ClientSet{ConfigClient: &dryRunConfigClient, SettingsClient: &dtclient.DryRunSettingsClient{}}

	clients := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: environmentName}: &clientSet,
	}

	errs := deploy.Deploy(context.TODO(), projects, clients, deploy.DeployConfigsOptions{ContinueOnErr: true})
	assert.Len(t, errs, 1)

	dashboards, found := dryRunConfigClient.GetEntries(api.NewAPIs()["dashboard"])
	assert.True(t, found, "expected entries for dashboard API to exist in dry-run client after deployment")
	assert.Len(t, dashboards, 1)

	assert.Equal(t, dashboards[0].Name, individualConfigName)
}

func TestDeployConfigsValidatesClassicAPINames(t *testing.T) {
	tests := []struct {
		name            string
		given           []project.Project
		wantErrsContain map[string][]string
	}{
		{
			name: "no duplicates if from different environment",
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
						"env2": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env2",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config2",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no duplicates if for different api",
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
								config.Config{
									Type:        config.ClassicApiType{Api: "custom-service-php"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config2",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no duplicates if for api allow duplicate names",
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "anomaly-detection-metrics"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
								config.Config{
									Type:        config.ClassicApiType{Api: "anomaly-detection-metrics"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config2",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate by value parameters",
			wantErrsContain: map[string][]string{
				"env1": {"::config2", "::config1"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config2",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate by environment parameter",
			wantErrsContain: map[string][]string{
				"env1": {"::config2", "::config1"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &environment.EnvironmentVariableParameter{
											Name:            "ENV_1",
											HasDefaultValue: true,
											DefaultValue:    "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config2",
									},
									Parameters: config.Parameters{
										config.NameParameter: &environment.EnvironmentVariableParameter{
											Name:            "ENV_1",
											HasDefaultValue: true,
											DefaultValue:    "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate by mix of value and environment parameter",
			wantErrsContain: map[string][]string{
				"env1": {"::config2", "::config1"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &environment.EnvironmentVariableParameter{
											Name:            "ENV_1",
											HasDefaultValue: true,
											DefaultValue:    "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config2",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate by reference parameter",
			wantErrsContain: map[string][]string{
				"env1": {"::config2", "::config1"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &reference.ReferenceParameter{
											ParameterReference: parameter.ParameterReference{
												Config: coordinate.Coordinate{
													Project:  "projA",
													Type:     "typeA",
													ConfigId: "ID",
												},
												Property: "property",
											},
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config2",
									},
									Parameters: config.Parameters{
										config.NameParameter: &reference.ReferenceParameter{
											ParameterReference: parameter.ParameterReference{
												Config: coordinate.Coordinate{
													Project:  "projA",
													Type:     "typeA",
													ConfigId: "ID",
												},
												Property: "property",
											},
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate in different projects",
			wantErrsContain: map[string][]string{
				"env1": {"p2:type:config2", "p1:type:config1"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"p1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										Project:  "p1",
										Type:     "type",
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
							"p2": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										Project:  "p2",
										Type:     "type",
										ConfigId: "config2",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
									Template: testutils.GenerateDummyTemplate(t),
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			dryRunConfigClient := dtclient.DryRunConfigClient{}
			clientSet := client.ClientSet{ConfigClient: &dryRunConfigClient, SettingsClient: &dtclient.DryRunSettingsClient{}}

			c := dynatrace.EnvironmentClients{
				dynatrace.EnvironmentInfo{Name: "env1"}: &clientSet,
				dynatrace.EnvironmentInfo{Name: "env2"}: &clientSet,
			}

			err := deploy.Deploy(context.TODO(), tc.given, c, deploy.DeployConfigsOptions{})
			if len(tc.wantErrsContain) == 0 {
				assert.NoError(t, err)
			} else {
				for env, errStrings := range tc.wantErrsContain {
					assert.ErrorContains(t, err, env)
					for _, s := range errStrings {
						assert.ErrorContains(t, err, s)
					}
				}
			}
		})
	}
}

func TestDeployConfigGraph_CollectsAllErrors(t *testing.T) {

	p := []project.Project{
		{
			Id: "project",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					"app-detection-rule": []config.Config{
						{
							Type:     config.ClassicApiType{Api: "app-detection-rule"},
							Template: testutils.GenerateFaultyTemplate(t), // deploying this will fail, and should result in the dependent dashboard not being deployed either
							Coordinate: coordinate.Coordinate{
								Project:  "project",
								Type:     "dashboard",
								ConfigId: "WILL FAIL DEPLOYMENT",
							},
							Environment: "env",
							Parameters: testutils.ToParameterMap(
								[]parameter.NamedParameter{
									{
										Name:      config.NameParameter,
										Parameter: &parameter.DummyParameter{Value: "Some Name"},
									},
								}),
							Skip: false,
						},
						{
							Type:     config.ClassicApiType{Api: "app-detection-rule"},
							Template: testutils.GenerateDummyTemplate(t),
							Coordinate: coordinate.Coordinate{
								Project:  "project",
								Type:     "app-detection-rule",
								ConfigId: "WILL FAIL VALIDATION - Overlapping Name #1",
							},
							Environment: "env",
							Parameters: testutils.ToParameterMap(
								[]parameter.NamedParameter{
									{
										Name:      config.NameParameter,
										Parameter: &parameter.DummyParameter{Value: "DUPLICATE NAME"},
									},
								}),
							Skip: false,
						},
						{
							Type:     config.ClassicApiType{Api: "app-detection-rule"},
							Template: testutils.GenerateDummyTemplate(t),
							Coordinate: coordinate.Coordinate{
								Project:  "project",
								Type:     "app-detection-rule",
								ConfigId: "WILL FAIL VALIDATION - Overlapping Name #2",
							},
							Environment: "env",
							Parameters: testutils.ToParameterMap(
								[]parameter.NamedParameter{
									{
										Name:      config.NameParameter,
										Parameter: &parameter.DummyParameter{Value: "DUPLICATE NAME"},
									},
								}),
							Skip: false,
						},
					},
				},
			},
		},
	}

	dryRunConfigClient := dtclient.DryRunConfigClient{}
	clientSet := client.ClientSet{ConfigClient: &dryRunConfigClient, SettingsClient: &dtclient.DryRunSettingsClient{}}

	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &clientSet,
	}

	t.Run("stop on error - returns validation errors", func(t *testing.T) {
		errs := deploy.Deploy(context.TODO(), p, c, deploy.DeployConfigsOptions{})
		assert.Error(t, errs)

		var envErrs errors.EnvironmentDeploymentErrors
		assert.ErrorAs(t, errs, &envErrs)
		assert.Len(t, envErrs["env"], 1)
		assert.ErrorContains(t, envErrs["env"][0], "duplicated config name")
		assert.ErrorContains(t, envErrs["env"][0], "WILL FAIL VALIDATION")
	})

	t.Run("continue on error - returns validation and deployment", func(t *testing.T) {
		errs := deploy.Deploy(context.TODO(), p, c, deploy.DeployConfigsOptions{ContinueOnErr: true})
		assert.Error(t, errs)

		var envErrs errors.EnvironmentDeploymentErrors
		assert.ErrorAs(t, errs, &envErrs)
		assert.Len(t, envErrs["env"], 2)
		assert.ErrorContains(t, envErrs["env"][0], "WILL FAIL VALIDATION")
		var depErr errors.DeploymentErrors
		assert.ErrorAs(t, envErrs["env"][1], &depErr)
		assert.Equal(t, 1, depErr.ErrorCount, "Expected one deployment error to be counted")
	})

}

func TestDeployConfigFF(t *testing.T) {
	testClientSet := client.ClientSet{
		SegmentClient:               client.TestSegmentsClient{},
		ServiceLevelObjectiveClient: client.TestServiceLevelObjectiveClient{},
	}
	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env"}: &testClientSet,
	}
	tests := []struct {
		name              string
		projects          []project.Project
		featureFlag       string
		configType        config.TypeID
		expectedErrString string
	}{
		{
			name: "segments FF test",
			projects: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env": project.ConfigsPerType{
							"p1": {
								config.Config{
									Type:        config.Segment{},
									Environment: "env",
									Coordinate: coordinate.Coordinate{
										Project:  "p1",
										Type:     "type",
										ConfigId: "config1",
									},
								},
							},
						},
					},
				},
			},
			featureFlag: featureflags.Segments.EnvName(),
			configType:  config.SegmentID,
		},
		{
			name: "SLO FF test",
			projects: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env": project.ConfigsPerType{
							"p1": {
								config.Config{
									Type:        config.ServiceLevelObjective{},
									Environment: "env",
									Coordinate: coordinate.Coordinate{
										Project:  "p1",
										Type:     "type",
										ConfigId: "config1",
									},
								},
							},
						},
					},
				},
			},
			featureFlag: featureflags.ServiceLevelObjective.EnvName(),
			configType:  config.ServiceLevelObjectiveID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" | FF Enabled", func(t *testing.T) {
			t.Setenv(tt.featureFlag, "true")
			err := deploy.Deploy(context.Background(), tt.projects, c, deploy.DeployConfigsOptions{})
			//fakeClient returns unimplemented error on every execution of any method
			assert.Errorf(t, err, "unimplemented")
		})
		t.Run(tt.name+" | FF Disabled", func(t *testing.T) {
			t.Setenv(tt.featureFlag, "false")
			err := deploy.Deploy(context.Background(), tt.projects, c, deploy.DeployConfigsOptions{})
			assert.Errorf(t, err, fmt.Sprintf("unknown config-type (ID: %q)", tt.configType))
		})
	}
}

func TestDeployDryRun(t *testing.T) {
	c := dynatrace.EnvironmentClients{
		dynatrace.EnvironmentInfo{Name: "env", Group: "group"}: &client.DryRunClientSet,
	}
	projects := []project.Project{
		{
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					"p1": {
						config.Config{
							Type:        config.Segment{},
							Environment: "env",
							Coordinate: coordinate.Coordinate{
								Project:  "p1",
								Type:     "segment",
								ConfigId: "config1",
							},
							Template: testutils.GenerateDummyTemplate(t),
						},
					},
				},
			},
		},
		{
			Configs: project.ConfigsPerTypePerEnvironments{
				"env": project.ConfigsPerType{
					"p1": {
						config.Config{
							Type:        config.ServiceLevelObjective{},
							Environment: "env",
							Coordinate: coordinate.Coordinate{
								Project:  "p1",
								Type:     "segment",
								ConfigId: "config1",
							},
							Template: testutils.GenerateDummyTemplate(t),
						},
					},
				},
			},
		},
	}
	t.Setenv(featureflags.Segments.EnvName(), "true")
	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")
	t.Run("dry-run", func(t *testing.T) {
		err := deploy.Deploy(context.Background(), projects, c, deploy.DeployConfigsOptions{DryRun: true})
		assert.Empty(t, err)
	})
}
