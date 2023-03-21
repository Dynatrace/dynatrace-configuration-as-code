// @license
// Copyright 2023 Dynatrace LLC
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

package match

import (
	"sort"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
)

type MatchProcessingEnv struct {
	RawMatchList           RawMatchList
	ConfigType             config.Type
	CurrentremainingMatch  *[]int
	RemainingMatch         []int
	remainingMatchSeeded   []int
	remainingMatchUnSeeded []int
}

func (e *MatchProcessingEnv) genSeededMatch(resultList []CompareResult, getId func(CompareResult) int) {
	e.remainingMatchSeeded = []int{}

	if len(resultList) == 0 {
		return
	}

	e.remainingMatchSeeded = make([]int, 1, len(resultList))
	e.remainingMatchSeeded[0] = getId(resultList[0])

	for _, result := range resultList[1:] {
		id := getId(result)
		if id == e.remainingMatchSeeded[len(e.remainingMatchSeeded)-1] {
			// pass
		} else {
			e.remainingMatchSeeded = append(e.remainingMatchSeeded, id)
		}
	}

}

func (e *MatchProcessingEnv) genUnSeededMatch(resultList []CompareResult, getId func(CompareResult) int, remainingItems *[]int) {
	e.remainingMatchUnSeeded = []int{}

	if len(*remainingItems) == 0 {
		return
	}

	e.remainingMatchUnSeeded = make([]int, 0, len(*remainingItems))

	resultIdx := 0
	id := 0

	for _, remainingItem := range *remainingItems {
		for ; resultIdx < len(resultList); resultIdx++ {
			id = getId(resultList[resultIdx])
			if id >= remainingItem {
				break
			}
		}
		if remainingItem != id || resultIdx >= len(resultList) {
			e.remainingMatchUnSeeded = append(e.remainingMatchUnSeeded, remainingItem)
		}
	}

}

func (e *MatchProcessingEnv) reduceRemainingMatchList(singleToSingleMatch []CompareResult, getId func(CompareResult) int) {
	idList := make([]int, len(singleToSingleMatch))

	i := 0
	for _, result := range singleToSingleMatch {

		idList[i] = getId(result)
		i++
	}

	e.trimremainingItems(idList)
}

func (e *MatchProcessingEnv) trimremainingItems(idsToDrop []int) {

	sort.Slice(idsToDrop, func(i, j int) bool { return idsToDrop[i] < idsToDrop[j] })

	nbRemaining := len((*e).RemainingMatch) - len(idsToDrop)
	newremainingItems := make([]int, nbRemaining)

	dropI := 0
	oldI := 0
	newI := 0

	for oldI < len((*e).RemainingMatch) {

		if dropI < len(idsToDrop) {
			if idsToDrop[dropI] < (*e).RemainingMatch[oldI] {
				log.Error("Dropping a non-remaining ID?? dropI: %d idsToDrop[dropI]: %d", dropI, idsToDrop[dropI])
				dropI++
				continue

			} else if idsToDrop[dropI] == (*e).RemainingMatch[oldI] {
				dropI++
				oldI++
				continue
			}
		}

		newremainingItems[newI] = (*e).RemainingMatch[oldI]
		newI++

		oldI++

	}

	if newI != nbRemaining {

		log.Error("Did not trim properly?? nbRemaining: %d newI: %d", nbRemaining, newI)
		log.Error("Did not trim properly?? len(e.remainingItems): %d len(idsToDrop): %d", len((*e).RemainingMatch), len(idsToDrop))

	}

	(*e).RemainingMatch = newremainingItems
}
