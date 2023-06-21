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

package deploy

import (
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	config "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/topologysort"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeployConfigShouldFailOnAnAlreadyKnownEntityName(t *testing.T) {
	name := "test"
	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: name,
			},
		},
	}

	client := &dtclient.DummyClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}
	entityMap := newEntityMap(testApiMap)
	entityMap.put(parameter.ResolvedEntity{EntityName: name, Coordinate: coordinate.Coordinate{Type: "dashboard"}})
	_, errors := deployClassicConfig(context.TODO(), client, testApiMap, entityMap, nil, "", &conf)

	assert.NotEmpty(t, errors)
}

func TestDeployConfigShouldFailCyclicParameterDependencies(t *testing.T) {
	ownerParameterName := "owner"
	configCoordinates := coordinate.Coordinate{
		Project:  "project1",
		Type:     "dashboard",
		ConfigId: "dashboard-1",
	}

	parameters := []topologysort.ParameterWithName{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   configCoordinates,
						Property: ownerParameterName,
					},
				},
			},
		},
		{
			Name: ownerParameterName,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   configCoordinates,
						Property: config.NameParameter,
					},
				},
			},
		},
	}

	client := &dtclient.DummyClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}

	_, errors := deployClassicConfig(context.TODO(), client, testApiMap, newEntityMap(testApiMap), nil, "", &conf)
	assert.NotEmpty(t, errors)
}

func TestDeployConfigShouldFailOnMissingNameParameter(t *testing.T) {
	parameters := []topologysort.ParameterWithName{}

	client := &dtclient.DummyClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}

	_, errors := deployClassicConfig(context.TODO(), client, testApiMap, newEntityMap(testApiMap), nil, "", &conf)
	assert.NotEmpty(t, errors)
}

func TestDeployConfigShouldFailOnReferenceOnUnknownConfig(t *testing.T) {
	parameters := []topologysort.ParameterWithName{
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

	client := &dtclient.DummyClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}

	_, errors := deployClassicConfig(context.TODO(), client, testApiMap, newEntityMap(testApiMap), nil, "", &conf)
	assert.NotEmpty(t, errors)
}

func TestDeployConfigShouldFailOnReferenceOnSkipConfig(t *testing.T) {
	referenceCoordinates := coordinate.Coordinate{
		Project:  "project2",
		Type:     "dashboard",
		ConfigId: "dashboard",
	}

	parameters := []topologysort.ParameterWithName{
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

	client := &dtclient.DummyClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "dashboard"},
		Template: generateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  toParameterMap(parameters),
		Skip:        false,
	}

	_, errors := deployClassicConfig(context.TODO(), client, testApiMap, newEntityMap(testApiMap), nil, "", &conf)
	assert.NotEmpty(t, errors)
}
