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
)

type IndexRuleType struct {
	IsSeed      bool
	WeightValue int
	IndexRules  []IndexRule
}

type IndexRule struct {
	Name              string
	Path              []string
	WeightValue       int
	SelfMatchDisabled bool
}

// ByWeightTypeValue implements sort.Interface for []IndexRule based on
// the WeightTypeValue field.
type ByWeightTypeValue []IndexRuleType

func (a ByWeightTypeValue) Len() int           { return len(a) }
func (a ByWeightTypeValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeightTypeValue) Less(i, j int) bool { return a[j].WeightValue < a[i].WeightValue }

type IndexRuleMapGenerator struct {
	SelfMatch    bool
	baseRuleList []IndexRuleType
}

func NewIndexRuleMapGenerator(selfMatch bool, ruleList []IndexRuleType) *IndexRuleMapGenerator {
	i := new(IndexRuleMapGenerator)
	i.SelfMatch = selfMatch
	i.baseRuleList = ruleList
	return i
}

func (i *IndexRuleMapGenerator) genActiveList() []IndexRuleType {

	activeList := make([]IndexRuleType, 0, len(i.baseRuleList))

	for _, confType := range i.baseRuleList {
		ruleType := IndexRuleType{
			IsSeed:      confType.IsSeed,
			WeightValue: confType.WeightValue,
			IndexRules:  make([]IndexRule, 0, len(confType.IndexRules)),
		}
		for _, conf := range confType.IndexRules {
			if conf.SelfMatchDisabled && i.SelfMatch {
				continue
			}
			ruleType.IndexRules = append(ruleType.IndexRules, conf)
		}
		if len(ruleType.IndexRules) >= 1 {
			activeList = append(activeList, ruleType)
		}
	}

	return activeList
}

func (i *IndexRuleMapGenerator) genSortedActiveList() []IndexRuleType {

	activeList := i.genActiveList()

	sort.Sort(ByWeightTypeValue(activeList))

	return activeList
}

func (i *IndexRule) runIndexRule(entityProcessingPtr *MatchProcessing, resultListPtr *IndexCompareResultList) {

	sortedIndexSource := genSortedItemsIndex(*i, &(*entityProcessingPtr).Source)
	sortedIndexTarget := genSortedItemsIndex(*i, &(*entityProcessingPtr).Target)

	compareIndexes(resultListPtr, sortedIndexSource, sortedIndexTarget, *i)

}

func keepMatches(matchedEntities map[int]int, singleToSingleMatch []CompareResult) map[int]int {
	for _, result := range singleToSingleMatch {
		_, found := matchedEntities[result.LeftId]

		if found {
			log.Error("Should never find multiple exact matches for an entity, %v", result)
		}

		matchedEntities[result.LeftId] = result.RightId
	}

	return matchedEntities
}

func (i *IndexRuleMapGenerator) RunIndexRuleAll(matchProcessingPtr *MatchProcessing) (*IndexCompareResultList, *map[int]int) {
	matchedEntities := map[int]int{}
	remainingResultsPtr := &IndexCompareResultList{}

	ruleTypes := i.genSortedActiveList()

	log.Info("Type: %s -> nb source %d and nb target %d", matchProcessingPtr.GetEntitiesType(),
		matchProcessingPtr.Source.RawMatchList.Len(), matchProcessingPtr.Target.RawMatchList.Len())

	for _, indexRuleType := range ruleTypes {
		resultListPtr := newIndexCompareResultList()
		matchProcessingPtr.PrepareRemainingMatch(true, indexRuleType.IsSeed, remainingResultsPtr)

		for _, indexRule := range indexRuleType.IndexRules {
			indexRule.runIndexRule(matchProcessingPtr, resultListPtr)
		}

		resultListPtr.MergeRemainingWeightType(remainingResultsPtr)
		singleToSingleMatchEntities := resultListPtr.ProcessMatches()
		remainingResultsPtr = resultListPtr

		matchProcessingPtr.adjustremainingMatch(&singleToSingleMatchEntities, &resultListPtr.CompareResults)

		matchedEntities = keepMatches(matchedEntities, singleToSingleMatchEntities)
	}

	log.Info("Type: %s -> nb source %d and nb target %d -> Matched: %d",
		matchProcessingPtr.GetEntitiesType(), len(*matchProcessingPtr.Source.RawMatchList.GetValues()),
		len(*matchProcessingPtr.Target.RawMatchList.GetValues()), len(matchedEntities))

	return remainingResultsPtr, &matchedEntities
}
