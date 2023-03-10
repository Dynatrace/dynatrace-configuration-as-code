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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_AllDefinedApiFiltersHaveApis(t *testing.T) {
	definedApis := api.NewAPIs()

	for apiId := range apiFilters {
		_, found := definedApis[apiId]

		assert.True(t, found, "Filtered API '%v' not defined in apis", apiId)
	}
}

func Test_ShouldBePersisted(t *testing.T) {
	t.Run("dashboard -  should not be persisted if its a preset", func(t *testing.T) {
		assert.False(t, apiFilters["dashboard"].shouldConfigBePersisted(map[string]interface{}{
			"dashboardMetadata": map[string]interface{}{
				"preset": true,
			},
		}))
	})

	t.Run("dashboard -  should be persisted if dashboardMetadata is missing", func(t *testing.T) {
		assert.True(t, apiFilters["dashboard"].shouldConfigBePersisted(map[string]interface{}{}))
	})

	t.Run("dashboard -  should be persisted if it's not a preset", func(t *testing.T) {
		assert.True(t, apiFilters["dashboard"].shouldConfigBePersisted(map[string]interface{}{
			"dashboardMetadata": map[string]interface{}{
				"preset": false,
			},
		}))
	})

	t.Run("dashboard -  should be persisted if dashboardMetadata.preset is missing", func(t *testing.T) {
		assert.True(t, apiFilters["dashboard"].shouldConfigBePersisted(map[string]interface{}{
			"dashboardMetadata": map[string]interface{}{},
		}))
	})

	t.Run("synthetic-location - should be persisted if it's private", func(t *testing.T) {
		assert.True(t, apiFilters["synthetic-location"].shouldConfigBePersisted(map[string]interface{}{
			"type": "PRIVATE",
		}))
	})

	t.Run("synthetic-location - should not be persisted if it's public", func(t *testing.T) {
		assert.False(t, apiFilters["synthetic-location"].shouldConfigBePersisted(map[string]interface{}{
			"type": "PUBLIC",
		}))
	})

	t.Run("hosts-auto-update - Empty update windows are not persisted", func(t *testing.T) {
		assert.False(t, apiFilters["hosts-auto-update"].shouldConfigBePersisted(map[string]interface{}{
			"updateWindows": map[string]interface{}{
				"windows": []interface{}{},
			},
		}))
	})

	t.Run("hosts-auto-update - Missing update windows are is persisted", func(t *testing.T) {
		assert.True(t, apiFilters["hosts-auto-update"].shouldConfigBePersisted(map[string]interface{}{}))
	})

	t.Run("hosts-auto-update - Windows with values are persisted", func(t *testing.T) {
		assert.True(t, apiFilters["hosts-auto-update"].shouldConfigBePersisted(map[string]interface{}{
			"updateWindows": map[string]interface{}{
				"windows": []interface{}{"1", "2", "3"},
			},
		}))
	})

}

func TestShouldConfigBeSkipped(t *testing.T) {

	t.Run("dashboard - Owner 'Dynatrace' is skipped", func(t *testing.T) {
		owner := "Dynatrace"
		assert.True(t, apiFilters["dashboard"].shouldBeSkippedPreDownload(client.Value{Owner: &owner}))
	})

	t.Run("dashboard - Owner 'Not Dynatrace' is not skipped", func(t *testing.T) {
		owner := "Not Dynatrace"
		assert.False(t, apiFilters["dashboard"].shouldBeSkippedPreDownload(client.Value{Owner: &owner}))
	})

	t.Run("dashboard - No owner is not skipped", func(t *testing.T) {
		assert.False(t, apiFilters["dashboard"].shouldBeSkippedPreDownload(client.Value{}))
	})

	t.Run("anomaly-detection-metrics - ruxit. should be skipped", func(t *testing.T) {
		assert.True(t, apiFilters["anomaly-detection-metrics"].shouldBeSkippedPreDownload(client.Value{
			Id: "ruxit.",
		}))
	})

	t.Run("anomaly-detection-metrics - dynatrace. should be skipped", func(t *testing.T) {
		assert.True(t, apiFilters["anomaly-detection-metrics"].shouldBeSkippedPreDownload(client.Value{
			Id: "dynatrace.",
		}))
	})

	t.Run("anomaly-detection-metrics - ids should not be skipped", func(t *testing.T) {
		assert.False(t, apiFilters["anomaly-detection-metrics"].shouldBeSkippedPreDownload(client.Value{
			Id: "b836ff25-24e3-496d-8dce-d94110815ab5",
		}))
	})

	t.Run("anomaly-detection-metrics - random strings should not be skipped", func(t *testing.T) {
		assert.False(t, apiFilters["anomaly-detection-metrics"].shouldBeSkippedPreDownload(client.Value{
			Id: "test.something",
		}))
	})
}
