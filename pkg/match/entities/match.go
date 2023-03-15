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
	"encoding/json"
	"fmt"
	"path/filepath"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"
	"github.com/spf13/afero"
)

func CompareConfigs(fs afero.Fs, matchParameters MatchParameters, entityPerTypeSource project.ConfigsPerType, entityPerTypeTarget project.ConfigsPerType) ([]string, int, int, error) {
	nbEntitiesSource := 0
	nbEntitiesTarget := 0
	stats := []string{fmt.Sprintf("%65s %10s %10s %10s %10s %10s", "Type", "Matched", "MultiMatched", "UnMatched", "Total", "Source")}

	for entitiesType := range entityPerTypeTarget {

		log.Debug("Processing Type: %s", entitiesType)

		entityProcessingPtr, err := genEntityProcessing(entityPerTypeSource, entityPerTypeTarget, entitiesType)
		if err != nil {
			return []string{}, 0, 0, err
		}
		nbEntitiesSourceType := len(entityProcessingPtr.target.remainingEntities)
		nbEntitiesSource += nbEntitiesSourceType
		nbEntitiesTargetType := len(entityProcessingPtr.target.remainingEntities)
		nbEntitiesTarget += nbEntitiesTargetType

		var output MatchOutputType
		output, err = runIndexRuleAll(entityProcessingPtr, entitiesType, matchParameters)
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

func genEntityProcessing(entityPerTypeSource project.ConfigsPerType, entityPerTypeTarget project.ConfigsPerType, entitiesType string) (*EntityProcessing, error) {

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

	return newEntityProcessing(rawEntitiesSource, sourceType, rawEntitiesTarget, targetType), nil
}

func unmarshalEntities(entityPerType []config.Config) (*RawEntityList, error) {
	rawEntityList := new(RawEntityList)
	var err error = nil

	if len(entityPerType) > 0 {
		err = json.Unmarshal([]byte(entityPerType[0].Template.Content()), rawEntityList)
	}

	return rawEntityList, err
}

func runIndexRuleAll(entityProcessingPtr *EntityProcessing, entitiesType string, matchParameters MatchParameters) (MatchOutputType, error) {

	matchedEntities := map[int]int{}
	oldResultsPtr := &IndexCompareResultList{}

	sortedActiveIndexRuleTypes := newIndexRuleMapGenerator(matchParameters.SelfMatch).genSortedActiveList()

	log.Info("Type: %s -> nb source %d and nb target %d",
		entitiesType,
		len(*entityProcessingPtr.source.rawEntityListPtr), len(*entityProcessingPtr.target.rawEntityListPtr))

	for _, indexRuleType := range sortedActiveIndexRuleTypes {
		resultListPtr := newIndexCompareResultList(indexRuleType)
		entityProcessingPtr.prepareremainingEntities(true, indexRuleType.isSeed, oldResultsPtr)

		for _, indexRule := range indexRuleType.indexRules {
			runIndexRule(indexRule, entityProcessingPtr, resultListPtr)
		}

		resultListPtr.MergeOldWeightType(oldResultsPtr)
		singleToSingleMatchEntities := resultListPtr.processMatches()
		oldResultsPtr = resultListPtr

		entityProcessingPtr.adjustremainingEntities(singleToSingleMatchEntities, resultListPtr.compareResults)

		matchedEntities = keepMatches(matchedEntities, singleToSingleMatchEntities)
	}

	log.Info("Type: %s -> nb source %d and nb target %d -> Matched: %d",
		entitiesType, len(*entityProcessingPtr.source.rawEntityListPtr),
		len(*entityProcessingPtr.target.rawEntityListPtr), len(matchedEntities))

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

func genOutputPayload(entitiesType string, entityProcessingPtr *EntityProcessing, oldResultsPtr *IndexCompareResultList, matchedEntities map[int]int) MatchOutputType {

	multiMatchedMap := getMultiMatched(oldResultsPtr, entityProcessingPtr)
	entityProcessingPtr.prepareremainingEntities(false, true, oldResultsPtr)

	matchOutput := MatchOutputType{
		Type: entitiesType,
		MatchKey: MatchKey{
			Source: ExtractionInfo{
				From: entityProcessingPtr.source.configType.From,
				To:   entityProcessingPtr.source.configType.To,
			},
			Target: ExtractionInfo{
				From: entityProcessingPtr.target.configType.From,
				To:   entityProcessingPtr.target.configType.To,
			},
		},
		Matches:      make(map[string]string, len(matchedEntities)),
		MultiMatched: multiMatchedMap,
		UnMatched:    make([]string, len(*entityProcessingPtr.source.currentremainingEntities)),
	}

	for sourceI, targetI := range matchedEntities {
		matchOutput.Matches[(*(entityProcessingPtr.target.rawEntityListPtr))[targetI].(map[string]interface{})["entityId"].(string)] =
			(*(entityProcessingPtr.source.rawEntityListPtr))[sourceI].(map[string]interface{})["entityId"].(string)
	}

	for idx, sourceI := range *entityProcessingPtr.source.currentremainingEntities {
		matchOutput.UnMatched[idx] = (*(entityProcessingPtr.source.rawEntityListPtr))[sourceI].(map[string]interface{})["entityId"].(string)
	}

	return matchOutput
}

func getMultiMatched(oldResultsPtr *IndexCompareResultList, entityProcessingPtr *EntityProcessing) map[string][]string {
	printMultiMatchedSample(oldResultsPtr, entityProcessingPtr)

	return genMultiMatchedMap(oldResultsPtr, entityProcessingPtr)

}

func genMultiMatchedMap(oldResultsPtr *IndexCompareResultList, entityProcessingPtr *EntityProcessing) map[string][]string {

	multiMatched := map[string][]string{}

	if len(oldResultsPtr.compareResults) <= 0 {
		return multiMatched
	}

	firstIdx := 0
	currentId := oldResultsPtr.compareResults[0].leftId

	addMatchingMultiMatched := func(nbMatches int) {
		multiMatchedMatches := make([]string, nbMatches)
		for j := 0; j < nbMatches; j++ {
			compareResult := oldResultsPtr.compareResults[(j + firstIdx)]
			targetId := compareResult.rightId

			multiMatchedMatches[j] = (*(entityProcessingPtr.target.rawEntityListPtr))[targetId].(map[string]interface{})["entityId"].(string)
		}
		multiMatched[(*(entityProcessingPtr.source.rawEntityListPtr))[currentId].(map[string]interface{})["entityId"].(string)] = multiMatchedMatches
	}

	for i := 1; i < len(oldResultsPtr.compareResults); i++ {
		result := oldResultsPtr.compareResults[i]
		if result.leftId == currentId {

		} else {
			nbMatches := i - firstIdx
			addMatchingMultiMatched(nbMatches)

			currentId = result.leftId
			firstIdx = i
		}
	}
	nbMatches := len(oldResultsPtr.compareResults) - firstIdx
	addMatchingMultiMatched(nbMatches)

	return multiMatched

}

func printMultiMatchedSample(oldResultsPtr *IndexCompareResultList, entityProcessingPtr *EntityProcessing) {
	nbMultiMatched := len(oldResultsPtr.compareResults)

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
		result := oldResultsPtr.compareResults[i]
		log.Debug("Left: %v, Source: %v, Target: %v", result,
			(*entityProcessingPtr.source.rawEntityListPtr)[result.leftId],
			(*entityProcessingPtr.target.rawEntityListPtr)[result.rightId])
	}

}

func runIndexRule(indexRule IndexRule, entityProcessingPtr *EntityProcessing,
	resultListPtr *IndexCompareResultList) {

	sortedIndexSource := genSortedEntitiesIndex(indexRule, &(*entityProcessingPtr).source)
	sortedIndexTarget := genSortedEntitiesIndex(indexRule, &(*entityProcessingPtr).target)

	compareIndexes(resultListPtr, sortedIndexSource, sortedIndexTarget, indexRule)

}

func keepMatches(matchedEntities map[int]int, singleToSingleMatch []CompareResult) map[int]int {
	for _, result := range singleToSingleMatch {
		_, found := matchedEntities[result.leftId]

		if found {
			log.Error("Should never find multiple exact matches for an entity, %v", result)
		}

		matchedEntities[result.leftId] = result.rightId
	}

	return matchedEntities
}

func writeMatches(fs afero.Fs, matchParameters MatchParameters, entitiesType string, output MatchOutputType) error {

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

	sanitizedType := util.SanitizeName(entitiesType)
	fullMatchPath := filepath.Join(sanitizedOutputDir, fmt.Sprintf("%s.json", sanitizedType))

	err = afero.WriteFile(fs, fullMatchPath, outputAsJson, 0664)

	if err != nil {
		return err
	}

	return nil

}
