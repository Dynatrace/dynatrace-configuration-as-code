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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
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

func TestDeployAutomation(t *testing.T) {
	t.Run("base case", func(t *testing.T) {
		conf := &config.Config{
			Coordinate: coordinate.Coordinate{ConfigId: "config-id"},
			Type:       config.AutomationType{Resource: config.BusinessCalendar},
		}

		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(automation.BusinessCalendars, idutils.GenerateUuidFromName(conf.Coordinate.String()), []byte("{}")).Times(1).Return(&automation.Response{ID: "returned-id"}, nil)

		actual, err := deployAutomation(client, parameter.Properties{}, "{}", conf)

		assert.NotNil(t, actual)
		assert.Equal(t, "[UNKNOWN NAME]returned-id", actual.EntityName)
		assert.Equal(t, "returned-id", actual.Properties[config.IdParameter])
		assert.False(t, actual.Skip)
		assert.Empty(t, err)
	})

	t.Run("unescape jinja for workflows", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))

		{
			conf := &config.Config{
				OriginObjectId: "objectID",
				Type:           config.AutomationType{Resource: config.Workflow},
			}

			client.EXPECT().Upsert(automation.Workflows, conf.OriginObjectId, []byte(`{{ .unescaped.jinja }}`)).Times(1).Return(&automation.Response{ID: conf.OriginObjectId}, nil)

			actual, err := deployAutomation(client, parameter.Properties{}, `\{\{ .unescaped.jinja \}\}`, conf)
			assert.NotNil(t, actual)
			assert.Empty(t, err)
		}

		{
			conf := &config.Config{
				OriginObjectId: "objectID",
				Type:           config.AutomationType{Resource: config.BusinessCalendar},
			}

			client.EXPECT().Upsert(automation.BusinessCalendars, conf.OriginObjectId, []byte(`{{ .unescaped.jinja }}`)).Times(1).Return(&automation.Response{ID: conf.OriginObjectId}, nil)

			actual, err := deployAutomation(client, parameter.Properties{}, `\{\{ .unescaped.jinja \}\}`, conf)
			assert.NotNil(t, actual)
			assert.Empty(t, err)
		}

		{
			conf := &config.Config{
				OriginObjectId: "objectID",
				Type:           config.AutomationType{Resource: config.SchedulingRule},
			}

			client.EXPECT().Upsert(automation.SchedulingRules, conf.OriginObjectId, []byte(`{{ .unescaped.jinja }}`)).Times(1).Return(&automation.Response{ID: conf.OriginObjectId}, nil)

			actual, err := deployAutomation(client, parameter.Properties{}, `\{\{ .unescaped.jinja \}\}`, conf)
			assert.NotNil(t, actual)
			assert.Empty(t, err)
		}
	})

	t.Run("TestDeployAutomation - Workflow Upsert fails", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(automation.Workflows, gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{Resource: config.Workflow},
		}
		res, err := deployAutomation(client, nil, "", conf)
		assert.Nil(t, res)
		assert.Error(t, err)
	})

	t.Run("TestDeployAutomation - BusinessCalendar Upsert fails", func(t *testing.T) {
		client := NewMockautomationClient(gomock.NewController(t))
		client.EXPECT().Upsert(automation.BusinessCalendars, gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

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
		client.EXPECT().Upsert(automation.SchedulingRules, gomock.Any(), nil).Times(1).Return(nil, errors.New("UPSERT_FAIL"))

		conf := &config.Config{
			Type: config.AutomationType{Resource: config.SchedulingRule},
		}
		res, err := deployAutomation(client, nil, "", conf)
		assert.Nil(t, res)
		assert.Error(t, err)
	})

}
