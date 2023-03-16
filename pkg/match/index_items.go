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

import "sort"

type IndexMap map[string][]int

type IndexEntry struct {
	indexValue string
	matchedIds []int
}

type ByIndexValue []IndexEntry

func (a ByIndexValue) Len() int           { return len(a) }
func (a ByIndexValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByIndexValue) Less(i, j int) bool { return a[i].indexValue < a[j].indexValue }

func genSortedItemsIndex(indexRule IndexRule, items *MatchProcessingEnv) []IndexEntry {

	index := IndexMap{}

	for _, entityIdx := range *(items.CurrentremainingMatch) {

		value := getValueFromPath((*items.RawMatchList.GetValues())[entityIdx], indexRule.path)
		if value == nil {
			// pass
		} else {
			addValueToIndex(&index, value, entityIdx)
		}

	}

	flatSortedIndex := flattenSortIndex(&index)

	return flatSortedIndex
}

func flattenSortIndex(index *IndexMap) []IndexEntry {

	flatIndex := make([]IndexEntry, len(*index))
	idx := 0

	for indexValue, matchedIds := range *index {
		flatIndex[idx] = IndexEntry{
			indexValue: indexValue,
			matchedIds: matchedIds,
		}
		idx++
	}

	sort.Sort(ByIndexValue(flatIndex))

	return flatIndex
}

func getValueFromPath(entity interface{}, path []string) interface{} {

	if len(path) <= 0 {
		return nil
	}

	var current interface{}
	current = entity

	for _, field := range path {

		fieldValue, ok := (current.(map[string]interface{}))[field]
		if ok {
			current = fieldValue
		} else {
			current = nil
			break
		}

	}

	if current == nil {
		return nil
	} else {
		return current
	}
}

func addValueToIndex(index *IndexMap, value interface{}, entityId int) {

	stringValue, isString := value.(string)

	if isString {
		addSingleValueToIndex(
			index, stringValue, entityId)
	} else {
		sliceValue := value.([]interface{})

		for _, singleValue := range sliceValue {
			addSingleValueToIndex(
				index, singleValue.(string), entityId)
		}
	}
}

func addSingleValueToIndex(index *IndexMap, value string, entityId int) {

	if value == "" {
		return
	}

	(*index)[value] = append((*index)[value], entityId)

}
