//go:build unit

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package classic

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveIdentifiers(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		api          string
		expectedJson string
	}{
		{
			"metadata is removed",
			`{"metadata":""}`,
			"does-not-matter",
			"{}",
		},
		{
			"id is removed",
			`{"id":""}`,
			"does-not-matter",
			"{}",
		},
		{
			"identifier is removed",
			`{"identifier":""}`,
			"does-not-matter",
			"{}",
		},
		{
			"entityId is removed",
			`{"entityId":""}`,
			"does-not-matter",
			"{}",
		},
		{
			"applicationId is removed",
			`{"applicationId":""}`,
			"does-not-matter",
			"{}",
		},
		{
			"rules.id is removed",
			`{"rules": {"id": ""}}`,
			"does-not-matter",
			`{"rules":{}}`,
		},
		{
			"rules[].id is removed",
			`{"rules": [{"id": ""}]}`,
			"does-not-matter",
			`{"rules":[{}]}`,
		},
		{
			"rules.methodRules.id is removed",
			`{"rules": {"methodRules": {"id": ""}}}`,
			"does-not-matter",
			`{"rules": {"methodRules":{}}}`,
		},
		{
			"rules[].methodRules.id is removed",
			`{"rules": [{"methodRules": {"id": ""}}]}`,
			"does-not-matter",
			`{"rules": [{"methodRules":{}}]}`,
		},
		{
			"rules[].methodRules.id is removed",
			`{"rules": [{"methodRules": {"id": ""}}]}`,
			"does-not-matter",
			`{"rules": [{"methodRules":{}}]}`,
		},
		{
			"rules[].methodRules[].id is removed",
			`{"rules": [{"methodRules": [{"id": ""}]}]}`,
			"does-not-matter",
			`{"rules": [{"methodRules":[{}]}]}`,
		},
		{
			"rules.methodRules[].id is removed",
			`{"rules": {"methodRules": [{"id": ""}]}}`,
			"does-not-matter",
			`{"rules": {"methodRules":[{}]}}`,
		},
		{
			"other properties are not removed",
			`{"x":""}`,
			"does-not-matter",
			`{"x":""}`,
		},
		{
			"multiple others are not removed",
			`{"x":"", "y": 1234, "z": null}`,
			"does-not-matter",
			`{"x":"", "y": 1234, "z": null}`,
		},
		{
			"order property is removed for service-detection-full-web-service",
			`{"some_prop":"some_val", "order": 42}`,
			"service-detection-full-web-service",
			`{"some_prop":"some_val"}`,
		},
		{
			"order property is removed for service-detection-full-web-request",
			`{"some_prop":"some_val", "order": 42}`,
			"service-detection-full-web-request",
			`{"some_prop":"some_val"}`,
		},
		{
			"order property is removed for service-detection-opaque-web-service",
			`{"some_prop":"some_val", "order": 42}`,
			"service-detection-opaque-web-service",
			`{"some_prop":"some_val"}`,
		},
		{
			"order property is removed for service-detection-opaque-web-request",
			`{"some_prop":"some_val", "order": 42}`,
			"service-detection-opaque-web-request",
			`{"some_prop":"some_val"}`,
		},
		{
			"order property is NOT removed for other APIs",
			`{"some_prop":"some_val", "order": 42}`,
			"altering-profile",
			`{"some_prop":"some_val", "order": 42}`,
		},
		{
			"empty scopes.entities is removed from maintenance-window",
			`{"some_prop":"some_val", "scope": {"entities":[], "matches": ["some_match"] } }`,
			"maintenance-window",
			`{"some_prop":"some_val", "scope": {"matches": ["some_match"] } }`,
		},
		{
			"empty scopes.matches is removed from maintenance-window",
			`{"some_prop":"some_val", "scope": {"entities":["some_entity"], "matches": [] } }`,
			"maintenance-window",
			`{"some_prop":"some_val", "scope": {"entities": ["some_entity"] } }`,
		},
		{
			"empty scopes is removed from maintenance-window",
			`{"some_prop":"some_val", "scope": {"entities":[], "matches": [] } }`,
			"maintenance-window",
			`{"some_prop":"some_val"}`,
		},
		{
			"maintenance-window without any scopes is unchanged",
			`{"some_prop":"some_val"}`,
			"maintenance-window",
			`{"some_prop":"some_val"}`,
		},
		{
			"scopes property is NOT changed for other APIs",
			`{"some_prop":"some_val", "scope": {"entities":[], "matches": [] } }`,
			"alerting-profile",
			`{"some_prop":"some_val", "scope": {"entities":[], "matches": [] } }`,
		},
		{
			"name is replaced with template",
			`{"name":"asdf"}`,
			"does-not-matter",
			`{"name": "{{.name}}"}`,
		},
		{
			"displayName is replaced with name",
			`{"displayName":"asdf"}`,
			"does-not-matter",
			`{"displayName": "{{.name}}"}`,
		},
		{
			"dashboardMetadata.name is replaced with name",
			`{"dashboardMetadata": {"name": "something"}}`,
			"does-not-matter",
			`{"dashboardMetadata": {"name": "{{.name}}"}}`,
		},
		{
			"mixed works",
			`{"x":"", "y": 1234, "z": null, "id": "", "rules": {"methodRules": {"id": ""}}, "dashboardMetadata": {"name": "{{.name}}"}}`,
			"does-not-matter",
			`{"x":"", "y": 1234, "z": null, "rules": {"methodRules":{}}, "dashboardMetadata": {"name": "{{.name}}"}}`,
		},
		{
			"entity id is not removed for CMS, but all other ids are",
			`{"entityId": "some-id", "id": "must be removed"}`,
			"calculated-metrics-service",
			`{"entityId": "some-id"}`,
		},
		{
			"conversion goal IDs are removed from application-web payloads",
			`{
					"conversionGoals": [
						{
							"id": "424242424242",
							"name": "The Answer",
							"type": "VisitDuration",
							"visitDurationDetails": {
								"durationInMillis": 16962000
							}
						}
					],
					"costControlUserSessionPercentage": 100 }`,
			"application-web",
			`{
					"conversionGoals": [
						{
							"name": "The Answer",
							"type": "VisitDuration",
							"visitDurationDetails": {
								"durationInMillis": 16962000
							}
						}
					],
					"costControlUserSessionPercentage": 100 }`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := sanitizeProperties(unmarshal(t, test.json), test.api)

			expected := unmarshal(t, test.expectedJson)

			require.Equal(t, result, expected)
		})
	}
}
