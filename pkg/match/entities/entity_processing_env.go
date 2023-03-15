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
	rawEntityListPtr          *RawEntityList
	configType                config.Type
	currentremainingEntities  *[]int
	remainingEntities         []int
	remainingEntitiesSeeded   []int
	remainingEntitiesUnSeeded []int
}

func (e *EntityProcessingEnv) genSeededEntities(resultList []CompareResult, getId func(CompareResult) int) {
	e.remainingEntitiesSeeded = []int{}

	if len(resultList) == 0 {
		return
	}

	e.remainingEntitiesSeeded = make([]int, 1, len(resultList))
	e.remainingEntitiesSeeded[0] = getId(resultList[0])

	for _, result := range resultList[1:] {
		entityId := getId(result)
		if entityId == e.remainingEntitiesSeeded[len(e.remainingEntitiesSeeded)-1] {
			// pass
		} else {
			e.remainingEntitiesSeeded = append(e.remainingEntitiesSeeded, entityId)
		}
	}

}

func (e *EntityProcessingEnv) genUnSeededEntities(resultList []CompareResult, getId func(CompareResult) int, remainingEntities *[]int) {
	e.remainingEntitiesUnSeeded = []int{}

	if len(*remainingEntities) == 0 {
		return
	}

	e.remainingEntitiesUnSeeded = make([]int, 0, len(*remainingEntities))

	resultIdx := 0
	entityId := 0

	for _, remainingEntity := range *remainingEntities {
		for ; resultIdx < len(resultList); resultIdx++ {
			entityId = getId(resultList[resultIdx])
			if entityId >= remainingEntity {
				break
			}
		}
		if remainingEntity != entityId || resultIdx >= len(resultList) {
			e.remainingEntitiesUnSeeded = append(e.remainingEntitiesUnSeeded, remainingEntity)
		}
	}

}

func (e *EntityProcessingEnv) reduceRemainingEntityList(singleToSingleMatch []CompareResult, getId func(CompareResult) int) {
	entityIdList := make([]int, len(singleToSingleMatch))

	i := 0
	for _, result := range singleToSingleMatch {

		entityIdList[i] = getId(result)
		i++
	}

	e.trimremainingEntities(entityIdList)
}

func (e *EntityProcessingEnv) trimremainingEntities(idsToDrop []int) {

	sort.Slice(idsToDrop, func(i, j int) bool { return idsToDrop[i] < idsToDrop[j] })

	nbRemaining := len(e.remainingEntities) - len(idsToDrop)
	newremainingEntities := make([]int, nbRemaining)

	dropI := 0
	oldI := 0
	newI := 0

	for oldI < len(e.remainingEntities) {

		if dropI < len(idsToDrop) {
			if idsToDrop[dropI] < e.remainingEntities[oldI] {
				log.Error("Dropping a non-remaining ID?? dropI: %d idsToDrop[dropI]: %d", dropI, idsToDrop[dropI])
				dropI++
				continue

			} else if idsToDrop[dropI] == e.remainingEntities[oldI] {
				dropI++
				oldI++
				continue
			}
		}

		newremainingEntities[newI] = e.remainingEntities[oldI]
		newI++

		oldI++

	}

	if newI != nbRemaining {

		log.Error("Did not trim properly?? nbRemaining: %d newI: %d", nbRemaining, newI)
		log.Error("Did not trim properly?? len(e.remainingEntities): %d len(idsToDrop): %d", len(e.remainingEntities), len(idsToDrop))

	}

	e.remainingEntities = newremainingEntities
}
