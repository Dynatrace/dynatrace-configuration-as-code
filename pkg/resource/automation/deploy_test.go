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

package automation_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/automation"
)

func TestDeployAutomation_WrongType(t *testing.T) {
	client := &client.DummyAutomationClient{}

	conf := &config.Config{
		Type:     config.ClassicApiType{},
		Template: testutils.GenerateFaultyTemplate(t),
	}

	_, errs := automation.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
	assert.NotEmpty(t, errs)
}

func TestDeployAutomation_UnknownResourceType(t *testing.T) {
	client := &client.DummyAutomationClient{}
	conf := &config.Config{
		Type: config.AutomationType{
			Resource: config.AutomationResource("unkown"),
		},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
	}
	_, errs := automation.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
	assert.NotEmpty(t, errs)
}

func TestDeployAutomation_ClientUpsertFails(t *testing.T) {
	t.Run("TestDeployAutomation - Workflow Upsert fails", func(t *testing.T) {
		client := automation.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.Workflow,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		resp, err := automation.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
		assert.Zero(t, resp)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - Workflow Upsert fails - HTTP Err", func(t *testing.T) {
		client := automation.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{StatusCode: 400}, nil)

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.Workflow,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := automation.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - BusinessCalendar Upsert fails", func(t *testing.T) {
		client := automation.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.BusinessCalendar,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := automation.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - BusinessCalendar Upsert fails - HTTP Error", func(t *testing.T) {
		client := automation.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{StatusCode: 400}, nil)

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.BusinessCalendar,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := automation.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - Scheduling Rule Upsert fails", func(t *testing.T) {
		client := automation.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.SchedulingRule,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := automation.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - Scheduling Rule Upsert fails - HTTP Error", func(t *testing.T) {
		client := automation.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{StatusCode: 400}, nil)

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.SchedulingRule,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := automation.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
		assert.Error(t, err)
	})
}

func TestDeployAutomation(t *testing.T) {
	client := automation.NewMockDeploySource(gomock.NewController(t))
	client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{
		StatusCode: 200,
		Data:       []byte("{ \"id\": \"config-id\" }"),
	}, nil)
	conf := &config.Config{
		Coordinate: coordinate.Coordinate{
			ConfigId: "config-id",
		},
		Type: config.AutomationType{
			Resource: config.Workflow,
		},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
	}
	resolvedEntity, errs := automation.NewDeployAPI(client).Deploy(t.Context(), parameter.Properties{}, "{}", conf)
	assert.NotNil(t, resolvedEntity)
	assert.Equal(t, "config-id", resolvedEntity.Properties[config.IdParameter])
	assert.False(t, resolvedEntity.Skip)
	assert.Empty(t, errs)
}

func TestDeployAutomation_WithGivenObjectId(t *testing.T) {
	client := automation.NewMockDeploySource(gomock.NewController(t))
	objectId := "custom-object-id"
	client.EXPECT().Upsert(gomock.Any(), gomock.Any(), objectId, gomock.Any()).Times(1).Return(api.Response{
		StatusCode: 200,
		Data:       []byte(`{ "id": "config-id" }`),
	}, nil)
	conf := &config.Config{
		OriginObjectId: objectId,
		Coordinate: coordinate.Coordinate{
			ConfigId: "config-id",
		},
		Type: config.AutomationType{
			Resource: config.Workflow,
		},
	}
	resolvedEntity, errs := automation.NewDeployAPI(client).Deploy(t.Context(), parameter.Properties{}, "{}", conf)
	require.NoError(t, errs)
	assert.Equal(t, "config-id", resolvedEntity.Properties[config.IdParameter])
}
