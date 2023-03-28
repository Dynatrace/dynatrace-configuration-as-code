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

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match/rules"
)

func TestCompareIndex(t *testing.T) {

	tests := []struct {
		name             string
		input            IndexCompareResultList
		indexEntrySource []IndexEntry
		indexEntryTarget []IndexEntry
		indexRule        rules.IndexRule
		want             IndexCompareResultList
	}{
		{
			name:  "Test Compare 3 to 3 = 9",
			input: IndexCompareResultList{},
			indexEntrySource: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
			},
			indexEntryTarget: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
			},
			indexRule: rules.IndexRule{
				Name:              "Detected Name",
				Path:              []string{"properties", "detectedName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
			want: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{1, 1, 1},
					CompareResult{1, 2, 1},
					CompareResult{1, 3, 1},
					CompareResult{2, 1, 1},
					CompareResult{2, 2, 1},
					CompareResult{2, 3, 1},
					CompareResult{3, 1, 1},
					CompareResult{3, 2, 1},
					CompareResult{3, 3, 1},
				},
			},
		},
		{
			name:  "Test Compare 3 to 3 = 9, plus orphan in Source",
			input: IndexCompareResultList{},
			indexEntrySource: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
				IndexEntry{"Orphan", []int{1, 2, 3}},
			},
			indexEntryTarget: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
			},
			indexRule: rules.IndexRule{
				Name:              "Detected Name",
				Path:              []string{"properties", "detectedName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
			want: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{1, 1, 1},
					CompareResult{1, 2, 1},
					CompareResult{1, 3, 1},
					CompareResult{2, 1, 1},
					CompareResult{2, 2, 1},
					CompareResult{2, 3, 1},
					CompareResult{3, 1, 1},
					CompareResult{3, 2, 1},
					CompareResult{3, 3, 1},
				},
			},
		},
		{
			name:  "Test Compare 3 to 3 = 9, plus orphan in Target",
			input: IndexCompareResultList{},
			indexEntrySource: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
			},
			indexEntryTarget: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
				IndexEntry{"Orphan", []int{1, 2, 3}},
			},
			indexRule: rules.IndexRule{
				Name:              "Detected Name",
				Path:              []string{"properties", "detectedName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
			want: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{1, 1, 1},
					CompareResult{1, 2, 1},
					CompareResult{1, 3, 1},
					CompareResult{2, 1, 1},
					CompareResult{2, 2, 1},
					CompareResult{2, 3, 1},
					CompareResult{3, 1, 1},
					CompareResult{3, 2, 1},
					CompareResult{3, 3, 1},
				},
			},
		},
		{
			name:  "Test Compare 3 to 3 = 9, plus orphans in Source and Target",
			input: IndexCompareResultList{},
			indexEntrySource: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
				IndexEntry{"Orphan1", []int{1, 2, 3}},
			},
			indexEntryTarget: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
				IndexEntry{"Orphan2", []int{1, 2, 3}},
			},
			indexRule: rules.IndexRule{
				Name:              "Detected Name",
				Path:              []string{"properties", "detectedName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
			want: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{1, 1, 1},
					CompareResult{1, 2, 1},
					CompareResult{1, 3, 1},
					CompareResult{2, 1, 1},
					CompareResult{2, 2, 1},
					CompareResult{2, 3, 1},
					CompareResult{3, 1, 1},
					CompareResult{3, 2, 1},
					CompareResult{3, 3, 1},
				},
			},
		},
		{
			name:  "Test Skip too many matches (32 x 32 = 1024, but max is 1000)",
			input: IndexCompareResultList{},
			indexEntrySource: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
				IndexEntry{"1000+ matches", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}},
			},
			indexEntryTarget: []IndexEntry{
				IndexEntry{"Test", []int{1, 2, 3}},
				IndexEntry{"1000+ matches", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}},
			},
			indexRule: rules.IndexRule{
				Name:              "Detected Name",
				Path:              []string{"properties", "detectedName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
			want: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{1, 1, 1},
					CompareResult{1, 2, 1},
					CompareResult{1, 3, 1},
					CompareResult{2, 1, 1},
					CompareResult{2, 2, 1},
					CompareResult{2, 3, 1},
					CompareResult{3, 1, 1},
					CompareResult{3, 2, 1},
					CompareResult{3, 3, 1},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareIndexes(&tt.input, tt.indexEntrySource, tt.indexEntryTarget, tt.indexRule)

			if !reflect.DeepEqual(tt.input.CompareResults, tt.want.CompareResults) {
				t.Errorf("compareIndexes() gotRemainingResultList = %v, want %v", tt.input.CompareResults, tt.want.CompareResults)
			}

		})
	}
}
