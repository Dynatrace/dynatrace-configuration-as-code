/*
 * @license
 * Copyright 2023 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package entitymap

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"sync"
)

type EntityMap struct {
	lock             sync.RWMutex
	resolvedEntities map[coordinate.Coordinate]config.ResolvedEntity
}

func New() *EntityMap {
	return &EntityMap{
		resolvedEntities: make(map[coordinate.Coordinate]config.ResolvedEntity),
	}
}

func (r *EntityMap) Put(resolvedEntity config.ResolvedEntity) {
	r.lock.Lock()
	defer r.lock.Unlock()
	// memorize resolved entity
	r.resolvedEntities[resolvedEntity.Coordinate] = resolvedEntity
}

func (r *EntityMap) Get() map[coordinate.Coordinate]config.ResolvedEntity {

	r.lock.RLock()
	defer r.lock.RUnlock()

	entityCopy := make(map[coordinate.Coordinate]config.ResolvedEntity, len(r.resolvedEntities))
	for k, v := range r.resolvedEntities {
		entityCopy[k] = v
	}

	return entityCopy
}

func (r *EntityMap) GetResolvedProperty(coordinate coordinate.Coordinate, propertyName string) (any, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if e, f := r.resolvedEntities[coordinate]; f {
		if p, f := e.Properties[propertyName]; f {
			return p, true
		}
	}

	return nil, false
}

func (r *EntityMap) GetResolvedEntity(config coordinate.Coordinate) (config.ResolvedEntity, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	v, f := r.resolvedEntities[config]
	return v, f
}
