/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package v1

import (
	"fmt"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v1"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"
)

func sortConfigurations(configs []config.Config) (sorted []config.Config, err error) {
	sorted = []config.Config{}
	incomingDeps, inDegrees := calculateIncomingConfigDependencies(configs)
	reverse, err, errorOn := topologySort(incomingDeps, inDegrees)
	if err != nil {
		log.Debug(err.Error())
		return sorted, fmt.Errorf("failed to sort configs, circular dependency on config %s detected, please check dependencies", configs[errorOn].GetFullQualifiedId())
	}

	for i := len(reverse) - 1; i >= 0; i-- {
		sorted = append(sorted, configs[reverse[i]])
		log.Debug("\t\t%s", configs[reverse[i]].GetFullQualifiedId())
	}
	return sorted, nil
}

func calculateIncomingConfigDependencies(configs []config.Config) (adjacencyMatrix [][]bool, inDegrees []int) {
	adjacencyMatrix = make([][]bool, len(configs))
	inDegrees = make([]int, len(configs))

	for i := range configs {
		c1 := configs[i]
		adjacencyMatrix[i] = make([]bool, len(configs))
		for j := range configs {
			if i != j {
				c2 := configs[j]
				if c2.HasDependencyOn(c1) {
					logDependency(c2.GetFullQualifiedId(), c1.GetFullQualifiedId())
					adjacencyMatrix[i][j] = true
					inDegrees[i]++
				}
			}
		}
	}

	return adjacencyMatrix, inDegrees
}

// https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
func topologySort(incomingEdges [][]bool, inDegrees []int) (topoSorted []int, err error, errorOnId int) {

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
	for i := range inDegrees {
		if inDegrees[i] != 0 {
			return topoSorted, fmt.Errorf("circular Dependency in Topology Sort, could not resolve dependencies still pointing to index %d", i), i
		}
	}

	return topoSorted, nil, -1
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

func logDependency(depending string, dependedOn string) {
	log.Debug("%s has dependency on %s", depending, dependedOn)
}
