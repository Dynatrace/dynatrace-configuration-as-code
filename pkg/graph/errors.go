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

package graph

import (
	"fmt"
	"strings"

	"gonum.org/v1/gonum/graph"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

// SortingErrors is a slice of errors that implements the error interface.
// It may contain general errors as well as CyclicDependencyError.
type SortingErrors []error

func (errs SortingErrors) Error() string {
	b := strings.Builder{}
	for _, e := range errs {
		_, _ = b.WriteString(fmt.Sprintf("%s\n", e.Error()))
	}
	return b.String()
}

// CyclicDependencyError is returned if sorting a graph failed due to cyclic dependencies between configurations.
type CyclicDependencyError struct {
	//A slice of all dependency cycles between configurations as slices of DependencyLocation. Each cycle slice is returned in order of dependencies.
	ConfigsInDependencyCycle [][]DependencyLocation `json:"configsInDependencyCycle"`
}

// DependencyLocation is a short from location pointing to the coordinate and (if available) file of the configuration.
type DependencyLocation struct {
	// The coordinate.Coordinate of the configuration that is part of a Dependency Cycle
	Coordinate coordinate.Coordinate `json:"coordinate"`
	// The filepath this configuration was loaded from. May be empty.
	Filepath string `json:"filepath,omitempty"`
}

func (e CyclicDependencyError) Error() string {
	b := strings.Builder{}
	_, _ = b.WriteString(fmt.Sprintf("There are %d dependency cycles between the configurations.\n", len(e.ConfigsInDependencyCycle)))
	for _, cycle := range e.ConfigsInDependencyCycle {
		_, _ = b.WriteString("Please check the following configuration's references and break the cycle:\n")

		for _, c := range cycle {
			_, _ = b.WriteString(fmt.Sprintf("%q", c.Coordinate))
			if c.Filepath != "" {
				_, _ = b.WriteString(fmt.Sprintf(" (%s)", c.Filepath))
			}
			_, _ = b.WriteString(" -> ")
		}
		_, _ = b.WriteString(fmt.Sprintf("%q", cycle[0].Coordinate))

	}

	return b.String()
}

func newCyclicDependencyError(cycles [][]graph.Node) CyclicDependencyError {
	cfgCycles := make([][]DependencyLocation, len(cycles))
	for i, cycle := range cycles {
		cfgCycles[i] = make([]DependencyLocation, len(cycle))
		for j, node := range cycle {
			coord := node.(ConfigNode).Config.Coordinate
			filepath := ""
			if t, ok := node.(ConfigNode).Config.Template.(*template.FileBasedTemplate); ok {
				filepath = t.FilePath()
			}

			cfgCycles[i][j] = DependencyLocation{
				Coordinate: coord,
				Filepath:   filepath,
			}
		}
	}
	return CyclicDependencyError{
		ConfigsInDependencyCycle: cfgCycles,
	}
}
