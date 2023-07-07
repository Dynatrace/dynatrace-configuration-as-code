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

package topologysort

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/topologysort"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort/errors"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"gotest.tools/assert"
)

func TestIsReferencing(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := parameter.NamedParameter{
		Name: "name",
		Parameter: &parameter.DummyParameter{
			References: []parameter.ParameterReference{
				{Config: referencingConfig, Property: referencingProperty},
			},
		},
	}

	referencedParameter := parameter.NamedParameter{
		Name:      referencingProperty,
		Parameter: &parameter.DummyParameter{},
	}

	result := parameterReference(param, referencingConfig, referencedParameter)

	assert.Assert(t, result, "should reference parameter")
}

func TestIsReferencingShouldReturnFalseForNotReferencing(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	referencingProperty := "managementZoneName"

	param := parameter.NamedParameter{
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

	referencedParameter := parameter.NamedParameter{
		Name:      "name",
		Parameter: &parameter.DummyParameter{},
	}

	result := parameterReference(param, referencingConfig, referencedParameter)

	assert.Assert(t, !result, "should not reference parameter")
}

func TestIsReferencingShouldReturnFalseForParameterWithoutReferences(t *testing.T) {
	referencingConfig := coordinate.Coordinate{
		Project:  "project-1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	param := parameter.NamedParameter{
		Name:      "name",
		Parameter: &parameter.DummyParameter{},
	}

	referencedParameter := parameter.NamedParameter{
		Name:      "name",
		Parameter: &parameter.DummyParameter{},
	}

	result := parameterReference(param, referencingConfig, referencedParameter)

	assert.Assert(t, !result, "should not reference parameter")
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
		depErr, ok := err.(errors.CircularDependencyConfigSortError)
		assert.Assert(t, ok, "expected errors of type CircularDependencyConfigSortError")
		if depErr.Location.Match(config1Coordinates) {
			assert.Assert(t, depErr.DependsOn[0] == config2Coordinates)
		}
		if depErr.Location.Match(config2Coordinates) {
			assert.Assert(t, depErr.DependsOn[0] == config3Coordinates)
		}
		if depErr.Location.Match(config3Coordinates) {
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

// TODO move up!
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
		sortErrs []topologysort.TopologySortError
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
				[]topologysort.TopologySortError{},
				testConfigs,
			},
			[]error{},
		},
		{
			"parses simple errors into list",
			args{
				[]topologysort.TopologySortError{
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
				errors.CircularDependencyConfigSortError{
					Location:  testConfigs[0].Coordinate,
					DependsOn: []coordinate.Coordinate{testConfigs[2].Coordinate},
				},
				errors.CircularDependencyConfigSortError{
					Location:  testConfigs[1].Coordinate,
					DependsOn: []coordinate.Coordinate{testConfigs[0].Coordinate},
				},
				errors.CircularDependencyConfigSortError{
					Location:  testConfigs[2].Coordinate,
					DependsOn: []coordinate.Coordinate{testConfigs[0].Coordinate},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseConfigSortErrors(tt.args.sortErrs, tt.args.configs)
			assert.DeepEqual(t, got, tt.want, cmpopts.SortSlices(func(a, b error) bool {
				depErrA := a.(errors.CircularDependencyConfigSortError)
				depErrB := b.(errors.CircularDependencyConfigSortError)
				return depErrA.Location.String() < depErrB.Location.String()
			}))
		})
	}
}
