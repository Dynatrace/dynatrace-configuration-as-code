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
	"github.com/spf13/afero"
)

func genMultiMatchedMap(remainingResultsPtr *match.IndexCompareResultList, entityProcessingPtr *match.MatchProcessing) map[string][]string {

	multiMatched := map[string][]string{}

	if len(remainingResultsPtr.CompareResults) <= 0 {
		return multiMatched
	}

	firstIdx := 0
	currentId := remainingResultsPtr.CompareResults[0].LeftId

	addMatchingMultiMatched := func(matchCount int) {
		multiMatchedMatches := make([]string, matchCount)
		for j := 0; j < matchCount; j++ {
			compareResult := remainingResultsPtr.CompareResults[(j + firstIdx)]
			targetId := compareResult.RightId

			multiMatchedMatches[j] = (*entityProcessingPtr.Target.RawMatchList.GetValues())[targetId].(map[string]interface{})["entityId"].(string)
		}
		multiMatched[(*entityProcessingPtr.Source.RawMatchList.GetValues())[currentId].(map[string]interface{})["entityId"].(string)] = multiMatchedMatches
	}

	for i := 1; i < len(remainingResultsPtr.CompareResults); i++ {
		result := remainingResultsPtr.CompareResults[i]
		if result.LeftId == currentId {

		} else {
			matchCount := i - firstIdx
			addMatchingMultiMatched(matchCount)

			currentId = result.LeftId
			firstIdx = i
		}
	}
	matchCount := len(remainingResultsPtr.CompareResults) - firstIdx
	addMatchingMultiMatched(matchCount)

	return multiMatched

}

func printMultiMatchedSample(remainingResultsPtr *match.IndexCompareResultList, entityProcessingPtr *match.MatchProcessing) {
	multiMatchedCount := len(remainingResultsPtr.CompareResults)

	if multiMatchedCount <= 0 {
		return
	}

	var maxPrint int
	if multiMatchedCount > 10 {
		maxPrint = 10
	} else {
		maxPrint = multiMatchedCount
	}

	for i := 0; i < maxPrint; i++ {
		result := remainingResultsPtr.CompareResults[i]
		log.Debug("Left: %v, Source: %v, Target: %v", result,
			(*entityProcessingPtr.Source.RawMatchList.GetValues())[result.LeftId],
			(*entityProcessingPtr.Target.RawMatchList.GetValues())[result.RightId])
	}

}

func getMultiMatched(remainingResultsPtr *match.IndexCompareResultList, entityProcessingPtr *match.MatchProcessing) map[string][]string {
	printMultiMatchedSample(remainingResultsPtr, entityProcessingPtr)

	return genMultiMatchedMap(remainingResultsPtr, entityProcessingPtr)

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

func genOutputPayload(entityProcessingPtr *match.MatchProcessing, remainingResultsPtr *match.IndexCompareResultList, matchedEntities *map[int]int) MatchOutputType {

	multiMatchedMap := getMultiMatched(remainingResultsPtr, entityProcessingPtr)
	entityProcessingPtr.PrepareRemainingMatch(false, true, remainingResultsPtr)

	matchOutput := MatchOutputType{
		Type: entityProcessingPtr.GetEntitiesType(),
		MatchKey: MatchKey{
			Source: ExtractionInfo{
				From: (*entityProcessingPtr).Source.ConfigType.From,
				To:   (*entityProcessingPtr).Source.ConfigType.To,
			},
			Target: ExtractionInfo{
				From: (*entityProcessingPtr).Target.ConfigType.From,
				To:   (*entityProcessingPtr).Target.ConfigType.To,
			},
		},
		Matches:      make(map[string]string, len(*matchedEntities)),
		MultiMatched: multiMatchedMap,
		UnMatched:    make([]string, len(*entityProcessingPtr.Source.CurrentRemainingMatch)),
	}

	for sourceI, targetI := range *matchedEntities {
		matchOutput.Matches[(*entityProcessingPtr.Target.RawMatchList.GetValues())[targetI].(map[string]interface{})["entityId"].(string)] =
			(*entityProcessingPtr.Source.RawMatchList.GetValues())[sourceI].(map[string]interface{})["entityId"].(string)
	}

	for idx, sourceI := range *entityProcessingPtr.Source.CurrentRemainingMatch {
		matchOutput.UnMatched[idx] = (*entityProcessingPtr.Source.RawMatchList.GetValues())[sourceI].(map[string]interface{})["entityId"].(string)
	}

	return matchOutput
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
