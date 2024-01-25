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

package pointer

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// DeletePointer contains all data needed to identify an object to be deleted from a Dynatrace environment.
// DeletePointer is similar but not fully equivalent to config.Coordinate as it may contain an Identifier that is either
// a Name or a ConfigID - only in case of a ConfigID is it actually equivalent to a Coordinate
type DeletePointer struct {
	Project string
	Type    string

	//Identifier will either be the Name of a classic Config API object, or a configID for newer types like Settings
	Identifier string

	// Scope is the Entity ID / information necessary to delete the entity. This is required for sub-path entities.
	Scope string
}

func (d DeletePointer) AsCoordinate() coordinate.Coordinate {
	return coordinate.Coordinate{
		Project:  d.Project,
		Type:     d.Type,
		ConfigId: d.Identifier,
	}
}

func (d DeletePointer) String() string {
	if d.Project != "" {
		return d.AsCoordinate().String()
	}
	return fmt.Sprintf("%s:%s", d.Type, d.Identifier)
}
