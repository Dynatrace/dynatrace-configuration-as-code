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
	IsSeed      bool
	WeightValue int
	IndexRules  []IndexRule
}

type IndexRule struct {
	Name              string
	Path              []string
	WeightValue       int
	SelfMatchDisabled bool
}

var INDEX_CONFIG_LIST_ALL = []IndexRuleType{
	{
		IsSeed:      true,
		WeightValue: 100,
		IndexRules: []IndexRule{
			{
				Name:              "DetectedName",
				Path:              []string{"properties", "detectedName"},
				WeightValue:       1,
				SelfMatchDisabled: false,
			},
			{
				Name:              "oneAgentCustomHostName",
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
				Name:              "displayName",
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
				Name:              "ipAddress",
				Path:              []string{"properties", "ipAddress"},
				WeightValue:       2,
				SelfMatchDisabled: false,
			},
		},
	},
}

// ByWeightTypeValue implements sort.Interface for []IndexRule based on
// the WeightTypeValue field.
type ByWeightTypeValue []IndexRuleType

func (a ByWeightTypeValue) Len() int           { return len(a) }
func (a ByWeightTypeValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeightTypeValue) Less(i, j int) bool { return a[j].WeightValue < a[i].WeightValue }

type ClassifiedIndexRule struct {
	WeightTypeValue int
	IndexRules      []IndexRule
}

type IndexRuleMapGenerator struct {
	SelfMatch bool
}

func NewIndexRuleMapGenerator(selfMatch bool) *IndexRuleMapGenerator {
	i := new(IndexRuleMapGenerator)
	i.SelfMatch = selfMatch
	return i
}

func (i *IndexRuleMapGenerator) genActiveList() []IndexRuleType {

	activeList := make([]IndexRuleType, 0, len(INDEX_CONFIG_LIST_ALL))

	for _, confType := range INDEX_CONFIG_LIST_ALL {
		ruleType := IndexRuleType{
			IsSeed:      confType.IsSeed,
			WeightValue: confType.WeightValue,
			IndexRules:  make([]IndexRule, 0, len(confType.IndexRules)),
		}
		for _, conf := range confType.IndexRules {
			if conf.SelfMatchDisabled && i.SelfMatch {
				continue
			}
			ruleType.IndexRules = append(ruleType.IndexRules, conf)
		}
		if len(ruleType.IndexRules) >= 1 {
			activeList = append(activeList, ruleType)
		}
	}

	return activeList
}

func (i *IndexRuleMapGenerator) GenSortedActiveList() []IndexRuleType {

	activeList := i.genActiveList()

	sort.Sort(ByWeightTypeValue(activeList))

	return activeList
}
