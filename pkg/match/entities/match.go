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
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

func CompareEntities(fs afero.Fs, matchParameters match.MatchParameters, entityPerTypeSource project.ConfigsPerType, entityPerTypeTarget project.ConfigsPerType) ([]string, int, int, error) {
	nbEntitiesSource := 0
	nbEntitiesTarget := 0
	stats := []string{fmt.Sprintf("%65s %10s %10s %10s %10s %10s", "Type", "Matched", "MultiMatched", "UnMatched", "Total", "Source")}

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
		output, err = runRules(entityProcessingPtr, entitiesType, matchParameters)
		if err != nil {
			return []string{}, 0, 0, err
		}

		err = writeMatches(fs, matchParameters, entitiesType, output)
		if err != nil {
			return []string{}, 0, 0, fmt.Errorf("failed to persist matches of type: %s, see error: %s", entitiesType, err)
		}

		stats = append(stats, fmt.Sprintf("%65s %10d %10d %10d %10d %10d", entitiesType, len(output.Matches), len(output.MultiMatched), len(output.UnMatched), nbEntitiesTargetType, nbEntitiesSourceType))
	}

	return stats, nbEntitiesSource, nbEntitiesTarget, nil
}

func genEntityProcessing(entityPerTypeSource project.ConfigsPerType, entityPerTypeTarget project.ConfigsPerType, entitiesType string) (*match.MatchProcessing, error) {

	rawEntitiesSource, err := unmarshalEntities(entityPerTypeSource[entitiesType])
	if err != nil {
		return nil, err
	}
	sourceType := config.Type{}
	if len(entityPerTypeSource[entitiesType]) > 0 {
		sourceType = entityPerTypeSource[entitiesType][0].Type
	}

	rawEntitiesTarget, err := unmarshalEntities(entityPerTypeTarget[entitiesType])
	if err != nil {
		return nil, err
	}
	targetType := config.Type{}
	if len(entityPerTypeTarget[entitiesType]) > 0 {
		targetType = entityPerTypeTarget[entitiesType][0].Type
	}

	return match.NewMatchProcessing(rawEntitiesSource, sourceType, rawEntitiesTarget, targetType), nil
}

func unmarshalEntities(entityPerType []config.Config) (*RawEntityList, error) {
	rawEntityList := &RawEntityList{
		Values: new([]interface{}),
	}
	var err error = nil

	if len(entityPerType) > 0 {
		err = json.Unmarshal([]byte(entityPerType[0].Template.Content()), rawEntityList.Values)
	}

	return rawEntityList, err
}

func runRules(entityProcessingPtr *match.MatchProcessing, entitiesType string, matchParameters match.MatchParameters) (MatchOutputType, error) {

	activeIndexRuleTypes := match.NewIndexRuleMapGenerator(matchParameters.SelfMatch, INDEX_CONFIG_LIST_ENTITIES)

	oldResultsPtr, matchedEntities := activeIndexRuleTypes.RunIndexRuleAll(entitiesType, entityProcessingPtr)

	outputPayload := genOutputPayload(entitiesType, entityProcessingPtr, oldResultsPtr, matchedEntities)

	return outputPayload, nil

}

type MatchOutputType struct {
	Type         string              `json:"type"`
	MatchKey     MatchKey            `json:"matchKey"`
	Matches      map[string]string   `json:"matches"`
	MultiMatched map[string][]string `json:"multiMatched"`
	UnMatched    []string            `json:"unmatched"`
}

type MatchKey struct {
	Source ExtractionInfo `json:"source"`
	Target ExtractionInfo `json:"target"`
}

type ExtractionInfo struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func genOutputPayload(entitiesType string, entityProcessingPtr *match.MatchProcessing, oldResultsPtr *match.IndexCompareResultList, matchedEntities *map[int]int) MatchOutputType {

	multiMatchedMap := getMultiMatched(oldResultsPtr, entityProcessingPtr)
	entityProcessingPtr.PrepareRemainingMatch(false, true, oldResultsPtr)

	matchOutput := MatchOutputType{
		Type: entitiesType,
		MatchKey: MatchKey{
			Source: ExtractionInfo{
				From: entityProcessingPtr.Source.ConfigType.From,
				To:   entityProcessingPtr.Source.ConfigType.To,
			},
			Target: ExtractionInfo{
				From: entityProcessingPtr.Target.ConfigType.From,
				To:   entityProcessingPtr.Target.ConfigType.To,
			},
		},
		Matches:      make(map[string]string, len(*matchedEntities)),
		MultiMatched: multiMatchedMap,
		UnMatched:    make([]string, len(*entityProcessingPtr.Source.CurrentremainingMatch)),
	}

	for sourceI, targetI := range *matchedEntities {
		matchOutput.Matches[(*entityProcessingPtr.Target.RawMatchList.GetValues())[targetI].(map[string]interface{})["entityId"].(string)] =
			(*entityProcessingPtr.Source.RawMatchList.GetValues())[sourceI].(map[string]interface{})["entityId"].(string)
	}

	for idx, sourceI := range *entityProcessingPtr.Source.CurrentremainingMatch {
		matchOutput.UnMatched[idx] = (*entityProcessingPtr.Source.RawMatchList.GetValues())[sourceI].(map[string]interface{})["entityId"].(string)
	}

	return matchOutput
}

func getMultiMatched(oldResultsPtr *match.IndexCompareResultList, entityProcessingPtr *match.MatchProcessing) map[string][]string {
	printMultiMatchedSample(oldResultsPtr, entityProcessingPtr)

	return genMultiMatchedMap(oldResultsPtr, entityProcessingPtr)

}

func genMultiMatchedMap(oldResultsPtr *match.IndexCompareResultList, entityProcessingPtr *match.MatchProcessing) map[string][]string {

	multiMatched := map[string][]string{}

	if len(oldResultsPtr.CompareResults) <= 0 {
		return multiMatched
	}

	firstIdx := 0
	currentId := oldResultsPtr.CompareResults[0].LeftId

	addMatchingMultiMatched := func(nbMatches int) {
		multiMatchedMatches := make([]string, nbMatches)
		for j := 0; j < nbMatches; j++ {
			compareResult := oldResultsPtr.CompareResults[(j + firstIdx)]
			targetId := compareResult.RightId

			multiMatchedMatches[j] = (*entityProcessingPtr.Target.RawMatchList.GetValues())[targetId].(map[string]interface{})["entityId"].(string)
		}
		multiMatched[(*entityProcessingPtr.Source.RawMatchList.GetValues())[currentId].(map[string]interface{})["entityId"].(string)] = multiMatchedMatches
	}

	for i := 1; i < len(oldResultsPtr.CompareResults); i++ {
		result := oldResultsPtr.CompareResults[i]
		if result.LeftId == currentId {

		} else {
			nbMatches := i - firstIdx
			addMatchingMultiMatched(nbMatches)

			currentId = result.LeftId
			firstIdx = i
		}
	}
	nbMatches := len(oldResultsPtr.CompareResults) - firstIdx
	addMatchingMultiMatched(nbMatches)

	return multiMatched

}

func printMultiMatchedSample(oldResultsPtr *match.IndexCompareResultList, entityProcessingPtr *match.MatchProcessing) {
	nbMultiMatched := len(oldResultsPtr.CompareResults)

	if nbMultiMatched <= 0 {
		return
	}

	var maxPrint int
	if nbMultiMatched > 10 {
		maxPrint = 10
	} else {
		maxPrint = nbMultiMatched
	}

	for i := 0; i < maxPrint; i++ {
		result := oldResultsPtr.CompareResults[i]
		log.Debug("Left: %v, Source: %v, Target: %v", result,
			(*entityProcessingPtr.Source.RawMatchList.GetValues())[result.LeftId],
			(*entityProcessingPtr.Target.RawMatchList.GetValues())[result.RightId])
	}

}

func writeMatches(fs afero.Fs, matchParameters match.MatchParameters, entitiesType string, output MatchOutputType) error {

	sanitizedOutputDir := filepath.Clean(matchParameters.OutputDir)

	if sanitizedOutputDir != "." {
		err := fs.MkdirAll(sanitizedOutputDir, 0777)
		if err != nil {
			return err
		}
	}

	outputAsJson, err := json.Marshal(output)

	if err != nil {
		return err
	}

	sanitizedType := config.Sanitize(entitiesType)
	fullMatchPath := filepath.Join(sanitizedOutputDir, fmt.Sprintf("%s.json", sanitizedType))

	err = afero.WriteFile(fs, fullMatchPath, outputAsJson, 0664)

	if err != nil {
		return err
	}

	return nil

}
