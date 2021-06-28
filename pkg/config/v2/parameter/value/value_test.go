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

package value

import (
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"

	"gotest.tools/assert"
)

func TestParseValueParameter(t *testing.T) {
	value := "test"

	param, err := parseValueParameter(parameter.ParameterParserContext{
		Coordinate: coordinate.Coordinate{
			Project: "projectA",
			Api:     "dashboard",
			Config:  "super-important",
		},
		ParameterName: "title",
		Value: map[string]interface{}{
			"value": value,
		},
	})

	assert.NilError(t, err)

	valueParam, ok := param.(*ValueParameter)

	assert.Assert(t, ok, "parsed parameter is value parameter")
	assert.Equal(t, value, valueParam.Value)
}

func TestParseValueParameterMissingValueParameterShouldReturnError(t *testing.T) {
	value := "test"

	_, err := parseValueParameter(parameter.ParameterParserContext{
		Coordinate: coordinate.Coordinate{
			Project: "projectA",
			Api:     "dashboard",
			Config:  "super-important",
		},
		ParameterName: "title",
		Value: map[string]interface{}{
			"title": value,
		},
	})

	assert.Assert(t, err != nil)
}

func TestGetReferencesShouldNotReturnAnything(t *testing.T) {
	fixture := ValueParameter{
		Value: "test",
	}

	refs := fixture.GetReferences()

	assert.Assert(t, len(refs) == 0)
}

func TestResolveValue(t *testing.T) {
	value := "test"
	fixture := ValueParameter{
		Value: value,
	}

	result, err := fixture.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project: "projectA",
			Api:     "dashboard",
			Config:  "super-important",
		},
		ParameterName: "test",
	})

	assert.NilError(t, err)
	assert.Equal(t, value, result)
}
