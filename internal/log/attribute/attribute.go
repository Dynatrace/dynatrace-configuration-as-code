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
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// LogCoordinate is the type to be used to log coordinates as context attributes
type LogCoordinate struct {
	Reference string `json:"reference"`
	Project   string `json:"project"`
	Type      string `json:"type"`
	ConfigID  string `json:"configID"`
}

// Coordinate builds an attribute containing information taken from the provided coordinate
func Coordinate(coordinate coordinate.Coordinate) slog.Attr {
	return slog.Any("coordinate",
		LogCoordinate{
			coordinate.String(),
			coordinate.Project,
			coordinate.Type,
			coordinate.ConfigId,
		})
}

// Type builds an attribute containing information about a config type. This is used in cases where no full coordinate exists,
// but only a config type is known - for example in download or deletion
func Type[X ~string](t X) slog.Attr {
	return slog.Any("type", t)
}

// Environment builds an attribute containing environment information for structured logging
func Environment(environment, group string) slog.Attr {
	return slog.Any("environment",
		struct {
			Group string `json:"group"`
			Name  string `json:"name"`
		}{
			group,
			environment,
		})
}

// Error builds an attribute containing error information for structured logging
func Error(err error) slog.Attr {
	return slog.Any(
		"error",
		struct {
			Type    string `json:"type"`
			Details string `json:"details"`
		}{
			Type:    fmt.Sprintf("%T", err),
			Details: err.Error(),
		})
}

const deploymentStatus = "deploymentStatus"

func StatusDeploying() slog.Attr {
	return slog.Any(deploymentStatus, "deploying")
}

func StatusDeployed() slog.Attr {
	return slog.Any(deploymentStatus, "deployed")
}

func StatusDeploymentFailed() slog.Attr {
	return slog.Any(deploymentStatus, "failed")
}

func StatusDeploymentSkipped() slog.Attr {
	return slog.Any(deploymentStatus, "skipped")
}
