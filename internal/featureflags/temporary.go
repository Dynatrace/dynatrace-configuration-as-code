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
