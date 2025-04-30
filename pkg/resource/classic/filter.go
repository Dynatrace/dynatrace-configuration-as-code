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
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

// ContentFilter defines whether a given API value should be skipped - either already PreDownload or based on it's full json content
type ContentFilter struct {
	// ShouldBeSkippedPreDownload is an optional callback indicating that a config should not be downloaded after the list of the configs
	ShouldBeSkippedPreDownload func(value dtclient.Value) bool

	// ShouldConfigBePersisted is an optional callback to check whether a config should be persisted after being downloaded
	ShouldConfigBePersisted func(json map[string]interface{}) bool
}

type ContentFilters map[string]ContentFilter

// ApiContentFilters defines default ContentFilter rules per API identifier
var ApiContentFilters = map[string]ContentFilter{
	api.Dashboard: {
		ShouldBeSkippedPreDownload: func(value dtclient.Value) bool {
			return value.Owner != nil && *value.Owner == "Dynatrace"
		},
		ShouldConfigBePersisted: func(json map[string]interface{}) bool {
			if json["dashboardMetadata"] != nil {
				metadata := json["dashboardMetadata"].(map[string]interface{})

				if metadata["preset"] != nil && metadata["preset"] == true && metadata["owner"] == "Dynatrace" {
					return false
				}
			}

			return true
		},
	},
	api.SyntheticLocation: {
		ShouldConfigBePersisted: func(json map[string]interface{}) bool {
			return json["type"] == "PRIVATE"
		},
	},
	api.HostsAutoUpdate: {
		// check that the property 'updateWindows' is not empty, otherwise discard.
		ShouldConfigBePersisted: func(json map[string]interface{}) bool {
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
	api.AnomalyDetectionMetrics: {
		ShouldBeSkippedPreDownload: func(value dtclient.Value) bool {
			return strings.HasPrefix(value.Id, "dynatrace.") || strings.HasPrefix(value.Id, "ruxit.")
		},
	},
	api.NetworkZone: {
		ShouldBeSkippedPreDownload: func(value dtclient.Value) bool {
			return value.Id == "default"
		},
	},
}
