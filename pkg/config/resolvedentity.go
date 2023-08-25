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

package config

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

// ResolvedEntity represents the Dynatrace configuration entity of a config.Config
type ResolvedEntity struct {
	// EntityName is the name returned by the Dynatrace api. In theory should be the
	// same as the `name` property defined in the configuration, but
	// can differ.
	EntityName string

	// Coordinate of the config.Config this entity represents
	Coordinate coordinate.Coordinate

	// Properties defines a map of all already resolved parameters
	Properties parameter.Properties

	// Skip flag indicating that this entity was skipped
	// if an entity is skipped, there will be no properties
	Skip bool
}
