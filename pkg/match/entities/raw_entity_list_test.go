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
	"encoding/json"
	"gotest.tools/assert"
	"reflect"
	"testing"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

func getRawEntityListFromJson(jsonData string) RawEntityList {
	rawEntityList := &RawEntityList{
		Values: new([]interface{}),
	}
	json.Unmarshal([]byte(jsonData), rawEntityList.Values)

	return *rawEntityList
}

func convertRawEntityToMatchList(rawMatchList match.RawMatchList) match.RawMatchList {
	return rawMatchList
}

func getRawMatchListFromJson(jsonData string) match.RawMatchList {
	rawEntityList := getRawEntityListFromJson(jsonData)
	return convertRawEntityToMatchList(&rawEntityList)
}

var entityListJson = `[{
	"entityId": "AZURE_VM-109729BAB28C66E8",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-109729BAB28C66E8",
	"firstSeenTms": 1663004173751,
	"lastSeenTms": 1674246562180,
	"properties": {},
	"toRelationships": {}
}, {
	"entityId": "AZURE_VM-06C38A40104F9FB2",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-06C38A40104F9FB2",
	"firstSeenTms": 1663004439413,
	"lastSeenTms": 1674246569091,
	"properties": {},
	"toRelationships": {}
}, {
	"entityId": "AZURE_VM-2BBAEC9A7D21833E",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-2BBAEC9A7D21833E",
	"firstSeenTms": 1662997868374,
	"lastSeenTms": 1674246568646,
	"properties": {},
	"toRelationships": {}
}]`

var entityListJsonSorted = `[{
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
	"entityId": "AZURE_VM-2BBAEC9A7D21833E",
	"type": "AZURE_VM",
	"displayName": "UNKNOWN AZURE_VM-2BBAEC9A7D21833E",
	"firstSeenTms": 1662997868374,
	"lastSeenTms": 1674246568646,
	"properties": {},
	"toRelationships": {}
}]`

func TestSortRawEntityList(t *testing.T) {

	tests := []struct {
		name       string
		entityList RawEntityList
		want       RawEntityList
	}{
		{
			name:       "Test sort raw entity list",
			entityList: getRawEntityListFromJson(entityListJson),
			want:       getRawEntityListFromJson(entityListJsonSorted),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.entityList.Sort()

			if !reflect.DeepEqual(tt.entityList, tt.want) {
				t.Errorf("SortRawEntityList() entityList = %v, want %v", tt.entityList, tt.want)
			}

		})
	}
}

func TestUnmarshalEntities(t *testing.T) {

	tests := []struct {
		name          string
		entityPerType []config.Config
		want          RawEntityList
	}{
		{
			name: "unmarshalEntities",
			entityPerType: []config.Config{
				config.Config{
					Template: template.NewDownloadTemplate("AZURE_VM", "AZURE_VM", entityListJsonSorted),
				},
			},
			want: getRawEntityListFromJson(entityListJsonSorted),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalEntities(tt.entityPerType)

			assert.NilError(t, err)

			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("unmarshalEntities() got = %v, want %v", *got, tt.want)
			}

		})
	}
}

func TestGenEntityProcessing(t *testing.T) {

	tests := []struct {
		name                string
		entityPerTypeSource project.ConfigsPerType
		entityPerTypeTarget project.ConfigsPerType
		entitiesType        string
		want                match.MatchProcessing
	}{
		{
			name: "genEntityProcessing",
			entityPerTypeSource: project.ConfigsPerType{
				"AZURE_VM": []config.Config{
					config.Config{
						Template: template.NewDownloadTemplate("AZURE_VM", "AZURE_VM", entityListJsonSorted),
						Type: config.EntityType{
							EntitiesType: "AZURE_VM",
							From:         "1",
							To:           "2",
						},
					},
				},
			},
			entityPerTypeTarget: project.ConfigsPerType{
				"AZURE_VM": []config.Config{
					config.Config{
						Template: template.NewDownloadTemplate("AZURE_VM", "AZURE_VM", entityListJson),
						Type: config.EntityType{
							EntitiesType: "AZURE_VM",
							From:         "2",
							To:           "3",
						},
					},
				},
			},
			entitiesType: "AZURE_VM",
			want: *match.NewMatchProcessing(
				getRawMatchListFromJson(entityListJsonSorted),
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genEntityProcessing(tt.entityPerTypeSource, tt.entityPerTypeTarget, tt.entitiesType)

			assert.NilError(t, err)

			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("genEntityProcessing() got = %v, want %v", *got, tt.want)
			}

		})
	}
}
