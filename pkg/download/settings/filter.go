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

package settings

// noOpFilter is a settings 2.0 filter that does nothing
var noOpFilter = Filter{
	ShouldDiscard: func(settingsValue map[string]interface{}) bool { return false },
}

// Filter can be used to filter/discard settings 2.0
type Filter struct {
	// ShouldDiscard contains logic whether a settings object should be discarded
	// based on specific criteria on the settings value payload
	ShouldDiscard func(settingsValue map[string]interface{}) bool
}

// Filters represents a map of settings 2.0 Filters
type Filters map[string]Filter

// Get returns the filter for a given key
func (f Filters) Get(schemaID string) Filter {
	if filter, ok := f[schemaID]; ok {
		return filter
	}
	return noOpFilter
}

// defaultSettingsFilters is the default Filters used for settings 2.0
var defaultSettingsFilters = Filters{
	"builtin:logmonitoring.logs-on-grail-activate": {
		ShouldDiscard: func(json map[string]interface{}) bool {
			return json["activated"] != false
		},
	},
}
