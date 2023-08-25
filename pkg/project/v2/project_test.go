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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasDependencyOn(t *testing.T) {
	environment := "dev"
	referencedProjectId := "projct2"

	p := project.Project{
		Id: "project1",
		Dependencies: project.DependenciesPerEnvironment{
			environment: []string{
				referencedProjectId,
			},
		},
	}

	referencedProject := project.Project{
		Id: referencedProjectId,
	}

	result := p.HasDependencyOn(environment, referencedProject)

	assert.True(t, result, "should have dependency")
}

func TestHasDependencyOnShouldReturnFalseIfNoDependenciesForEnvironmentAreDefined(t *testing.T) {
	environment := "dev"

	p := project.Project{
		Id: "project1",
	}

	p2 := project.Project{
		Id: "project2",
	}

	result := p.HasDependencyOn(environment, p2)

	assert.False(t, result, "should not have dependency")
}

func TestHasDependencyOnShouldReturnFalseIfNoDependencyDefined(t *testing.T) {
	environment := "dev"

	p := project.Project{
		Id: "project1",
		Dependencies: project.DependenciesPerEnvironment{
			environment: []string{
				"project3",
			},
		},
	}

	project2 := project.Project{
		Id: "project2",
	}

	result := p.HasDependencyOn(environment, project2)

	assert.False(t, result, "should not have dependency")
}

func TestProject_ForEveryConfigDo(t *testing.T) {
	t.Run("simple case", func(t *testing.T) {
		given := project.Project{
			Id:      "projectID",
			GroupId: "groupID",
			Configs: map[project.EnvironmentName]project.ConfigsPerType{
				"env1": map[string][]config.Config{
					"type1": {
						{Coordinate: coordinate.Coordinate{Project: "projectID", Type: "type1", ConfigId: "config1"}},
						{Coordinate: coordinate.Coordinate{Project: "projectID", Type: "type1", ConfigId: "config2"}},
					},
					"type2": {
						{Coordinate: coordinate.Coordinate{Project: "projectID", Type: "type2", ConfigId: "config3"}},
					},
				},
				"env2": map[string][]config.Config{
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
