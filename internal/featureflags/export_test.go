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

// New creates a new FeatureFlag
// envName is the environment variable the feature flag is loading the values from when evaluated
// defaultEnabled defines whether the feature flag is enabled or not by default
func New(envName string, defaultEnabled bool) FeatureFlag {
	return FeatureFlag{
		envName:        envName,
		defaultEnabled: defaultEnabled,
	}
}
