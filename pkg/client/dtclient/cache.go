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
	entries map[string][]T
	mutex   sync.RWMutex
}

func (s *cache[T]) set(key string, entries []T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.entries == nil {
		s.entries = make(map[string][]T)
	}
	s.entries[key] = entries
}

func (s *cache[T]) get(id string, filter func(T) bool) ([]T, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if _, ok := s.entries[id]; !ok {
		return nil, false
	}

	if filter == nil {
		return s.entries[id], true
	}

	result := make([]T, 0)
	for _, i := range s.entries[id] {
		if filter(i) {
			result = append(result, i)
		}
	}
	return result, true
}

func (s *cache[T]) delete(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.entries, id)
}
