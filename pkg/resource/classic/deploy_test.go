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
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/classic"
)

var (
	dashboardApi = api.API{ID: "dashboard", URLPath: "dashboard"}
	testApiMap   = api.APIs{"dashboard": dashboardApi}
)

type clientStub struct {
	upsertByName               func(ctx context.Context, a api.API, name string, payload []byte) (dtclient.DynatraceEntity, error)
	upsertByNonUniqueNameAndId func(ctx context.Context, a api.API, entityID string, name string, payload []byte, duplicate bool) (dtclient.DynatraceEntity, error)
}

func (c clientStub) UpsertByName(ctx context.Context, a api.API, name string, payload []byte) (dtclient.DynatraceEntity, error) {
	return c.upsertByName(ctx, a, name, payload)
}

func (c clientStub) UpsertByNonUniqueNameAndId(ctx context.Context, a api.API, entityID string, name string, payload []byte, duplicate bool) (dtclient.DynatraceEntity, error) {
	return c.upsertByNonUniqueNameAndId(ctx, a, entityID, name, payload, duplicate)
}

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

func TestDeploy_FailOnInvalidConfigType(t *testing.T) {
	client := &dtclient.DummyConfigClient{}
	conf := config.Config{
		Type:     config.Segment{},
		Template: testutils.GenerateDummyTemplate(t),
	}
	_, err := classic.NewDeployAPI(client, testApiMap).Deploy(t.Context(), nil, "", &conf)
	assert.ErrorContains(t, err, "not of expected type")
}

func TestDeploy_FailOnInvalidAPI(t *testing.T) {
	client := &dtclient.DummyConfigClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: "invalid"},
		Template: testutils.GenerateDummyTemplate(t),
	}
	_, err := classic.NewDeployAPI(client, api.NewAPIs()).Deploy(t.Context(), nil, "", &conf)
	assert.ErrorContains(t, err, "unknown API")
}

func TestDeploy_FailOnMissingScopeForParentAPI(t *testing.T) {
	client := &dtclient.DummyConfigClient{}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: api.DashboardShareSettings},
		Template: testutils.GenerateDummyTemplate(t),
	}
	_, err := classic.NewDeployAPI(client, api.NewAPIs()).Deploy(t.Context(), nil, "", &conf)
	assert.ErrorContains(t, err, "failed to extract scope")
}

func TestDeploy_ReplacesWithParentAPI_ID(t *testing.T) {
	parentID := "parentID"
	c := clientStub{
		upsertByName: func(ctx context.Context, a api.API, name string, payload []byte) (dtclient.DynatraceEntity, error) {
			require.Equal(t, a.URLPath, fmt.Sprintf("/api/config/v1/dashboards/%s/shareSettings", parentID))
			require.Equal(t, a.AppliedParentObjectID, parentID)
			return dtclient.DynatraceEntity{
				Id: "custom-id",
			}, nil
		},
	}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: api.DashboardShareSettings},
		Template: testutils.GenerateDummyTemplate(t),
	}
	properties := parameter.Properties{
		config.ScopeParameter: parentID,
	}
	_, err := classic.NewDeployAPI(c, api.NewAPIs()).Deploy(t.Context(), properties, "", &conf)
	assert.NoError(t, err)
}

func TestDeploy_FailsOnDeploy(t *testing.T) {
	customErr := errors.New("custom error")
	c := clientStub{
		upsertByName: func(ctx context.Context, a api.API, name string, payload []byte) (dtclient.DynatraceEntity, error) {
			return dtclient.DynatraceEntity{}, customErr
		},
	}
	conf := config.Config{
		Type:     config.ClassicApiType{Api: api.Dashboard},
		Template: testutils.GenerateDummyTemplate(t),
	}
	_, err := classic.NewDeployAPI(c, testApiMap).Deploy(t.Context(), parameter.Properties{config.NameParameter: "name"}, "", &conf)
	assert.ErrorIs(t, err, customErr)
}

func TestDeploy_UpdatesByNonUniqueName(t *testing.T) {
	nameParameter := parameter.Properties{config.NameParameter: "name"}
	scopeAndNameParameter := parameter.Properties{config.ScopeParameter: "scope", config.NameParameter: "name"}

	t.Run("Updates with duplicate 'true'", func(t *testing.T) {
		c := clientStub{
			upsertByNonUniqueNameAndId: func(ctx context.Context, a api.API, entityID string, name string, payload []byte, duplicate bool) (dtclient.DynatraceEntity, error) {
				require.True(t, duplicate)
				return dtclient.DynatraceEntity{}, nil
			},
		}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: api.Dashboard},
			Template: testutils.GenerateDummyTemplate(t),
			Parameters: config.Parameters{
				config.NonUniqueNameConfigDuplicationParameter: &valueParam.ValueParameter{Value: true},
			},
		}
		_, err := classic.NewDeployAPI(c, api.NewAPIs()).Deploy(t.Context(), nameParameter, "", &conf)
		assert.NoError(t, err)
	})

	t.Run("Update with invalid duplicate parameter fails during resolve", func(t *testing.T) {
		c := clientStub{}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: api.Dashboard},
			Template: testutils.GenerateDummyTemplate(t),
			Parameters: config.Parameters{
				config.NonUniqueNameConfigDuplicationParameter: &reference.ReferenceParameter{},
			},
		}
		_, err := classic.NewDeployAPI(c, api.NewAPIs()).Deploy(t.Context(), nameParameter, "", &conf)
		expectedErr := &reference.UnresolvedReferenceError{}
		assert.ErrorAs(t, err, &expectedErr)
	})

	t.Run("Update with invalid duplicate parameter type fails", func(t *testing.T) {
		c := clientStub{}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: api.Dashboard},
			Template: testutils.GenerateDummyTemplate(t),
			Parameters: config.Parameters{
				config.NonUniqueNameConfigDuplicationParameter: &valueParam.ValueParameter{Value: "true"},
			},
		}
		_, err := classic.NewDeployAPI(c, api.NewAPIs()).Deploy(t.Context(), nameParameter, "", &conf)
		assert.ErrorContains(t, err, "invalid boolean")
	})

	t.Run("Updates with duplicate 'false' if there isn't any parameter set", func(t *testing.T) {
		c := clientStub{
			upsertByNonUniqueNameAndId: func(ctx context.Context, a api.API, entityID string, name string, payload []byte, duplicate bool) (dtclient.DynatraceEntity, error) {
				require.False(t, duplicate)
				return dtclient.DynatraceEntity{}, nil
			},
		}
		conf := config.Config{
			Type:     config.ClassicApiType{Api: api.Dashboard},
			Template: testutils.GenerateDummyTemplate(t),
		}
		_, err := classic.NewDeployAPI(c, api.NewAPIs()).Deploy(t.Context(), nameParameter, "", &conf)
		assert.NoError(t, err)
	})

	t.Run("Doesn't update via set objectID if API type is not UserActionAndSessionPropertiesMobile", func(t *testing.T) {
		objectID := "object-id"
		c := clientStub{
			upsertByNonUniqueNameAndId: func(ctx context.Context, a api.API, entityID string, name string, payload []byte, duplicate bool) (dtclient.DynatraceEntity, error) {
				require.Equal(t, entityID, "c28406e6-ef82-362f-81d2-2da0825d64f7")
				require.NotEqual(t, entityID, objectID)
				return dtclient.DynatraceEntity{}, nil
			},
		}
		conf := config.Config{
			Type:           config.ClassicApiType{Api: api.Dashboard},
			Template:       testutils.GenerateDummyTemplate(t),
			OriginObjectId: objectID,
		}
		_, err := classic.NewDeployAPI(c, api.NewAPIs()).Deploy(t.Context(), scopeAndNameParameter, "", &conf)
		assert.NoError(t, err)
	})
}
