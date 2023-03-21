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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"strings"
)

type apiFilter struct {
	// shouldBeSkippedPreDownload is an optional callback indicating that a config should not be downloaded after the list of the configs
	shouldBeSkippedPreDownload func(value client.Value) bool

	// shouldConfigBePersisted is an optional callback to check whether a config should be persisted after being downloaded
	shouldConfigBePersisted func(json map[string]interface{}) bool
}

var apiFilters = map[string]apiFilter{
	"dashboard": {
		shouldBeSkippedPreDownload: func(value client.Value) bool {
			return value.Owner != nil && *value.Owner == "Dynatrace"
		},
		shouldConfigBePersisted: func(json map[string]interface{}) bool {
			if json["dashboardMetadata"] != nil {
				metadata := json["dashboardMetadata"].(map[string]interface{})

				if metadata["preset"] != nil && metadata["preset"] == true {
					return false
				}
			}

			return true
		},
	},
	"synthetic-location": {
		shouldConfigBePersisted: func(json map[string]interface{}) bool {
			return json["type"] == "PRIVATE"
		},
	},
	"hosts-auto-update": {
		// check that the property 'updateWindows' is not empty, otherwise discard.
		shouldConfigBePersisted: func(json map[string]interface{}) bool {
			autoUpdates, ok := json["updateWindows"]
			if !ok {
				return true
			}

			windows, ok := autoUpdates.(map[string]interface{})["windows"].([]interface{})
			if !ok {
				return true
			}

			return len(windows) > 0
		},
	},
	"anomaly-detection-metrics": {
		shouldBeSkippedPreDownload: func(value client.Value) bool {
			return strings.HasPrefix(value.Id, "dynatrace.") || strings.HasPrefix(value.Id, "ruxit.")
		},
	},
}
