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

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
)

var testRules = []IndexRuleType{
	{
		IsSeed:      true,
		WeightValue: 80,
		IndexRules: []IndexRule{
			{
				Name:              "Test self-match alone",
				Path:              []string{"test"},
				WeightValue:       1,
				SelfMatchDisabled: true,
			},
		},
	},
	{
		IsSeed:      true,
		WeightValue: 90,
		IndexRules: []IndexRule{
			{
				Name:              "Entity Id",
				Path:              []string{"entityId"},
				WeightValue:       1,
				SelfMatchDisabled: true,
			},
			{
				Name:              "Display Name",
				Path:              []string{"displayName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
		},
	},
	{
		IsSeed:      true,
		WeightValue: 100,
		IndexRules: []IndexRule{
			{
				Name:              "Detected Name",
				Path:              []string{"properties", "detectedName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
		},
	},
}

var testGenerator = IndexRuleMapGenerator{
	SelfMatch:    false,
	baseRuleList: testRules,
}

func TestNewIndexRuleMapGenerator(t *testing.T) {

	tests := []struct {
		name      string
		selfMatch bool
		ruleList  []IndexRuleType
		want      IndexRuleMapGenerator
	}{
		{
			name:      "newIndexRuleMap",
			selfMatch: false,
			ruleList:  testRules,
			want:      testGenerator,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewIndexRuleMapGenerator(tt.selfMatch, tt.ruleList)

			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("NewIndexRuleMapGenerator() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestGenActiveList(t *testing.T) {

	tests := []struct {
		name      string
		selfMatch bool
		ruleList  []IndexRuleType
		want      []IndexRuleType
	}{
		{
			name:      "genActiveList - not self match",
			selfMatch: false,
			ruleList:  testRules,
			want:      testRules,
		},
		{
			name:      "genActiveList - self match",
			selfMatch: true,
			ruleList:  testRules,
			want: []IndexRuleType{
				{
					IsSeed:      true,
					WeightValue: 90,
					IndexRules: []IndexRule{
						{
							Name:              "Display Name",
							Path:              []string{"displayName"},
							WeightValue:       1,
							SelfMatchDisabled: false,
						},
					},
				},
				{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules: []IndexRule{
						{
							Name:              "Detected Name",
							Path:              []string{"properties", "detectedName"},
							WeightValue:       1,
							SelfMatchDisabled: false,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewIndexRuleMapGenerator(tt.selfMatch, tt.ruleList)
			got := generator.genActiveList()

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genActiveList() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestGenSortedActiveList(t *testing.T) {

	tests := []struct {
		name      string
		selfMatch bool
		ruleList  []IndexRuleType
		want      []IndexRuleType
	}{
		{
			name:      "genSortedActiveList - not self match",
			selfMatch: false,
			ruleList:  testRules,
			want: []IndexRuleType{
				{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules: []IndexRule{
						{
							Name:              "Detected Name",
							Path:              []string{"properties", "detectedName"},
							WeightValue:       1,
							SelfMatchDisabled: false,
						},
					},
				},
				{
					IsSeed:      true,
					WeightValue: 90,
					IndexRules: []IndexRule{
						{
							Name:              "Entity Id",
							Path:              []string{"entityId"},
							WeightValue:       1,
							SelfMatchDisabled: true,
						},
						{
							Name:              "Display Name",
							Path:              []string{"displayName"},
							WeightValue:       1,
							SelfMatchDisabled: false,
						},
					},
				},
				{
					IsSeed:      true,
					WeightValue: 80,
					IndexRules: []IndexRule{
						{
							Name:              "Test self-match alone",
							Path:              []string{"test"},
							WeightValue:       1,
							SelfMatchDisabled: true,
						},
					},
				},
			},
		},
		{
			name:      "genSortedActiveList - self match",
			selfMatch: true,
			ruleList:  testRules,
			want: []IndexRuleType{
				{
					IsSeed:      true,
					WeightValue: 100,
					IndexRules: []IndexRule{
						{
							Name:              "Detected Name",
							Path:              []string{"properties", "detectedName"},
							WeightValue:       1,
							SelfMatchDisabled: false,
						},
					},
				},
				{
					IsSeed:      true,
					WeightValue: 90,
					IndexRules: []IndexRule{
						{
							Name:              "Display Name",
							Path:              []string{"displayName"},
							WeightValue:       1,
							SelfMatchDisabled: false,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewIndexRuleMapGenerator(tt.selfMatch, tt.ruleList)
			got := generator.genSortedActiveList()

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genSortedActiveList() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestRunIndexRule(t *testing.T) {

	tests := []struct {
		name             string
		rule             IndexRule
		entityProcessing MatchProcessing
		resultList       IndexCompareResultList
		want             IndexCompareResultList
	}{
		{
			name: "runIndexRule",
			rule: IndexRule{
				Name:              "Detected Name",
				Path:              []string{"displayName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
			entityProcessing: MatchProcessing{
				Source: MatchProcessingEnv{
					RawMatchList:           getRawMatchListFromJson(entityListJsonSorted),
					ConfigType:             config.Type{},
					CurrentRemainingMatch:  &[]int{0, 1, 2},
					RemainingMatch:         []int{},
					remainingMatchSeeded:   []int{},
					remainingMatchUnSeeded: []int{},
				},
				Target: MatchProcessingEnv{
					RawMatchList:           getRawMatchListFromJson(entityListJsonSorted),
					ConfigType:             config.Type{},
					CurrentRemainingMatch:  &[]int{0, 1, 2},
					RemainingMatch:         []int{},
					remainingMatchSeeded:   []int{},
					remainingMatchUnSeeded: []int{},
				},
				matchedMap: map[int]int{},
			},
			resultList: IndexCompareResultList{
				CompareResults: []CompareResult{},
			},
			want: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{0, 0, 1},
					CompareResult{1, 1, 1},
					CompareResult{2, 2, 1},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rule.runIndexRule(&tt.entityProcessing, &tt.resultList)

			if !reflect.DeepEqual(tt.resultList, tt.want) {
				t.Errorf("runIndexRule() got = %v, want %v", tt.resultList, tt.want)
			}

		})
	}
}

func TestKeepMatches(t *testing.T) {

	tests := []struct {
		name                string
		matchedEntities     map[int]int
		singleToSingleMatch []CompareResult
		want                map[int]int
	}{
		{
			name:            "keepMatches",
			matchedEntities: map[int]int{},
			singleToSingleMatch: []CompareResult{
				CompareResult{0, 0, 1},
				CompareResult{1, 1, 1},
				CompareResult{2, 2, 1},
			},
			want: map[int]int{
				0: 0,
				1: 1,
				2: 2,
			},
		},
		{
			name: "keepMatches - existing results",
			matchedEntities: map[int]int{
				40: 40,
				41: 41,
				42: 42,
			},
			singleToSingleMatch: []CompareResult{
				CompareResult{0, 0, 1},
				CompareResult{1, 1, 1},
				CompareResult{2, 2, 1},
			},
			want: map[int]int{
				0:  0,
				1:  1,
				2:  2,
				40: 40,
				41: 41,
				42: 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keepMatches(tt.matchedEntities, tt.singleToSingleMatch)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("keepMatches() got = %v, want %v", got, tt.want)
			}

		})
	}
}

var entityListJsonSortedTopMatch = `[{
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
	"entityId": "AZURE_VM-2BBAEC9A7D2-TEST",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-2BBAEC9A7D21833E",
	"firstSeenTms": 1662997868374,
	"lastSeenTms": 1674246568646,
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

var entityListJsonSortedMultiMatch = `[{
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
}, {
	"entityId": "AZURE_VM-2BBAEC9A7D21833E",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-2BBAEC9A7D21833E",
	"firstSeenTms": 1642997868374,
	"lastSeenTms": 1654246568646,
	"properties": {},
	"toRelationships": {}
}]`

func TestRunIndexRuleAll(t *testing.T) {

	tests := []struct {
		name                    string
		indexRuleMapGenerator   IndexRuleMapGenerator
		matchProcessing         *MatchProcessing
		wantRemainingResultList IndexCompareResultList
		wantMatchedEntities     map[int]int
	}{
		{
			name:                  "RunIndexRuleAll",
			indexRuleMapGenerator: testGenerator,
			matchProcessing: NewMatchProcessing(
				getRawMatchListFromJson(entityListJsonSorted),
				config.Type{},
				getRawMatchListFromJson(entityListJsonSorted),
				config.Type{},
			),
			wantRemainingResultList: IndexCompareResultList{
				CompareResults: []CompareResult{},
			},
			wantMatchedEntities: map[int]int{
				0: 0,
				1: 1,
				2: 2,
			},
		},
		{
			name:                  "RunIndexRuleAll - keep only top match",
			indexRuleMapGenerator: testGenerator,
			matchProcessing: NewMatchProcessing(
				getRawMatchListFromJson(entityListJsonSorted),
				config.Type{},
				getRawMatchListFromJson(entityListJsonSortedTopMatch),
				config.Type{},
			),
			wantRemainingResultList: IndexCompareResultList{
				CompareResults: []CompareResult{},
			},
			wantMatchedEntities: map[int]int{
				0: 0,
				1: 1,
				2: 3,
			},
		},
		{
			name:                  "RunIndexRuleAll - multi-match remaining",
			indexRuleMapGenerator: testGenerator,
			matchProcessing: NewMatchProcessing(
				getRawMatchListFromJson(entityListJsonSorted),
				config.Type{},
				getRawMatchListFromJson(entityListJsonSortedMultiMatch),
				config.Type{},
			),
			wantRemainingResultList: IndexCompareResultList{
				CompareResults: []CompareResult{
					CompareResult{2, 2, 2},
					CompareResult{2, 3, 2},
				},
			},
			wantMatchedEntities: map[int]int{
				0: 0,
				1: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRemainingResultList, gotMatchedEntities := tt.indexRuleMapGenerator.RunIndexRuleAll(tt.matchProcessing)

			if !reflect.DeepEqual(*gotRemainingResultList, tt.wantRemainingResultList) {
				t.Errorf("RunIndexRuleAll() gotRemainingResultList = %v, want %v", *gotRemainingResultList, tt.wantRemainingResultList)
			}
			if !reflect.DeepEqual(*gotMatchedEntities, tt.wantMatchedEntities) {
				t.Errorf("RunIndexRuleAll() gotMatchedEntities = %v, want %v", *gotMatchedEntities, tt.wantMatchedEntities)
			}

		})
	}
}
