//go:build unit

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

package resolve

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/entitymap"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/testutils"
	"gotest.tools/assert"
	"testing"
)

func TestResolveParameterValues(t *testing.T) {
	name := "test"
	owner := "hansi"
	ownerParameterName := "owner"
	timeout := 5
	timeoutParameterName := "timeout"
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: name,
			},
		},
		{
			Name: ownerParameterName,
			Parameter: &parameter.DummyParameter{
				Value: owner,
			},
		},
		{
			Name: timeoutParameterName,
			Parameter: &parameter.DummyParameter{
				Value: timeout,
			},
		},
	}

	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  testutils.ToParameterMap(parameters),
		Skip:        false,
	}

	entities := entitymap.New()

	values, errs := Properties(&conf, entities)

	assert.Assert(t, len(errs) == 0, "there should be no errors (errors: %s)", errs)
	assert.Equal(t, name, values[config.NameParameter])
	assert.Equal(t, owner, values[ownerParameterName])
	assert.Equal(t, timeout, values[timeoutParameterName])
}

func TestResolveParameterValuesShouldFailWhenReferencingNonExistingConfig(t *testing.T) {
	nonExistingConfig := coordinate.Coordinate{
		Project:  "non-existing",
		Type:     "management-zone",
		ConfigId: "zone1",
	}
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   nonExistingConfig,
						Property: "name",
					},
				},
			},
		},
	}

	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  testutils.ToParameterMap(parameters),
		Skip:        false,
	}

	entities := entitymap.New()

	_, errs := Properties(&conf, entities)

	assert.Assert(t, len(errs) > 0, "there should be errors (no errors: %d)", len(errs))
}

func TestResolveParameterValuesShouldFailWhenReferencingSkippedConfig(t *testing.T) {
	referenceCoordinate := coordinate.Coordinate{
		Project:  "project1",
		Type:     "management-zone",
		ConfigId: "zone1",
	}

	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   referenceCoordinate,
						Property: "name",
					},
				},
			},
		},
	}

	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  testutils.ToParameterMap(parameters),
		Skip:        false,
	}

	entities := entitymap.New()
	entities.Put(config.ResolvedEntity{
		EntityName: "zone1",
		Coordinate: referenceCoordinate,
		Properties: parameter.Properties{},
		Skip:       true,
	})

	_, errs := Properties(&conf, entities)

	assert.Assert(t, len(errs) > 0, "there should be errors (no errors: %d)", len(errs))
}

func TestResolveParameterValuesShouldFailWhenParameterResolveReturnsError(t *testing.T) {
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Err: errors.New("error"),
			},
		},
	}

	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  testutils.ToParameterMap(parameters),
		Skip:        false,
	}

	entities := entitymap.New()

	_, errs := Properties(&conf, entities)

	assert.Assert(t, len(errs) > 0, "there should be errors (no errors: %d)", len(errs))
}

func TestExtractConfigName(t *testing.T) {
	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	name := "test"

	properties := parameter.Properties{
		config.NameParameter: name,
	}

	val, err := ExtractConfigName(&conf, properties)

	assert.NilError(t, err)
	assert.Equal(t, name, val)
}

func TestExtractConfigNameShouldFailOnMissingName(t *testing.T) {
	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	properties := parameter.Properties{}

	_, err := ExtractConfigName(&conf, properties)

	assert.Assert(t, err != nil, "error should not be nil (error val: %s)", err)
}

func TestExtractConfigNameShouldFailOnNameWithNonStringType(t *testing.T) {
	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	properties := parameter.Properties{
		config.NameParameter: 1,
	}

	_, err := ExtractConfigName(&conf, properties)

	assert.Assert(t, err != nil, "error should not be nil (error val: %s)", err)
}
