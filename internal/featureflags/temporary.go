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

const (
	// SkipReadOnlyAccountGroupUpdates toggles whether updates to read-only account groups are skipped or not.
	// Introduced: 2024-03-29; v2.13.0
	SkipReadOnlyAccountGroupUpdates FeatureFlag = "MONACO_SKIP_READ_ONLY_ACCOUNT_GROUP_UPDATES"
	// PersistSettingsOrder toggles whether insertAfter config parameter is persisted for ordered settings.
	// Introduced: 2024-05-15; v2.14.0
	PersistSettingsOrder FeatureFlag = "MONACO_FEAT_PERSIST_SETTINGS_ORDER"
	// OpenPipeline toggles whether openpipeline configurations are downloaded and / or deployed.
	// Introduced: 2024-06-10; v2.15.0
	OpenPipeline FeatureFlag = "MONACO_FEAT_OPENPIPELINE"
	// IgnoreSkippedConfigs toggles whether configurations that are marked to be skipped should also be excluded
	// from the dependency graph created by Monaco. These configs are not only skipped during deployment but also
	// not validated prior to deployment. Further, other configs cannot reference properties of this config anymore.
	IgnoreSkippedConfigs FeatureFlag = "MONACO_FEAT_IGNORE_SKIPPED_CONFIGS"
	// Segments toggles whether segment configurations are downloaded and / or deployed.
	// Introduced: v2.18.0
	Segments FeatureFlag = "MONACO_FEAT_SEGMENTS"
	// ServiceUsers toggles whether account service users configurations are downloaded and / or deployed.
	// Introduced: v2.18.0
	ServiceUsers FeatureFlag = "MONACO_FEAT_SERVICE_USERS"
	// OnlyCreateReferencesInStringValues toggles whether references are created arbitarily in JSON templates
	// or when enabled only in string values within the JSON.
	// Introduced: v2.19.0
	OnlyCreateReferencesInStringValues FeatureFlag = "MONACO_FEAT_ONLY_CREATE_REFERENCES_IN_STRINGS"
	// ServiceLevelObjective toggles whether slo configurations are downloaded and / or deployed.
	// Introduced: v2.19.0
	ServiceLevelObjective FeatureFlag = "MONACO_FEAT_SLO_V2"
)

// temporaryDefaultValues defines temporary feature flags and their default values.
// It is suitable for features that are hidden during development or have some uncertainty.
// These should always be removed after release of a feature, or some stabilization period if needed.
var temporaryDefaultValues = map[FeatureFlag]defaultValue{
	SkipReadOnlyAccountGroupUpdates:    false,
	PersistSettingsOrder:               true,
	OpenPipeline:                       true,
	IgnoreSkippedConfigs:               false,
	Segments:                           true,
	ServiceUsers:                       false,
	OnlyCreateReferencesInStringValues: false,
	ServiceLevelObjective:              false,
}
