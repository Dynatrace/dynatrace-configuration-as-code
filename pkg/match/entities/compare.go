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
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"
)

func CompareIndexes(resultListPtr *IndexCompareResultList, indexSource []IndexEntry, indexTarget []IndexEntry, indexRule IndexRule) {

	srcI := 0
	tgtI := 0

	for srcI < len(indexSource) && tgtI < len(indexTarget) {
		diff := strings.Compare(indexSource[srcI].IndexValue, indexTarget[tgtI].IndexValue)

		if diff < 0 {
			srcI++

		} else if diff == 0 {
			totalMatches := len(indexSource[srcI].MatchedIds) * len(indexTarget[tgtI].MatchedIds)
			if totalMatches > 1000 {
				log.Debug("too many matches for: %s, Nb of matches: %d", indexSource[srcI].IndexValue, totalMatches)
			} else {
				for _, entityIdSource := range indexSource[srcI].MatchedIds {
					for _, entityIdTarget := range indexTarget[tgtI].MatchedIds {
						(*resultListPtr).AddResult(entityIdSource, entityIdTarget, indexRule.WeightValue)
					}
				}

			}

			srcI++
			tgtI++

		} else {
			tgtI++

		}
	}

}
