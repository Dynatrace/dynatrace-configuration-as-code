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
	"gotest.tools/assert"
	"reflect"
	"testing"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
)

func TestNewMatchProcessing(t *testing.T) {

	tests := []struct {
		name               string
		rawMatchListSource RawMatchList
		sourceType         config.Type
		rawMatchListTarget RawMatchList
		targetType         config.Type
		want               MatchProcessing
	}{
		{
			name:               "NewMatchProcessing",
			rawMatchListSource: getRawMatchListFromJson(entityListJsonSorted),
			sourceType: config.Type{
				EntitiesType: "HOST",
				From:         "1",
				To:           "2",
			},
			rawMatchListTarget: getRawMatchListFromJson(entityJsonSortedValueString),
			targetType: config.Type{
				EntitiesType: "HOST",
				From:         "2",
				To:           "3",
			},
			want: MatchProcessing{
				Source: MatchProcessingEnv{
					RawMatchList: getRawMatchListFromJson(entityListJsonSorted),
					ConfigType: config.Type{
						EntitiesType: "HOST",
						From:         "1",
						To:           "2",
					},
					RemainingMatch: []int{0, 1, 2},
				},
				Target: MatchProcessingEnv{
					RawMatchList: getRawMatchListFromJson(entityJsonSortedValueString),
					ConfigType: config.Type{
						EntitiesType: "HOST",
						From:         "2",
						To:           "3",
					},
					RemainingMatch: []int{0},
				},
				matchedMap: map[int]int{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMatchProcessing(tt.rawMatchListSource, tt.sourceType, tt.rawMatchListTarget, tt.targetType)

			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("NewMatchProcessing() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestGenremainingMatchList(t *testing.T) {

	tests := []struct {
		name         string
		rawMatchList RawMatchList
		want         []int
	}{
		{
			name:         "genremainingMatchList",
			rawMatchList: getRawMatchListFromJson(entityListJsonSorted),
			want:         []int{0, 1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := genremainingMatchList(tt.rawMatchList)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genremainingMatchList() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestGetEntitiesType(t *testing.T) {

	tests := []struct {
		name            string
		matchProcessing MatchProcessing
		want            string
	}{
		{
			name: "GetEntitiesType - Both",
			matchProcessing: MatchProcessing{
				Source: MatchProcessingEnv{
					RawMatchList: getRawMatchListFromJson(entityListJsonSorted),
					ConfigType: config.Type{
						EntitiesType: "HOST",
						From:         "1",
						To:           "2",
					},
					RemainingMatch: []int{0, 1, 2},
				},
				Target: MatchProcessingEnv{
					RawMatchList: getRawMatchListFromJson(entityJsonSortedValueString),
					ConfigType: config.Type{
						EntitiesType: "HOST",
						From:         "2",
						To:           "3",
					},
					RemainingMatch: []int{0},
				},
				matchedMap: map[int]int{},
			},
			want: "HOST",
		},
		{
			name: "GetEntitiesType - Target Only",
			matchProcessing: MatchProcessing{
				Source: MatchProcessingEnv{},
				Target: MatchProcessingEnv{
					RawMatchList: getRawMatchListFromJson(entityJsonSortedValueString),
					ConfigType: config.Type{
						EntitiesType: "HOST",
						From:         "2",
						To:           "3",
					},
					RemainingMatch: []int{0},
				},
				matchedMap: map[int]int{},
			},
			want: "HOST",
		},
		{
			name: "GetEntitiesType - Source Only",
			matchProcessing: MatchProcessing{
				Source: MatchProcessingEnv{
					RawMatchList: getRawMatchListFromJson(entityListJsonSorted),
					ConfigType: config.Type{
						EntitiesType: "HOST",
						From:         "1",
						To:           "2",
					},
					RemainingMatch: []int{0, 1, 2},
				},
				Target:     MatchProcessingEnv{},
				matchedMap: map[int]int{},
			},
			want: "HOST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.matchProcessing.GetEntitiesType()

			assert.Equal(t, got, tt.want)

		})
	}
}
