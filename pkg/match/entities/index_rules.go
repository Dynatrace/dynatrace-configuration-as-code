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

package entities

import "github.com/dynatrace/dynatrace-configuration-as-code/pkg/match"

var INDEX_CONFIG_LIST_ENTITIES = []match.IndexRuleType{
	{
		IsSeed:      true,
		WeightValue: 100,
		IndexRules: []match.IndexRule{
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
		IndexRules: []match.IndexRule{
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
		IndexRules: []match.IndexRule{
			{
				Name:              "Ip Addresses List",
				Path:              []string{"properties", "ipAddress"},
				WeightValue:       2,
				SelfMatchDisabled: false,
			},
		},
	},
}
