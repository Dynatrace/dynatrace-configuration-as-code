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

package entities

import (
	"sort"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"
)

type EntityProcessingEnv struct {
	RawEntityListPtr         *RawEntityList
	Type                     config.Type
	CurrentRemainingEntities *[]int
	RemainingEntities        []int
	RemainingEntitiesSeeded  []int
}

func (e *EntityProcessingEnv) GenSeededEntities(resultList []CompareResult, getId func(CompareResult) int) {
	e.RemainingEntitiesSeeded = []int{}

	if len(resultList) == 0 {
		return
	}

	e.RemainingEntitiesSeeded = make([]int, 1, len(resultList))
	e.RemainingEntitiesSeeded[0] = getId(resultList[0])

	for _, result := range resultList[1:] {
		entityId := getId(result)
		if entityId == e.RemainingEntitiesSeeded[len(e.RemainingEntitiesSeeded)-1] {
			// pass
		} else {
			e.RemainingEntitiesSeeded = append(e.RemainingEntitiesSeeded, entityId)
		}
	}

}

func (e *EntityProcessingEnv) ReduceRemainingEntityList(singleToSingleMatch []CompareResult, getId func(CompareResult) int) {
	entityIdList := make([]int, len(singleToSingleMatch))

	i := 0
	for _, result := range singleToSingleMatch {

		entityIdList[i] = getId(result)
		i++
	}

	e.trimRemainingEntities(entityIdList)
}

func (e *EntityProcessingEnv) trimRemainingEntities(idsToDrop []int) {

	sort.Slice(idsToDrop, func(i, j int) bool { return idsToDrop[i] < idsToDrop[j] })

	nbRemaining := len(e.RemainingEntities) - len(idsToDrop)
	newRemainingEntities := make([]int, nbRemaining)

	dropI := 0
	oldI := 0
	newI := 0

	for oldI < len(e.RemainingEntities) {

		if dropI < len(idsToDrop) {
			if idsToDrop[dropI] < e.RemainingEntities[oldI] {
				log.Error("Dropping a non-remaining ID?? dropI: %d idsToDrop[dropI]: %d", dropI, idsToDrop[dropI])
				dropI++
				continue

			} else if idsToDrop[dropI] == e.RemainingEntities[oldI] {
				dropI++
				oldI++
				continue
			}
		}

		newRemainingEntities[newI] = e.RemainingEntities[oldI]
		newI++

		oldI++

	}

	if newI != nbRemaining {

		log.Error("Did not trim properly?? nbRemaining: %d newI: %d", nbRemaining, newI)
		log.Error("Did not trim properly?? len(e.RemainingEntities): %d len(idsToDrop): %d", len(e.RemainingEntities), len(idsToDrop))

	}

	e.RemainingEntities = newRemainingEntities
}
