//go:build unit

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

package classic

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/golang/mock/gomock"
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
		{
			"unregistered api should not be skipped",
			"api",
			api.Value{},
			false,
		},
		{
			"ruxit. should be skipped",
			"anomaly-detection-metrics",
			api.Value{Id: "ruxit.a"},
			true,
		},
		{
			"dynatrace. should be skipped",
			"anomaly-detection-metrics",
			api.Value{Id: "dynatrace.b"},
			true,
		},
		{
			"ids should not be skipped",
			"anomaly-detection-metrics",
			api.Value{Id: "b836ff25-24e3-496d-8dce-d94110815ab5"},
			false,
		},
		{
			"random strings should not be skipped",
			"anomaly-detection-metrics",
			api.Value{Id: "test.something"},
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
			"Synthetic-location should be persisted if it's private",
			"synthetic-location",
			`{"type": "PRIVATE"}`,
			true,
		},
		{
			"Synthetic-location should not be persisted if it's public",
			"synthetic-location",
			`{"type": "PUBLIC"}`,
			false,
		},
		{
			"Empty update windows are not persisted",
			"hosts-auto-update",
			`{"updateWindows": {"windows": []}}`,
			false,
		},
		{
			"Missing updateWindows is persisted",
			"hosts-auto-update",
			`{}`,
			true,
		},
		{
			"Missing windows is persisted",
			"hosts-auto-update",
			`{"updateWindows": {}}`,
			true,
		},
		{
			"Windows with values are persisted",
			"hosts-auto-update",
			`{"updateWindows": {"windows": ["1-2-3"]}}`,
			true,
		},
		{
			"unregistered api should be persisted",
			"some-api",
			"{}",
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

func TestFilterConfigsToSkip(t *testing.T) {
	a := api.NewMockApi(gomock.NewController(t))
	a.EXPECT().GetId().AnyTimes().Return("dashboard")

	var ownerDynatrace = "Dynatrace"
	var ownerSomebody = "Somebody"

	values := []api.Value{
		{Owner: &ownerDynatrace}, // should be removed
		{Owner: &ownerSomebody},
		{Owner: nil},
	}

	results := filterConfigsToSkip(a, values)

	assert.Equal(t, len(results), 2)
	assert.Equal(t, *results[0].Owner, ownerSomebody)
	assert.Equal(t, results[1].Owner, (*string)(nil))
}
