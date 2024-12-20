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
	// Documents toggles whether documents are downloaded and / or deployed.
	// Introduced: 2024-04-16; v2.14.0
	Documents FeatureFlag = "MONACO_FEAT_DOCUMENTS"
	// DeleteDocuments toggles whether documents are deleted
	// Introduced: 2024-04-16; v2.14.2
	DeleteDocuments FeatureFlag = "MONACO_FEAT_DELETE_DOCUMENTS"
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
)

// temporary FeatureFlags - for features that are hidden during development or have some uncertainty.
// These should always be removed after release of a feature, or some stabilization period if needed.
var temporary = map[FeatureFlag]defaultValue{
	SkipReadOnlyAccountGroupUpdates: false,
	Documents:                       true,
	DeleteDocuments:                 true,
	PersistSettingsOrder:            true,
	OpenPipeline:                    true,
	IgnoreSkippedConfigs:            false,
	Segments:                        false,
}
