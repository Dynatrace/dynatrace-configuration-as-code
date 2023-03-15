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

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"
)

type IndexCompareResultList struct {
	ruleType       IndexRuleType
	compareResults []CompareResult
}

func newIndexCompareResultList(ruleType IndexRuleType) *IndexCompareResultList {
	i := new(IndexCompareResultList)
	i.ruleType = ruleType
	i.compareResults = []CompareResult{}
	return i
}

func newReversedIndexCompareResultList(sourceList *IndexCompareResultList) *IndexCompareResultList {
	i := new(IndexCompareResultList)
	i.ruleType = sourceList.ruleType
	size := len(sourceList.compareResults)
	i.compareResults = make([]CompareResult, size)
	resI := 0

	for _, result := range sourceList.compareResults {
		i.compareResults[resI] = CompareResult{result.rightId, result.leftId, result.weight}
		resI++
	}

	if resI != size {
		log.Error("Did not reverse properly!")
	}
	return i
}

func (i *IndexCompareResultList) addResult(entityIdSource int, entityIdTarget int, weightValue int) {
	i.compareResults = append(i.compareResults, CompareResult{entityIdSource, entityIdTarget, weightValue})
}

func (i *IndexCompareResultList) processMatches() []CompareResult {

	if len(i.compareResults) == 0 {
		return []CompareResult{}
	}

	i.sumMatchWeightValues()
	reverseResults := i.reduceBothForwardAndBackward()
	singleToSingleMatchEntities := keepSingleToSingleMatchEntitiesLeftRight(i, reverseResults)

	i.trimSingleToSingleMatches(singleToSingleMatchEntities)

	return singleToSingleMatchEntities

}

func (i *IndexCompareResultList) reduceBothForwardAndBackward() *IndexCompareResultList {

	i.keepTopMatchesOnly()

	reverseResults := newReversedIndexCompareResultList(i)
	reverseResults.keepTopMatchesOnly()

	i.compareResults = newReversedIndexCompareResultList(reverseResults).compareResults

	return reverseResults
}

func (i *IndexCompareResultList) keepSingleMatchEntities() []CompareResult {

	if len(i.compareResults) == 0 {
		return []CompareResult{}
	}

	i.sort()

	singleMatchEntities := []CompareResult{}

	prevResult := i.compareResults[0]
	prevTotalSeen := 1

	keepSingleMatch := func() {
		if prevTotalSeen == 1 {
			singleMatchEntities = append(singleMatchEntities, prevResult)
		}
	}

	for _, compareResult := range i.compareResults[1:] {
		if compareResult.leftId == prevResult.leftId {
			prevTotalSeen += 1
		} else {
			keepSingleMatch()
			prevResult = compareResult
			prevTotalSeen = 1
		}
	}
	keepSingleMatch()

	return singleMatchEntities
}

func (i *IndexCompareResultList) sort() {

	sort.Sort(ByLeftRight(i.compareResults))

}

func (i *IndexCompareResultList) sumMatchWeightValues() {

	i.sort()

	summedMatchResults := []CompareResult{}
	prevTotal := i.compareResults[0]

	aI := 0
	bI := 1

	for bI < len(i.compareResults) {
		a := i.compareResults[aI]
		b := i.compareResults[bI]

		if a.areIdsEqual(b) {
			prevTotal.weight += b.weight
		} else {
			summedMatchResults = append(summedMatchResults, prevTotal)
			prevTotal = b
		}

		aI++
		bI++
	}

	summedMatchResults = append(summedMatchResults, prevTotal)

	i.compareResults = summedMatchResults

}

func (i *IndexCompareResultList) getMaxWeight() int {
	var max_weight int = 0
	for _, result := range i.compareResults {
		if result.weight > max_weight {
			max_weight = result.weight
		}
	}

	return max_weight
}

func (i *IndexCompareResultList) elevateWeight(lowerMaxWeight int) {
	for _, result := range i.compareResults {
		result.weight += lowerMaxWeight
	}
}

func (i *IndexCompareResultList) keepTopMatchesOnly() {

	if len(i.compareResults) == 0 {
		return
	}

	i.sortTopMatches()

	topMatchesResults := []CompareResult{}
	prevTop := i.compareResults[0]

	for _, result := range i.compareResults {

		if result.leftId == prevTop.leftId {
			if result.weight == prevTop.weight {

			} else {
				continue
			}
		} else {
			prevTop = result
		}

		topMatchesResults = append(topMatchesResults, result)

	}

	i.compareResults = topMatchesResults

}

func (i *IndexCompareResultList) trimSingleToSingleMatches(singleToSingleMatchEntities []CompareResult) {

	newLen := len(i.compareResults) - len(singleToSingleMatchEntities)
	trimmedList := make([]CompareResult, newLen)

	i.sort()
	sort.Sort(ByLeftRight(singleToSingleMatchEntities))

	curI := 0
	sglI := 0
	trmI := 0
	var diff int

	for curI < len(i.compareResults) {

		if sglI >= len(singleToSingleMatchEntities) {
			diff = -1
		} else {
			diff = compareCompareResults(i.compareResults[curI], singleToSingleMatchEntities[sglI])
		}

		if diff < 0 {
			trimmedList[trmI] = i.compareResults[curI]
			trmI++
			curI++

		} else if diff == 0 {
			curI++
			sglI++

		} else {
			sglI++

		}
	}

	if trmI != newLen {
		log.Error("Did not trim properly?? newLen: %d trmI: %d", newLen, trmI)
		log.Error("Did not trim properly?? len(i.compareResults): %d len(singleToSingleMatchEntities): %d", len(i.compareResults), len(singleToSingleMatchEntities))
	}

	i.compareResults = trimmedList

}

func (i *IndexCompareResultList) MergeOldWeightType(oldResults *IndexCompareResultList) {
	i.sort()
	oldResults.sort()

	lowerMaxWeight := i.getMaxWeight()
	oldResults.elevateWeight(lowerMaxWeight)

	i.compareResults = append(i.compareResults, oldResults.compareResults...)
}

func (i *IndexCompareResultList) sortTopMatches() {

	sort.Sort(ByTopMatch(i.compareResults))

}
