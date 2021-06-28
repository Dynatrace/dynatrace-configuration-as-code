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

// +build unit

package v2

import (
	"testing"

	"gotest.tools/assert"
)

func TestHasDependencyOn(t *testing.T) {
	environment := "dev"
	referencedProjectId := "projct2"

	project := Project{
		Id: "project1",
		Dependencies: DependenciesPerEnvironment{
			environment: []string{
				referencedProjectId,
			},
		},
	}

	referencedProject := Project{
		Id: referencedProjectId,
	}

	result := project.HasDependencyOn(environment, referencedProject)

	assert.Assert(t, result, "should have dependency")
}

func TestHasDependencyOnShouldReturnFalseIfNoDependenciesForEnvironmentAreDefined(t *testing.T) {
	environment := "dev"

	project := Project{
		Id: "project1",
	}

	project2 := Project{
		Id: "project2",
	}

	result := project.HasDependencyOn(environment, project2)

	assert.Assert(t, !result, "should not have dependency")
}

func TestHasDependencyOnShouldReturnFalseIfNoDependencyDefined(t *testing.T) {
	environment := "dev"

	project := Project{
		Id: "project1",
		Dependencies: DependenciesPerEnvironment{
			environment: []string{
				"project3",
			},
		},
	}

	project2 := Project{
		Id: "project2",
	}

	result := project.HasDependencyOn(environment, project2)

	assert.Assert(t, !result, "should not have dependency")
}
