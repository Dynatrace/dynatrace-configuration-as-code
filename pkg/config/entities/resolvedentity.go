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

package entities

import (
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

// ResolvedEntity represents the Dynatrace configuration entity of a config.Config
type ResolvedEntity struct {
	// Coordinate of the config.Config this entity represents
	Coordinate coordinate.Coordinate

	// Properties defines a map of all already resolved parameters
	Properties parameter.Properties

	// Skip flag indicating that this entity was skipped
	// if an entity is skipped, there will be no properties
	Skip bool
}

// ResolvePropValue retrieves the value associated with the specified key in a nested map.

// The ResolvePropValue function is designed to work with nested maps where keys are
// structured using dots to represent nested levels. It recursively traverses the
// nested maps to find the value associated with the specified key.
//
// If the key is found, the function returns the associated value and true. If the
// key is not found, it returns nil and false.
func ResolvePropValue(key string, props any) (any, bool) {
	first, rest, _ := strings.Cut(key, ".")

	switch valMap := props.(type) {
	case map[any]any:
		if p, found := valMap[first]; found {
			if rest == "" {
				return p, true
			}
			return ResolvePropValue(rest, p)
		}
	case map[string]any:
		if p, found := valMap[first]; found {
			if rest == "" {
				return p, true
			}
			return ResolvePropValue(rest, p)
		}
	}
	// Key not found
	return nil, false
}
