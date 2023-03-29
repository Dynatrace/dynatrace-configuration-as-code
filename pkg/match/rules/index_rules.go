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

package rules

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/slices"
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

func GenExtraFieldsL2(ruleList []IndexRuleType) map[string][]string {

	extraFields := map[string][]string{}
	depthL2 := 2

	for _, confType := range ruleList {

		for _, conf := range confType.IndexRules {

			if len(conf.Path) != depthL2 {
				continue
			}

			key := conf.Path[0]
			value := conf.Path[1]

			_, found := extraFields[key]

			if !found {
				extraFields[key] = []string{}
			}

			if !slices.Contains(extraFields[key], value) {
				extraFields[key] = append(extraFields[key], value)
			}
		}
	}

	return extraFields
}
