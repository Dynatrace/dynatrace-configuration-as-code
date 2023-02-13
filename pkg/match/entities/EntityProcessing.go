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
	Source          EntityProcessingEnv
	Target          EntityProcessingEnv
	MatchedEntities map[int]int
}

func NewEntityProcessing(rawEntityListPtrSource *RawEntityList, sourceType config.Type, rawEntityListPtrTarget *RawEntityList, targetType config.Type) *EntityProcessing {
	e := new(EntityProcessing)
	e.MatchedEntities = map[int]int{}

	rawEntityListPtrSource.Sort()
	rawEntityListPtrTarget.Sort()

	e.Source = EntityProcessingEnv{
		RawEntityListPtr:  rawEntityListPtrSource,
		Type:              sourceType,
		RemainingEntities: genRemainingEntitiesList(rawEntityListPtrSource),
	}
	e.Target = EntityProcessingEnv{
		RawEntityListPtr:  rawEntityListPtrTarget,
		Type:              targetType,
		RemainingEntities: genRemainingEntitiesList(rawEntityListPtrTarget),
	}

	return e
}

func genRemainingEntitiesList(rawEntityListPtr *RawEntityList) []int {
	remainingEntitiesList := make([]int, len(*rawEntityListPtr))
	for i := range *rawEntityListPtr {
		remainingEntitiesList[i] = i
	}

	return remainingEntitiesList
}

func (e *EntityProcessing) AdjustRemainingEntities(singleToSingleMatch []CompareResult, resultList []CompareResult) {

	sort.Sort(ByLeft(singleToSingleMatch))
	e.Source.ReduceRemainingEntityList(singleToSingleMatch, GetLeftId)
	sort.Sort(ByRight(singleToSingleMatch))
	e.Target.ReduceRemainingEntityList(singleToSingleMatch, GetRightId)

}

func (e *EntityProcessing) PrepareRemainingEntities(indexRuleType IndexRuleType, resultListPtr *IndexCompareResultList) {

	if indexRuleType.IsSeed {
		e.Source.CurrentRemainingEntities = &(e.Source.RemainingEntities)
		e.Target.CurrentRemainingEntities = &(e.Target.RemainingEntities)
	} else {
		sort.Sort(ByLeft(resultListPtr.CompareResults))
		e.Source.GenSeededEntities(resultListPtr.CompareResults, GetLeftId)
		e.Source.CurrentRemainingEntities = &(e.Source.RemainingEntitiesSeeded)

		sort.Sort(ByRight(resultListPtr.CompareResults))
		e.Target.GenSeededEntities(resultListPtr.CompareResults, GetRightId)
		e.Target.CurrentRemainingEntities = &(e.Target.RemainingEntitiesSeeded)
	}

}
