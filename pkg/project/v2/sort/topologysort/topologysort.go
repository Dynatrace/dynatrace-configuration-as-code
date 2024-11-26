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
	"slices"
	s "sort"
	"strings"
	"sync"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/topologysort"
	errors2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort/errors"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type projectsPerEnvironment map[string][]project.Project

func SortProjects(projects []project.Project, environments []string) (map[string][]config.Config, []error) {
	sortedProjectsPerEnvironment, errs := sortProjects(projects, environments)
	if len(errs) > 0 {
		return nil, errs
	}

	result := make(map[string][]config.Config)

	for env, sortedProject := range sortedProjectsPerEnvironment {
		sortedConfigResult := make([]config.Config, 0)

		for _, p := range sortedProject {
			configs := p.Configs[env]
			sortedConfigs, cfgSortErrs := sortConfigs(getConfigs(configs))

			errs = append(errs, cfgSortErrs...)

			sortedConfigResult = append(sortedConfigResult, sortedConfigs...)
		}

		result[env] = sortedConfigResult
	}

	if errs != nil {
		return nil, errs
	}

	return result, nil
}

func getConfigs(m map[string][]config.Config) []config.Config {
	result := make([]config.Config, 0)

	for _, v := range m {
		result = append(result, v...)
	}

	s.SliceStable(result, func(i, j int) bool {
		return strings.Compare(result[i].Coordinate.ConfigId, result[j].Coordinate.ConfigId) < 0
	})

	return result
}

func sortConfigs(configs []config.Config) ([]config.Config, []error) {
	matrix, inDegrees := configsToSortData(configs)

	sorted, sortErrs := topologysort.TopologySort(matrix, inDegrees)

	if len(sortErrs) > 0 {
		return nil, parseConfigSortErrors(sortErrs, configs)
	}

	result := make([]config.Config, 0, len(configs))

	for i := len(sorted) - 1; i >= 0; i-- {
		result = append(result, configs[sorted[i]])
	}

	return result, nil
}

// referencesLookup is a double lookup map to check dependencies between configs using their coordinates
type referencesLookup map[coordinate.Coordinate]map[coordinate.Coordinate]struct{}

func configsToSortData(configs []config.Config) ([][]bool, []int) {
	numConfigs := len(configs)
	matrix := make([][]bool, numConfigs)
	inDegrees := make([]int, len(configs))

	// build lookup tables for References between configs.
	// with this we need to calculate the references only once and can use the map-lookup with takes O(1)
	refLookup := make(referencesLookup, len(configs))
	for i := range configs {
		refs := configs[i].References()
		c := make(map[coordinate.Coordinate]struct{}, len(refs))
		for ir := range refs {
			c[refs[ir]] = struct{}{}
		}

		refLookup[configs[i].Coordinate] = c
	}

	wg := sync.WaitGroup{}
	wg.Add(len(configs))

	for i := range configs {
		go func(i int) {
			row := make([]bool, numConfigs)

			for j := range configs {
				// don't check the same config
				if i == j {
					continue
				}

				// we do not care about skipped configs
				if configs[j].Skip {
					continue
				}

				// check if we have a reference between the configs.
				// We check the inner config-loop against the outer one
				if _, f := refLookup[configs[j].Coordinate][configs[i].Coordinate]; f {
					logDependency("Configuration", configs[j].Coordinate.String(), configs[i].Coordinate.String())
					row[j] = true
					inDegrees[i]++
				}
			}

			matrix[i] = row
			wg.Done()
		}(i)
	}

	wg.Wait()

	return matrix, inDegrees
}

// parseConfigSortErrors turns [topologysort.TopologySortError] into [CircularDependencyConfigSortError]
// for each config still has an edge to another after sorting an error will be created by aggregating the sort errors
func parseConfigSortErrors(sortErrs []topologysort.TopologySortError, configs []config.Config) []error {
	depErrs := make(map[coordinate.Coordinate]errors2.CircularDependencyConfigSortError)

	for _, sortErr := range sortErrs {
		conf := configs[sortErr.OnId]

		for _, index := range sortErr.UnresolvedIncomingEdgesFrom {
			dependingConfig := configs[index]

			if err, exists := depErrs[dependingConfig.Coordinate]; exists {
				err.DependsOn = append(err.DependsOn, conf.Coordinate)
				depErrs[dependingConfig.Coordinate] = err
			} else {
				depErrs[dependingConfig.Coordinate] = errors2.CircularDependencyConfigSortError{
					Location:    dependingConfig.Coordinate,
					Environment: dependingConfig.Environment,
					DependsOn:   []coordinate.Coordinate{conf.Coordinate},
				}
			}
		}
	}

	errs := make([]error, len(depErrs))
	i := 0
	for _, depErr := range depErrs {
		errs[i] = depErr
		i++
	}

	return errs
}

func sortProjects(projects []project.Project, environments []string) (projectsPerEnvironment, []error) {
	var errs []error

	resultByEnvironment := make(projectsPerEnvironment)

	for _, env := range environments {
		matrix, inDegrees := projectsToSortData(projects, env)

		sorted, sortErrs := topologysort.TopologySort(matrix, inDegrees)

		if len(sortErrs) > 0 {
			for _, sortErr := range sortErrs {
				p := projects[sortErr.OnId]

				errs = append(errs, &errors2.CircualDependencyProjectSortError{
					Environment:       env,
					Project:           p.Id,
					DependsOnProjects: toDependenciesPerEnvironment(p)[env],
				})
			}
		}

		result := make([]project.Project, 0, len(sorted))

		for i := len(sorted) - 1; i >= 0; i-- {
			result = append(result, projects[sorted[i]])
		}

		resultByEnvironment[env] = result
	}

	if errs != nil {
		return nil, errs
	}

	return resultByEnvironment, nil
}

type projectID = string

// dependenciesPerEnvironment is a map of EnvironmentName to project IDs
type dependenciesPerEnvironment map[project.EnvironmentName][]projectID

func toDependenciesPerEnvironment(p project.Project) dependenciesPerEnvironment {
	result := make(dependenciesPerEnvironment)

	for _, c := range p.ConfigList() {
		// ignore skipped configs
		if c.Skip {
			continue
		}

		for _, ref := range c.References() {
			// ignore project on same project
			if p.Id == ref.Project {
				continue
			}

			if !slices.Contains(result[c.Environment], ref.Project) {
				result[c.Environment] = append(result[c.Environment], ref.Project)
			}
		}
	}

	return result
}

func projectsToSortData(projects []project.Project, environment string) ([][]bool, []int) {
	numProjects := len(projects)
	matrix := make([][]bool, numProjects)
	inDegrees := make([]int, len(projects))

	for i, prj := range projects {
		matrix[i] = make([]bool, numProjects)

		for j, p := range projects {
			if i == j {
				continue
			}

			if hasDependencyOn(p, environment, prj) {
				logDependency("Project", p.Id, prj.Id)
				matrix[i][j] = true
				inDegrees[i]++
			}
		}
	}

	return matrix, inDegrees
}

func logDependency(prefix string, depending string, dependedOn string) {
	log.Debug("%s: %s has dependency on %s", prefix, depending, dependedOn)
}

// hasDependencyOn returns whether the project it is called on, has a dependency on the given project, for the given environment
func hasDependencyOn(orig project.Project, environment string, project project.Project) bool {
	for c := range slices.Values(orig.ConfigList()) {
		if c.Environment == environment {
			for r := range slices.Values(c.References()) {
				if r.Project == project.Id {
					return true
				}
			}
		}
	}
	return false
}
