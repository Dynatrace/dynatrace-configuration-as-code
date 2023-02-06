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

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShouldDiscard(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		json    map[string]interface{}
		discard bool
	}{
		{
			name:    "log-on-grail-activate - should not be persisted when activated false",
			schema:  "builtin:logmonitoring.logs-on-grail-activate",
			json:    map[string]interface{}{"activated": false},
			discard: true,
		},
		{
			name:    "log-on-grail-activate - should be persisted when activated true",
			schema:  "builtin:logmonitoring.logs-on-grail-activate",
			json:    map[string]interface{}{"activated": true},
			discard: false,
		},
		{
			name:    "logmonitoring.log-buckets-rules - should not be persisted when name is 'default'",
			schema:  "builtin:logmonitoring.log-buckets-rules",
			json:    map[string]interface{}{"ruleName": "default"},
			discard: true,
		},
		{
			name:    "logmonitoring.log-buckets-rules - should be persisted when name is not 'default'",
			schema:  "builtin:logmonitoring.log-buckets-rules",
			json:    map[string]interface{}{"ruleName": "something"},
			discard: false,
		},
		{
			name:    "bizevents-processing-buckets.rule - should be not persisted when name is 'default'",
			schema:  "builtin:bizevents-processing-buckets.rule",
			json:    map[string]interface{}{"ruleName": "default"},
			discard: true,
		},
		{
			name:    "bizevents-processing-buckets.rule - should be persisted when name is not 'default'",
			schema:  "builtin:bizevents-processing-buckets.rule",
			json:    map[string]interface{}{"ruleName": "something"},
			discard: false,
		},
		{
			name:    "alerting.profile - should not be persisted when name is 'Default'",
			schema:  "builtin:alerting.profile",
			json:    map[string]interface{}{"name": "Default"},
			discard: true,
		},
		{
			name:    "alerting.profile - should not be persisted when name is 'Default'",
			schema:  "builtin:alerting.profile",
			json:    map[string]interface{}{"name": "Default for ActiveGate Token Expiry"},
			discard: true,
		},
		{
			name:    "alerting.profile - should be persisted when name is 'Something'",
			schema:  "builtin:alerting.profile",
			json:    map[string]interface{}{"name": "Something"},
			discard: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filter, found := defaultSettingsFilters[tc.schema]
			assert.True(t, found, "filter for schema %q not found", tc.schema)

			shouldDiscard, reason := filter.ShouldDiscard(tc.json)

			assert.Equal(t, shouldDiscard, tc.discard)
			if shouldDiscard {
				assert.NotEmpty(t, reason)
			}
		})
	}
}

func TestGetFilter(t *testing.T) {
	assert.NotNil(t, Filters{"id": noOpFilter}.Get("id"))
}

func TestNoOpFilterDoesNothing(t *testing.T) {
	shouldDiscard, reason := noOpFilter.ShouldDiscard(map[string]interface{}{})
	assert.False(t, shouldDiscard)
	assert.Empty(t, reason)
}
