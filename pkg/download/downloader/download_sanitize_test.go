//go:build unit

// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package downloader

import (
	"gotest.tools/assert"
	"testing"
)

func TestRemoveIdentifiers(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		expectedJson string
	}{
		{
			"metadata is removed",
			`{"metadata":""}`,
			"{}",
		},
		{
			"id is removed",
			`{"id":""}`,
			"{}",
		},
		{
			"identifier is removed",
			`{"identifier":""}`,
			"{}",
		},
		{
			"entityId is removed",
			`{"entityId":""}`,
			"{}",
		},
		{
			"rules.id is removed",
			`{"rules": {"id": ""}}`,
			`{"rules":{}}`,
		},
		{
			"rules[].id is removed",
			`{"rules": [{"id": ""}]}`,
			`{"rules":[{}]}`,
		},
		{
			"rules.methodRules.id is removed",
			`{"rules": {"methodRules": {"id": ""}}}`,
			`{"rules": {"methodRules":{}}}`,
		},
		{
			"rules[].methodRules.id is removed",
			`{"rules": [{"methodRules": {"id": ""}}]}`,
			`{"rules": [{"methodRules":{}}]}`,
		},
		{
			"rules[].methodRules.id is removed",
			`{"rules": [{"methodRules": {"id": ""}}]}`,
			`{"rules": [{"methodRules":{}}]}`,
		},
		{
			"rules[].methodRules[].id is removed",
			`{"rules": [{"methodRules": [{"id": ""}]}]}`,
			`{"rules": [{"methodRules":[{}]}]}`,
		},
		{
			"rules.methodRules[].id is removed",
			`{"rules": {"methodRules": [{"id": ""}]}}`,
			`{"rules": {"methodRules":[{}]}}`,
		},
		{
			"other properties are not removed",
			`{"x":""}`,
			`{"x":""}`,
		},
		{
			"multiple others are not removed",
			`{"x":"", "y": 1234, "z": null}`,
			`{"x":"", "y": 1234, "z": null}`,
		},
		{
			"name is replaced with template",
			`{"name":"asdf"}`,
			`{"name": "{{.name}}"}`,
		},
		{
			"displayName is replaced with name",
			`{"displayName":"asdf"}`,
			`{"displayName": "{{.name}}"}`,
		},
		{
			"dashboardMetadata.name is replaced with name",
			`{"dashboardMetadata": {"name": "something"}}`,
			`{"dashboardMetadata": {"name": "{{.name}}"}}`,
		},
		{
			"mixed works",
			`{"x":"", "y": 1234, "z": null, "id": "", "rules": {"methodRules": {"id": ""}}, "dashboardMetadata": {"name": "{{.name}}"}}`,
			`{"x":"", "y": 1234, "z": null, "rules": {"methodRules":{}}, "dashboardMetadata": {"name": "{{.name}}"}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := sanitizeProperties(unmarshal(t, test.json))

			expected := unmarshal(t, test.expectedJson)

			assert.DeepEqual(t, result, expected)
		})
	}
}
