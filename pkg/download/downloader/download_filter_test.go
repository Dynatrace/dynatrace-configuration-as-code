//go:build unit
// +build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package downloader

import (
	"encoding/json"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"gotest.tools/assert"
	"testing"
)

func Test_ShouldConfigBeSkipped(t *testing.T) {
	dynatrace := "Dynatrace"
	notDynatrace := "Not Dynatrace"

	tests := []struct {
		name            string
		apiId           string
		value           api.Value
		shouldBeSkipped bool
	}{
		{
			"owner 'Dynatrace' is skipped",
			"dashboard",
			api.Value{Owner: &dynatrace},
			true,
		},
		{
			"owner 'Not Dynatrace' is not skipped",
			"dashboard",
			api.Value{Owner: &notDynatrace},
			false,
		},
		{
			"no owner is not skipped",
			"dashboard",
			api.Value{Owner: nil},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.apiId+" "+test.name, func(t *testing.T) {
			fac, finish := api.CreateAPIMockWithId(t, test.apiId)
			defer finish()

			skip := shouldConfigBeSkipped(fac, test.value)

			assert.Equal(t, skip, test.shouldBeSkipped)
		})
	}
}

func Test_ShouldBePersisted(t *testing.T) {
	tests := []struct {
		name    string
		apiId   string
		json    string
		persist bool
	}{
		{
			"Dashboard should not be persisted if its a preset",
			"dashboard",
			`{"dashboardMetadata": {"preset": true}}`,
			false,
		},
		{
			"Dashboard should be persisted if dashboardMetadata is missing",
			"dashboard",
			`{}`,
			true,
		},
		{
			"Dashboard should be persisted if's not a preset",
			"dashboard",
			`{"dashboardMetadata": {"preset": false}}`,
			true,
		},
		{
			"Dashboard should be persisted if dashboardMetadata.preset is missing",
			"dashboard",
			`{"dashboardMetadata": {}}`,
			true,
		},
		{
			"Synthetic-location should be persisted if it's not private",
			"synthetic-location",
			`{"type": "PRIVATE"}`,
			false,
		},
		{
			"Synthetic-location should not be persisted if it's private",
			"synthetic-location",
			`{"type": "PUBLIC"}`,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.apiId+" "+test.name, func(t *testing.T) {
			fac, finish := api.CreateAPIMockWithId(t, test.apiId)
			defer finish()

			mappedJson := unmarshal(t, test.json)

			persist := shouldConfigBePersisted(fac, mappedJson)

			assert.Equal(t, persist, test.persist)
		})
	}
}

func Test_AllDefinedApiFiltersHaveApis(t *testing.T) {
	definedApis := api.NewApis()

	for apiId := range apiFilters {
		_, found := definedApis[apiId]

		assert.Equal(t, found, true, "Filtered API '%v' not defined in apis", apiId)
	}
}

func unmarshal(t *testing.T, content string) map[string]interface{} {
	mapped := map[string]interface{}{}
	err := json.Unmarshal([]byte(content), &mapped)

	assert.NilError(t, err, "Error during test definition")

	return mapped
}
