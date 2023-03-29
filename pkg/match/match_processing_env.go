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
	RawMatchList          RawMatchList
	ConfigType            config.EntityType
	CurrentRemainingMatch *[]int
	RemainingMatch        []int
}

func (e *MatchProcessingEnv) genSeededMatch(resultList *[]CompareResult, getId func(CompareResult) int) {
	remainingMatchSeeded := make([]int, 0, len(*resultList))

	if len(*resultList) == 0 {
		e.CurrentRemainingMatch = &remainingMatchSeeded
		return
	}

	remainingMatchSeeded = append(remainingMatchSeeded, getId((*resultList)[0]))

	for _, result := range (*resultList)[1:] {
		id := getId(result)
		if id == remainingMatchSeeded[len(remainingMatchSeeded)-1] {
			// pass
		} else {
			remainingMatchSeeded = append(remainingMatchSeeded, id)
		}
	}

	e.CurrentRemainingMatch = &remainingMatchSeeded
}

func (e *MatchProcessingEnv) genUnSeededMatch(resultList *[]CompareResult, getId func(CompareResult) int) {
	remainingMatchUnSeeded := make([]int, 0, len(e.RemainingMatch))

	if len(e.RemainingMatch) == 0 {
		e.CurrentRemainingMatch = &remainingMatchUnSeeded
		return
	}

	resultIdx := 0
	id := 0

	for _, remainingItem := range e.RemainingMatch {
		for ; resultIdx < len(*resultList); resultIdx++ {
			id = getId((*resultList)[resultIdx])
			if id >= remainingItem {
				break
			}
		}
		if remainingItem != id || resultIdx >= len(*resultList) {
			remainingMatchUnSeeded = append(remainingMatchUnSeeded, remainingItem)
		}
	}

	e.CurrentRemainingMatch = &remainingMatchUnSeeded
}

func (e *MatchProcessingEnv) trimremainingItems(idsToDrop *[]int) {

	sort.Slice((*idsToDrop), func(i, j int) bool { return (*idsToDrop)[i] < (*idsToDrop)[j] })

	remainingCount := len((*e).RemainingMatch) - len(*idsToDrop)
	newremainingItems := make([]int, remainingCount)

	dropI := 0
	oldI := 0
	newI := 0

	for oldI < len((*e).RemainingMatch) {

		if dropI < len(*idsToDrop) {
			if (*idsToDrop)[dropI] < (*e).RemainingMatch[oldI] {
				log.Error("Dropping a non-remaining ID?? dropI: %d idsToDrop[dropI]: %d", dropI, (*idsToDrop)[dropI])
				dropI++
				continue

			} else if (*idsToDrop)[dropI] == (*e).RemainingMatch[oldI] {
				dropI++
				oldI++
				continue
			}
		}

		newremainingItems[newI] = (*e).RemainingMatch[oldI]
		newI++

		oldI++

	}

	if newI != remainingCount {

		log.Error("Did not trim properly?? remainingCount: %d newI: %d", remainingCount, newI)
		log.Error("Did not trim properly?? len(e.remainingItems): %d len(idsToDrop): %d", len((*e).RemainingMatch), len(*idsToDrop))

	}

	(*e).RemainingMatch = newremainingItems
	(*e).CurrentRemainingMatch = &(*e).RemainingMatch
}

func (e *MatchProcessingEnv) reduceRemainingMatchList(uniqueMatch *[]CompareResult, getId func(CompareResult) int) {
	idList := make([]int, len(*uniqueMatch))

	i := 0
	for _, result := range *uniqueMatch {

		idList[i] = getId(result)
		i++
	}

	e.trimremainingItems(&idList)
}
