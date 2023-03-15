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
)

type EntityProcessing struct {
	source          EntityProcessingEnv
	target          EntityProcessingEnv
	matchedEntities map[int]int
}

func newEntityProcessing(rawEntityListPtrSource *RawEntityList, sourceType config.Type, rawEntityListPtrTarget *RawEntityList, targetType config.Type) *EntityProcessing {
	e := new(EntityProcessing)
	e.matchedEntities = map[int]int{}

	rawEntityListPtrSource.Sort()
	rawEntityListPtrTarget.Sort()

	e.source = EntityProcessingEnv{
		rawEntityListPtr:  rawEntityListPtrSource,
		configType:        sourceType,
		remainingEntities: genremainingEntitiesList(rawEntityListPtrSource),
	}
	e.target = EntityProcessingEnv{
		rawEntityListPtr:  rawEntityListPtrTarget,
		configType:        targetType,
		remainingEntities: genremainingEntitiesList(rawEntityListPtrTarget),
	}

	return e
}

func genremainingEntitiesList(rawEntityListPtr *RawEntityList) []int {
	remainingEntitiesList := make([]int, len(*rawEntityListPtr))
	for i := range *rawEntityListPtr {
		remainingEntitiesList[i] = i
	}

	return remainingEntitiesList
}

func (e *EntityProcessing) adjustremainingEntities(singleToSingleMatch []CompareResult, resultList []CompareResult) {

	sort.Sort(ByLeft(singleToSingleMatch))
	e.source.reduceRemainingEntityList(singleToSingleMatch, getLeftId)
	sort.Sort(ByRight(singleToSingleMatch))
	e.target.reduceRemainingEntityList(singleToSingleMatch, getRightId)

}

func (e *EntityProcessing) prepareremainingEntities(keepSeeded bool, keepUnseeded bool, resultListPtr *IndexCompareResultList) {

	if keepSeeded && keepUnseeded {
		e.source.currentremainingEntities = &(e.source.remainingEntities)
		e.target.currentremainingEntities = &(e.target.remainingEntities)
	} else if keepSeeded {
		sort.Sort(ByLeft(resultListPtr.compareResults))
		e.source.genSeededEntities(resultListPtr.compareResults, getLeftId)
		e.source.currentremainingEntities = &(e.source.remainingEntitiesSeeded)

		sort.Sort(ByRight(resultListPtr.compareResults))
		e.target.genSeededEntities(resultListPtr.compareResults, getRightId)
		e.target.currentremainingEntities = &(e.target.remainingEntitiesSeeded)
	} else if keepUnseeded {
		sort.Sort(ByLeft(resultListPtr.compareResults))
		e.source.genUnSeededEntities(resultListPtr.compareResults, getLeftId, &(e.source.remainingEntities))
		e.source.currentremainingEntities = &(e.source.remainingEntitiesUnSeeded)

		sort.Sort(ByRight(resultListPtr.compareResults))
		e.target.genUnSeededEntities(resultListPtr.compareResults, getRightId, &(e.target.remainingEntities))
		e.target.currentremainingEntities = &(e.target.remainingEntitiesUnSeeded)
	}

}
