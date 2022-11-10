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

package sort

import "fmt"

type TopologySortError struct {
	OnId int
}

func (e TopologySortError) Error() string {
	return fmt.Sprintf("circular Dependency in Topology Sort, could not resolve dependencies still pointing to index %d", e.OnId)
}

var (
	_ error = (*TopologySortError)(nil)
)

// https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
func TopologySort(incomingEdges [][]bool, inDegrees []int) (topoSorted []int, errs []TopologySortError) {

	nodes := getAllLeaves(inDegrees)

	topoSorted = []int{}
	for len(nodes) > 0 {
		cur := nodes[0]
		nodes = nodes[1:]
		topoSorted = append(topoSorted, cur)
		for i := range inDegrees {
			if incomingEdges[i][cur] {
				incomingEdges[i][cur] = false
				inDegrees[i]--
				if inDegrees[i] <= 0 {
					nodes = append(nodes, i)
				}
			}
		}
	}

	errs = []TopologySortError{}
	for i := range inDegrees {
		if inDegrees[i] != 0 {
			errs = append(errs, TopologySortError{OnId: i})
		}
	}

	return topoSorted, errs
}

func getAllLeaves(inDegrees []int) []int {
	var nodes []int
	for i := range inDegrees {
		if inDegrees[i] == 0 {
			nodes = append(nodes, i)
		}
	}
	return nodes
}
