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

package sort

import "fmt"

// TopologySortError is an error returned for any unresolved dependency after sorting
// The error marks the ID of the node left with unresolved incoming edges after sorting,
// as well as the IDs of the nodes still pointing to it.
type TopologySortError struct {
	OnId                        int
	UnresolvedIncomingEdgesFrom []int
}

func (e TopologySortError) Error() string {
	return fmt.Sprintf("circular Dependency in Topology Sort, could not resolve dependencies still pointing to index %d from %v", e.OnId, e.UnresolvedIncomingEdgesFrom)
}

var (
	_ error = (*TopologySortError)(nil)
)

// TopologySort implements [Kahn's algorithm for topological sorting]
// As an input to the algorithm it takes a directed graph represented as a dependency matrix of incoming edges.
// To simplify checking full resolution/sort failure a precomputed slice of inDegrees per node must be provided as well.
// Indices in the incomingEdges dependency matrix and inDegrees slice need to match - e.g. if incomingEdges[i] marks two
// edges as 'true', inDegrees[i] must equal 2.
//
// Successful sorting - which is guaranteed for acyclic directed graphs - will see all edges resolved,
// leaving each node with an inDegree of zero.
//
// In case of cycles in the graph some nodes will be left with unresolved edges (inDegree > 0) - in this case a list of
// [TopologySortError] will be returned, marking each node id with unresolved incoming edges and which node ids are still
// pointing to it.
//
// [Kahn's algorithm for topological sorting]: https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
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
			errs = append(errs, TopologySortError{
				OnId:                        i,
				UnresolvedIncomingEdgesFrom: getSourceOfIncomingEdges(incomingEdges[i]),
			})
		}
	}

	return topoSorted, errs
}

// getAllLeaves returns all leaves in the graph to sort - by returning all node ids with no incoming edges
func getAllLeaves(inDegrees []int) []int {
	var nodes []int
	for i := range inDegrees {
		if inDegrees[i] == 0 {
			nodes = append(nodes, i)
		}
	}
	return nodes
}

// getSourceOfIncomingEdges transforms a row of the dependency matrix into the ids of all nodes that are the source of an incoming edge
func getSourceOfIncomingEdges(incomingEdges []bool) []int {
	var sources []int
	for i, hasEdge := range incomingEdges {
		if hasEdge {
			sources = append(sources, i)
		}
	}
	return sources
}
