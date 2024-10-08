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
	"fmt"
)

// noOpFilter is a settings 2.0 filter that does nothing
var noOpFilter = Filter{
	ShouldDiscard: func(settingsValue map[string]interface{}) (bool, string) { return false, "" },
}

// Filter can be used to filter/discard settings 2.0
type Filter struct {
	// ShouldDiscard contains logic whether a settings object should be discarded
	// based on specific criteria on the settings value payload. It returns true or false
	// depending on the specific implementation and a reason that gives more context to the
	// evaluation result, e.g. a filter that discards settings that contain a field "foo"
	// with a value "bar" in their payload would be implemented like:
	// func (json map[string]interface{}) (bool, string) { return json["foo"] == "bar",  "foo is set to bar" }
	ShouldDiscard func(map[string]interface{}) (discard bool, reason string)
}

// Filters represents a map of settings 2.0 Filters
type Filters map[string]Filter

// Get returns the filter for a given key
func (f Filters) Get(schemaID string) Filter {
	if filter, ok := f[schemaID]; ok {
		return filter
	}
	return noOpFilter
}

func formatDefaultDiscardReasonMsg(entityName interface{}) string {
	return fmt.Sprintf("%q cannot be managed via configuration as code", entityName)
}

// DefaultSettingsFilters is the default Filters used for settings 2.0
var DefaultSettingsFilters = Filters{
	"builtin:logmonitoring.logs-on-grail-activate": {
		ShouldDiscard: func(json map[string]interface{}) (bool, string) {
			return json["activated"] == false, "'activated' field is set to false"
		},
	},
	// following settings20 obj needs to be discarded bc of error during deploy:
	// "Given property 'matcher' with value: '*' Invalid DQL query: token recognition error at: '*' at 1:0"
	"builtin:logmonitoring.log-buckets-rules": {
		ShouldDiscard: func(json map[string]interface{}) (bool, string) {
			return json["ruleName"] == "default", formatDefaultDiscardReasonMsg(json["ruleName"])
		},
	},
	// following settings20 obj needs to be discarded bc of error during deploy:
	// "Given property 'matcher' with value: '*' Invalid DQL query: token recognition error at: '*' at 1:0"
	"builtin:bizevents-processing-buckets.rule": {
		ShouldDiscard: func(json map[string]interface{}) (bool, string) {
			return json["ruleName"] == "default", formatDefaultDiscardReasonMsg(json["ruleName"])
		},
	},
	// following settings20 obj needs to be discarded bc it is strictly read only and causes problems during deploy:
	// "cannot be modified"
	"builtin:alerting.profile": {
		ShouldDiscard: func(json map[string]interface{}) (bool, string) {
			return json["name"] == "Default" || json["name"] == "Default for ActiveGate Token Expiry", formatDefaultDiscardReasonMsg(json["name"])
		},
	},
	// following settings20 obj needs to be discarded bc it is strictly read only and causes problems during deploy:
	// "cannot be modified"
	"builtin:logmonitoring.log-events": {
		ShouldDiscard: func(json map[string]interface{}) (bool, string) {
			return json["summary"] == "Default Kubernetes Log Events", formatDefaultDiscardReasonMsg("Default Kubernetes Log Events")
		},
	},

	// builtin:host.monitoring.mode is not reliable during download, what's why we decided to skip it by default
	"builtin:host.monitoring.mode": {
		ShouldDiscard: func(json map[string]interface{}) (bool, string) {
			return true, formatDefaultDiscardReasonMsg("Monitoring mode")
		},
	},
}
