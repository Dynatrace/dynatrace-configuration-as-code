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

package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldDiscard(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		json    map[string]interface{}
		discard bool
	}{
		{
			name:    "builtin:logmonitoring.logs-on-grail-activate - discarded if 'activated' is 'false'",
			schema:  "builtin:logmonitoring.logs-on-grail-activate",
			json:    map[string]interface{}{"activated": false},
			discard: true,
		},
		{
			name:    "builtin:logmonitoring.logs-on-grail-activate - not discarded if 'activated' is 'true'",
			schema:  "builtin:logmonitoring.logs-on-grail-activate",
			json:    map[string]interface{}{"activated": true},
			discard: false,
		},
		{
			name:    "builtin:logmonitoring.log-buckets-rules - discarded if 'name' is 'default'",
			schema:  "builtin:logmonitoring.log-buckets-rules",
			json:    map[string]interface{}{"ruleName": "default"},
			discard: true,
		},
		{
			name:    "builtin:logmonitoring.log-buckets-rules - not discarded if 'name' is not 'default'",
			schema:  "builtin:logmonitoring.log-buckets-rules",
			json:    map[string]interface{}{"ruleName": "something"},
			discard: false,
		},
		{
			name:    "builtin:bizevents-processing-buckets.rule - discarded if 'name' is 'default'",
			schema:  "builtin:bizevents-processing-buckets.rule",
			json:    map[string]interface{}{"ruleName": "default"},
			discard: true,
		},
		{
			name:    "builtin:bizevents-processing-buckets.rule - not discarded if 'name' is not 'default'",
			schema:  "builtin:bizevents-processing-buckets.rule",
			json:    map[string]interface{}{"ruleName": "something"},
			discard: false,
		},
		{
			name:    "builtin:alerting.profile - discarded if name is 'Default'",
			schema:  "builtin:alerting.profile",
			json:    map[string]interface{}{"name": "Default"},
			discard: true,
		},
		{
			name:    "builtin:alerting.profile - discarded if name is 'Default'",
			schema:  "builtin:alerting.profile",
			json:    map[string]interface{}{"name": "Default for ActiveGate Token Expiry"},
			discard: true,
		},
		{
			name:    "builtin:alerting.profile - not discarded if 'name' is 'Something'",
			schema:  "builtin:alerting.profile",
			json:    map[string]interface{}{"name": "Something"},
			discard: false,
		},
		{
			name:    "builtin:logmonitoring.log-events - discarded if 'summary' is equal to 'Default Kubernetes Log Events'",
			schema:  "builtin:logmonitoring.log-events",
			json:    map[string]interface{}{"summary": "Default Kubernetes Log Events"},
			discard: true,
		},
		{
			name:    "builtin:logmonitoring.log-events - not discarded if 'summary' is not equal to 'Default Kubernetes Log Events'",
			schema:  "builtin:logmonitoring.log-events",
			json:    map[string]interface{}{"summary": "my log event"},
			discard: false,
		},
		{
			name:    "all builtin:host.monitoring.mode objects are discarded",
			schema:  "builtin:host.monitoring.mode",
			json:    map[string]interface{}{},
			discard: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filter, found := DefaultSettingsFilters[tc.schema]
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
