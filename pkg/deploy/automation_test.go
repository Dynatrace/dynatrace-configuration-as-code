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
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestDeployAutomation_WrongType(t *testing.T) {
	client := &dummyAutomationClient{}

	conf := &config.Config{
		Type:     config.ClassicApiType{},
		Template: generateFaultyTemplate(t),
	}

	_, errors := deployAutomation(context.TODO(), client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeployAutomation_UnknownResourceType(t *testing.T) {
	client := &dummyAutomationClient{}
	conf := &config.Config{
		Type: config.AutomationType{
			Resource: config.AutomationResource("unkown"),
		},
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap([]parameter.NamedParameter{}),
	}
	_, errors := deployAutomation(context.TODO(), client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeployAutomation_ClientUpsertFails(t *testing.T) {

	t.Run("TestDeployAutomation - Workflow Upsert fails", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.Workflow,
			},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap([]parameter.NamedParameter{}),
		}
		_, err := deployAutomation(context.TODO(), client, nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - BusinessCalendar Upsert fails", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.BusinessCalendar,
			},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap([]parameter.NamedParameter{}),
		}
		_, err := deployAutomation(context.TODO(), client, nil, "", conf)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - Scheduling Rule Upsert fails", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.SchedulingRule,
			},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap([]parameter.NamedParameter{}),
		}
		_, err := deployAutomation(context.TODO(), client, nil, "", conf)
		assert.Error(t, err)
	})
}

func TestDeployAutomation(t *testing.T) {
	client := NewMockautomationClient(gomock.NewController(t))
	client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&automation.Response{
		ID: "config-id",
	}, nil)
	conf := &config.Config{
		Coordinate: coordinate.Coordinate{
			ConfigId: "config-id",
		},
		Type: config.AutomationType{
			Resource: config.Workflow,
		},
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap([]parameter.NamedParameter{}),
	}
	resolvedEntity, errors := deployAutomation(context.TODO(), client, parameter.Properties{}, "{}", conf)
	assert.NotNil(t, resolvedEntity)
	assert.Equal(t, "[UNKNOWN NAME]config-id", resolvedEntity.EntityName)
	assert.Equal(t, "config-id", resolvedEntity.Properties[config.IdParameter])
	assert.False(t, resolvedEntity.Skip)
	assert.Empty(t, errors)
}
