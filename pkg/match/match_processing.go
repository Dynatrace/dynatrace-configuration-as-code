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

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
)

type MatchProcessing struct {
	Source     MatchProcessingEnv
	Target     MatchProcessingEnv
	matchedMap map[int]int
}

type RawMatchList interface {
	Sort()
	Len() int
	GetValues() *[]interface{}
}
type ByRawItemId interface {
	Len() int
	Swap(i, j int)
	Less(i, j int) bool
}

func NewMatchProcessing(rawMatchListSource RawMatchList, SourceType config.Type, rawMatchListTarget RawMatchList, TargetType config.Type) *MatchProcessing {
	e := new(MatchProcessing)
	e.matchedMap = map[int]int{}

	rawMatchListSource.Sort()
	rawMatchListTarget.Sort()

	e.Source = MatchProcessingEnv{
		RawMatchList:   rawMatchListSource,
		ConfigType:     SourceType,
		RemainingMatch: genremainingMatchList(rawMatchListSource),
	}
	e.Target = MatchProcessingEnv{
		RawMatchList:   rawMatchListTarget,
		ConfigType:     TargetType,
		RemainingMatch: genremainingMatchList(rawMatchListTarget),
	}

	return e
}

func genremainingMatchList(rawMatchList RawMatchList) []int {
	remainingMatchList := make([]int, rawMatchList.Len())
	for i := range *rawMatchList.GetValues() {
		remainingMatchList[i] = i
	}

	return remainingMatchList
}

func (e *MatchProcessing) GetEntitiesType() string {

	if (e.Target.ConfigType == config.Type{}) {
		return e.Source.ConfigType.EntitiesType
	}
	return e.Target.ConfigType.EntitiesType

}

func (e *MatchProcessing) adjustremainingMatch(singleToSingleMatch []CompareResult, resultList []CompareResult) {

	sort.Sort(ByLeft(singleToSingleMatch))
	e.Source.reduceRemainingMatchList(singleToSingleMatch, getLeftId)
	sort.Sort(ByRight(singleToSingleMatch))
	e.Target.reduceRemainingMatchList(singleToSingleMatch, getRightId)

}

func (e *MatchProcessing) PrepareRemainingMatch(keepSeeded bool, keepUnseeded bool, resultListPtr *IndexCompareResultList) {

	if keepSeeded && keepUnseeded {
		e.Source.CurrentRemainingMatch = &(e.Source.RemainingMatch)
		e.Target.CurrentRemainingMatch = &(e.Target.RemainingMatch)
	} else if keepSeeded {
		sort.Sort(ByLeft(resultListPtr.CompareResults))
		e.Source.genSeededMatch(resultListPtr.CompareResults, getLeftId)
		e.Source.CurrentRemainingMatch = &(e.Source.remainingMatchSeeded)

		sort.Sort(ByRight(resultListPtr.CompareResults))
		e.Target.genSeededMatch(resultListPtr.CompareResults, getRightId)
		e.Target.CurrentRemainingMatch = &(e.Target.remainingMatchSeeded)
	} else if keepUnseeded {
		sort.Sort(ByLeft(resultListPtr.CompareResults))
		e.Source.genUnSeededMatch(resultListPtr.CompareResults, getLeftId, &(e.Source.RemainingMatch))
		e.Source.CurrentRemainingMatch = &(e.Source.remainingMatchUnSeeded)

		sort.Sort(ByRight(resultListPtr.CompareResults))
		e.Target.genUnSeededMatch(resultListPtr.CompareResults, getRightId, &(e.Target.RemainingMatch))
		e.Target.CurrentRemainingMatch = &(e.Target.remainingMatchUnSeeded)
	}

}
