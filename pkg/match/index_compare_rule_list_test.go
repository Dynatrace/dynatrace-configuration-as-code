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
	"reflect"
	"testing"

	"gotest.tools/assert"
)

func TestNewIndexCompareResultList(t *testing.T) {

	tests := []struct {
		name string
		want IndexCompareResultList
	}{
		{
			name: "newIndexCompareResultList",
			want: IndexCompareResultList{
				CompareResults: []CompareResult{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newIndexCompareResultList()

			if !reflect.DeepEqual(*result, tt.want) {
				t.Errorf("newIndexCompareResultList() result = %v, want %v", result, tt.want)
			}
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
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{5, 6, 1},
					CompareResult{10, 12, 1}},
			},
			want: IndexCompareResultList{
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

			if !reflect.DeepEqual(*result, tt.want) {
				t.Errorf("newReversedIndexCompareResultList() result = %v, want %v", result, tt.want)
			}
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
				CompareResults: []CompareResult{
					CompareResult{3, 4, 1},
					CompareResult{5, 6, 1},
					CompareResult{10, 12, 1},
				},
			},
			toAdd: CompareResult{7, 3, 1},
			want: IndexCompareResultList{
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
			tt.original.addResult(tt.toAdd.LeftId, tt.toAdd.RightId, tt.toAdd.Weight)

			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("addResult() original = %v, want %v", tt.original, tt.want)
			}
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

			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("sortTopMatches() original = %v, want %v", tt.original, tt.want)
			}
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

			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("keepTopMatchesOnly() original = %v, want %v", tt.original, tt.want)
			}
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
				CompareResults: []CompareResult{
					CompareResult{5, 6, 3},
					CompareResult{3, 8, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			wantReturned: IndexCompareResultList{
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
			if !reflect.DeepEqual(tt.original, tt.wantChanged) {
				t.Errorf("reduceBothForwardAndBackward() original = %v, wantChanged %v", tt.original, tt.wantChanged)
			}
			if !reflect.DeepEqual(*got, tt.wantReturned) {
				t.Errorf("reduceBothForwardAndBackward() got = %v, wantReturned %v", got, tt.wantReturned)
			}
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
				CompareResults: []CompareResult{
					CompareResult{1, 2, 3},
					CompareResult{3, 2, 2},
					CompareResult{2, 1, 1},
					CompareResult{3, 1, 2},
					CompareResult{1, 3, 3},
				},
			},
			want: IndexCompareResultList{
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

			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("sort() original = %v, want %v", tt.original, tt.want)
			}
		})
	}
}

func TestGetUniqueMatchItems(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		want     []CompareResult
	}{
		{
			name: "getUniqueMatchItems - ordered",
			original: IndexCompareResultList{
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
			name: "getUniqueMatchItems - unordered",
			original: IndexCompareResultList{
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
			got := tt.original.getUniqueMatchItems()

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUniqueMatchItems() got = %v, want %v", got, tt.want)
			}
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

			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("sumMatchWeightValues() original = %v, want %v", tt.original, tt.want)
			}
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

			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("elevateWeight() original = %v, want %v", tt.original, tt.want)
			}
		})
	}
}

func TestTrimUniqueMatches(t *testing.T) {

	tests := []struct {
		name          string
		original      IndexCompareResultList
		uniqueMatches []CompareResult
		want          IndexCompareResultList
	}{
		{
			name: "trimUniqueMatches",
			original: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{5, 6, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			uniqueMatches: []CompareResult{
				CompareResult{3, 8, 3},
				CompareResult{5, 6, 3},
			},
			want: IndexCompareResultList{
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
			tt.original.trimUniqueMatches(tt.uniqueMatches)

			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("trimUniqueMatches() original = %v, want %v", tt.original, tt.want)
			}
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

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProcessMatches() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeRemainingWeightType(t *testing.T) {

	tests := []struct {
		name     string
		original IndexCompareResultList
		toMerge  IndexCompareResultList
		want     IndexCompareResultList
	}{
		{
			name: "MergeRemainingWeightType - unsorted",
			original: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{13, 18, 2},
					CompareResult{15, 16, 1},
					CompareResult{20, 22, 2},
					CompareResult{20, 26, 1},
					CompareResult{20, 27, 2},
				},
			},
			toMerge: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{5, 6, 3},
					CompareResult{10, 12, 8},
					CompareResult{10, 16, 8},
					CompareResult{10, 17, 8},
				},
			},
			want: IndexCompareResultList{
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
			name: "MergeRemainingWeightType - sorted",
			original: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{13, 18, 2},
					CompareResult{20, 26, 1},
					CompareResult{20, 22, 2},
					CompareResult{15, 16, 1},
					CompareResult{20, 27, 2},
				},
			},
			toMerge: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{3, 8, 3},
					CompareResult{10, 16, 8},
					CompareResult{10, 12, 8},
					CompareResult{5, 6, 3},
					CompareResult{10, 17, 8},
				},
			},
			want: IndexCompareResultList{
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
			tt.original.MergeRemainingWeightType(&tt.toMerge)

			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("MergeRemainingWeightType() original = %v, want %v", tt.original, tt.want)
			}
		})
	}
}
