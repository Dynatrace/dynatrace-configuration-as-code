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
	"testing"

	"gotest.tools/assert"
)

func compareIndexCompareResultList(t *testing.T, i *IndexCompareResultList, j *IndexCompareResultList) {
	assert.Equal(t, i.ruleType.IsSeed, j.ruleType.IsSeed)
	assert.Equal(t, i.ruleType.WeightValue, j.ruleType.WeightValue)
	assert.Equal(t, len(i.ruleType.IndexRules), len(j.ruleType.IndexRules))
	for k, _ := range i.ruleType.IndexRules {
		assert.Equal(t, i.ruleType.IndexRules[k].Name, j.ruleType.IndexRules[k].Name)
		assert.Equal(t, i.ruleType.IndexRules[k].Path, j.ruleType.IndexRules[k].Path)
		assert.Equal(t, i.ruleType.IndexRules[k].WeightValue, j.ruleType.IndexRules[k].WeightValue)
		assert.Equal(t, i.ruleType.IndexRules[k].SelfMatchDisabled, j.ruleType.IndexRules[k].SelfMatchDisabled)
	}

	compareCompareResult(t, i.CompareResults, j.CompareResults)

}

func compareCompareResult(t *testing.T, i []CompareResult, j []CompareResult) {

	assert.Equal(t, len(i), len(j))
	for k, _ := range i {
		assert.Equal(t, i[k].LeftId, j[k].LeftId)
		assert.Equal(t, i[k].RightId, j[k].RightId)
		assert.Equal(t, i[k].weight, j[k].weight)
	}

}

func TestNewIndexCompareResultList(t *testing.T) {

	tests := []struct {
		name     string
		ruleType IndexRuleType
		want     IndexCompareResultList
	}{
		{
			name: "newIndexCompareResultList",
			ruleType: IndexRuleType{
				IsSeed:      true,
				WeightValue: 100,
				IndexRules:  []IndexRule{},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newIndexCompareResultList(tt.ruleType)
			compareIndexCompareResultList(t, result, &tt.want)
		})
	}
}
func TestNewReversedIndexCompareResultList(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     IndexCompareResultList
	}{
		{
			name: "newReversedIndexCompareResultList",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{5, 6, 1},
					CompareResult{10, 12, 1}},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{4, 3, 1},
					CompareResult{6, 5, 1},
					CompareResult{12, 10, 1}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newReversedIndexCompareResultList(&tt.original)
			compareIndexCompareResultList(t, result, &tt.want)
		})
	}
}

func TestAddResult(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		toAdd    CompareResult
		want     IndexCompareResultList
	}{
		{
			name: "addResult",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{5, 6, 1},
					CompareResult{10, 12, 1},
				},
			},
			toAdd: CompareResult{7, 3, 1},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{5, 6, 1},
					CompareResult{10, 12, 1},
					CompareResult{7, 3, 1},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.addResult(tt.toAdd.LeftId, tt.toAdd.RightId, tt.toAdd.weight)
			compareIndexCompareResultList(t, &tt.original, &tt.want)
		})
	}
}

func TestSortTopMatches(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     IndexCompareResultList
	}{
		{
			name: "KeepTopMatchesOnly",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{3, 8, 3},
					CompareResult{3, 9, 1},
					CompareResult{5, 6, 3},
					CompareResult{5, 12, 3},
					CompareResult{5, 14, 1},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{3, 4, 1},
					CompareResult{3, 9, 1},
					CompareResult{5, 6, 3},
					CompareResult{5, 12, 3},
					CompareResult{5, 14, 1},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.sortTopMatches()
			compareIndexCompareResultList(t, &tt.original, &tt.want)
		})
	}
}

func TestKeepTopMatchesOnly(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     IndexCompareResultList
	}{
		{
			name: "KeepTopMatchesOnly",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{3, 8, 3},
					CompareResult{3, 9, 1},
					CompareResult{5, 6, 3},
					CompareResult{5, 12, 3},
					CompareResult{5, 14, 1},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{5, 6, 3},
					CompareResult{5, 12, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.keepTopMatchesOnly()
			compareIndexCompareResultList(t, &tt.original, &tt.want)
		})
	}
}

func TestReduceBothForwardAndBackward(t *testing.T) {

	tests := []struct {
		name         string
		original     IndexCompareResultList
		wantChanged  IndexCompareResultList
		wantReturned IndexCompareResultList
	}{
		{
			name: "reduceBothForwardAndBackward",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{3, 8, 3},
					CompareResult{3, 9, 1},
					CompareResult{5, 6, 3},
					CompareResult{5, 12, 3},
					CompareResult{5, 14, 1},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			wantChanged: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{5, 6, 3},
					CompareResult{3, 8, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			wantReturned: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{6, 5, 3},
					CompareResult{8, 3, 3},
					CompareResult{12, 10, 8},
					CompareResult{16, 10, 8},
					CompareResult{17, 10, 8},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.original.reduceBothForwardAndBackward()
			compareIndexCompareResultList(t, &tt.original, &tt.wantChanged)
			compareIndexCompareResultList(t, got, &tt.wantReturned)
		})
	}
}

func TestSortCompareResults(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     IndexCompareResultList
	}{
		{
			name: "sort",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{1, 2, 3},
					CompareResult{3, 2, 2},
					CompareResult{2, 1, 1},
					CompareResult{3, 1, 2},
					CompareResult{1, 3, 3},
				},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{1, 2, 3},
					CompareResult{1, 3, 3},
					CompareResult{2, 1, 1},
					CompareResult{3, 1, 2},
					CompareResult{3, 2, 2},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.sort()
			compareIndexCompareResultList(t, &tt.original, &tt.want)
		})
	}
}

func TestGetSingleMatchItems(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     []CompareResult
	}{
		{
			name: "getSingleMatchItems - ordered",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{5, 6, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			want: []CompareResult{
				CompareResult{3, 8, 3},
				CompareResult{5, 6, 3},
			},
		},
		{
			name: "getSingleMatchItems - unordered",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{5, 6, 3},
					CompareResult{3, 8, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			want: []CompareResult{
				CompareResult{3, 8, 3},
				CompareResult{5, 6, 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.original.getSingleMatchItems()
			compareCompareResult(t, got, tt.want)
		})
	}
}

func TestSumMatchWeightValues(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     IndexCompareResultList
	}{
		{
			name: "sumMatchWeightValues - ordered",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{1, 2, 2},
					CompareResult{1, 2, 1},
					CompareResult{1, 3, 1},
					CompareResult{1, 3, 1},
					CompareResult{1, 3, 1},
					CompareResult{2, 1, 1},
					CompareResult{3, 1, 2},
					CompareResult{3, 2, 2},
					CompareResult{3, 2, 5},
				},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{1, 2, 3},
					CompareResult{1, 3, 3},
					CompareResult{2, 1, 1},
					CompareResult{3, 1, 2},
					CompareResult{3, 2, 7},
				},
			},
		},
		{
			name: "sumMatchWeightValues - unordered",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{2, 1, 1},
					CompareResult{1, 2, 1},
					CompareResult{3, 2, 5},
					CompareResult{1, 3, 1},
					CompareResult{1, 3, 1},
					CompareResult{3, 1, 2},
					CompareResult{1, 2, 2},
					CompareResult{3, 2, 2},
					CompareResult{1, 3, 1},
				},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{1, 2, 3},
					CompareResult{1, 3, 3},
					CompareResult{2, 1, 1},
					CompareResult{3, 1, 2},
					CompareResult{3, 2, 7},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.sumMatchWeightValues()
			compareIndexCompareResultList(t, &tt.original, &tt.want)
		})
	}
}

func TestGetMaxWeight(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     int
	}{
		{
			name: "getMaxWeight",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{1, 2, 3},
					CompareResult{1, 3, 3},
					CompareResult{2, 1, 1},
					CompareResult{3, 1, 2},
					CompareResult{3, 2, 7},
				},
			},
			want: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			assert.Equal(t, tt.original.getMaxWeight(), tt.want)
		})
	}
}

func TestElevateWeight(t *testing.T) {

	tests := []struct {
		name         string
		original     IndexCompareResultList
		elevateValue int
		want         IndexCompareResultList
	}{
		{
			name: "elevateWeight",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{1, 2, 3},
					CompareResult{1, 3, 3},
					CompareResult{2, 1, 1},
					CompareResult{3, 1, 2},
					CompareResult{3, 2, 7},
				},
			},
			elevateValue: 10,
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{1, 2, 13},
					CompareResult{1, 3, 13},
					CompareResult{2, 1, 11},
					CompareResult{3, 1, 12},
					CompareResult{3, 2, 17},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.elevateWeight(tt.elevateValue)
			compareIndexCompareResultList(t, &tt.original, &tt.want)
		})
	}
}

func TestTrimSingleToSingleMatches(t *testing.T) {

	tests := []struct {
		name                  string
		original              IndexCompareResultList
		singleToSingleMatches []CompareResult
		want                  IndexCompareResultList
	}{
		{
			name: "trimSingleToSingleMatches",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{5, 6, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			singleToSingleMatches: []CompareResult{
				CompareResult{3, 8, 3},
				CompareResult{5, 6, 3},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.trimSingleToSingleMatches(tt.singleToSingleMatches)
			compareIndexCompareResultList(t, &tt.original, &tt.want)
		})
	}
}

func TestProcessMatches(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     []CompareResult
	}{
		{
			name: "ProcessMatches - unsorted",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{3, 8, 2},
					CompareResult{10, 16, 4},
					CompareResult{3, 9, 1},
					CompareResult{5, 6, 3},
					CompareResult{10, 17, 4},
					CompareResult{5, 12, 3},
					CompareResult{5, 14, 1},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 4},
					CompareResult{3, 8, 1},
					CompareResult{10, 17, 4},
				},
			},
			want: []CompareResult{
				CompareResult{3, 8, 3},
				CompareResult{5, 6, 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.original.ProcessMatches()
			compareCompareResult(t, got, tt.want)
		})
	}
}

func TestMergeOldWeightType(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		toMerge  IndexCompareResultList
		want     IndexCompareResultList
	}{
		{
			name: "MergeOldWeightType - unsorted",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 90,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{13, 18, 2},
					CompareResult{15, 16, 1},
					CompareResult{20, 22, 2},
					CompareResult{20, 26, 1},
					CompareResult{20, 27, 2},
				},
			},
			toMerge: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{5, 6, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 90,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 8, 5},
					CompareResult{5, 6, 5},
					CompareResult{10, 12, 10},
					CompareResult{10, 16, 10},
					CompareResult{10, 17, 10},
					CompareResult{13, 18, 2},
					CompareResult{15, 16, 1},
					CompareResult{20, 22, 2},
					CompareResult{20, 26, 1},
					CompareResult{20, 27, 2},
				},
			},
		},
		{
			name: "MergeOldWeightType - sorted",
			original: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 90,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{13, 18, 2},
					CompareResult{20, 26, 1},
					CompareResult{20, 22, 2},
					CompareResult{15, 16, 1},
					CompareResult{20, 27, 2},
				},
			},
			toMerge: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{10, 16, 8},
					CompareResult{10, 12, 8},
					CompareResult{5, 6, 3},
					CompareResult{10, 17, 8},
				},
			},
			want: IndexCompareResultList{
				ruleType: IndexRuleType{
					IsSeed:      true,
					WeightValue: 90,
					IndexRules:  []IndexRule{},
				},
				CompareResults: []CompareResult{
					CompareResult{3, 8, 5},
					CompareResult{5, 6, 5},
					CompareResult{10, 12, 10},
					CompareResult{10, 16, 10},
					CompareResult{10, 17, 10},
					CompareResult{13, 18, 2},
					CompareResult{15, 16, 1},
					CompareResult{20, 22, 2},
					CompareResult{20, 26, 1},
					CompareResult{20, 27, 2},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.MergeOldWeightType(&tt.toMerge)
			compareIndexCompareResultList(t, &tt.original, &tt.want)
		})
	}
}
