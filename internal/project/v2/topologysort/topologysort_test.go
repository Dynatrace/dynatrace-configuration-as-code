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

//go:build unit

package topologysort

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/util/sort"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"

	config "github.com/dynatrace/dynatrace-configuration-as-code/internal/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/config/v2/parameter"
	project "github.com/dynatrace/dynatrace-configuration-as-code/internal/project/v2"
	"gotest.tools/assert"
)

func TestIsReferencing(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := ParameterWithName{
		Name: "name",
		Parameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{Config: referencingConfig, Property: referencingProperty},
			},
		},
	}

	referencedParameter := ParameterWithName{
		Name:      referencingProperty,
		Parameter: &parameter.DummyParameter{},
	}

	result := param.IsReferencing(referencingConfig, referencedParameter)

	assert.Assert(t, result, "should reference parameter")
}

func TestIsReferencingShouldReturnFalseForNotReferencing(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := ParameterWithName{
		Name: "name",
		Parameter: &parameter.DummyParameter{
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
		Parameter: &parameter.DummyParameter{},
	}

	result := param.IsReferencing(referencingConfig, referencedParameter)

	assert.Assert(t, !result, "should not reference parameter")
}

func TestIsReferencingShouldReturnFalseForParameterWithoutReferences(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	param := ParameterWithName{
		Name:      "name",
		Parameter: &parameter.DummyParameter{},
	}

	referencedParameter := ParameterWithName{
		Name:      "name",
		Parameter: &parameter.DummyParameter{},
	}

	result := param.IsReferencing(referencingConfig, referencedParameter)

	assert.Assert(t, !result, "should not reference parameter")
}

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

	sortedParams, errs := SortParameters("", "dev", configCoordinates, parameters)

	assert.Equal(t, len(errs), 0, "expected zero errors when sorting")
	assert.Assert(t, len(sortedParams) == len(parameters), "the same number of parameters should be sorted")

	indexName := indexOfParam(t, sortedParams, config.NameParameter)
	indexOwner := indexOfParam(t, sortedParams, ownerParameterName)

	assert.Assert(t, indexName > indexOwner, "parameter name (index %d) must be after parameter owner (%d)", indexName, indexOwner)
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

	_, errs := SortParameters("", "dev", configCoordinates, parameters)

	assert.Assert(t, len(errs) > 0, "should fail")
}

func TestSortConfigs(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}
	configCoordinates2 := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-2",
	}
	referencedConfigCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "auto-tags",
		ConfigId: "tags",
	}

	configs := []config.Config{
		{
			Coordinate:  configCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			Skip:        false,
		},
		{
			Coordinate:  configCoordinates2,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			Skip:        false,
		},
		{
			Coordinate:  referencedConfigCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			Skip:        false,
		},
	}

	sorted, errs := sortConfigs(configs)

	assert.Equal(t, len(errs), 0, "expected zero errors when sorting")
	assert.Assert(t, len(configs) == len(sorted), "len configs (%d) == len sorted (%d)", len(configs), len(sorted))

	indexConfig := indexOfConfig(t, sorted, configCoordinates)
	indexReferenced := indexOfConfig(t, sorted, referencedConfigCoordinates)

	assert.Assert(t, indexReferenced < indexConfig, "referenced config (index %d) should be before config (index %d)", indexReferenced, indexConfig)
}

func TestSortConfigsShouldFailOnCyclicDependency(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}
	referencedConfigCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "auto-tags",
		ConfigId: "tags",
	}

	configs := []config.Config{
		{
			Coordinate:  configCoordinates,
			Environment: "development",
			Parameters: config.Parameters{
				"p": parameter.NewDummy(referencedConfigCoordinates),
			},
			Skip: false,
		},
		{
			Coordinate:  referencedConfigCoordinates,
			Environment: "development",
			Parameters: config.Parameters{
				"p": parameter.NewDummy(configCoordinates),
			},
			Skip: false,
		},
	}

	_, errs := sortConfigs(configs)

	assert.Assert(t, len(errs) > 0, "should fail")
}

func TestSortConfigsShouldReportAllLinksOfCyclicDependency(t *testing.T) {
	config1Coordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}
	config2Coordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "auto-tags",
		ConfigId: "tags",
	}
	config3Coordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "management-zone",
		ConfigId: "zone-1",
	}

	configs := []config.Config{
		{
			Coordinate:  config1Coordinates,
			Environment: "development",
			Parameters: config.Parameters{
				"p": parameter.NewDummy(config2Coordinates),
			},
			Skip: false,
		},
		{
			Coordinate:  config2Coordinates,
			Environment: "development",
			Parameters: config.Parameters{
				"p": parameter.NewDummy(config3Coordinates),
			},
			Skip: false,
		},
		{
			Coordinate:  config3Coordinates,
			Environment: "development",
			Parameters: config.Parameters{
				"p": parameter.NewDummy(config1Coordinates),
			},
			Skip: false,
		},
	}

	_, errs := sortConfigs(configs)

	assert.Assert(t, len(errs) > 0, "should report cyclic dependency errors")
	assert.Assert(t, len(errs) == 3, "should report an error for each config")
	for _, err := range errs {
		depErr, ok := err.(CircularDependencyConfigSortError)
		assert.Assert(t, ok, "expected errors of type CircularDependencyConfigSortError")
		if depErr.Config.Match(config1Coordinates) {
			assert.Assert(t, depErr.DependsOn[0] == config2Coordinates)
		}
		if depErr.Config.Match(config2Coordinates) {
			assert.Assert(t, depErr.DependsOn[0] == config3Coordinates)
		}
		if depErr.Config.Match(config3Coordinates) {
			assert.Assert(t, depErr.DependsOn[0] == config1Coordinates)
		}
	}
}

func TestSortConfigsShouldNotFailOnCyclicDependencyWhichAreSkip(t *testing.T) {
	configCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}
	referencedConfigCoordinates := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "auto-tags",
		ConfigId: "tags",
	}

	configs := []config.Config{
		{
			Coordinate:  configCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			Skip:        true,
		},
		{
			Coordinate:  referencedConfigCoordinates,
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			Skip:        true,
		},
		{
			Coordinate: coordinate.Coordinate{
				Project:  "project-1",
				Type:     "dashboard",
				ConfigId: "dashboard-2",
			},
			Environment: "development",
			Parameters:  map[string]parameter.Parameter{},
			Skip:        false,
		},
	}

	_, errs := sortConfigs(configs)

	assert.Equal(t, len(errs), 0, "expected zero errors when sorting")
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

	t.Fatalf("no config `%s` found", coordinate)
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

func Test_parseConfigSortErrors(t *testing.T) {
	testConfigs := []config.Config{
		{Coordinate: coordinate.Coordinate{
			Project:  "p1",
			Type:     "a1",
			ConfigId: "c1",
		}},
		{Coordinate: coordinate.Coordinate{
			Project:  "p1",
			Type:     "a1",
			ConfigId: "c2",
		}},
		{Coordinate: coordinate.Coordinate{
			Project:  "p1",
			Type:     "a1",
			ConfigId: "c3",
		}},
		{Coordinate: coordinate.Coordinate{
			Project:  "p1",
			Type:     "a2",
			ConfigId: "c1",
		}},
	}

	type args struct {
		sortErrs []sort.TopologySortError
		configs  []config.Config
	}
	tests := []struct {
		name string
		args args
		want []error
	}{
		{
			"returns empty list for empty input",
			args{
				[]sort.TopologySortError{},
				testConfigs,
			},
			[]error{},
		},
		{
			"parses simple errors into list",
			args{
				[]sort.TopologySortError{
					{
						OnId:                        0,
						UnresolvedIncomingEdgesFrom: []int{1, 2},
					},
					{
						OnId:                        2,
						UnresolvedIncomingEdgesFrom: []int{0},
					},
				},
				testConfigs,
			},
			[]error{
				CircularDependencyConfigSortError{
					Config:    testConfigs[0].Coordinate,
					DependsOn: []coordinate.Coordinate{testConfigs[2].Coordinate},
				},
				CircularDependencyConfigSortError{
					Config:    testConfigs[1].Coordinate,
					DependsOn: []coordinate.Coordinate{testConfigs[0].Coordinate},
				},
				CircularDependencyConfigSortError{
					Config:    testConfigs[2].Coordinate,
					DependsOn: []coordinate.Coordinate{testConfigs[0].Coordinate},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseConfigSortErrors(tt.args.sortErrs, tt.args.configs)
			assert.DeepEqual(t, got, tt.want, cmpopts.SortSlices(func(a, b error) bool {
				depErrA := a.(CircularDependencyConfigSortError)
				depErrB := b.(CircularDependencyConfigSortError)
				return depErrA.Config.String() < depErrB.Config.String()
			}))
		})
	}
}

func TestHasDependencyOn(t *testing.T) {
	referencedConfig := coordinate.Coordinate{
		Project:  "project1",
		Type:     "auto-tag",
		ConfigId: "tag",
	}

	conf := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard1",
		},
		Environment: "dev",
		Parameters: config.Parameters{
			"p": parameter.NewDummy(referencedConfig),
		},
	}

	referencedConf := config.Config{
		Coordinate:  referencedConfig,
		Environment: "dev",
	}

	result := hasDependencyOn(conf, referencedConf)

	assert.Assert(t, result, "should have dependency")
}

func TestHasDependencyOnShouldReturnFalseIfNoDependenciesAreDefined(t *testing.T) {
	conf := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard1",
		},
		Environment: "dev",
	}

	conf2 := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "auto-tag",
			ConfigId: "tag",
		},
		Environment: "dev",
	}

	result := hasDependencyOn(conf, conf2)

	assert.Assert(t, !result, "should not have dependency")
}
