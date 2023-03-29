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

//go:build unit

package entities

import (
	"path/filepath"
	"reflect"
	"testing"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match"
)

var entityListJsonSortedMultiMatch = `[{
	"entityId": "AZURE_VM-06C38A40104F9FB2",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-06C38A40104F9FB2",
	"firstSeenTms": 1663004439413,
	"lastSeenTms": 1674246569091,
	"properties": {},
	"toRelationships": {}
}, {
	"entityId": "AZURE_VM-109729BAB28C66E8",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-109729BAB28C66E8",
	"firstSeenTms": 1663004173751,
	"lastSeenTms": 1674246562180,
	"properties": {},
	"toRelationships": {}
}, {
	"entityId": "AZURE_VM-2BBAEC9A7D21833A",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-2BBAEC9A7D21833E",
	"firstSeenTms": 1662997868374,
	"lastSeenTms": 1674246568646,
	"properties": {},
	"toRelationships": {}
}, {
	"entityId": "AZURE_VM-2BBAEC9A7D21833B",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-2BBAEC9A7D21833E",
	"firstSeenTms": 1642997868374,
	"lastSeenTms": 1654246568646,
	"properties": {},
	"toRelationships": {}
}]`

const workingDirPath = `/home/test/monaco/match`
const name = `match-test`
const matchType = `entities`
const manifestFileName = `manifest.yaml`
const projectEnvName = `projectEnvName`
const matchFileName = "match.yaml"
const tenantUrl = "https://xxxnnnnn.live.Dynatrace.com"
const tokenType = "environment"
const tokenName = "TEST_TOKEN"
const tokenValue = "TEST"
const groupName = "default"

var workingDir = filepath.FromSlash(workingDirPath)
var outputPath = filepath.Join(workingDir, `output`)
var sourceManifestDir = filepath.Join(workingDir, `source`)
var sourceManifestPath = filepath.Join(sourceManifestDir, manifestFileName)
var targetManifestDir = filepath.Join(workingDir, `target`)
var targetManifestPath = filepath.Join(targetManifestDir, manifestFileName)
var matchFilePath = filepath.Join(workingDir, matchFileName)

const rawMatchYAMLContent = `name: %s
type: %s
outputPath: %s
sourceInfo:
  manifestPath: %s
  project: %s
  environment: %s
targetInfo:
  manifestPath: %s
  project: %s
  environment: %s
`

const rawManifestYAMLContent = `
manifestVersion: "1.0"
projects:
- name: %s
environmentGroups:
- name: %s
  environments:
  - name: %s
    url:
      value: %s
    auth:
      token:
        type: %s
        name: %s
`

func genTestMatchParameters() match.MatchParameters {
	return match.MatchParameters{
		Name:       name,
		Type:       matchType,
		WorkingDir: workingDirPath,
		OutputDir:  outputPath,
		SelfMatch:  false,
		Source: match.MatchParametersEnv{
			EnvType:     match.SOURCE_ENV,
			WorkingDir:  sourceManifestDir,
			Project:     projectEnvName,
			Environment: projectEnvName,
			Manifest: manifest.Manifest{
				Projects: manifest.ProjectDefinitionByProjectID{
					projectEnvName: manifest.ProjectDefinition{
						Name: projectEnvName,
						Path: projectEnvName,
					},
				},
				Environments: manifest.Environments{
					projectEnvName: manifest.EnvironmentDefinition{
						Name: projectEnvName,
						Type: 0,
						URL: manifest.URLDefinition{
							Type:  0,
							Value: tenantUrl,
						},
						Group: groupName,
						Auth: manifest.Auth{
							Token: manifest.AuthSecret{
								Name:  tokenName,
								Value: tokenValue,
							},
						},
					},
				},
			},
		},
		Target: match.MatchParametersEnv{
			EnvType:     match.TARGET_ENV,
			WorkingDir:  targetManifestDir,
			Project:     projectEnvName,
			Environment: projectEnvName,
			Manifest: manifest.Manifest{
				Projects: manifest.ProjectDefinitionByProjectID{
					projectEnvName: manifest.ProjectDefinition{
						Name: projectEnvName,
						Path: projectEnvName,
					},
				},
				Environments: manifest.Environments{
					projectEnvName: manifest.EnvironmentDefinition{
						Name: projectEnvName,
						Type: 0,
						URL: manifest.URLDefinition{
							Type:  0,
							Value: tenantUrl,
						},
						Group: groupName,
						Auth: manifest.Auth{
							Token: manifest.AuthSecret{
								Name:  tokenName,
								Value: tokenValue,
							},
						},
					},
				},
			},
		},
	}
}

func TestRunRules(t *testing.T) {

	tests := []struct {
		name            string
		matchProcessing match.MatchProcessing
		matchParameters match.MatchParameters
		want            MatchOutputType
	}{
		{
			name: "runRules",
			matchProcessing: *match.NewMatchProcessing(
				getRawMatchListFromJson(entityListJsonSortedMultiMatch),
				config.EntityType{
					EntitiesType: "AZURE_VM",
					From:         "1",
					To:           "2",
				},
				getRawMatchListFromJson(entityListJson),
				config.EntityType{
					EntitiesType: "AZURE_VM",
					From:         "2",
					To:           "3",
				},
			),
			matchParameters: genTestMatchParameters(),
			want: MatchOutputType{
				Type: "AZURE_VM",
				MatchKey: MatchKey{
					Source: ExtractionInfo{
						From: "1",
						To:   "2",
					},
					Target: ExtractionInfo{
						From: "2",
						To:   "3",
					},
				},
				Matches: map[string]string{
					"AZURE_VM-06C38A40104F9FB2": "AZURE_VM-06C38A40104F9FB2",
					"AZURE_VM-109729BAB28C66E8": "AZURE_VM-109729BAB28C66E8",
				},
				MultiMatched: map[string][]string{
					"AZURE_VM-2BBAEC9A7D21833A": []string{
						"AZURE_VM-2BBAEC9A7D21833E",
					},
					"AZURE_VM-2BBAEC9A7D21833B": []string{
						"AZURE_VM-2BBAEC9A7D21833E",
					},
				},
				UnMatched: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runRules(&tt.matchProcessing, tt.matchParameters)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("runRules() got = %v, want %v", got, tt.want)
			}

		})
	}
}
