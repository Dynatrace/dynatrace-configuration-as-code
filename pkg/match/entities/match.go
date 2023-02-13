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

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"
)

func CompareConfigs(entityPerTypeSource project.ConfigsPerType, entityPerTypeTarget project.ConfigsPerType) (int, int, error) {
	nbEntitiesSource := 0
	nbEntitiesTarget := 0

	for entitiesType := range entityPerTypeTarget {

		log.Debug("Processing Type: %s", entitiesType)

		entityProcessingPtr, err := genEntityProcessing(entityPerTypeSource, entityPerTypeTarget, entitiesType)
		if err != nil {
			return 0, 0, err
		}
		nbEntitiesSource += len(entityProcessingPtr.Source.RemainingEntities)
		nbEntitiesTarget += len(entityProcessingPtr.Target.RemainingEntities)

		err = runIndexRuleAll(entityProcessingPtr, entitiesType)
		if err != nil {
			return 0, 0, err
		}

		log.Debug("Completed Type: %s", entitiesType)
	}

	return nbEntitiesSource, nbEntitiesTarget, nil
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

func runIndexRuleAll(entityProcessingPtr *EntityProcessing, entitiesType string) error {

	matchedEntities := map[int]int{}
	oldResultsPtr := &IndexCompareResultList{}

	sortedActiveIndexRuleTypes := NewIndexRuleMapGenerator(false).GenSortedActiveList()

	for _, indexRuleType := range sortedActiveIndexRuleTypes {
		resultListPtr := NewIndexCompareResultList(indexRuleType)
		entityProcessingPtr.PrepareRemainingEntities(indexRuleType, oldResultsPtr)

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

	printLeftoverSample(oldResultsPtr, entityProcessingPtr)

	outputPayload := genOutputPayload(entityProcessingPtr, matchedEntities)
	log.Debug("outputPayload: %v", outputPayload)

	return nil

}

type MatchOutputType struct {
	Source  ExtractionInfo
	Target  ExtractionInfo
	Matches map[string]string
}

type ExtractionInfo struct {
	From string
	To   string
}

func genOutputPayload(entityProcessingPtr *EntityProcessing, matchedEntities map[int]int) MatchOutputType {

	matchOutput := MatchOutputType{
		Source: ExtractionInfo{
			From: entityProcessingPtr.Source.Type.From,
			To:   entityProcessingPtr.Source.Type.To,
		},
		Target: ExtractionInfo{
			From: entityProcessingPtr.Target.Type.From,
			To:   entityProcessingPtr.Target.Type.To,
		},
		Matches: make(map[string]string, len(matchedEntities)),
	}

	for sourceI, targetI := range matchedEntities {
		matchOutput.Matches[(*(entityProcessingPtr.Target.RawEntityListPtr))[targetI].(map[string]interface{})["entityId"].(string)] =
			(*(entityProcessingPtr.Source.RawEntityListPtr))[sourceI].(map[string]interface{})["entityId"].(string)
	}

	return matchOutput
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
