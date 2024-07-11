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

/*
 * This file groups 'permanent' flags - features we want to be able to toggle long-term, instead of removing them after a stabilization period.
 */

// DangerousCommands returns the feature flag that tells whether dangerous commands for the CLI are enabled or not
func DangerousCommands() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_ENABLE_DANGEROUS_COMMANDS",
		defaultEnabled: false,
	}
}

// FastDependencyResolver returns the feature flag controlling whether the fast (but memory intensive) Aho-Corasick
// algorithm based dependency resolver is used when downloading. If set to false, the old naive and CPU intensive resolver
// is used. This flag is permanent as the fast resolver has significant memory cost.
func FastDependencyResolver() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_FAST_DEPENDENCY_RESOLVER",
		defaultEnabled: false,
	}
}

// DownloadFilter returns the feature flag controlling whether download filters out configurations that we believe can't
// be managed by config-as-code. Some users may still want to download everything on an environment, and turning off the
// filters allows them to do so.
func DownloadFilter() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_DOWNLOAD_FILTER",
		defaultEnabled: true,
	}
}

// DownloadFilterSettings returns the feature flag controlling whether general filters are applied to Settings download.
func DownloadFilterSettings() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_DOWNLOAD_FILTER_SETTINGS",
		defaultEnabled: true,
	}
}

// DownloadFilterSettingsUnmodifiable returns the feature flag controlling whether Settings marked as unmodifiable by
// their dtclient.SettingsModificationInfo are filtered out on download.
func DownloadFilterSettingsUnmodifiable() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_DOWNLOAD_FILTER_SETTINGS_UNMODIFIABLE",
		defaultEnabled: true,
	}
}

// DownloadFilterClassicConfigs returns the feature flag controlling whether download filters are applied to Classic Config API download.
func DownloadFilterClassicConfigs() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_DOWNLOAD_FILTER_CLASSIC_CONFIGS",
		defaultEnabled: true,
	}
}

// SkipVersionCheck returns the feature flag to control disabling the version check that happens at the end of each monaco run
func SkipVersionCheck() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_SKIP_VERSION_CHECK",
		defaultEnabled: false,
	}
}

// ExtractScopeAsParameter returns the feature flag to controlling whether the scope field of setting 2.0 objects shall be extracted as monaco parameter
func ExtractScopeAsParameter() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_EXTRACT_SCOPE_AS_PARAMETER",
		defaultEnabled: false,
	}
}

// BuildSimpleClassicURL returns the feature flag to controlling whether we attempt to create the Classic URL of a platform environment via string replacement before using the metadata API.
// As there may be networking/DNS edge-cases where the replaced URL is valid (GET returns 200) but is not actually a Classic environment, this feature flag allows deactivation of the feature.
func BuildSimpleClassicURL() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_SIMPLE_CLASSIC_URL",
		defaultEnabled: true,
	}
}

// LogToFile returns the feature flag to control whether log files shall be created or not
func LogToFile() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_LOG_FILE_ENABLED",
		defaultEnabled: true,
	}
}

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
