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

// CoordinateAttr returns an attribute containing information taken from the provided coordinate.
func CoordinateAttr(coordinate coordinate.Coordinate) slog.Attr {
	return slog.Any("coordinate", coordinate)
}

// TypeAttr returns an attribute containing information about a config type. This is used in cases where no full coordinate exists,
// but only a config type is known - for example in download or deletion.
func TypeAttr[X ~string](t X) slog.Attr {
	return slog.Any("type", t)
}

// EnvironmentAttr returns an attribute containing environment information for structured logging.
func EnvironmentAttr(environment, group string) slog.Attr {
	return slog.Any("environment",
		slog.GroupValue(
			slog.String("group", group),
			slog.String("name", environment)))
}

// ErrorAttr returns an attribute containing error information for structured logging.
func ErrorAttr(err error) slog.Attr {
	return slog.Any(
		"error",
		slog.GroupValue(
			slog.String("type", fmt.Sprintf("%T", err)),
			slog.String("details", err.Error())))
}

const deploymentStatus = "deploymentStatus"

// StatusDeployingAttr returns an attribute with deploymentStatus set to deploying.
func StatusDeployingAttr() slog.Attr {
	return slog.Any(deploymentStatus, "deploying")
}

// StatusDeployedAttr returns an attribute with deploymentStatus set to deployed.
func StatusDeployedAttr() slog.Attr {
	return slog.Any(deploymentStatus, "deployed")
}

// StatusDeploymentFailedAttr returns an attribute with deploymentStatus set to failed.
func StatusDeploymentFailedAttr() slog.Attr {
	return slog.Any(deploymentStatus, "failed")
}

// StatusDeploymentSkippedAttr returns an attribute with deploymentStatus set to skipped.
func StatusDeploymentSkippedAttr() slog.Attr {
	return slog.Any(deploymentStatus, "skipped")
}
