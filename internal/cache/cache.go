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

package cache

import "sync"

type Cache[T any] interface {
	Get(key string) (T, bool)
	Set(key string, entries T)
	Delete(key string)
	Clear()
}

// NoopCache is an implementation of Cache that doesn't actually do anything.
type NoopCache[T interface{}] struct{}

func (n NoopCache[T]) Get(_ string) (T, bool) {
	var res T
	return res, false
}

func (n NoopCache[T]) Set(_ string, _ T) {
	// no-op
}

func (n NoopCache[T]) Delete(_ string) {
	// no-op
}

func (n NoopCache[T]) Clear() {
	// no-op
}

// DefaultCache is an implementation of Cache that stores all values in a map.
type DefaultCache[T any] struct {
	entries map[string]T
	mutex   sync.RWMutex
}

// Get retrieves the value associated with the given key from the cache.
// It acquires a read lock to allow concurrent access from multiple goroutines.
// It returns the value and a boolean indicating if the value exists in the cache.
func (s *DefaultCache[T]) Get(key string) (T, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	e, ok := s.entries[key]
	return e, ok
}

// Set adds or updates an entry in the cache with the specified key and value.
// It acquires an exclusive write lock to ensure exclusive access during the update.
func (s *DefaultCache[T]) Set(key string, entries T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.entries == nil {
		s.entries = make(map[string]T)
	}
	s.entries[key] = entries
}

// Delete removes an entry from the cache with the specified key.
// It acquires an exclusive write lock to ensure exclusive access during the deletion.
func (s *DefaultCache[T]) Delete(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.entries, key)
}

func (s *DefaultCache[T]) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.entries = nil
}
