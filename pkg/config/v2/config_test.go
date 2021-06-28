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

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"gotest.tools/assert"
)

func TestHasDependencyOn(t *testing.T) {
	referencedConfig := coordinate.Coordinate{
		Project: "project1",
		Api:     "auto-tag",
		Config:  "tag",
	}

	conf := Config{
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard1",
		},
		Environment: "dev",
		References: []coordinate.Coordinate{
			referencedConfig,
		},
	}

	referencedConf := Config{
		Coordinate:  referencedConfig,
		Environment: "dev",
	}

	result := conf.HasDependencyOn(referencedConf)

	assert.Assert(t, result, "should have dependency")
}

func TestHasDependencyOnShouldReturnFalseIfNoDependenciesAreDefined(t *testing.T) {
	conf := Config{
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard1",
		},
		Environment: "dev",
	}

	conf2 := Config{
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "auto-tag",
			Config:  "tag",
		},
		Environment: "dev",
	}

	result := conf.HasDependencyOn(conf2)

	assert.Assert(t, !result, "should not have dependency")
}

func TestMatchReference(t *testing.T) {
	conf := Config{
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard1",
		},
		Environment: "dev",
	}

	result := conf.MatchReference(coordinate.Coordinate{
		Project: "project1",
		Api:     "dashboard",
		Config:  "dashboard1",
	})

	assert.Assert(t, result, "should match reference")
}

func TestMatchReferenceShouldReturnFalseIfNotMatching(t *testing.T) {
	conf := Config{
		Coordinate: coordinate.Coordinate{
			Project: "project1",
			Api:     "dashboard",
			Config:  "dashboard1",
		},
		Environment: "dev",
	}

	result := conf.MatchReference(coordinate.Coordinate{
		Project: "project2",
		Api:     "auto-tag",
		Config:  "tag",
	})

	assert.Assert(t, !result, "should not match reference")
}
