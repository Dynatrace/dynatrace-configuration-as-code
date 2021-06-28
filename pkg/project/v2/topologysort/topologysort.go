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

package topologysort

import (
	"fmt"
	"strings"

	s "sort"

	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/sort"
)

type ProjectsPerEnvironment map[string][]project.Project

type CircularDependencyParameterSortError struct {
	Config             coordinate.Coordinate
	EnvironmentDetails errors.EnvironmentDetails
	Parameter          string
	DependsOn          []parameter.ParameterReference
}

func (e *CircularDependencyParameterSortError) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e *CircularDependencyParameterSortError) LocationDetails() errors.EnvironmentDetails {
	return e.EnvironmentDetails
}

var (
	_ errors.DetailedConfigError = (*CircularDependencyParameterSortError)(nil)
)

func (e *CircularDependencyParameterSortError) Error() string {
	return fmt.Sprintf("%s: circular dependency detected. check parameter dependencies: %s",
		e.Parameter, joinParameterReferencesToString(e.DependsOn))
}

func joinParameterReferencesToString(refs []parameter.ParameterReference) string {
	switch len(refs) {
	case 0:
		return ""
	case 1:
		return refs[0].ToString()
	}

	result := strings.Builder{}

	for _, ref := range refs {
		result.WriteString(ref.ToString())
		result.WriteString(", ")
	}

	return result.String()
}

type CircualDependencyProjectSortError struct {
	Environment string
	Project     string
	// slice of project ids
	DependsOn []string
}

func (e *CircualDependencyProjectSortError) Error() string {
	return fmt.Sprintf("%s:%s: circular dependency detected. check project dependencies: %s",
		e.Environment, e.Project, strings.Join(e.DependsOn, ", "))
}

type CircularDependencyConfigSortError struct {
	Config      coordinate.Coordinate
	Environment string
	DependsOn   []coordinate.Coordinate
}

func (e *CircularDependencyConfigSortError) Error() string {
	return fmt.Sprintf("%s:%s: circular dependency detected. check configs dependencies: %s",
		e.Environment, e.Config.ToString(), joinCoordinatesToString(e.DependsOn))
}

func joinCoordinatesToString(coordinates []coordinate.Coordinate) string {
	switch len(coordinates) {
	case 0:
		return ""
	case 1:
		return coordinates[0].ToString()
	}

	result := strings.Builder{}

	for _, coordinate := range coordinates {
		result.WriteString(coordinate.ToString())
		result.WriteString(", ")
	}

	return result.String()
}

var (
	_ error = (*CircularDependencyConfigSortError)(nil)
	_ error = (*CircualDependencyProjectSortError)(nil)
	_ error = (*CircularDependencyParameterSortError)(nil)
)

type ParameterWithName struct {
	Name      string
	Parameter parameter.Parameter
}

func (p *ParameterWithName) IsReferencing(config coordinate.Coordinate, param ParameterWithName) bool {
	for _, ref := range p.Parameter.GetReferences() {
		if ref.Config == config && ref.Property == param.Name {
			return true
		}
	}

	return false
}

func SortParameters(group string, environment string, conf coordinate.Coordinate, parameters config.Parameters) ([]ParameterWithName, error) {
	parametersWithName := make([]ParameterWithName, 0, len(parameters))

	for name, param := range parameters {
		parametersWithName = append(parametersWithName, ParameterWithName{
			Name:      name,
			Parameter: param,
		})
	}

	s.SliceStable(parametersWithName, func(i, j int) bool {
		return strings.Compare(parametersWithName[i].Name, parametersWithName[j].Name) < 0
	})

	matrix, inDegrees := parametersToSortData(conf, parametersWithName)
	sorted, err, errorOn := sort.TopologySort(matrix, inDegrees)

	if err != nil {
		param := parametersWithName[errorOn]

		return nil, &CircularDependencyParameterSortError{
			Config: conf,
			EnvironmentDetails: errors.EnvironmentDetails{
				Group:       group,
				Environment: environment,
			},
			Parameter: param.Name,
			DependsOn: param.Parameter.GetReferences(),
		}
	}

	result := make([]ParameterWithName, 0, len(parametersWithName))

	for i := len(sorted) - 1; i >= 0; i-- {
		result = append(result, parametersWithName[sorted[i]])
	}

	return result, nil
}

func parametersToSortData(conf coordinate.Coordinate, parameters []ParameterWithName) ([][]bool, []int) {
	numParameters := len(parameters)
	matrix := make([][]bool, numParameters)
	inDegrees := make([]int, numParameters)

	for i, param := range parameters {
		matrix[i] = make([]bool, numParameters)

		for j, p := range parameters {
			if i == j {
				continue
			}

			if p.IsReferencing(conf, param) {
				util.Log.Debug("\t\t%s has dep on %s", p.Name, param.Name)
				matrix[i][j] = true
				inDegrees[i]++
			}
		}
	}

	return matrix, inDegrees
}

func GetSortedConfigsForEnvironments(projects []project.Project, environments []string) (map[string][]config.Config, []error) {
	sortedProjectsPerEnvironment, errs := sortProjects(projects, environments)

	if len(errs) > 0 {
		return nil, errs
	}

	result := make(map[string][]config.Config)
	var errors []error

	for env, sortedProject := range sortedProjectsPerEnvironment {
		sortedConfigResult := make([]config.Config, 0)

		for _, project := range sortedProject {
			configs := project.Configs[env]
			sortedConfigs, err := sortConfigs(getConfigs(configs))

			if err != nil {
				errors = append(errors, err)
			}

			sortedConfigResult = append(sortedConfigResult, sortedConfigs...)
		}

		result[env] = sortedConfigResult
	}

	if errors != nil {
		return nil, errors
	}

	return result, nil
}

func getConfigs(m map[string][]config.Config) []config.Config {
	result := make([]config.Config, 0)

	for _, v := range m {
		result = append(result, v...)
	}

	s.SliceStable(result, func(i, j int) bool {
		return strings.Compare(result[i].Coordinate.Config, result[j].Coordinate.Config) < 0
	})

	return result
}

func sortConfigs(configs []config.Config) ([]config.Config, error) {
	matrix, inDegrees := configsToSortData(configs)

	sorted, err, errorOn := sort.TopologySort(matrix, inDegrees)

	if err != nil {
		conf := configs[errorOn]

		return nil, &CircularDependencyConfigSortError{
			Config:      conf.Coordinate,
			Environment: conf.Environment,
			DependsOn:   conf.References,
		}
	}

	result := make([]config.Config, 0, len(configs))

	for i := len(sorted) - 1; i >= 0; i-- {
		result = append(result, configs[sorted[i]])
	}

	return result, nil
}

func configsToSortData(configs []config.Config) ([][]bool, []int) {
	numConfigs := len(configs)
	matrix := make([][]bool, numConfigs)
	inDegrees := make([]int, len(configs))

	for i, config := range configs {
		matrix[i] = make([]bool, numConfigs)

		for j, c := range configs {
			if i == j {
				continue
			}

			// we do not care about skipped configs
			if c.Skip {
				continue
			}

			if c.HasDependencyOn(config) {
				util.Log.Debug("\t\t%s has dep on %s", c.Coordinate.ToString(), config.Coordinate.ToString())
				matrix[i][j] = true
				inDegrees[i]++
			}
		}
	}

	return matrix, inDegrees
}

func sortProjects(projects []project.Project, environments []string) (ProjectsPerEnvironment, []error) {
	var errors []error

	resultByEnvironment := make(ProjectsPerEnvironment)

	for _, env := range environments {
		matrix, inDegrees := projectsToSortData(projects, env)

		sorted, err, errorOn := sort.TopologySort(matrix, inDegrees)

		if err != nil {
			project := projects[errorOn]

			errors = append(errors, &CircualDependencyProjectSortError{
				Environment: env,
				Project:     project.Id,
				DependsOn:   project.Dependencies[env],
			})
		}

		result := make([]project.Project, 0, len(sorted))

		for i := len(sorted) - 1; i >= 0; i-- {
			result = append(result, projects[sorted[i]])
		}

		resultByEnvironment[env] = result
	}

	if errors != nil {
		return nil, errors
	}

	return resultByEnvironment, nil
}

func projectsToSortData(projects []project.Project, environment string) ([][]bool, []int) {
	numProjects := len(projects)
	matrix := make([][]bool, numProjects)
	inDegrees := make([]int, len(projects))

	for i, project := range projects {
		matrix[i] = make([]bool, numProjects)

		for j, p := range projects {
			if i == j {
				continue
			}

			if p.HasDependencyOn(environment, project) {
				util.Log.Debug("\t\t%s has dep on %s", p.Id, project.Id)
				matrix[i][j] = true
				inDegrees[i]++
			}
		}
	}

	return matrix, inDegrees
}
