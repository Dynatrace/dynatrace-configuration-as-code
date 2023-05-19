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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
)

//go:generate mockgen -source=automation.go -destination=automation_mock.go -package=deploy automationClient
type automationClient interface {
	Upsert(resourceType automation.ResourceType, id string, data []byte) (result *automation.Response, err error)
}

type dummyAutomationClient struct {
}

func (c *dummyAutomationClient) Upsert(_ automation.ResourceType, id string, _ []byte) (*automation.Response, error) {
	return &automation.Response{Id: id}, nil
}

func deployAutomation(client automationClient, properties parameter.Properties, renderedConfig string, c *config.Config) (*parameter.ResolvedEntity, error) {
	t, ok := c.Type.(config.AutomationType)
	if !ok {
		return &parameter.ResolvedEntity{}, fmt.Errorf("config was not of expected type %q, but %q", config.AutomationType{}.ID(), c.Type.ID())
	}

	var id string

	if c.OriginObjectId != "" {
		id = c.OriginObjectId
	} else {
		id = idutils.GenerateUUIDFromCoordinate(c.Coordinate)
	}

	var err error
	var resp *automation.Response
	switch t.Resource {
	case config.Workflow:
		resp, err = client.Upsert(automation.Workflows, id, []byte(renderedConfig))
	case config.BusinessCalendar:
		resp, err = client.Upsert(automation.BusinessCalendars, id, []byte(renderedConfig))
	case config.SchedulingRule:
		resp, err = client.Upsert(automation.SchedulingRules, id, []byte(renderedConfig))
	default:
		err = fmt.Errorf("unkonwn rsource type %q", t.Resource)
	}
	if resp == nil || err != nil {
		return nil, fmt.Errorf("failed to upsert automation object of type %s with id %s: %w", t.Resource, id, err)
	}

	name := fmt.Sprintf("[UNKNOWN NAME]%s", resp.Id)
	if configName, err := extractConfigName(c, properties); err == nil {
		name = configName
	} else {
		log.Warn("failed to extract name for automation object %q - ID will be used", resp.Id)
	}

	properties[config.IdParameter] = resp.Id
	resolved := parameter.ResolvedEntity{
		EntityName: name,
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}
	return &resolved, err

}
