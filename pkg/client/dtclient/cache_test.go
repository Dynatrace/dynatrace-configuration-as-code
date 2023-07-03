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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache_HasCache(t *testing.T) {
	cache := &cache[DownloadSettingsObject]{
		cachedItems: map[string][]DownloadSettingsObject{
			"schemaID1": {},
			"schemaID2": {{}, {}},
		},
	}

	hasCache1 := cache.hasCache("schemaID1")
	hasCache2 := cache.hasCache("schemaID2")
	hasCache3 := cache.hasCache("schemaID3")

	assert.True(t, hasCache1, "Expected schemaID1 cache to exist")
	assert.True(t, hasCache2, "Expected schemaID2 cache to exist")
	assert.False(t, hasCache3, "Expected schemaID3 cache to not exist")
}

func TestCache_Set(t *testing.T) {
	cache := &cache[DownloadSettingsObject]{}
	settings := []DownloadSettingsObject{{ExternalId: "1"}, {ExternalId: "2"}}

	cache.set("schemaID", settings)

	assert.Equal(t, settings, cache.cachedItems["schemaID"], "Expected settings to be set")
}

func TestCache_Filter(t *testing.T) {
	cache := &cache[DownloadSettingsObject]{
		cachedItems: map[string][]DownloadSettingsObject{
			"schemaID": {
				{ExternalId: "1"},
				{ExternalId: "2"},
				{ExternalId: "3"},
			},
		},
	}
	filter := func(object DownloadSettingsObject) bool {
		return object.ExternalId != "2"
	}

	filtered := cache.filter("schemaID", filter)

	assert.Len(t, filtered, 2, "Expected 2 filtered settings")
	assert.Equal(t, "1", filtered[0].ExternalId, "Expected first object to have ExternalId = 1")
	assert.Equal(t, "3", filtered[1].ExternalId, "Expected second object to have ExternalId = 3")
}

func TestCache_Filter_NilFilter(t *testing.T) {
	cache := &cache[DownloadSettingsObject]{
		cachedItems: map[string][]DownloadSettingsObject{
			"schemaID": {
				{ExternalId: "1"},
				{ExternalId: "2"},
				{ExternalId: "3"},
			},
		},
	}

	filtered := cache.filter("schemaID", nil)

	assert.Len(t, filtered, 3, "Expected 3 settings with default filter")
}

func TestCache_Invalidate(t *testing.T) {
	cache := &cache[DownloadSettingsObject]{
		cachedItems: map[string][]DownloadSettingsObject{
			"schemaID": {},
		},
	}

	cache.invalidate("schemaID")

	_, exists := cache.cachedItems["schemaID"]
	assert.False(t, exists, "Expected schemaID cache to be invalidated")
}
