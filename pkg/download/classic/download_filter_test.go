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
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_AllDefinedApiFiltersHaveApis(t *testing.T) {
	definedApis := api.NewApis()

	for apiId := range apiFilters {
		_, found := definedApis[apiId]

		assert.True(t, found, "Filtered API '%v' not defined in apis", apiId)
	}
}

func Test_APIFiltersLogic(t *testing.T) {
	assert.False(t, apiFilters["dashboard"].shouldConfigBePersisted(map[string]interface{}{
		"dashboardMetadata": map[string]interface{}{
			"preset": true,
		},
	}))
	assert.True(t, apiFilters["dashboard"].shouldConfigBePersisted(map[string]interface{}{
		"dashboardMetadata": map[string]interface{}{
			"preset": false,
		},
	}))

	owner := "Dynatrace"
	owner2 := "NotDynatrace"
	assert.True(t, apiFilters["dashboard"].shouldBeSkippedPreDownload(api.Value{Owner: &owner}))
	assert.False(t, apiFilters["dashboard"].shouldBeSkippedPreDownload(api.Value{Owner: &owner2}))
	assert.False(t, apiFilters["dashboard"].shouldBeSkippedPreDownload(api.Value{}))

	assert.True(t, apiFilters["synthetic-location"].shouldConfigBePersisted(map[string]interface{}{
		"type": "PRIVATE",
	}))
	assert.False(t, apiFilters["synthetic-location"].shouldConfigBePersisted(map[string]interface{}{
		"type": "NOT-PRIVATE",
	}))

	assert.True(t, apiFilters["hosts-auto-update"].shouldConfigBePersisted(map[string]interface{}{}))
	assert.True(t, apiFilters["hosts-auto-update"].shouldConfigBePersisted(map[string]interface{}{
		"updateWindows": map[string]interface{}{},
	}))
	assert.False(t, apiFilters["hosts-auto-update"].shouldConfigBePersisted(map[string]interface{}{
		"updateWindows": map[string]interface{}{
			"windows": []interface{}{},
		},
	}))
	assert.True(t, apiFilters["hosts-auto-update"].shouldConfigBePersisted(map[string]interface{}{
		"updateWindows": map[string]interface{}{
			"windows": []interface{}{"some-entry"},
		},
	}))

	assert.True(t, apiFilters["anomaly-detection-metrics"].shouldBeSkippedPreDownload(api.Value{
		Id: "dynatrace.",
	}))
	assert.True(t, apiFilters["anomaly-detection-metrics"].shouldBeSkippedPreDownload(api.Value{
		Id: "ruxit.",
	}))
	assert.False(t, apiFilters["anomaly-detection-metrics"].shouldBeSkippedPreDownload(api.Value{
		Id: "somtehing.",
	}))
}
