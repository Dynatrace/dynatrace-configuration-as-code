//go:build unit

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package classic_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/classic"
)

var dashboardApi = api.API{ID: "dashboard", URLPath: "dashboard"}
var testApiMap = api.APIs{"dashboard": dashboardApi}

func TestDeployConfigShouldFailOnAnAlreadyKnownEntityName(t *testing.T) {
	name := "test"
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: name,
			},
		},
	}

	client := &dtclient.DummyConfigClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
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
	entityMap := entities.New()
	entityMap.Put(entities.ResolvedEntity{Coordinate: coordinate.Coordinate{Type: "dashboard"}})
	_, errors := classic.NewDeployAPI(client, testApiMap).Deploy(t.Context(), nil, "", &conf)

	assert.NotEmpty(t, errors)
}

func TestDeployConfigShouldFailOnMissingNameParameter(t *testing.T) {
	parameters := []parameter.NamedParameter{}

	client := &dtclient.DummyConfigClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
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

	_, errors := classic.NewDeployAPI(client, testApiMap).Deploy(t.Context(), nil, "", &conf)
	assert.NotEmpty(t, errors)
}

func TestDeployConfigShouldFailOnReferenceOnUnknownConfig(t *testing.T) {
	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config: coordinate.Coordinate{
							Project:  "project2",
							Type:     "dashboard",
							ConfigId: "dashboard",
						},
						Property: "managementZoneId",
					},
				},
			},
		},
	}

	client := &dtclient.DummyConfigClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
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

	_, errors := classic.NewDeployAPI(client, testApiMap).Deploy(t.Context(), nil, "", &conf)
	assert.NotEmpty(t, errors)
}

func TestDeployConfigShouldFailOnReferenceOnSkipConfig(t *testing.T) {
	referenceCoordinates := coordinate.Coordinate{
		Project:  "project2",
		Type:     "dashboard",
		ConfigId: "dashboard",
	}

	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   referenceCoordinates,
						Property: "managementZoneId",
					},
				},
			},
		},
	}

	client := &dtclient.DummyConfigClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
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

	_, errors := classic.NewDeployAPI(client, testApiMap).Deploy(t.Context(), nil, "", &conf)
	assert.NotEmpty(t, errors)
}
