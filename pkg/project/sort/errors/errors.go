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
	"strings"
)

type CircualDependencyProjectSortError struct {
	// Environment is the name of the environment the project sorting failed for
	Environment string `json:"environment"`
	// Project is the name of the project that is part of a cycle
	Project string `json:"project"`
	// DependsOnProjects is a list of project IDs the Project depends on
	DependsOnProjects []string `json:"dependsOnProjects"`
}

func (e CircualDependencyProjectSortError) Error() string {
	return fmt.Sprintf("%s:%s: circular dependency detected.\n check project dependencies: %s",
		e.Environment, e.Project, strings.Join(e.DependsOnProjects, ", "))
}

type CircularDependencyConfigSortError struct {
	// Location (coordinate) of the config.Config which part of a cycle
	Location coordinate.Coordinate `json:"location"`
	// Environment is the name of the environment the project sorting failed for
	Environment string `json:"environment"`
	// DependsOn is a list of coordinates the configuration (Location) depends on
	DependsOn []coordinate.Coordinate `json:"dependsOn"`
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
)
