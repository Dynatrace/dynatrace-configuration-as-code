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

package topologysort

import (
	"testing"

	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/test"
	"gotest.tools/assert"
)

func TestIsReferencing(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := ParameterWithName{
		Name: "name",
		Parameter: &test.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   referencingConfig,
					Property: referencingProperty,
				},
			},
		},
	}

	referencedParameter := ParameterWithName{
		Name:      referencingProperty,
		Parameter: &test.DummyParameter{},
	}

	result := param.IsReferencing(referencingConfig, referencedParameter)

	assert.Assert(t, result, "should reference paramter")
}

func TestIsReferencingShouldReturnFalseForNotReferencing(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := ParameterWithName{
		Name: "name",
		Parameter: &test.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   referencingConfig,
					Property: referencingProperty,
				},
			},
		},
	}

	referencedParameter := ParameterWithName{
		Name:      "name",
		Parameter: &test.DummyParameter{},
	}

	result := param.IsReferencing(referencingConfig, referencedParameter)

	assert.Assert(t, !result, "should not reference paramter")
}

func TestIsReferencingShouldReturnFalseForParameterWithoutReferences(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}

	param := ParameterWithName{
		Name:      "name",
		Parameter: &test.DummyParameter{},
	}

	referencedParameter := ParameterWithName{
		Name:      "name",
		Parameter: &test.DummyParameter{},
	}

	result := param.IsReferencing(referencingConfig, referencedParameter)

	assert.Assert(t, !result, "should not reference paramter")
}

func TestSortParameters(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashbord",
		Config:  "dashboard-1",
	}

	ownerParameterName := "owner"
	timeoutParameterName := "timeout"

	parameters := config.Parameters{
		config.NameParameter: &test.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: ownerParameterName,
				},
			},
		},
		ownerParameterName:   &test.DummyParameter{},
		timeoutParameterName: &test.DummyParameter{},
	}

	sortedParams, err := SortParameters("", "dev", configCoordinates, parameters)

	assert.NilError(t, err)
	assert.Assert(t, len(sortedParams) == len(parameters), "the same number of parameters should be sorted")

	indexName := indexOfParam(t, sortedParams, config.NameParameter)
	indexOwner := indexOfParam(t, sortedParams, ownerParameterName)

	assert.Assert(t, indexName > indexOwner, "parameter name (index %d) must be after parameter owner (%d)", indexName, indexOwner)
}

func TestSortParametersShouldFailOnCircularDependency(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashbord",
		Config:  "dashboard-1",
	}

	ownerParameterName := "owner"

	parameters := config.Parameters{
		config.NameParameter: &test.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: ownerParameterName,
				},
			},
		},
		ownerParameterName: &test.DummyParameter{
			References: []parameter.ParameterReference{
				{
					Config:   configCoordinates,
					Property: config.NameParameter,
				},
			},
		},
	}

	_, err := SortParameters("", "dev", configCoordinates, parameters)

	assert.Assert(t, err != nil, "should fail")
}

func TestSortConfigs(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}
	configCoordinates2 := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashboard",
		Config:  "dashboard-2",
	}
	referencedConfigCoordinates := coordinate.Coordinate{
		Project: "project-1",
		Api:     "auto-tags",
		Config:  "tags",
	}

	configs := []config.Config{
		{
			Coordinate:  configCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			References: []coordinate.Coordinate{
				referencedConfigCoordinates,
			},
			Skip: false,
		},
		{
			Coordinate:  configCoordinates2,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			References:  []coordinate.Coordinate{},
			Skip:        false,
		},
		{
			Coordinate:  referencedConfigCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			References:  []coordinate.Coordinate{},
			Skip:        false,
		},
	}

	sorted, err := sortConfigs(configs)

	assert.NilError(t, err)
	assert.Assert(t, len(configs) == len(sorted), "len configs (%d) == len sorted (%d)", len(configs), len(sorted))

	indexConfig := indexOfConfig(t, sorted, configCoordinates)
	indexReferenced := indexOfConfig(t, sorted, referencedConfigCoordinates)

	assert.Assert(t, indexReferenced < indexConfig, "referenced config (index %d) should be before config (index %d)", indexReferenced, indexConfig)
}

func TestSortConfigsShouldFailOnCyclicDependency(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}
	referencedConfigCoordinates := coordinate.Coordinate{
		Project: "project-1",
		Api:     "auto-tags",
		Config:  "tags",
	}

	configs := []config.Config{
		{
			Coordinate:  configCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			References: []coordinate.Coordinate{
				referencedConfigCoordinates,
			},
			Skip: false,
		},
		{
			Coordinate:  referencedConfigCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			References: []coordinate.Coordinate{
				configCoordinates,
			},
			Skip: false,
		},
	}

	_, err := sortConfigs(configs)

	assert.Assert(t, err != nil, "should fail")
}

func TestSortConfigsShouldNotFailOnCyclicDependencyWhichAreSkip(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project: "project-1",
		Api:     "dashboard",
		Config:  "dashboard-1",
	}
	referencedConfigCoordinates := coordinate.Coordinate{
		Project: "project-1",
		Api:     "auto-tags",
		Config:  "tags",
	}

	configs := []config.Config{
		{
			Coordinate:  configCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			References: []coordinate.Coordinate{
				referencedConfigCoordinates,
			},
			Skip: true,
		},
		{
			Coordinate:  referencedConfigCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			References: []coordinate.Coordinate{
				configCoordinates,
			},
			Skip: true,
		},
		{
			Coordinate: coordinate.Coordinate{
				Project: "project-1",
				Api:     "dashboard",
				Config:  "dashboard-2",
			},
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			References:  []coordinate.Coordinate{},
			Skip:        false,
		},
	}

	_, err := sortConfigs(configs)

	assert.NilError(t, err)
}

func TestSortProjects(t *testing.T) {
	projectId := "project-1"
	projectId2 := "project-2"
	referencedProjectId := "project-3"

	environmentName := "dev"

	environments := []string{
		environmentName,
	}

	projects := []project.Project{
		{
			Id: projectId,
			Dependencies: project.DependenciesPerEnvironment{
				environmentName: []string{
					referencedProjectId,
				},
			},
		},
		{
			Id: projectId2,
		},
		{
			Id: referencedProjectId,
		},
	}

	sorted, errors := sortProjects(projects, environments)

	assert.Assert(t, len(errors) == 0, "there should be no errors (no errors %d)", len(errors))
	assert.Assert(t, len(sorted) == 1, "there should be exactly one environments")

	projectsForEnvironment := sorted[environmentName]
	assert.Assert(t, len(projectsForEnvironment) == len(projects), "there should be exactly the same amount of environments")

	indexProject := indexOfProject(t, projectsForEnvironment, projectId)
	indexReferenced := indexOfProject(t, projectsForEnvironment, referencedProjectId)

	assert.Assert(t, indexReferenced < indexProject,
		"referenced project (index %d) should be before project (index %d)", indexReferenced, indexProject)
}

func TestSortProjectsShouldFailOnCyclicDependency(t *testing.T) {
	projectId := "project-1"
	referencedProjectId := "project-3"

	environmentName := "dev"

	environments := []string{
		environmentName,
	}

	projects := []project.Project{
		{
			Id: projectId,
			Dependencies: project.DependenciesPerEnvironment{
				environmentName: []string{
					referencedProjectId,
				},
			},
		},
		{
			Id: referencedProjectId,
			Dependencies: project.DependenciesPerEnvironment{
				environmentName: []string{
					projectId,
				},
			},
		},
	}

	_, errors := sortProjects(projects, environments)

	assert.Assert(t, len(errors) > 0, "there should be errors (no errors %d)", len(errors))
}

func TestGetSortedConfigsForEnvironments(t *testing.T) {
	projectId := "project1"
	referencedProjectId := "project2"
	environmentName := "dev"

	dashboardApiId := "dashboard"
	dashboardConfigCoordinate := coordinate.Coordinate{
		Project: projectId,
		Api:     dashboardApiId,
		Config:  "sample dashboard",
	}

	autoTagApiId := "auto-tag"
	autoTagConfigId := "tag"
	autoTagCoordinates := coordinate.Coordinate{
		Project: referencedProjectId,
		Api:     autoTagApiId,
		Config:  autoTagConfigId,
	}

	referencedPropertyName := "tagId"

	projects := []project.Project{
		{
			Id: projectId,
			Configs: project.ConfigsPerApisPerEnvironments{
				environmentName: {
					dashboardApiId: []config.Config{
						{
							Coordinate:  dashboardConfigCoordinate,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"autoTagId": &test.DummyParameter{
									References: []parameter.ParameterReference{
										{
											Config:   autoTagCoordinates,
											Property: referencedPropertyName,
										},
									},
								},
							},
							References: []coordinate.Coordinate{
								autoTagCoordinates,
							},
						},
						{
							Coordinate: coordinate.Coordinate{
								Project: projectId,
								Api:     dashboardApiId,
								Config:  "Random Dashboard",
							},
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"name": &test.DummyParameter{
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
			Configs: project.ConfigsPerApisPerEnvironments{
				environmentName: {
					autoTagApiId: []config.Config{
						{
							Coordinate:  autoTagCoordinates,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								referencedPropertyName: &test.DummyParameter{
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

	sortedPerEnvironment, errors := GetSortedConfigsForEnvironments(projects, environments)

	assert.Assert(t, len(errors) == 0, "should not return error")
	assert.Assert(t, len(sortedPerEnvironment) == 1)

	sorted := sortedPerEnvironment[environmentName]

	assert.Assert(t, len(sorted) == 3)

	dashboardIndex := indexOfConfig(t, sorted, dashboardConfigCoordinate)
	autoTagIndex := indexOfConfig(t, sorted, autoTagCoordinates)

	assert.Assert(t, autoTagIndex < dashboardIndex,
		"auto-tag (index %d) should be deployed before dashboard (index %d)", autoTagIndex, dashboardIndex)
}

func indexOfProject(t *testing.T, projects []project.Project, projectId string) int {
	for i, p := range projects {
		if p.Id == projectId {
			return i
		}
	}

	t.Fatalf("no project with name `%s` found", projectId)
	return -1
}

func indexOfConfig(t *testing.T, configs []config.Config, coordinate coordinate.Coordinate) int {
	for i, c := range configs {
		if c.Coordinate == coordinate {
			return i
		}
	}

	t.Fatalf("no config `%s` found", coordinate.ToString())
	return -1
}

func indexOfParam(t *testing.T, params []ParameterWithName, name string) int {
	for i, p := range params {
		if p.Name == name {
			return i
		}
	}

	t.Fatalf("no parameter with name `%s` found", name)
	return -1
}
