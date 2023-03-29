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

package entities

import (
	"reflect"
	"testing"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match"
)

func TestGenMultiMatchedMap(t *testing.T) {

	tests := []struct {
		name             string
		remainingResults match.IndexCompareResultList
		matchProcessing  match.MatchProcessing
		want             map[string][]string
	}{
		{
			name: "genMultiMatchedMap",
			remainingResults: match.IndexCompareResultList{
				CompareResults: []match.CompareResult{
					match.CompareResult{2, 2, 1},
					match.CompareResult{3, 2, 1},
				},
			},
			matchProcessing: *match.NewMatchProcessing(
				getRawMatchListFromJson(entityListJsonSortedMultiMatch),
				config.EntityType{
					EntitiesType: "AZURE_VM",
					From:         "1",
					To:           "2",
				},
				getRawMatchListFromJson(entityListJson),
				config.EntityType{
					EntitiesType: "AZURE_VM",
					From:         "2",
					To:           "3",
				},
			),
			want: map[string][]string{
				"AZURE_VM-2BBAEC9A7D21833A": []string{
					"AZURE_VM-2BBAEC9A7D21833E",
				},
				"AZURE_VM-2BBAEC9A7D21833B": []string{
					"AZURE_VM-2BBAEC9A7D21833E",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := genMultiMatchedMap(&tt.remainingResults, &tt.matchProcessing)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genMultiMatchedMap() got = %v, want %v", got, tt.want)
			}

		})
	}
}
