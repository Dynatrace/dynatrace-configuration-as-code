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

package errors

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"strings"
)

type CircularDependencyParameterSortError struct {
	Location           coordinate.Coordinate          `json:"location"`
	EnvironmentDetails errors.EnvironmentDetails      `json:"environmentDetails"`
	ParameterName      string                         `json:"parameterName"`
	DependsOn          []parameter.ParameterReference `json:"dependsOn"`
}

func (e CircularDependencyParameterSortError) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e CircularDependencyParameterSortError) LocationDetails() errors.EnvironmentDetails {
	return e.EnvironmentDetails
}

var (
	_ errors.DetailedConfigError = (*CircularDependencyParameterSortError)(nil)
)

func (e CircularDependencyParameterSortError) Error() string {
	return fmt.Sprintf("%s: circular dependency detected. check parameter dependencies: %s",
		e.ParameterName, joinParameterReferencesToString(e.DependsOn))
}

func joinParameterReferencesToString(refs []parameter.ParameterReference) string {
	switch len(refs) {
	case 0:
		return ""
	case 1:
		return refs[0].String()
	}

	result := strings.Builder{}

	for _, ref := range refs {
		result.WriteString(ref.String())
		result.WriteString(", ")
	}

	return result.String()
}

type CircualDependencyProjectSortError struct {
	Environment string `json:"environment"`
	Project     string `json:"project"`
	// slice of project ids
	DependsOnProjects []string `json:"dependsOnProjects"`
}

func (e CircualDependencyProjectSortError) Error() string {
	return fmt.Sprintf("%s:%s: circular dependency detected.\n check project dependencies: %s",
		e.Environment, e.Project, strings.Join(e.DependsOnProjects, ", "))
}

type CircularDependencyConfigSortError struct {
	Location    coordinate.Coordinate   `json:"location"`
	Environment string                  `json:"environment"`
	DependsOn   []coordinate.Coordinate `json:"dependsOn"`
}

func (e CircularDependencyConfigSortError) Error() string {
	return fmt.Sprintf("%s:%s: is part of circular dependency.\n depends on: %s",
		e.Environment, e.Location, joinCoordinatesToString(e.DependsOn))
}

func joinCoordinatesToString(coordinates []coordinate.Coordinate) string {
	switch len(coordinates) {
	case 0:
		return ""
	case 1:
		return coordinates[0].String()
	}

	result := strings.Builder{}

	for _, c := range coordinates {
		result.WriteString(c.String())
		result.WriteString(", ")
	}

	return result.String()
}

var (
	_ error = (*CircularDependencyConfigSortError)(nil)
	_ error = (*CircualDependencyProjectSortError)(nil)
	_ error = (*CircularDependencyParameterSortError)(nil)
)
