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
	"testing"

	"gotest.tools/assert"
)

func getRawEntityListFromJson(jsonData string) RawEntityList {
	rawEntityList := &RawEntityList{
		Values: new([]interface{}),
	}
	json.Unmarshal([]byte(jsonData), rawEntityList.Values)

	return *rawEntityList
}

func TestSortRawEntityList(t *testing.T) {

	entityListJson := `[{
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

	entityListJsonSorted := `[{
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

			assert.Equal(t, tt.entityList.Len(), tt.want.Len())
			for i, _ := range *tt.entityList.GetValues() {
				assert.Equal(t, ((*tt.entityList.GetValues())[i].(map[string]interface{}))["entityId"].(string),
					((*tt.want.GetValues())[i].(map[string]interface{}))["entityId"].(string))
			}

		})
	}
}
