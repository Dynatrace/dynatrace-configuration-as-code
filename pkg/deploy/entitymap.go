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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"sync"
)

// ResolvedEntities defines a map representing resolved configs. this includes the
// api `ID` of a config.
type ResolvedEntities map[coordinate.Coordinate]ResolvedEntity

// ResolvedEntity struct representing an already deployed entity
type ResolvedEntity struct {
	// EntityName is the name returned by the Dynatrace api. In theory should be the
	// same as the `name` property defined in the configuration, but
	// can differ.
	EntityName string

	// Coordinate of the config this entity represents
	Coordinate coordinate.Coordinate

	// Properties defines a map of all already resolved parameters
	Properties parameter.Properties

	// Skip flag indicating that this entity was skipped
	// if an entity is skipped, there will be no properties
	Skip bool
}

type entityMap struct {
	lock             sync.RWMutex
	resolvedEntities ResolvedEntities
}

func newEntityMap() *entityMap {
	return &entityMap{
		resolvedEntities: make(ResolvedEntities),
	}
}

func (r *entityMap) put(resolvedEntity ResolvedEntity) {
	r.lock.Lock()
	defer r.lock.Unlock()
	// memorize resolved entity
	r.resolvedEntities[resolvedEntity.Coordinate] = resolvedEntity
}

func (r *entityMap) Property(config coordinate.Coordinate, property string) (any, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if e, f := r.resolvedEntities[config]; f {
		if p, f := e.Properties[property]; f {
			return p, true
		}
	}

	return nil, false
}

func (r *entityMap) Entity(config coordinate.Coordinate) (ResolvedEntity, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	v, f := r.resolvedEntities[config]
	return v, f
}
