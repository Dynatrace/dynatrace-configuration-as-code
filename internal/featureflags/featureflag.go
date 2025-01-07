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
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
)

type (
	// FeatureFlag represents a command line switch to turn certain features
	// ON or OFF. Values are read from environment variables defined by
	// the feature flag. The feature flag can have default value which is used
	// when the resp. environment variable does not exist
	FeatureFlag string

	defaultValue = bool
)

func (ff FeatureFlag) String() string {
	return ff.EnvName()
}

// EnvName gives back the environment variable name for the feature flag
func (ff FeatureFlag) EnvName() string {
	return string(ff)
}

// Enabled look up between known temporary and permanent flags and evaluates it.
// Feature flags are considered to be "enabled" if their resp. environment variable
// is set to 1, t, T, TRUE, true or True.
// Feature flags are considered to be "disabled" if their resp. environment variable
// is set to 0, f, F, FALSE, false or False.
func (ff FeatureFlag) Enabled() bool {
	v, exists := temporaryDefaultValues[ff]
	if exists {
		_, exists = permanentDefaultValues[ff]
		if exists {
			panic(fmt.Sprintf("feature flag %s defined as temporary and permanent", ff))
		}
		return enabled(ff, v)
	}

	v, exists = permanentDefaultValues[ff]
	if exists {
		return enabled(ff, v)
	}

	panic(fmt.Sprintf("unknown feature flag %s", ff))
}

// enabled evaluates the feature flag.
// Feature flags are considered to be "enabled" if their resp. environment variable
// is set to 1, t, T, TRUE, true or True.
// Feature flags are considered to be "disabled" if their resp. environment variable
// is set to 0, f, F, FALSE, false or False.
func enabled(ff FeatureFlag, d defaultValue) bool {
	if val, ok := os.LookupEnv(ff.EnvName()); ok {
		value, err := strconv.ParseBool(strings.ToLower(val))
		if err != nil {
			log.Warn("Unsupported value %q for feature flag %q. Using default value: %v", val, ff, d)
			return d
		}
		return value
	}
	return d
}
