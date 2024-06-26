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

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"os"
	"strconv"
	"strings"
)

// FeatureFlag represents a command line switch to turn certain features
// ON or OFF. Values are read from environment variables defined by
// the feature flag. The feature flag can have default values that are used
// when the resp. environment variable does not exist
type FeatureFlag struct {
	// envName is the environment variable name
	// that is used to read the value from
	envName string
	// defaultEnabled states whether this feature flag
	// is enabled or disabled by default
	defaultEnabled bool
}

// Enabled evaluates the feature flag.
// Feature flags are considered to be "enabled" if their resp. environment variable
// is set to 1, t, T, TRUE, true or True.
// Feature flags are considered to be "disabled" if their resp. environment variable
// is set to 0, f, F, FALSE, false or False.
func (ff FeatureFlag) Enabled() bool {
	if val, ok := os.LookupEnv(ff.envName); ok {
		enabled, err := strconv.ParseBool(strings.ToLower(val))
		if err != nil {
			log.Warn("Unsupported value %q for feature flag %q. Using default value: %v", val, ff.envName, ff.defaultEnabled)
			return ff.defaultEnabled
		}
		return enabled
	}
	return ff.defaultEnabled
}

// EnvName gives back the environment variable name for
// the feature flag
func (ff FeatureFlag) EnvName() string {
	return ff.envName
}

// Value returns the current value and default value of a FeatureFlag
func (ff FeatureFlag) Value() (enabled bool, defaultVal bool) {
	return ff.Enabled(), ff.defaultEnabled
}
