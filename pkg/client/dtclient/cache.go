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

package dtclient

import "sync"

type cache[T any] struct {
	cachedItems map[string][]T
	mutex       sync.RWMutex
}

func (s *cache[T]) hasCache(id string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, ok := s.cachedItems[id]
	return ok
}

func (s *cache[T]) set(id string, settings []T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cachedItems == nil {
		s.cachedItems = make(map[string][]T)
	}
	s.cachedItems[id] = settings
}

func (s *cache[T]) filter(id string, filter func(T) bool) []T {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if filter == nil {
		filter = func(object T) bool { return true }
	}
	result := make([]T, 0)
	for _, i := range s.cachedItems[id] {
		if filter(i) {
			result = append(result, i)
		}
	}
	return result
}

func (s *cache[T]) invalidate(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.cachedItems, id)
}
