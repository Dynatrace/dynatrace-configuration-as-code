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
	"sort"
	"testing"

	"gotest.tools/assert"
)

func TestAreIdsEqual(t *testing.T) {

	tests := []struct {
		name        string
		compareFrom CompareResult
		compareTo   CompareResult
		want        bool
	}{
		{
			name:        "Equal",
			compareFrom: CompareResult{1, 1, 1},
			compareTo:   CompareResult{1, 1, 1},
			want:        true,
		},
		{
			name:        "Equal, different weight",
			compareFrom: CompareResult{1, 1, 1},
			compareTo:   CompareResult{1, 1, 2},
			want:        true,
		},
		{
			name:        "DifferentRight",
			compareFrom: CompareResult{1, 1, 1},
			compareTo:   CompareResult{1, 2, 2},
			want:        false,
		},
		{
			name:        "DifferentLeft",
			compareFrom: CompareResult{1, 1, 1},
			compareTo:   CompareResult{2, 1, 2},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.compareFrom.areIdsEqual(tt.compareTo), tt.want)

		})
	}
}

func TestSort(t *testing.T) {

	byLeft := []CompareResult{
		CompareResult{3, 2, 1},
		CompareResult{2, 1, 1},
		CompareResult{1, 3, 1},
	}
	byLeftRight := []CompareResult{
		CompareResult{1, 2, 3},
		CompareResult{3, 2, 2},
		CompareResult{2, 1, 1},
		CompareResult{3, 1, 2},
		CompareResult{1, 3, 3},
	}
	byRight := []CompareResult{
		CompareResult{3, 2, 1},
		CompareResult{2, 1, 1},
		CompareResult{1, 3, 1},
	}
	byRightLeft := []CompareResult{
		CompareResult{1, 2, 1},
		CompareResult{3, 2, 1},
		CompareResult{2, 1, 1},
		CompareResult{3, 1, 1},
		CompareResult{1, 3, 1},
	}
	byTopMatch := []CompareResult{
		CompareResult{1, 2, 3},
		CompareResult{3, 2, 2},
		CompareResult{2, 1, 1},
		CompareResult{3, 1, 1},
		CompareResult{1, 3, 1},
	}

	tests := []struct {
		name      string
		inputList sort.Interface
		inputPtr  *[]CompareResult
		want      []CompareResult
	}{
		{
			name:      "Sort ByLeft",
			inputList: ByLeft(byLeft),
			inputPtr:  &byLeft,
			want: []CompareResult{
				CompareResult{1, 3, 1},
				CompareResult{2, 1, 1},
				CompareResult{3, 2, 1},
			},
		},
		{
			name:      "Sort ByLeftRight",
			inputList: ByLeftRight(byLeftRight),
			inputPtr:  &byLeftRight,
			want: []CompareResult{
				CompareResult{1, 2, 3},
				CompareResult{1, 3, 3},
				CompareResult{2, 1, 1},
				CompareResult{3, 1, 2},
				CompareResult{3, 2, 2},
			},
		},
		{
			name:      "Sort ByRight",
			inputList: ByRight(byRight),
			inputPtr:  &byRight,
			want: []CompareResult{
				CompareResult{2, 1, 1},
				CompareResult{3, 2, 1},
				CompareResult{1, 3, 1},
			},
		},
		{
			name:      "Sort ByRightLeft",
			inputList: ByRightLeft(byRightLeft),
			inputPtr:  &byRightLeft,
			want: []CompareResult{
				CompareResult{2, 1, 1},
				CompareResult{3, 1, 1},
				CompareResult{1, 2, 1},
				CompareResult{3, 2, 1},
				CompareResult{1, 3, 1},
			},
		},
		{
			name:      "Sort ByTopMatch",
			inputList: ByTopMatch(byTopMatch),
			inputPtr:  &byTopMatch,
			want: []CompareResult{
				CompareResult{1, 2, 3},
				CompareResult{1, 3, 1},
				CompareResult{2, 1, 1},
				CompareResult{3, 2, 2},
				CompareResult{3, 1, 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort.Sort(tt.inputList)

			if !reflect.DeepEqual(*tt.inputPtr, tt.want) {
				t.Errorf("Sort() inputPtr = %v, want %v", *tt.inputPtr, tt.want)
			}

		})
	}
}

func TestCompareLeftRightToRightLeft(t *testing.T) {

	tests := []struct {
		name        string
		compareFrom CompareResult
		compareTo   CompareResult
		want        int
	}{
		{
			name:        "Equal",
			compareFrom: CompareResult{1, 1, 1},
			compareTo:   CompareResult{1, 1, 1},
			want:        0,
		},
		{
			name:        "Equal, different weight",
			compareFrom: CompareResult{1, 1, 1},
			compareTo:   CompareResult{1, 1, 2},
			want:        0,
		},
		{
			name:        "left is smaller than inverted Left (so right)",
			compareFrom: CompareResult{1, 5, 1},
			compareTo:   CompareResult{5, 2, 2},
			want:        -2,
		},
		{
			name:        "left is smaller than inverted Left (so right), and right is bigger that inverted right (so left)",
			compareFrom: CompareResult{1, 10, 1},
			compareTo:   CompareResult{5, 2, 2},
			want:        -2,
		},
		{
			name:        "left is bigger than inverted Left (so right)",
			compareFrom: CompareResult{1, 5, 1},
			compareTo:   CompareResult{5, 0, 2},
			want:        2,
		},
		{
			name:        "right is smaller than inverted right (so left)",
			compareFrom: CompareResult{5, 1, 1},
			compareTo:   CompareResult{2, 5, 2},
			want:        -1,
		},
		{
			name:        "right is bigger than inverted right (so left)",
			compareFrom: CompareResult{5, 1, 1},
			compareTo:   CompareResult{0, 5, 2},
			want:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, compareLeftRightResult(tt.compareFrom, tt.compareTo), tt.want)

		})
	}
}

func TestCompareCompareResults(t *testing.T) {

	tests := []struct {
		name        string
		compareFrom CompareResult
		compareTo   CompareResult
		want        int
	}{
		{
			name:        "Equal",
			compareFrom: CompareResult{1, 1, 1},
			compareTo:   CompareResult{1, 1, 1},
			want:        0,
		},
		{
			name:        "Equal, different weight",
			compareFrom: CompareResult{1, 1, 1},
			compareTo:   CompareResult{1, 1, 2},
			want:        0,
		},
		{
			name:        "left is smaller than left)",
			compareFrom: CompareResult{1, 5, 1},
			compareTo:   CompareResult{2, 5, 2},
			want:        -2,
		},
		{
			name:        "left is smaller than left), and right is bigger",
			compareFrom: CompareResult{1, 5, 1},
			compareTo:   CompareResult{2, 10, 2},
			want:        -2,
		},
		{
			name:        "left is bigger than Left",
			compareFrom: CompareResult{1, 5, 1},
			compareTo:   CompareResult{0, 5, 2},
			want:        2,
		},
		{
			name:        "right is smaller right",
			compareFrom: CompareResult{5, 1, 1},
			compareTo:   CompareResult{5, 2, 2},
			want:        -1,
		},
		{
			name:        "right is bigger than right",
			compareFrom: CompareResult{5, 1, 1},
			compareTo:   CompareResult{5, 0, 2},
			want:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, compareCompareResults(tt.compareFrom, tt.compareTo), tt.want)

		})
	}
}

func TestExtractUniqueTopMatch(t *testing.T) {

	tests := []struct {
		name  string
		input IndexCompareResultList
		want  []CompareResult
	}{
		{
			name: "extractUniqueTopMatch",
			input: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{1, 3, 1},
					CompareResult{2, 1, 1},
					CompareResult{2, 2, 1},
					CompareResult{3, 4, 1},
					CompareResult{5, 6, 1},
					CompareResult{8, 3, 1},
					CompareResult{10, 12, 1},
				},
			},
			want: []CompareResult{
				CompareResult{3, 4, 1},
				CompareResult{5, 6, 1},
				CompareResult{10, 12, 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUniqueTopMatch(&tt.input)

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("extractUniqueTopMatch() result = %v, want %v", result, tt.want)
			}

		})
	}
}

func TestGetIds(t *testing.T) {

	tests := []struct {
		name        string
		compareFrom CompareResult
		wantLeftId  int
		wantRightId int
	}{
		{
			name:        "get left and right Ids: All 1s",
			compareFrom: CompareResult{1, 1, 1},
			wantLeftId:  1,
			wantRightId: 1,
		},
		{
			name:        "get left and right Ids: 1, 2",
			compareFrom: CompareResult{1, 2, 1},
			wantLeftId:  1,
			wantRightId: 2,
		},
		{
			name:        "get left and right Ids: 4, 3",
			compareFrom: CompareResult{4, 3, 1},
			wantLeftId:  4,
			wantRightId: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, getLeftId(tt.compareFrom), tt.wantLeftId)
			assert.Equal(t, getRightId(tt.compareFrom), tt.wantRightId)

		})
	}
}
