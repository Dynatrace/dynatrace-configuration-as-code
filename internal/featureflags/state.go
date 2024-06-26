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
	for _, v := range Permanent {
		enabled, def := v.Value()
		if enabled != def {
			return true
		}
	}
	for _, v := range Temporary {
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

	_, _ = fmt.Fprintf(&s, "Feature Flags:\n\n")
	permanentKeys := maps.Keys(Permanent)
	slices.Sort(permanentKeys)
	for _, k := range permanentKeys {
		v := Permanent[k]
		enabled, def := v.Value()
		modifiedStr := " "
		if enabled != def {
			modifiedStr = "!"
		}
		_, _ = fmt.Fprintf(&s, "%v\t%v: %v (default:%v)\n", modifiedStr, v.envName, enabled, def)
	}

	_, _ = fmt.Fprintf(&s, "\n\nDevelopment and Experimental Flags:\n\n")
	tempKeys := maps.Keys(Temporary)
	slices.Sort(tempKeys)
	for _, k := range tempKeys {
		v := Temporary[k]
		enabled, def := v.Value()
		modifiedStr := " "
		if enabled != def {
			modifiedStr = "!"
		}
		_, _ = fmt.Fprintf(&s, "%v\t%v: %v (default:%v)\n", modifiedStr, v.envName, enabled, def)
	}

	return s.String()
}
