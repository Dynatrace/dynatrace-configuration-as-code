//go:build unit

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

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCache_Get(t *testing.T) {
	cache := DefaultCache[int]{entries: map[string]int{"key": 100}}
	value, found := cache.Get("key")
	assert.True(t, found)
	assert.Equal(t, 100, value)

	value, found = cache.Get("nonexistent")
	assert.False(t, found)
	assert.Equal(t, 0, value)
}

func TestCache_Set(t *testing.T) {
	cache := DefaultCache[int]{}
	cache.Set("key", 100)
	value, found := cache.Get("key")
	assert.True(t, found)
	assert.Equal(t, 100, value)
}

func TestCache_Delete(t *testing.T) {
	cache := DefaultCache[int]{entries: map[string]int{"key": 100}}
	cache.Delete("key")
	value, found := cache.Get("key")
	assert.False(t, found)
	assert.Equal(t, 0, value)
}
