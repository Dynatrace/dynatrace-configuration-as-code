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
	"slices"
	"strings"

	"golang.org/x/exp/maps"
)

// AnyModified returns true if any feature flag value is different to its default.
func AnyModified() bool {
	return anyFeatureFlagModified(permanent) || anyFeatureFlagModified(temporary)
}

// anyFeatureFlagModified returns true if any feature flag value is different to its default.
func anyFeatureFlagModified(featureFlags map[FeatureFlag]defaultValue) bool {
	for ff, d := range featureFlags {
		if ff.Enabled() != d {
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
	s.WriteString(makeFeatureFlagTableString(permanent))

	s.WriteString("\n\nDevelopment and Experimental Flags:\n\n")
	s.WriteString(makeFeatureFlagTableString(temporary))

	return s.String()
}

func makeFeatureFlagTableString(featureFlags map[FeatureFlag]defaultValue) string {
	s := strings.Builder{}
	flags := maps.Keys(featureFlags)
	slices.Sort(flags)
	for _, f := range flags {
		modifiedStr := " "
		if f.Enabled() != featureFlags[f] {
			modifiedStr = "!"
		}
		_, _ = fmt.Fprintf(&s, "%v\t%v: %v (default:%v)\n", modifiedStr, f, f.Enabled(), featureFlags[f])
	}
	return s.String()
}
