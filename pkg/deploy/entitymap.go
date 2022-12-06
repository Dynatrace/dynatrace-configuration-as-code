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

package deploy

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
)

// EntityMap is holds information about known entity names entities already resolved by monaco
type EntityMap struct {
	resolvedEntities parameter.ResolvedEntities
	knownEntityNames map[string]map[string]struct{}
}

// NewEntityMap creates a new EntityMap from a given set of APIs
func NewEntityMap(apis api.ApiMap) *EntityMap {
	knownEntityNames := make(map[string]map[string]struct{})
	for _, api := range apis {
		knownEntityNames[api.GetId()] = make(map[string]struct{})
	}
	resolvedEntities := make(parameter.ResolvedEntities)
	return &EntityMap{
		resolvedEntities: resolvedEntities,
		knownEntityNames: knownEntityNames,
	}
}

func (k *EntityMap) PutResolved(coordinate coordinate.Coordinate, resolvedEntity parameter.ResolvedEntity) {
	k.resolvedEntities[coordinate] = resolvedEntity
	if resolvedEntity.Skip || resolvedEntity.EntityName == "" {
		return
	}
	if _, found := k.knownEntityNames[coordinate.Type]; !found {
		k.knownEntityNames[coordinate.Type] = make(map[string]struct{})
	}
	k.knownEntityNames[coordinate.Type][resolvedEntity.EntityName] = struct{}{}
}

func (k *EntityMap) Resolved() parameter.ResolvedEntities {
	return k.resolvedEntities
}

func (k *EntityMap) Known(entityType string, entityName string) bool {
	_, found := k.knownEntityNames[entityType][entityName]
	return found
}
