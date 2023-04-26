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
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2/topologysort"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeployAutomation_WrongType(t *testing.T) {
	client := &dummyAutomationClient{}

	conf := &config.Config{
		Type:     config.ClassicApiType{},
		Template: generateFaultyTemplate(t),
	}

	_, errors := deployAutomation(client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeployAutomation_UnknownResourceType(t *testing.T) {
	client := &dummyAutomationClient{}
	conf := &config.Config{
		Type: config.AutomationType{
			Resource: config.AutomationResource("unkown"),
		},
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap([]topologysort.ParameterWithName{}),
	}
	_, errors := deployAutomation(client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeployAutomation_ClientUpsertFails(t *testing.T) {

	t.Run("TestDeployAutomation - Workflow Upsert fails", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.Workflow,
			},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap([]topologysort.ParameterWithName{}),
		}
		res, err := deployAutomation(client, nil, "", conf)
		assert.Nil(t, res)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - BusinessCalendar Upsert fails", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.BusinessCalendar,
			},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap([]topologysort.ParameterWithName{}),
		}
		res, err := deployAutomation(client, nil, "", conf)
		assert.Nil(t, res)
		assert.Error(t, err)
	})
	t.Run("TestDeployAutomation - Scheduling Rule Upsert fails", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{
				Resource: config.SchedulingRule,
			},
			Template:   generateDummyTemplate(t),
			Parameters: toParameterMap([]topologysort.ParameterWithName{}),
		}
		res, err := deployAutomation(client, nil, "", conf)
		assert.Nil(t, res)
		assert.Error(t, err)
	})
}

func TestDeployAutomation(t *testing.T) {
	client := NewMockautomationClient(gomock.NewController(t))
	client.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&automation.Response{
		Id: "config-id",
	}, nil)
	conf := &config.Config{
		Coordinate: coordinate.Coordinate{
			ConfigId: "config-id",
		},
		Type: config.AutomationType{
			Resource: config.Workflow,
		},
		Template:   generateDummyTemplate(t),
		Parameters: toParameterMap([]topologysort.ParameterWithName{}),
	}
	resolvedEntity, errors := deployAutomation(client, parameter.Properties{}, "{}", conf)
	assert.NotNil(t, resolvedEntity)
	assert.Equal(t, "[UNKNOWN NAME]config-id", resolvedEntity.EntityName)
	assert.Equal(t, "config-id", resolvedEntity.Properties[config.IdParameter])
	assert.False(t, resolvedEntity.Skip)
	assert.Empty(t, errors)
}
