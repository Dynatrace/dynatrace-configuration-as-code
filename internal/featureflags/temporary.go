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

package featureflags

// VerifyEnvironmentType returns the feature flag that tells whether the environment check
// at the beginning of execution is enabled or not.
// Introduced: before 2023-04-27; v2.0.0
func VerifyEnvironmentType() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_VERIFY_ENV_TYPE",
		defaultEnabled: true,
	}
}

// ManagementZoneSettingsNumericIDs returns the feature flag that tells whether configs of settings type builtin:management-zones
// are addressed directly via their object ID or their resolved numeric ID when they are referenced.
// Introduced: 2023-04-18; v2.0.1
func ManagementZoneSettingsNumericIDs() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_USE_MZ_NUMERIC_ID",
		defaultEnabled: true,
	}
}

// ConsistentUUIDGeneration returns the feature flag controlling whether generated UUIDs use consistent separator characters regardless of OS
// This is default true and just exists to get old, technically buggy behavior on Windows again if needed.
// Introduced: 2023-05-25; v2.2.0
func ConsistentUUIDGeneration() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_CONSISTENT_UUID_GENERATION",
		defaultEnabled: true,
	}
}

// Buckets toggles whether the Grail bucket type can be used.
// Introduced: (inactive) 2023-08-09 ->
func Buckets() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_BUCKETS",
		defaultEnabled: true,
	}
}

// UnescapeOnConvert toggles whether converting will remove escape chars from v1 values.
// Introduced: 2023-09-01; v2.8.0
func UnescapeOnConvert() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_UNESCAPE_ON_CONVERT",
		defaultEnabled: true,
	}
}
