// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coordinate

import (
	"fmt"
	"log/slog"
)

// Coordinate struct used to specify the location of a certain configuration
type Coordinate struct {
	// Project specifies the id of a project
	Project string `json:"project"`

	// Type specifies the id of an api, or a schema id
	Type string `json:"type"`

	// ConfigId specifies the id of a monaco configuration definition
	ConfigId string `json:"configId"`
}

func (c Coordinate) String() string {
	return fmt.Sprintf("%s:%s:%s", c.Project, c.Type, c.ConfigId)
}

// Match tests if this coordinate is the same as the given one
func (c Coordinate) Match(coordinate Coordinate) bool {
	return c.Project == coordinate.Project &&
		c.Type == coordinate.Type &&
		c.ConfigId == coordinate.ConfigId
}

// LogValue implements slog.LogValuer.
// It returns a group containing the fields of the coordinate so that they appear together in log output
func (c Coordinate) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("reference", c.String()),
		slog.String("project", c.Project),
		slog.String("type", c.Type),
		slog.String("configId", c.ConfigId))
}
