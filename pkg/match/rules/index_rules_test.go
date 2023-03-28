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

package rules

import (
	"reflect"
	"testing"
)

var TEST_CONFIG_LIST = []IndexRuleType{
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
			{
				Name:              "One Agent Custom Host Name",
				Path:              []string{"properties", "oneAgentCustomHostName"},
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
	// ipAddress was tested with IsSeed = false on 5 million RBC Pre-Prod entities
	// All matches were identical, except for Network Interfaces the were not matching as well
	// Keeping IsSeed = true only has positive return
	{
		IsSeed:      true,
		WeightValue: 50,
		IndexRules: []IndexRule{
			{
				Name:              "Ip Addresses List",
				Path:              []string{"properties", "ipAddress"},
				WeightValue:       2,
				SelfMatchDisabled: false,
			},
		},
	},
}

func TestGenExtraFieldsL2(t *testing.T) {

	tests := []struct {
		name      string
		ruleTypes []IndexRuleType
		want      map[string][]string
	}{
		{
			name:      "GenExtraFieldsL2",
			ruleTypes: TEST_CONFIG_LIST,
			want: map[string][]string{
				"properties": []string{
					"detectedName",
					"oneAgentCustomHostName",
					"ipAddress",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenExtraFieldsL2(tt.ruleTypes)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenExtraFieldsL2() got = %v, want %v", got, tt.want)
			}

		})
	}
}
