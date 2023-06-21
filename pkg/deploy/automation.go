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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

//go:generate mockgen -source=automation.go -destination=automation_mock.go -package=deploy automationClient
type automationClient interface {
	Upsert(ctx context.Context, resourceType automation.ResourceType, id string, data []byte) (result *automation.Response, err error)
}
type dummyAutomationClient struct {
}

func (c *dummyAutomationClient) Upsert(_ context.Context, _ automation.ResourceType, id string, _ []byte) (*automation.Response, error) {
	return &automation.Response{ID: id}, nil
}

func deployAutomation(ctx context.Context, client automationClient, properties parameter.Properties, renderedConfig string, c *config.Config) (*parameter.ResolvedEntity, error) {
	t, ok := c.Type.(config.AutomationType)
	if !ok {
		return &parameter.ResolvedEntity{}, newConfigDeployErr(c, fmt.Sprintf("config was not of expected type %q, but %q", config.AutomationType{}.ID(), c.Type.ID()))
	}

	var id string

	if c.OriginObjectId != "" {
		id = c.OriginObjectId
	} else {
		id = idutils.GenerateUUIDFromCoordinate(c.Coordinate)
	}

	resourceType, err := automationutils.ClientResourceTypeFromConfigType(t.Resource)
	if err != nil {
		return nil, newConfigDeployErr(c, fmt.Sprintf("failed to upsert automation object of type %s with id %s", t.Resource, id)).withError(err)
	}

	var resp *automation.Response
	resp, err = client.Upsert(ctx, resourceType, id, []byte(renderedConfig))
	if resp == nil || err != nil {
		return nil, newConfigDeployErr(c, fmt.Sprintf("failed to upsert automation object of type %s with id %s", t.Resource, id)).withError(err)
	}

	name := fmt.Sprintf("[UNKNOWN NAME]%s", resp.ID)
	if configName, err := extractConfigName(c, properties); err == nil {
		name = configName
	} else {
		log.WithCtxFields(ctx).Warn("failed to extract name for automation object %q - ID will be used", resp.ID)
	}

	properties[config.IdParameter] = resp.ID
	resolved := parameter.ResolvedEntity{
		EntityName: name,
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}
	return &resolved, nil

}
