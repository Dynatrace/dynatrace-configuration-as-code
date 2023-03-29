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

//go:build unit

package match

import (
	"encoding/json"
	"sort"
	"testing"

	"gotest.tools/assert"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match/rules"
)

type RawMatchListImpl struct {
	Values *[]interface{}
}

// ByRawMatchId implements sort.Interface for []RawMatch] based on
// the EntityId string field.
type ByRawMatchId []interface{}

func (a ByRawMatchId) Len() int      { return len(a) }
func (a ByRawMatchId) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRawMatchId) Less(i, j int) bool {
	return (a[i].(map[string]interface{}))["entityId"].(string) < (a[j].(map[string]interface{}))["entityId"].(string)
}

func (r *RawMatchListImpl) Sort() {

	sort.Sort(ByRawMatchId(*r.GetValues()))

}

func (r *RawMatchListImpl) Len() int {

	return len(*r.GetValues())

}

func (r *RawMatchListImpl) GetValues() *[]interface{} {

	return r.Values

}

func getRawMatchListFromJson(jsonData string) RawMatchList {
	rawMatchList := &RawMatchListImpl{
		Values: new([]interface{}),
	}
	json.Unmarshal([]byte(jsonData), rawMatchList.Values)

	return rawMatchList
}

var entityListJsonSorted = `[{
	"entityId": "AZURE_VM-06C38A40104F9FB2",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-06C38A40104F9FB2",
	"firstSeenTms": 1663004439413,
	"lastSeenTms": 1674246569091,
	"properties": {},
	"toRelationships": {}
}, {
	"entityId": "AZURE_VM-109729BAB28C66E8",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-109729BAB28C66E8",
	"firstSeenTms": 1663004173751,
	"lastSeenTms": 1674246562180,
	"properties": {},
	"toRelationships": {}
}, {
	"entityId": "AZURE_VM-2BBAEC9A7D21833E",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-2BBAEC9A7D21833E",
	"firstSeenTms": 1662997868374,
	"lastSeenTms": 1674246568646,
	"properties": {},
	"toRelationships": {}
}]`

var entityJsonSortedValueString = `[{
	"entityId": "AZURE_VM-06C38A40104F9FB2",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-06C38A40104F9FB2",
	"firstSeenTms": 1663004439413,
	"lastSeenTms": 1674246569091,
	"properties": {},
	"toRelationships": {}
}]`

var entityJsonSortedValueList = `[{
	"entityId": "AZURE_VM-06C38A40104F9FB2",
	"type": "AZURE_VM",
	"displayName": ["UNKNOWN AZURE_VM-06C38A40104F9FB2", "KNOWN AZURE_VM-06C38A40104F9FB2"],
	"firstSeenTms": 1663004439413,
	"lastSeenTms": 1674246569091,
	"properties": {},
	"toRelationships": {}
}]`

func TestAddUniqueValueToIndex(t *testing.T) {

	tests := []struct {
		name     string
		indexMap IndexMap
		value    string
		itemId   int
		want     IndexMap
		wantLen  int
	}{
		{
			name:     "addUniqueValueToIndex",
			indexMap: IndexMap{},
			value:    "test",
			itemId:   0,
			want: IndexMap{
				"test": []int{0},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addUniqueValueToIndex(&tt.indexMap, tt.value, tt.itemId)

			assert.Equal(t, len(tt.indexMap), len(tt.want))
			assert.Equal(t, len(tt.indexMap[tt.value]), len(tt.want[tt.value]))
			assert.Equal(t, tt.indexMap[tt.value][0], tt.want[tt.value][0])

		})
	}
}

func TestAddValueToIndex(t *testing.T) {

	tests := []struct {
		name         string
		indexMap     IndexMap
		rawMatchList RawMatchList
		itemId       int
		want         IndexMap
	}{
		{
			name:         "addUniqueValueToIndex - string",
			indexMap:     IndexMap{},
			rawMatchList: getRawMatchListFromJson(entityJsonSortedValueString),
			itemId:       0,
			want: IndexMap{
				"UNKNOWN AZURE_VM-06C38A40104F9FB2": []int{0},
			},
		},
		{
			name:         "addUniqueValueToIndex - string slice",
			indexMap:     IndexMap{},
			rawMatchList: getRawMatchListFromJson(entityJsonSortedValueList),
			itemId:       0,
			want: IndexMap{
				"UNKNOWN AZURE_VM-06C38A40104F9FB2": []int{0},
				"KNOWN AZURE_VM-06C38A40104F9FB2":   []int{0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addValueToIndex(
				&tt.indexMap,
				getValueFromPath((*tt.rawMatchList.GetValues())[0], []string{"displayName"}),
				tt.itemId,
			)

			assert.Equal(t, len(tt.indexMap), len(tt.want))
			for k, _ := range tt.indexMap {
				assert.Equal(t, len(tt.indexMap[k]), len(tt.want[k]))
				for i, _ := range tt.indexMap[k] {
					assert.Equal(t, tt.indexMap[k][i], tt.want[k][i])
				}
			}

		})
	}
}

func stringSliceToInterfaceSlice(stringSl []string) []interface{} {
	slice := make([]interface{}, len(stringSl))
	for i, s := range stringSl {
		slice[i] = s
	}

	return slice
}

func TestGetValueFromPath(t *testing.T) {

	tests := []struct {
		name         string
		rawMatchList RawMatchList
		path         []string
		isSlice      bool
		want         interface{}
	}{
		{
			name:         "getValueFromPath - string",
			rawMatchList: getRawMatchListFromJson(entityJsonSortedValueString),
			path:         []string{"displayName"},
			isSlice:      false,
			want:         "UNKNOWN AZURE_VM-06C38A40104F9FB2",
		},
		{
			name:         "getValueFromPath - string slice",
			rawMatchList: getRawMatchListFromJson(entityJsonSortedValueList),
			path:         []string{"displayName"},
			isSlice:      true,
			want: stringSliceToInterfaceSlice([]string{
				"UNKNOWN AZURE_VM-06C38A40104F9FB2",
				"KNOWN AZURE_VM-06C38A40104F9FB2",
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getValueFromPath((*tt.rawMatchList.GetValues())[0], tt.path)

			if tt.isSlice {
				sliceGot := got.([]interface{})
				sliceWant := tt.want.([]interface{})

				assert.Equal(t, len(sliceGot), len(sliceWant))
				for i, _ := range sliceGot {
					assert.Equal(t, sliceGot[i].(string), sliceWant[i].(string))
				}
			} else {
				assert.Equal(t, got, tt.want)
			}

		})
	}
}

func compareIndexEntrySlice(t *testing.T, got []IndexEntry, want []IndexEntry) {

	assert.Equal(t, len(got), len(want))
	for i, _ := range want {
		assert.Equal(t, got[i].indexValue, want[i].indexValue)
		assert.Equal(t, len(got[i].matchedIds), len(want[i].matchedIds))
		for j, _ := range got[i].matchedIds {
			assert.Equal(t, got[i].matchedIds[j], want[i].matchedIds[j])
		}
	}
}

func TestFlattenSortIndex(t *testing.T) {

	tests := []struct {
		name     string
		indexMap IndexMap
		want     []IndexEntry
	}{
		{
			name: "flattenSortIndex - sorted",
			indexMap: IndexMap{
				"KNOWN AZURE_VM-06C38A40104F9FB2":   []int{1, 2, 3},
				"UNKNOWN AZURE_VM-06C38A40104F9FB2": []int{0},
			},
			want: []IndexEntry{
				IndexEntry{
					indexValue: "KNOWN AZURE_VM-06C38A40104F9FB2",
					matchedIds: []int{1, 2, 3},
				},
				IndexEntry{
					indexValue: "UNKNOWN AZURE_VM-06C38A40104F9FB2",
					matchedIds: []int{0},
				},
			},
		},
		{
			name: "flattenSortIndex - unsorted",
			indexMap: IndexMap{
				"UNKNOWN AZURE_VM-06C38A40104F9FB2": []int{0},
				"KNOWN AZURE_VM-06C38A40104F9FB2":   []int{1, 2, 3},
			},
			want: []IndexEntry{
				IndexEntry{
					indexValue: "KNOWN AZURE_VM-06C38A40104F9FB2",
					matchedIds: []int{1, 2, 3},
				},
				IndexEntry{
					indexValue: "UNKNOWN AZURE_VM-06C38A40104F9FB2",
					matchedIds: []int{0},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flattenSortIndex(&tt.indexMap)

			compareIndexEntrySlice(t, got, tt.want)

		})
	}
}

func TestGenSortedItemsIndex(t *testing.T) {

	tests := []struct {
		name            string
		indexRule       rules.IndexRule
		matchProcessing MatchProcessingEnv
		want            []IndexEntry
	}{
		{
			name: "genSortedItemsIndex",
			indexRule: rules.IndexRule{
				Name:              "test",
				Path:              []string{"displayName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
			matchProcessing: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2},
				RemainingMatch:        []int{},
			},
			want: []IndexEntry{
				IndexEntry{
					indexValue: "UNKNOWN AZURE_VM-06C38A40104F9FB2",
					matchedIds: []int{0},
				},
				IndexEntry{
					indexValue: "UNKNOWN AZURE_VM-109729BAB28C66E8",
					matchedIds: []int{1},
				},
				IndexEntry{
					indexValue: "UNKNOWN AZURE_VM-2BBAEC9A7D21833E",
					matchedIds: []int{2},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := genSortedItemsIndex(tt.indexRule, &tt.matchProcessing)

			compareIndexEntrySlice(t, got, tt.want)

		})
	}
}
