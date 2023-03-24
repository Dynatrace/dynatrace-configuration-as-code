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

import (
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

func MatchEntities(fs afero.Fs, matchParameters match.MatchParameters, entityPerTypeSource project.ConfigsPerType, entityPerTypeTarget project.ConfigsPerType) ([]string, int, int, error) {
	nbEntitiesSource := 0
	nbEntitiesTarget := 0
	stats := []string{fmt.Sprintf("%65s %10s %12s %10s %10s %10s", "Type", "Matched", "MultiMatched", "UnMatched", "Total", "Source")}

	for entitiesType := range entityPerTypeTarget {

		log.Debug("Processing Type: %s", entitiesType)

		entityProcessingPtr, err := genEntityProcessing(entityPerTypeSource, entityPerTypeTarget, entitiesType)
		if err != nil {
			return []string{}, 0, 0, err
		}
		nbEntitiesSourceType := len(entityProcessingPtr.Source.RemainingMatch)
		nbEntitiesSource += nbEntitiesSourceType
		nbEntitiesTargetType := len(entityProcessingPtr.Target.RemainingMatch)
		nbEntitiesTarget += nbEntitiesTargetType

		var output MatchOutputType
		output, err = runRules(entityProcessingPtr, matchParameters)
		if err != nil {
			return []string{}, 0, 0, err
		}

		err = writeMatches(fs, matchParameters, entitiesType, output)
		if err != nil {
			return []string{}, 0, 0, fmt.Errorf("failed to persist matches of type: %s, see error: %s", entitiesType, err)
		}

		stats = append(stats, fmt.Sprintf("%65s %10d %12d %10d %10d %10d", entitiesType, len(output.Matches), len(output.MultiMatched), len(output.UnMatched), nbEntitiesTargetType, nbEntitiesSourceType))
	}

	return stats, nbEntitiesSource, nbEntitiesTarget, nil
}
