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

package match

import (
	"sort"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"
)

type IndexRuleType struct {
	IsSeed      bool
	weightValue int
	IndexRules  []IndexRule
}

type IndexRule struct {
	name              string
	path              []string
	weightValue       int
	selfMatchDisabled bool
}

// ByWeightTypeValue implements sort.Interface for []IndexRule based on
// the WeightTypeValue field.
type ByWeightTypeValue []IndexRuleType

func (a ByWeightTypeValue) Len() int           { return len(a) }
func (a ByWeightTypeValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeightTypeValue) Less(i, j int) bool { return a[j].weightValue < a[i].weightValue }

type IndexRuleMapGenerator struct {
	SelfMatch bool
}

var INDEX_CONFIG_LIST_ALL = []IndexRuleType{
	{
		IsSeed:      true,
		weightValue: 100,
		IndexRules: []IndexRule{
			{
				name:              "Detected Name",
				path:              []string{"properties", "detectedName"},
				weightValue:       1,
				selfMatchDisabled: false,
			},
			{
				name:              "One Agent Custom Host Name",
				path:              []string{"properties", "oneAgentCustomHostName"},
				weightValue:       1,
				selfMatchDisabled: false,
			},
		},
	},
	{
		IsSeed:      true,
		weightValue: 90,
		IndexRules: []IndexRule{
			{
				name:              "Entity Id",
				path:              []string{"entityId"},
				weightValue:       1,
				selfMatchDisabled: true,
			},
			{
				name:              "Display Name",
				path:              []string{"displayName"},
				weightValue:       1,
				selfMatchDisabled: false,
			},
		},
	},
	// ipAddress was tested with IsSeed = false on 5 million RBC Pre-Prod entities
	// All matches were identical, except for Network Interfaces the were not matching as well
	// Keeping IsSeed = true only has positive return
	{
		IsSeed:      true,
		weightValue: 50,
		IndexRules: []IndexRule{
			{
				name:              "Ip Addresses List",
				path:              []string{"properties", "ipAddress"},
				weightValue:       2,
				selfMatchDisabled: false,
			},
		},
	},
}

func NewIndexRuleMapGenerator(selfMatch bool) *IndexRuleMapGenerator {
	i := new(IndexRuleMapGenerator)
	i.SelfMatch = selfMatch
	return i
}

func (i *IndexRuleMapGenerator) genActiveList() []IndexRuleType {

	activeList := make([]IndexRuleType, 0, len(INDEX_CONFIG_LIST_ALL))

	for _, confType := range INDEX_CONFIG_LIST_ALL {
		ruleType := IndexRuleType{
			IsSeed:      confType.IsSeed,
			weightValue: confType.weightValue,
			IndexRules:  make([]IndexRule, 0, len(confType.IndexRules)),
		}
		for _, conf := range confType.IndexRules {
			if conf.selfMatchDisabled && i.SelfMatch {
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

func (i *IndexRuleMapGenerator) RunIndexRuleAll(itemsType string, matchProcessingPtr *MatchProcessing) (*IndexCompareResultList, *map[int]int) {
	matchedEntities := map[int]int{}
	oldResultsPtr := &IndexCompareResultList{}

	ruleTypes := i.genSortedActiveList()

	log.Info("Type: %s -> nb source %d and nb target %d", itemsType,
		matchProcessingPtr.Source.RawMatchList.Len(), matchProcessingPtr.Target.RawMatchList.Len())

	for _, indexRuleType := range ruleTypes {
		resultListPtr := newIndexCompareResultList(indexRuleType)
		matchProcessingPtr.PrepareRemainingMatch(true, indexRuleType.IsSeed, oldResultsPtr)

		for _, indexRule := range indexRuleType.IndexRules {
			indexRule.runIndexRule(matchProcessingPtr, resultListPtr)
		}

		resultListPtr.MergeOldWeightType(oldResultsPtr)
		singleToSingleMatchEntities := resultListPtr.ProcessMatches()
		oldResultsPtr = resultListPtr

		matchProcessingPtr.adjustremainingMatch(singleToSingleMatchEntities, resultListPtr.CompareResults)

		matchedEntities = keepMatches(matchedEntities, singleToSingleMatchEntities)
	}

	log.Info("Type: %s -> nb source %d and nb target %d -> Matched: %d",
		itemsType, len(*matchProcessingPtr.Source.RawMatchList.GetValues()),
		len(*matchProcessingPtr.Target.RawMatchList.GetValues()), len(matchedEntities))

	return oldResultsPtr, &matchedEntities
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
