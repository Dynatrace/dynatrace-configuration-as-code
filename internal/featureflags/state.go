/*
 * @license
 * Copyright 2024 Dynatrace LLC
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
	"golang.org/x/exp/maps"
	"slices"
	"strings"
)

// AnyModified returns true if any Permanent or Temporary feature flag value is different to its default.
func AnyModified() bool {
	return anyFeatureFlagModified(Permanent) || anyFeatureFlagModified(Temporary)
}

// anyFeatureFlagModified returns true if any feature flag value is different to its default.
func anyFeatureFlagModified[K TemporaryFlag | PermanentFlag](featureFlags map[K]FeatureFlag) bool {
	for _, v := range featureFlags {
		enabled, def := v.Value()
		if enabled != def {
			return true
		}
	}
	return false
}

// StateInfo builds a string message describing the current and default values of all feature flags,
// noting especially if any flag has been changed off its default value.
func StateInfo() string {
	s := strings.Builder{}

	if AnyModified() {
		s.WriteString("Lines starting with '!' indicate that a flag has been modified from its default value.\n\n")
	}

	s.WriteString("Feature Flags:\n\n")
	s.WriteString(makeFeatureFlagTableString(Permanent))

	s.WriteString("\n\nDevelopment and Experimental Flags:\n\n")
	s.WriteString(makeFeatureFlagTableString(Temporary))

	return s.String()
}

func makeFeatureFlagTableString[K TemporaryFlag | PermanentFlag](featureFlags map[K]FeatureFlag) string {
	s := strings.Builder{}
	keys := maps.Keys(featureFlags)
	slices.Sort(keys)
	for _, k := range keys {
		v := featureFlags[k]
		enabled, def := v.Value()
		modifiedStr := " "
		if enabled != def {
			modifiedStr = "!"
		}
		_, _ = fmt.Fprintf(&s, "%v\t%v: %v (default:%v)\n", modifiedStr, v.envName, enabled, def)
	}
	return s.String()
}
