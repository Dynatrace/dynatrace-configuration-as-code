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

package automation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/testutils"
)

func TestDeployAutomation_WrongType(t *testing.T) {
	client := &client.DryRunAutomationClient{}

	conf := &config.Config{
		Type:     config.ClassicApiType{},
		Template: testutils.GenerateFaultyTemplate(t),
	}

	_, errors := Deploy(context.TODO(), client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeployAutomation_UnknownResourceType(t *testing.T) {
	client := &client.DryRunAutomationClient{}
	conf := &config.Config{
		Type: config.AutomationType{
			Resource: config.AutomationResource("unkown"),
		},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
	}
	_, errors := Deploy(context.TODO(), client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeployAutomation_ClientUpsertFails(t *testing.T) {
	t.Run("TestDeployAutomation - Workflow Upsert fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(automation.Response{}, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.Workflow,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		resp, err := Deploy(context.TODO(), client, nil, "", conf)
		assert.Zero(t, resp)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - Workflow Upsert fails - HTTP Err", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{StatusCode: 400}, nil)

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.Workflow,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := Deploy(context.TODO(), client, nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - BusinessCalendar Upsert fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(automation.Response{}, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.BusinessCalendar,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := Deploy(context.TODO(), client, nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - BusinessCalendar Upsert fails - HTTP Error", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(automation.Response{StatusCode: 400}, nil)

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.BusinessCalendar,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := Deploy(context.TODO(), client, nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - Scheduling Rule Upsert fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(automation.Response{}, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.SchedulingRule,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := Deploy(context.TODO(), client, nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - Scheduling Rule Upsert fails - HTTP Error", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(automation.Response{StatusCode: 400}, nil)

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.SchedulingRule,
			},
			Template:   testutils.GenerateDummyTemplate(t),
			Parameters: testutils.ToParameterMap([]parameter.NamedParameter{}),
		}
		_, err := Deploy(context.TODO(), client, nil, "", conf)
		assert.Error(t, err)
	})
}

func TestDeployAutomation(t *testing.T) {
	client := NewMockClient(gomock.NewController(t))
	client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(automation.Response{
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
	resolvedEntity, errors := Deploy(context.TODO(), client, parameter.Properties{}, "{}", conf)
	assert.NotNil(t, resolvedEntity)
	assert.Equal(t, "[UNKNOWN NAME]config-id", resolvedEntity.EntityName)
	assert.Equal(t, "config-id", resolvedEntity.Properties[config.IdParameter])
	assert.False(t, resolvedEntity.Skip)
	assert.Empty(t, errors)
}
