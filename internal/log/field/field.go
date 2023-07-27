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

package field

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// Field is an additional custom field that can be used for structural logging output
type Field struct {
	// Key is the key used for the field
	Key string
	// Value is the value used for the field and can be anything
	Value any
}

// F creates a new custom field for the logger
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// LogCoordinate is the type to be used to log coordinates as context fields
type LogCoordinate struct {
	Reference string `json:"reference"`
	Project   string `json:"project"`
	Type      string `json:"type"`
	ConfigID  string `json:"configID"`
}

// Coordinate builds a Field containing information taken from the provided coordinate
func Coordinate(coordinate coordinate.Coordinate) Field {
	return Field{"coordinate",
		LogCoordinate{
			coordinate.String(),
			coordinate.Project,
			coordinate.Type,
			coordinate.ConfigId,
		}}
}

// Type builds a Field containing information about a config type. This is used in cases where no full coordinate exists,
// but only a config type is known - for example in download or deletion
func Type(t string) Field {
	return Field{"type", t}
}

// Environment builds a Field containing environment information for structured logging
func Environment(environment, group string) Field {
	return Field{"environment",
		struct {
			Group string `json:"group"`
			Name  string `json:"name"`
		}{
			group,
			environment,
		}}
}

// Error builds a Field containing error information for structured logging
func Error(err error) Field {
	return Field{"error",
		struct {
			Type    string `json:"type"`
			Details error  `json:"details"`
		}{
			fmt.Sprintf("%T", err),
			err,
		}}
}
