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
	"sort"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

type RawEntityList struct {
	Values *[]interface{}
}

// ByRawEntityId implements sort.Interface for []RawEntity] based on
// the EntityId string field.
type ByRawEntityId []interface{}

func (a ByRawEntityId) Len() int      { return len(a) }
func (a ByRawEntityId) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRawEntityId) Less(i, j int) bool {
	return (a[i].(map[string]interface{}))["entityId"].(string) < (a[j].(map[string]interface{}))["entityId"].(string)
}

func (r *RawEntityList) Sort() {

	sort.Sort(ByRawEntityId(*r.GetValues()))

}

func (r *RawEntityList) Len() int {

	return len(*r.GetValues())

}

func (r *RawEntityList) GetValues() *[]interface{} {

	return r.Values

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

func genEntityProcessing(entityPerTypeSource project.ConfigsPerType, entityPerTypeTarget project.ConfigsPerType, entitiesType string) (*match.MatchProcessing, error) {

	rawEntitiesSource, err := unmarshalEntities(entityPerTypeSource[entitiesType])
	if err != nil {
		return nil, err
	}
	sourceType := config.EntityType{}
	if len(entityPerTypeSource[entitiesType]) > 0 {
		sourceType = entityPerTypeSource[entitiesType][0].Type.(config.EntityType)
	}

	rawEntitiesTarget, err := unmarshalEntities(entityPerTypeTarget[entitiesType])
	if err != nil {
		return nil, err
	}
	targetType := config.EntityType{}
	if len(entityPerTypeTarget[entitiesType]) > 0 {
		targetType = entityPerTypeTarget[entitiesType][0].Type.(config.EntityType)
	}

	return match.NewMatchProcessing(rawEntitiesSource, sourceType, rawEntitiesTarget, targetType), nil
}
