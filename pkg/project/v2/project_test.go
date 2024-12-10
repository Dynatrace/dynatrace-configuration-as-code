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

//go:build unit

package v2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

func TestGetConfigFor(t *testing.T) {
	tests := []struct {
		name            string
		givenCoordinate coordinate.Coordinate
		givenProject    project.Project
		wantConfig      config.Config
		wantFound       bool
	}{
		{
			name:            "Config found",
			givenCoordinate: coordinate.Coordinate{Project: "p1", Type: "t1", ConfigId: "c1"},
			givenProject: project.Project{

				Id: "p1",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{"t1": {config.Config{Coordinate: coordinate.Coordinate{ConfigId: "c1"}}}},
					"env2": project.ConfigsPerType{"t2": {config.Config{Coordinate: coordinate.Coordinate{ConfigId: "c2"}}}},
				},
			},

			wantFound:  true,
			wantConfig: config.Config{Coordinate: coordinate.Coordinate{ConfigId: "c1"}},
		},
		{
			name:            "Config not found - type mismatch",
			givenCoordinate: coordinate.Coordinate{Project: "p1", Type: "t2", ConfigId: "c1"},
			givenProject: project.Project{

				Id: "p1",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{"t1": {config.Config{Coordinate: coordinate.Coordinate{ConfigId: "c1"}}}},
					"env2": project.ConfigsPerType{"t2": {config.Config{Coordinate: coordinate.Coordinate{ConfigId: "c2"}}}},
				},
			},

			wantFound:  false,
			wantConfig: config.Config{},
		},
		{
			name:            "Config not found - id mismatch",
			givenCoordinate: coordinate.Coordinate{Project: "p1", Type: "t1", ConfigId: "c2"},
			givenProject: project.Project{

				Id: "p1",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{"t1": {config.Config{Coordinate: coordinate.Coordinate{ConfigId: "c1"}}}},
					"env2": project.ConfigsPerType{"t2": {config.Config{Coordinate: coordinate.Coordinate{ConfigId: "c2"}}}},
				},
			},

			wantFound:  false,
			wantConfig: config.Config{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, found := tc.givenProject.GetConfigFor(tc.givenCoordinate)
			assert.Equal(t, tc.wantConfig, cfg)
			assert.Equal(t, tc.wantFound, found)
		})
	}
}

func TestProject_ForEveryConfigDo(t *testing.T) {
	t.Run("simple case", func(t *testing.T) {
		given := project.Project{
			Id:      "projectID",
			GroupId: "groupID",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env1": project.ConfigsPerType{
					"type1": {
						{Coordinate: coordinate.Coordinate{Project: "projectID", Type: "type1", ConfigId: "config1"}},
						{Coordinate: coordinate.Coordinate{Project: "projectID", Type: "type1", ConfigId: "config2"}},
					},
					"type2": {
						{Coordinate: coordinate.Coordinate{Project: "projectID", Type: "type2", ConfigId: "config3"}},
					},
				},
				"env2": project.ConfigsPerType{
					"type3": {
						{Coordinate: coordinate.Coordinate{Project: "projectID", Type: "type3", ConfigId: "config4"}},
					},
				},
			},
		}

		var actual []string

		given.ForEveryConfigDo(func(c config.Config) {
			actual = append(actual, c.Coordinate.ConfigId)
		})

		assert.Contains(t, actual, "config1")
		assert.Contains(t, actual, "config2")
		assert.Contains(t, actual, "config3")
		assert.Contains(t, actual, "config4")
	})
}
