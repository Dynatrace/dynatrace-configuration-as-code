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
)

type CompareResult struct {
	LeftId  int
	RightId int
	Weight  int
}

func (a CompareResult) areIdsEqual(b CompareResult) bool {
	if a.LeftId == b.LeftId && a.RightId == b.RightId {
		return true
	}
	return false
}

// ByLeftRight implements sort.Interface for []CompareResult based on
// the SourceId and TargetId fields.
type ByLeft []CompareResult

func (a ByLeft) Len() int      { return len(a) }
func (a ByLeft) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLeft) Less(i, j int) bool {
	return a[i].LeftId < a[j].LeftId
}

// ByLeftRight implements sort.Interface for []CompareResult based on
// the SourceId and TargetId fields.
type ByLeftRight []CompareResult

func (a ByLeftRight) Len() int      { return len(a) }
func (a ByLeftRight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLeftRight) Less(i, j int) bool {
	if a[i].LeftId == a[j].LeftId {
		return a[i].RightId < a[j].RightId
	}

	return a[i].LeftId < a[j].LeftId
}

// ByLeftRight implements sort.Interface for []CompareResult based on
// the SourceId and TargetId fields.
type ByRight []CompareResult

func (a ByRight) Len() int      { return len(a) }
func (a ByRight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRight) Less(i, j int) bool {
	return a[i].RightId < a[j].RightId
}

// ByLeftRight implements sort.Interface for []CompareResult based on
// the SourceId and TargetId fields.
type ByRightLeft []CompareResult

func (a ByRightLeft) Len() int      { return len(a) }
func (a ByRightLeft) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRightLeft) Less(i, j int) bool {
	if a[i].RightId == a[j].RightId {
		return a[i].LeftId < a[j].LeftId
	}

	return a[i].RightId < a[j].RightId
}

// ByTopMatch implements sort.Interface for []CompareResult based on
// the SourceId asc and Weight desc fields.
type ByTopMatch []CompareResult

func (a ByTopMatch) Len() int      { return len(a) }
func (a ByTopMatch) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTopMatch) Less(i, j int) bool {
	if a[i].LeftId == a[j].LeftId {
		return a[j].Weight < a[i].Weight
	}
	return a[i].LeftId < a[j].LeftId
}

func compareLeftRightResult(leftRight CompareResult, rightLeft CompareResult) int {
	if leftRight.LeftId == rightLeft.RightId {
		if leftRight.RightId == rightLeft.LeftId {
			return 0
		} else if leftRight.RightId < rightLeft.LeftId {
			return -1
		} else {
			return 1
		}
	} else if leftRight.LeftId < rightLeft.RightId {
		return -2
	} else {
		return 2
	}
}

func CompareResults(a CompareResult, b CompareResult) int {
	if a.LeftId == b.LeftId {
		if a.RightId == b.RightId {
			return 0
		} else if a.RightId < b.RightId {
			return -1
		} else {
			return 1
		}
	} else if a.LeftId < b.LeftId {
		return -2
	} else {
		return 2
	}
}

func KeepSingleToSingleMatchEntitiesLeftRight(leftRight []CompareResult, rightLeft []CompareResult) []CompareResult {

	singleMatchEntities := []CompareResult{}

	sort.Sort(ByLeftRight(leftRight))
	sort.Sort(ByRightLeft(rightLeft))

	leftI := 0
	rightI := 0

	for leftI < len(leftRight) && rightI < len(rightLeft) {

		diff := compareLeftRightResult(leftRight[leftI], rightLeft[rightI])

		if diff < 0 {
			leftI++

		} else if diff == 0 {
			singleMatchEntities = append(singleMatchEntities, leftRight[leftI])

			leftI++
			rightI++

		} else {
			rightI++

		}
	}

	return singleMatchEntities
}

func GetLeftId(compareResult CompareResult) int {
	return compareResult.LeftId
}

func GetRightId(compareResult CompareResult) int {
	return compareResult.RightId
}
