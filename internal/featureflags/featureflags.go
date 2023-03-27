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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
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

// New creates a new FeatureFlag
// envName is the environment variable the feature flag is loading the values from when evaluated
// defaultEnabled defines whether the feature flag is enabled or not by default
func New(envName string, defaultEnabled bool) FeatureFlag {
	return FeatureFlag{
		envName:        envName,
		defaultEnabled: defaultEnabled,
	}
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

// Entities returns the feature flag that tells whether Dynatrace Entities download/matching is enabled or not
func Entities() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_ENTITIES",
		defaultEnabled: false,
	}
}

// DangerousCommands returns the feature flag that tells whether dangerous commands for the CLI are enabled or not
func DangerousCommands() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_ENABLE_DANGEROUS_COMMANDS",
		defaultEnabled: false,
	}
}

// VerifyEnvironmentType returns the feature flag that tells whether the environment check
// at the beginning of execution is enabled or not
func VerifyEnvironmentType() FeatureFlag {
	return FeatureFlag{
		envName:        "MONACO_FEAT_VERIFY_ENV_TYPE",
		defaultEnabled: true,
	}
}
