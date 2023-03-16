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
