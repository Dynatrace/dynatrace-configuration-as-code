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

package sort_test

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSortParameters(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	ownerParameterName := "owner"
	timeoutParameterName := "timeout"

	parameters := config.Parameters{
		config.NameParameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: ownerParameterName,
				},
			},
		},
		ownerParameterName:   &parameter.DummyParameter{},
		timeoutParameterName: &parameter.DummyParameter{},
	}

	sortedParams, errs := sort.SortParameters("", "dev", configCoordinates, parameters)

	assert.Len(t, errs, 0, "expected zero errors when sorting")
	assert.Equal(t, len(sortedParams), len(parameters), "the same number of parameters should be sorted")

	indexName := indexOfParam(t, sortedParams, config.NameParameter)
	indexOwner := indexOfParam(t, sortedParams, ownerParameterName)

	assert.Greaterf(t, indexName, indexOwner, "parameter name (index %d) must be after parameter owner (%d)", indexName, indexOwner)
}

func TestSortParametersShouldFailOnCircularDependency(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	ownerParameterName := "owner"

	parameters := config.Parameters{
		config.NameParameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: ownerParameterName,
				},
			},
		},
		ownerParameterName: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: config.NameParameter,
				},
			},
		},
	}

	_, errs := sort.SortParameters("", "dev", configCoordinates, parameters)

	assert.True(t, len(errs) > 0, "should fail")
}

func indexOfParam(t *testing.T, params []parameter.NamedParameter, name string) int {
	for i, p := range params {
		if p.Name == name {
			return i
		}
	}

	t.Fatalf("no parameter with name `%s` found", name)
	return -1
}

func TestGetSortedConfigsForEnvironments(t *testing.T) {
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
							Coordinate: coordinate.Coordinate{
								Project:  projectId,
								Type:     dashboardApiId,
								ConfigId: "Random Dashboard",
							},
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"name": &parameter.DummyParameter{
									Value: "sample",
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
						},
					},
				},
			},
		},
	}

	environments := []string{
		environmentName,
	}

	t.Run("Topology Sort", func(t *testing.T) {
		t.Setenv(featureflags.UseGraphs().EnvName(), "false")
		assertSortingWorks(t, projects, environments, environmentName, dashboardConfigCoordinate, autoTagCoordinates)
	})

	t.Run("Graph-based sort", func(t *testing.T) {
		t.Setenv(featureflags.UseGraphs().EnvName(), "true")
		assertSortingWorks(t, projects, environments, environmentName, dashboardConfigCoordinate, autoTagCoordinates)
	})

}

func assertSortingWorks(t *testing.T, projects []project.Project, environments []string, environmentName string, dashboardConfigCoordinate coordinate.Coordinate, autoTagCoordinates coordinate.Coordinate) {
	sortedPerEnvironment, errors := sort.GetSortedConfigsForEnvironments(projects, environments)

	assert.Len(t, errors, 0, "should not return error")
	assert.Len(t, sortedPerEnvironment, 1)

	sorted := sortedPerEnvironment[environmentName]

	assert.Len(t, sorted, 3)

	dashboardIndex := indexOfConfig(t, sorted, dashboardConfigCoordinate)
	autoTagIndex := indexOfConfig(t, sorted, autoTagCoordinates)

	assert.Less(t, autoTagIndex, dashboardIndex,
		"auto-tag (index %d) should be deployed before dashboard (index %d)", autoTagIndex, dashboardIndex)
}

func indexOfConfig(t *testing.T, configs []config.Config, coordinate coordinate.Coordinate) int {
	for i, c := range configs {
		if c.Coordinate == coordinate {
			return i
		}
	}

	t.Fatalf("no config `%s` found", coordinate)
	return -1
}
