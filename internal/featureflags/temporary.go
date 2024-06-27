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

// UnescapeOnConvert toggles whether converting will remove escape chars from v1 values.
// Introduced: 2023-09-01; v2.8.0
func UnescapeOnConvert() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_UNESCAPE_ON_CONVERT",
		defaultEnabled: true,
	}
}

// UpdateNonUniqueByNameIfSingleOneExists toggles whether we attempt update api.API configurations with NonUniqueName,
// by name if only a single one is found on the environment. As this causes issues if a project defines more than one config
// with the same name - they will overwrite each other, and keep a single on the environment - the feature flag is introduced
// to turn it off until a generally better solution is available.
// Introduced: 2023-09-01; v2.9.1
func UpdateNonUniqueByNameIfSingleOneExists() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_UPDATE_SINGLE_NON_UNIQUE_BY_NAME",
		defaultEnabled: true,
	}
}

func AccountManagement() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_ACCOUNT_MANAGEMENT",
		defaultEnabled: true,
	}
}

// GenerateJSONSchemas toggles whether the 'generate schema' command is available
func GenerateJSONSchemas() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_JSON_SCHEMAS",
		defaultEnabled: true,
	}
}

// DashboardShareSettings toggles whether the dashboard share settings are downloaded and / or deployed.
// Introduced: 2024-02-29; v2.12.0
func DashboardShareSettings() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_DASHBOARD_SHARE_SETTINGS",
		defaultEnabled: true,
	}
}

// KeyUserActionsWeb toggles whether the key user actions for web apps are downloaded and / or deployed.
// Introduced: 2024-03-21; v2.12.0
func KeyUserActionsWeb() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_KUA_WEB",
		defaultEnabled: true,
	}
}

// KeyUserActionsMobile toggles whether the key user actions for mobile apps are downloaded and / or deployed.
// Introduced: 2024-03-21; v2.12.0
func KeyUserActionsMobile() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_KUA_MOBILE",
		defaultEnabled: true,
	}
}

// UserActionSessionPropertiesMobile toggles whether user actions and session properties for mobile apps are downloaded and / or deployed.
// Introduced: 2024-03-21; v2.12.0
func UserActionSessionPropertiesMobile() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_UASP_MOBILE",
		defaultEnabled: true,
	}
}

// SkipReadOnlyAccountGroupUpdates toggles whether updates to read-only account groups are skipped or not.
// Introduced: 2024-03-29; v2.13.0
func SkipReadOnlyAccountGroupUpdates() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_SKIP_READ_ONLY_ACCOUNT_GROUP_UPDATES",
		defaultEnabled: false,
	}
}

// Documents toggles whether documents are downloaded and / or deployed.
// Introduced: 2024-04-16; v2.14.0
func Documents() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_DOCUMENTS",
		defaultEnabled: false,
	}
}

// DeleteDocuments toggles whether documents are deleted
// Introduced: 2024-04-16; v2.14.2
func DeleteDocuments() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_DELETE_DOCUMENTS",
		defaultEnabled: false,
	}
}

// EventQueue toggles whether the event reporting feature of monato is enabled or not
func EventQueue() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_EVENTQUEUE",
		defaultEnabled: false,
	}
}

// Documents toggles whether insertAfter config parameter is persisted for ordered settings.
// Introduced: 2024-05-15; v2.14.0
func PersistSettingsOrder() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_PERSIST_SETTINGS_ORDER",
		defaultEnabled: false,
	}
}

// OpenPipeline toggles whether openpipeline configurations are downloaded and / or deployed.
// Introduced: 2024-06-10; v2.15.0
func OpenPipeline() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_OPENPIPELINE",
		defaultEnabled: false,
	}
}
