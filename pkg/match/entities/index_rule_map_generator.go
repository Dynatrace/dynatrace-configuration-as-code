// @license
// Copyright 2021 Dynatrace LLC
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

import (
	"sort"
)

type IndexRuleType struct {
	isSeed      bool
	weightValue int
	indexRules  []IndexRule
}

type IndexRule struct {
	name              string
	path              []string
	weightValue       int
	selfMatchDisabled bool
}

// ByWeightTypeValue implements sort.Interface for []IndexRule based on
// the WeightTypeValue field.
type ByWeightTypeValue []IndexRuleType

func (a ByWeightTypeValue) Len() int           { return len(a) }
func (a ByWeightTypeValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeightTypeValue) Less(i, j int) bool { return a[j].weightValue < a[i].weightValue }

type IndexRuleMapGenerator struct {
	SelfMatch bool
}

var INDEX_CONFIG_LIST_ALL = []IndexRuleType{
	{
		isSeed:      true,
		weightValue: 100,
		indexRules: []IndexRule{
			{
				name:              "DetectedName",
				path:              []string{"properties", "detectedName"},
				weightValue:       1,
				selfMatchDisabled: false,
			},
			{
				name:              "oneAgentCustomHostName",
				path:              []string{"properties", "oneAgentCustomHostName"},
				weightValue:       1,
				selfMatchDisabled: false,
			},
		},
	},
	{
		isSeed:      true,
		weightValue: 90,
		indexRules: []IndexRule{
			{
				name:              "Entity Id",
				path:              []string{"entityId"},
				weightValue:       1,
				selfMatchDisabled: true,
			},
			{
				name:              "displayName",
				path:              []string{"displayName"},
				weightValue:       1,
				selfMatchDisabled: false,
			},
		},
	},
	// ipAddress was tested with isSeed = false on 5 million RBC Pre-Prod entities
	// All matches were identical, except for Network Interfaces the were not matching as well
	// Keeping isSeed = true only has positive return
	{
		isSeed:      true,
		weightValue: 50,
		indexRules: []IndexRule{
			{
				name:              "ipAddress",
				path:              []string{"properties", "ipAddress"},
				weightValue:       2,
				selfMatchDisabled: false,
			},
		},
	},
}

func newIndexRuleMapGenerator(selfMatch bool) *IndexRuleMapGenerator {
	i := new(IndexRuleMapGenerator)
	i.SelfMatch = selfMatch
	return i
}

func (i *IndexRuleMapGenerator) genActiveList() []IndexRuleType {

	activeList := make([]IndexRuleType, 0, len(INDEX_CONFIG_LIST_ALL))

	for _, confType := range INDEX_CONFIG_LIST_ALL {
		ruleType := IndexRuleType{
			isSeed:      confType.isSeed,
			weightValue: confType.weightValue,
			indexRules:  make([]IndexRule, 0, len(confType.indexRules)),
		}
		for _, conf := range confType.indexRules {
			if conf.selfMatchDisabled && i.SelfMatch {
				continue
			}
			ruleType.indexRules = append(ruleType.indexRules, conf)
		}
		if len(ruleType.indexRules) >= 1 {
			activeList = append(activeList, ruleType)
		}
	}

	return activeList
}

func (i *IndexRuleMapGenerator) genSortedActiveList() []IndexRuleType {

	activeList := i.genActiveList()

	sort.Sort(ByWeightTypeValue(activeList))

	return activeList
}
