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

package attribute

import (
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// Attr is an additional custom attribute that can be used for structural logging output
type Attr struct {
	// Key is the key used for the attribute
	Key string
	// Value is the value used for the attribute and can be anything
	Value any
}

// Any creates a new custom attribute for the logger
func Any(key string, value any) Attr {
	return Attr{Key: key, Value: value}
}

// LogCoordinate is the type to be used to log coordinates as context attributes
type LogCoordinate struct {
	Reference string `json:"reference"`
	Project   string `json:"project"`
	Type      string `json:"type"`
	ConfigID  string `json:"configID"`
}

// Coordinate builds an attribute containing information taken from the provided coordinate
func Coordinate(coordinate coordinate.Coordinate) Attr {
	return Attr{"coordinate",
		LogCoordinate{
			coordinate.String(),
			coordinate.Project,
			coordinate.Type,
			coordinate.ConfigId,
		}}
}

// Type builds an attribute containing information about a config type. This is used in cases where no full coordinate exists,
// but only a config type is known - for example in download or deletion
func Type[X ~string](t X) Attr {
	return Attr{"type", t}
}

// Environment builds an attribute containing environment information for structured logging
func Environment(environment, group string) Attr {
	return Attr{"environment",
		struct {
			Group string `json:"group"`
			Name  string `json:"name"`
		}{
			group,
			environment,
		}}
}

// Error builds an attribute containing error information for structured logging
func Error(err error) Attr {
	return Attr{
		Key: "error",
		Value: struct {
			Type    string `json:"type"`
			Details string `json:"details"`
		}{
			Type:    fmt.Sprintf("%T", err),
			Details: err.Error(),
		}}
}
func StatusDeploying() Attr {
	return statusAttr("deploying")
}

func StatusDeployed() Attr {
	return statusAttr("deployed")
}

func StatusDeploymentFailed() Attr {
	return statusAttr("failed")
}

func StatusDeploymentSkipped() Attr {
	return statusAttr("skipped")
}

func statusAttr(statusValue string) Attr {
	return Attr{"deploymentStatus", statusValue}
}
