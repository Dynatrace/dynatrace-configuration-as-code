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

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

func Test_AllDefinedApiFiltersHaveApis(t *testing.T) {
	definedApis := api.NewAPIs()

	for apiId := range ApiContentFilters {
		_, found := definedApis[apiId]

		assert.True(t, found, "Filtered API '%v' not defined in apis", apiId)
	}
}

func Test_ShouldBePersisted(t *testing.T) {
	t.Run("dashboard -  should not be persisted if its a preset owned by Dynatrace", func(t *testing.T) {
		assert.False(t, ApiContentFilters["dashboard"].ShouldConfigBePersisted(map[string]interface{}{
			"dashboardMetadata": map[string]interface{}{
				"preset": true,
				"owner":  "Dynatrace",
			},
		}))
	})

	t.Run("dashboard -  should  be persisted if its a preset owned by User", func(t *testing.T) {
		assert.True(t, ApiContentFilters["dashboard"].ShouldConfigBePersisted(map[string]interface{}{
			"dashboardMetadata": map[string]interface{}{
				"preset": true,
				"owner":  "Not Dynatrace",
			},
		}))
	})

	t.Run("dashboard -  should be persisted if dashboardMetadata is missing", func(t *testing.T) {
		assert.True(t, ApiContentFilters["dashboard"].ShouldConfigBePersisted(map[string]interface{}{}))
	})

	t.Run("dashboard -  should be persisted if it's not a preset", func(t *testing.T) {
		assert.True(t, ApiContentFilters["dashboard"].ShouldConfigBePersisted(map[string]interface{}{
			"dashboardMetadata": map[string]interface{}{
				"preset": false,
			},
		}))
	})

	t.Run("dashboard -  should be persisted if dashboardMetadata.preset is missing", func(t *testing.T) {
		assert.True(t, ApiContentFilters["dashboard"].ShouldConfigBePersisted(map[string]interface{}{
			"dashboardMetadata": map[string]interface{}{},
		}))
	})

	t.Run("synthetic-location - should be persisted if it's private", func(t *testing.T) {
		assert.True(t, ApiContentFilters["synthetic-location"].ShouldConfigBePersisted(map[string]interface{}{
			"type": "PRIVATE",
		}))
	})

	t.Run("synthetic-location - should not be persisted if it's public", func(t *testing.T) {
		assert.False(t, ApiContentFilters["synthetic-location"].ShouldConfigBePersisted(map[string]interface{}{
			"type": "PUBLIC",
		}))
	})

	t.Run("hosts-auto-update - Empty update windows are not persisted", func(t *testing.T) {
		assert.False(t, ApiContentFilters["hosts-auto-update"].ShouldConfigBePersisted(map[string]interface{}{
			"updateWindows": map[string]interface{}{
				"windows": []interface{}{},
			},
		}))
	})

	t.Run("hosts-auto-update - Missing update windows are is persisted", func(t *testing.T) {
		assert.True(t, ApiContentFilters["hosts-auto-update"].ShouldConfigBePersisted(map[string]interface{}{}))
	})

	t.Run("hosts-auto-update - Windows with values are persisted", func(t *testing.T) {
		assert.True(t, ApiContentFilters["hosts-auto-update"].ShouldConfigBePersisted(map[string]interface{}{
			"updateWindows": map[string]interface{}{
				"windows": []interface{}{"1", "2", "3"},
			},
		}))
	})

}

func TestShouldConfigBeSkipped(t *testing.T) {

	t.Run("dashboard - Owner 'Dynatrace' is skipped", func(t *testing.T) {
		owner := "Dynatrace"
		assert.True(t, ApiContentFilters["dashboard"].ShouldBeSkippedPreDownload(dtclient.Value{Owner: &owner}))
	})

	t.Run("dashboard - Owner 'Not Dynatrace' is not skipped", func(t *testing.T) {
		owner := "Not Dynatrace"
		assert.False(t, ApiContentFilters["dashboard"].ShouldBeSkippedPreDownload(dtclient.Value{Owner: &owner}))
	})

	t.Run("dashboard - No owner is not skipped", func(t *testing.T) {
		assert.False(t, ApiContentFilters["dashboard"].ShouldBeSkippedPreDownload(dtclient.Value{}))
	})

	t.Run("anomaly-detection-metrics - ruxit. should be skipped", func(t *testing.T) {
		assert.True(t, ApiContentFilters["anomaly-detection-metrics"].ShouldBeSkippedPreDownload(dtclient.Value{
			Id: "ruxit.",
		}))
	})

	t.Run("anomaly-detection-metrics - dynatrace. should be skipped", func(t *testing.T) {
		assert.True(t, ApiContentFilters["anomaly-detection-metrics"].ShouldBeSkippedPreDownload(dtclient.Value{
			Id: "dynatrace.",
		}))
	})

	t.Run("anomaly-detection-metrics - ids should not be skipped", func(t *testing.T) {
		assert.False(t, ApiContentFilters["anomaly-detection-metrics"].ShouldBeSkippedPreDownload(dtclient.Value{
			Id: "b836ff25-24e3-496d-8dce-d94110815ab5",
		}))
	})

	t.Run("anomaly-detection-metrics - random strings should not be skipped", func(t *testing.T) {
		assert.False(t, ApiContentFilters["anomaly-detection-metrics"].ShouldBeSkippedPreDownload(dtclient.Value{
			Id: "test.something",
		}))
	})

	t.Run("default network zone - should be skipped", func(t *testing.T) {
		assert.True(t, ApiContentFilters["network-zone"].ShouldBeSkippedPreDownload(dtclient.Value{
			Id: "default",
		}))
	})
}
