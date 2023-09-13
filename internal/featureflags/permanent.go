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
