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

type IndexCompareResultList struct {
	CompareResults []CompareResult
}

func newIndexCompareResultList() *IndexCompareResultList {
	i := new(IndexCompareResultList)
	i.CompareResults = []CompareResult{}
	return i
}

func newReversedIndexCompareResultList(sourceList *IndexCompareResultList) *IndexCompareResultList {
	i := new(IndexCompareResultList)
	size := len(sourceList.CompareResults)
	i.CompareResults = make([]CompareResult, size)
	resI := 0

	for _, result := range sourceList.CompareResults {
		i.CompareResults[resI] = CompareResult{result.RightId, result.LeftId, result.Weight}
		resI++
	}

	if resI != size {
		log.Error("Did not reverse properly!")
	}
	return i
}

func (i *IndexCompareResultList) addResult(entityIdSource int, entityIdTarget int, WeightValue int) {
	i.CompareResults = append(i.CompareResults, CompareResult{entityIdSource, entityIdTarget, WeightValue})
}

func (i *IndexCompareResultList) sortTopMatches() {

	sort.Sort(ByTopMatch(i.CompareResults))

}

func (i *IndexCompareResultList) keepTopMatchesOnly() {

	if len(i.CompareResults) == 0 {
		return
	}

	i.sortTopMatches()

	topMatchesResults := []CompareResult{}
	prevTop := i.CompareResults[0]

	for _, result := range i.CompareResults {

		if result.LeftId == prevTop.LeftId {
			if result.Weight == prevTop.Weight {

			} else {
				continue
			}
		} else {
			prevTop = result
		}

		topMatchesResults = append(topMatchesResults, result)

	}

	i.CompareResults = topMatchesResults

}

func (i *IndexCompareResultList) reduceBothForwardAndBackward() *IndexCompareResultList {

	i.keepTopMatchesOnly()

	reverseResults := newReversedIndexCompareResultList(i)
	reverseResults.keepTopMatchesOnly()

	i.CompareResults = newReversedIndexCompareResultList(reverseResults).CompareResults

	return reverseResults
}

func (i *IndexCompareResultList) sort() {

	sort.Sort(ByLeftRight(i.CompareResults))

}

func (i *IndexCompareResultList) getSingleMatchItems() []CompareResult {

	if len(i.CompareResults) == 0 {
		return []CompareResult{}
	}

	i.sort()

	singleMatchItems := []CompareResult{}

	prevResult := i.CompareResults[0]
	prevTotalSeen := 1

	keepSingleMatch := func() {
		if prevTotalSeen == 1 {
			singleMatchItems = append(singleMatchItems, prevResult)
		}
	}

	for _, compareResult := range i.CompareResults[1:] {
		if compareResult.LeftId == prevResult.LeftId {
			prevTotalSeen += 1
		} else {
			keepSingleMatch()
			prevResult = compareResult
			prevTotalSeen = 1
		}
	}
	keepSingleMatch()

	return singleMatchItems
}

func (i *IndexCompareResultList) sumMatchWeightValues() {

	i.sort()

	summedMatchResults := []CompareResult{}
	prevTotal := i.CompareResults[0]

	aI := 0
	bI := 1

	for bI < len(i.CompareResults) {
		a := i.CompareResults[aI]
		b := i.CompareResults[bI]

		if a.areIdsEqual(b) {
			prevTotal.Weight += b.Weight
		} else {
			summedMatchResults = append(summedMatchResults, prevTotal)
			prevTotal = b
		}

		aI++
		bI++
	}

	summedMatchResults = append(summedMatchResults, prevTotal)

	i.CompareResults = summedMatchResults

}

func (i *IndexCompareResultList) getMaxWeight() int {
	var max_weight int = 0
	for _, result := range i.CompareResults {
		if result.Weight > max_weight {
			max_weight = result.Weight
		}
	}

	return max_weight
}

func (i *IndexCompareResultList) elevateWeight(lowerMaxWeight int) {
	for idx, _ := range i.CompareResults {
		i.CompareResults[idx].Weight += lowerMaxWeight
	}
}

func (i *IndexCompareResultList) trimSingleToSingleMatches(singleToSingleMatchItems []CompareResult) {

	newLen := len(i.CompareResults) - len(singleToSingleMatchItems)
	trimmedList := make([]CompareResult, newLen)

	i.sort()
	sort.Sort(ByLeftRight(singleToSingleMatchItems))

	curI := 0
	sglI := 0
	trmI := 0
	var diff int

	for curI < len(i.CompareResults) {

		if sglI >= len(singleToSingleMatchItems) {
			diff = -1
		} else {
			diff = compareCompareResults(i.CompareResults[curI], singleToSingleMatchItems[sglI])
		}

		if diff < 0 {
			trimmedList[trmI] = i.CompareResults[curI]
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
		log.Error("Did not trim properly?? len(i.CompareResults): %d len(singleToSingleMatchItems): %d", len(i.CompareResults), len(singleToSingleMatchItems))
	}

	i.CompareResults = trimmedList

}

func (i *IndexCompareResultList) ProcessMatches() []CompareResult {

	if len(i.CompareResults) == 0 {
		return []CompareResult{}
	}

	i.sumMatchWeightValues()
	singleToSingleMatchItems := keepSingleToSingleMatchItemsLeftRight(i)

	i.trimSingleToSingleMatches(singleToSingleMatchItems)

	return singleToSingleMatchItems

}

func (i *IndexCompareResultList) MergeRemainingWeightType(remainingResults *IndexCompareResultList) {
	lowerMaxWeight := i.getMaxWeight()
	remainingResults.elevateWeight(lowerMaxWeight)

	i.CompareResults = append(i.CompareResults, remainingResults.CompareResults...)
	i.sort()
}
