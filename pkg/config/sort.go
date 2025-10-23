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

package config

import (
	"sort"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/topologysort"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

func getSortedParameters(c *Config) ([]parameter.NamedParameter, []error) {
	parametersWithName := make([]parameter.NamedParameter, 0, len(c.Parameters))

	for name, param := range c.Parameters {
		parametersWithName = append(parametersWithName, parameter.NamedParameter{
			Name:      name,
			Parameter: param,
		})
	}

	sort.SliceStable(parametersWithName, func(i, j int) bool {
		return strings.Compare(parametersWithName[i].Name, parametersWithName[j].Name) < 0
	})

	matrix, inDegrees := parametersToSortData(c.Coordinate, parametersWithName)
	sorted, sortErrs := topologysort.TopologySort(matrix, inDegrees)

	if len(sortErrs) > 0 {
		errs := make([]error, len(sortErrs))
		for i, sortErr := range sortErrs {
			param := parametersWithName[sortErr.OnId]

			errs[i] = &CircularDependencyParameterSortError{
				Location: c.Coordinate,
				EnvironmentDetails: errors.EnvironmentDetails{
					Group:       c.Group,
					Environment: c.Environment,
				},
				ParameterName: param.Name,
				DependsOn:     param.Parameter.GetReferences(),
			}
		}
		return nil, errs

	}

	result := make([]parameter.NamedParameter, 0, len(parametersWithName))

	for i := len(sorted) - 1; i >= 0; i-- {
		result = append(result, parametersWithName[sorted[i]])
	}

	return result, nil
}

func parametersToSortData(conf coordinate.Coordinate, parameters []parameter.NamedParameter) ([][]bool, []int) {
	numParameters := len(parameters)
	matrix := make([][]bool, numParameters)
	inDegrees := make([]int, numParameters)

	for i, param := range parameters {
		matrix[i] = make([]bool, numParameters)

		for j, p := range parameters {
			if i == j {
				continue
			}

			if parameterReference(p, conf, param) {
				logDependency("Config Parameter", p.Name, param.Name)
				matrix[i][j] = true
				inDegrees[i]++
			}
		}
	}

	return matrix, inDegrees
}

func parameterReference(sourceParam parameter.NamedParameter, config coordinate.Coordinate, targetParam parameter.NamedParameter) bool {
	for _, ref := range sourceParam.Parameter.GetReferences() {
		if ref.Config != config {
			continue
		}

		var match bool
		if vp, ok := targetParam.Parameter.(*value.ValueParameter); ok {
			match = searchValueParameterForKey(ref.Property, targetParam.Name, vp)
		} else {
			match = ref.Property == targetParam.Name
		}

		if match {
			return true // return if any ref matches, else continue loop and check others
		}
	}
	return false
}

func searchValueParameterForKey(key string, paramName string, param *value.ValueParameter) bool {
	if vpm, ok := param.Value.(map[string]any); ok {
		if first, rest, found := strings.Cut(key, "."); found && first == paramName {
			_, isRef := entities.ResolvePropValue(rest, vpm)
			return isRef
		}
	}
	return key == paramName
}

func logDependency(prefix string, depending string, dependedOn string) {
	log.Debug("%s: %s has dependency on %s", prefix, depending, dependedOn)
}
