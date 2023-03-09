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
	stats := []string{fmt.Sprintf("%65s %10s %10s %10s %10s %10s", "Type", "Matched", "LeftOvers", "UnMatched", "Total", "Source")}

	for entitiesType := range entityPerTypeTarget {

		log.Debug("Processing Type: %s", entitiesType)

		entityProcessingPtr, err := genEntityProcessing(entityPerTypeSource, entityPerTypeTarget, entitiesType)
		if err != nil {
			return []string{}, 0, 0, err
		}
		nbEntitiesSourceType := len(entityProcessingPtr.Target.RemainingEntities)
		nbEntitiesSource += nbEntitiesSourceType
		nbEntitiesTargetType := len(entityProcessingPtr.Target.RemainingEntities)
		nbEntitiesTarget += nbEntitiesTargetType

		var output MatchOutputType
		output, err = runIndexRuleAll(entityProcessingPtr, entitiesType)
		if err != nil {
			return []string{}, 0, 0, err
		}

		err = writeMatches(fs, matchParameters, entitiesType, output)
		if err != nil {
			return []string{}, 0, 0, fmt.Errorf("failed to persist matches of type: %s, see error: %s", entitiesType, err)
		}

		stats = append(stats, fmt.Sprintf("%65s %10d %10d %10d %10d %10d", entitiesType, len(output.Matches), len(output.LeftOvers), len(output.UnMatched), nbEntitiesTargetType, nbEntitiesSourceType))
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

	return NewEntityProcessing(rawEntitiesSource, sourceType, rawEntitiesTarget, targetType), nil
}

func unmarshalEntities(entityPerType []config.Config) (*RawEntityList, error) {
	rawEntityList := new(RawEntityList)
	err := json.Unmarshal([]byte(entityPerType[0].Template.Content()), rawEntityList)

	return rawEntityList, err
}

func runIndexRuleAll(entityProcessingPtr *EntityProcessing, entitiesType string) (MatchOutputType, error) {

	matchedEntities := map[int]int{}
	oldResultsPtr := &IndexCompareResultList{}

	sortedActiveIndexRuleTypes := NewIndexRuleMapGenerator(false).GenSortedActiveList()

	for _, indexRuleType := range sortedActiveIndexRuleTypes {
		resultListPtr := NewIndexCompareResultList(indexRuleType)
		entityProcessingPtr.PrepareRemainingEntities(true, indexRuleType.IsSeed, oldResultsPtr)

		for _, indexRule := range indexRuleType.IndexRules {
			runIndexRule(indexRule, entityProcessingPtr, resultListPtr)
		}

		resultListPtr.MergeOldWeightType(oldResultsPtr)
		singleToSingleMatchEntities := resultListPtr.ProcessMatches()
		oldResultsPtr = resultListPtr

		entityProcessingPtr.AdjustRemainingEntities(singleToSingleMatchEntities, resultListPtr.CompareResults)

		matchedEntities = keepMatches(matchedEntities, singleToSingleMatchEntities)
	}

	log.Info("Type: %s -> Matched: %d of source %d and target %d",
		entitiesType, len(matchedEntities),
		len(*entityProcessingPtr.Source.RawEntityListPtr), len(*entityProcessingPtr.Target.RawEntityListPtr))

	leftOverMap := getLeftOvers(oldResultsPtr, entityProcessingPtr)
	outputPayload := genOutputPayload(entitiesType, entityProcessingPtr, oldResultsPtr, matchedEntities, leftOverMap)

	return outputPayload, nil

}

type MatchOutputPerType map[string]MatchOutputType

type MatchOutputType struct {
	Type      string              `json:"type"`
	Source    ExtractionInfo      `json:"source"`
	Target    ExtractionInfo      `json:"target"`
	Matches   map[string]string   `json:"matches"`
	LeftOvers map[string][]string `json:"leftOvers"`
	UnMatched []string            `json:"unmatched"`
}

type ExtractionInfo struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func genOutputPayload(entitiesType string, entityProcessingPtr *EntityProcessing, oldResultsPtr *IndexCompareResultList, matchedEntities map[int]int, leftOverMap map[string][]string) MatchOutputType {

	entityProcessingPtr.PrepareRemainingEntities(false, true, oldResultsPtr)

	matchOutput := MatchOutputType{
		Type: entitiesType,
		Source: ExtractionInfo{
			From: entityProcessingPtr.Source.Type.From,
			To:   entityProcessingPtr.Source.Type.To,
		},
		Target: ExtractionInfo{
			From: entityProcessingPtr.Target.Type.From,
			To:   entityProcessingPtr.Target.Type.To,
		},
		LeftOvers: leftOverMap,
		Matches:   make(map[string]string, len(matchedEntities)),
		UnMatched: make([]string, len(*entityProcessingPtr.Source.CurrentRemainingEntities)),
	}

	for sourceI, targetI := range matchedEntities {
		matchOutput.Matches[(*(entityProcessingPtr.Target.RawEntityListPtr))[targetI].(map[string]interface{})["entityId"].(string)] =
			(*(entityProcessingPtr.Source.RawEntityListPtr))[sourceI].(map[string]interface{})["entityId"].(string)
	}

	for idx, targetI := range *entityProcessingPtr.Source.CurrentRemainingEntities {
		matchOutput.UnMatched[idx] = (*(entityProcessingPtr.Target.RawEntityListPtr))[targetI].(map[string]interface{})["entityId"].(string)
	}

	return matchOutput
}

func getLeftOvers(oldResultsPtr *IndexCompareResultList, entityProcessingPtr *EntityProcessing) map[string][]string {
	printLeftoverSample(oldResultsPtr, entityProcessingPtr)

	return genLeftOverMap(oldResultsPtr, entityProcessingPtr)

}

func genLeftOverMap(oldResultsPtr *IndexCompareResultList, entityProcessingPtr *EntityProcessing) map[string][]string {

	leftOvers := map[string][]string{}

	if len(oldResultsPtr.CompareResults) <= 0 {
		return leftOvers
	}

	firstIdx := 0
	currentId := oldResultsPtr.CompareResults[0].LeftId

	addMatchingLeftOvers := func(nbMatches int) {
		leftOverMatches := make([]string, nbMatches)
		for j := 0; j < nbMatches; j++ {
			sourceId := oldResultsPtr.CompareResults[(j + firstIdx)].RightId
			leftOverMatches[j] = (*(entityProcessingPtr.Target.RawEntityListPtr))[sourceId].(map[string]interface{})["entityId"].(string)
		}
		leftOvers[(*(entityProcessingPtr.Target.RawEntityListPtr))[currentId].(map[string]interface{})["entityId"].(string)] = leftOverMatches
	}

	for i, result := range oldResultsPtr.CompareResults[1:] {
		if result.LeftId == currentId {

		} else {
			nbMatches := i - firstIdx
			addMatchingLeftOvers(nbMatches)

			currentId = result.LeftId
			firstIdx = i
		}
	}
	nbMatches := len(oldResultsPtr.CompareResults) - firstIdx
	addMatchingLeftOvers(nbMatches)

	return leftOvers

}

func printLeftoverSample(oldResultsPtr *IndexCompareResultList, entityProcessingPtr *EntityProcessing) {
	nbLeftOvers := len(oldResultsPtr.CompareResults)

	if nbLeftOvers <= 0 {
		return
	}

	var maxPrint int
	if nbLeftOvers > 10 {
		maxPrint = 10
	} else {
		maxPrint = nbLeftOvers
	}

	for i := 0; i < maxPrint; i++ {
		result := oldResultsPtr.CompareResults[i]
		log.Debug("Left: %v, Source: %v, Target: %v", result,
			(*entityProcessingPtr.Source.RawEntityListPtr)[result.LeftId],
			(*entityProcessingPtr.Target.RawEntityListPtr)[result.RightId])
	}

}

func runIndexRule(indexRule IndexRule, entityProcessingPtr *EntityProcessing,
	resultListPtr *IndexCompareResultList) {

	sortedIndexSource := GenSortedEntitiesIndex(indexRule, &(*entityProcessingPtr).Source)
	sortedIndexTarget := GenSortedEntitiesIndex(indexRule, &(*entityProcessingPtr).Target)

	CompareIndexes(resultListPtr, sortedIndexSource, sortedIndexTarget, indexRule)

}

func keepMatches(matchedEntities map[int]int, singleToSingleMatch []CompareResult) map[int]int {
	for _, result := range singleToSingleMatch {
		_, found := matchedEntities[result.LeftId]

		if found {
			log.Error("Should never find multiple exact matches for an entity, %v", result)
		}

		matchedEntities[result.LeftId] = result.RightId
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
