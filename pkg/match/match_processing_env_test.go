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

func TestGenSeededMatch(t *testing.T) {

	tests := []struct {
		name               string
		matchProcessingEnv MatchProcessingEnv
		resultList         []CompareResult
		getId              func(CompareResult) int
		want               MatchProcessingEnv
	}{
		{
			name: "genSeededMatch - Left (Source)",
			matchProcessingEnv: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
			resultList: []CompareResult{
				CompareResult{0, 2, 1},
				CompareResult{1, 3, 1},
			},
			getId: getLeftId,
			want: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
		},
		{
			name: "genSeededMatch - Right (Target)",
			matchProcessingEnv: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2, 3, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
			resultList: []CompareResult{
				CompareResult{0, 2, 1},
				CompareResult{1, 3, 1},
			},
			getId: getRightId,
			want: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{2, 3},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.matchProcessingEnv.genSeededMatch(&tt.resultList, tt.getId)

			if !reflect.DeepEqual(tt.matchProcessingEnv, tt.want) {
				t.Errorf("genSeededMatch() matchProcessingEnv = %v, want %v", tt.matchProcessingEnv, tt.want)
			}

		})
	}
}

func TestGenUnSeededMatch(t *testing.T) {

	tests := []struct {
		name               string
		matchProcessingEnv MatchProcessingEnv
		resultList         []CompareResult
		getId              func(CompareResult) int
		want               MatchProcessingEnv
	}{
		{
			name: "genUnSeededMatch - Left (Source)",
			matchProcessingEnv: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
			resultList: []CompareResult{
				CompareResult{0, 2, 1},
				CompareResult{1, 3, 1},
			},
			getId: getLeftId,
			want: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{2, 3, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
		},
		{
			name: "genUnSeededMatch - Right (Target)",
			matchProcessingEnv: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2, 3, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
			resultList: []CompareResult{
				CompareResult{0, 2, 1},
				CompareResult{1, 3, 1},
			},
			getId: getRightId,
			want: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.matchProcessingEnv.genUnSeededMatch(&tt.resultList, tt.getId)

			if !reflect.DeepEqual(tt.matchProcessingEnv, tt.want) {
				t.Errorf("genUnSeededMatch() matchProcessingEnv = %v, want %v", tt.matchProcessingEnv, tt.want)
			}

		})
	}
}

func TestTrimremainingItems(t *testing.T) {

	tests := []struct {
		name               string
		matchProcessingEnv MatchProcessingEnv
		idsToDrop          []int
		want               MatchProcessingEnv
	}{
		{
			name: "trimremainingItems - Left (Source)",
			matchProcessingEnv: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2, 3, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
			idsToDrop: []int{0, 1},
			want: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{2, 3, 4, 5, 6},
				RemainingMatch:        []int{2, 3, 4, 5, 6},
			},
		},
		{
			name: "trimremainingItems - Right (Target)",
			matchProcessingEnv: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2, 3, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
			idsToDrop: []int{2, 3},
			want: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 4, 5, 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.matchProcessingEnv.trimremainingItems(&tt.idsToDrop)

			if !reflect.DeepEqual(tt.matchProcessingEnv, tt.want) {
				t.Errorf("trimremainingItems() matchProcessingEnv = %v, want %v", tt.matchProcessingEnv, tt.want)
			}

		})
	}
}

func TestReduceRemainingMatchList(t *testing.T) {

	tests := []struct {
		name               string
		matchProcessingEnv MatchProcessingEnv
		uniqueMatch        []CompareResult
		getId              func(CompareResult) int
		want               MatchProcessingEnv
	}{
		{
			name: "reduceRemainingMatchList - Left (Source)",
			matchProcessingEnv: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2, 3, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
			uniqueMatch: []CompareResult{
				CompareResult{0, 2, 1},
				CompareResult{1, 3, 1},
			},
			getId: getLeftId,
			want: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{2, 3, 4, 5, 6},
				RemainingMatch:        []int{2, 3, 4, 5, 6},
			},
		},
		{
			name: "reduceRemainingMatchList - Right (Target)",
			matchProcessingEnv: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 2, 3, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 2, 3, 4, 5, 6},
			},
			uniqueMatch: []CompareResult{
				CompareResult{0, 2, 1},
				CompareResult{1, 3, 1},
			},
			getId: getRightId,
			want: MatchProcessingEnv{
				RawMatchList:          getRawMatchListFromJson(entityListJsonSorted),
				ConfigType:            config.EntityType{},
				CurrentRemainingMatch: &[]int{0, 1, 4, 5, 6},
				RemainingMatch:        []int{0, 1, 4, 5, 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.matchProcessingEnv.reduceRemainingMatchList(&tt.uniqueMatch, tt.getId)

			if !reflect.DeepEqual(tt.matchProcessingEnv, tt.want) {
				t.Errorf("reduceRemainingMatchList() matchProcessingEnv = %v, want %v", tt.matchProcessingEnv, tt.want)
			}

		})
	}
}
