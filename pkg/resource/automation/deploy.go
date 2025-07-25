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

package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

//go:generate mockgen -source=deploy.go -destination=automation_mock.go -package=automation DeploySource
type DeploySource interface {
	Create(ctx context.Context, resourceType automation.ResourceType, data []byte) (api.Response, error)
	Update(ctx context.Context, resourceType automation.ResourceType, id string, data []byte) (api.Response, error)
}

type DeployAPI struct {
	source DeploySource
}

func NewDeployAPI(source DeploySource) *DeployAPI {
	return &DeployAPI{source}
}

func (d DeployAPI) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	t, ok := c.Type.(config.AutomationType)
	if !ok {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, fmt.Sprintf("config was not of expected type %q, but %q", config.AutomationType{}.ID(), c.Type.ID()))
	}

	var id string

	if c.OriginObjectId != "" {
		id = c.OriginObjectId
	} else {
		id = idutils.GenerateUUIDFromCoordinate(c.Coordinate)
	}

	resourceType, err := automationutils.ClientResourceTypeFromConfigType(t.Resource)
	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert automation object of type %s with id %s", t.Resource, id)).WithError(err)
	}

	resp, err := d.upsert(ctx, resourceType, id, []byte(renderedConfig))
	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert automation object of type %s with id %s", t.Resource, id)).WithError(err)
	}

	obj, err := automationutils.DecodeResponse(resp)
	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, fmt.Sprintf("failed to decode automation object response of type %s with id %s", t.Resource, id)).WithError(err)
	}

	properties[config.IdParameter] = obj.ID
	resolved := entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}
	return resolved, nil

}

func (d DeployAPI) upsert(ctx context.Context, resourceType automation.ResourceType, id string, data []byte) (api.Response, error) {
	resp, err := d.source.Update(ctx, resourceType, id, data)

	// return response if there is no error
	if err == nil {
		return resp, nil
	}

	// NotFound would mean that we need to create it, if not, something else is happening
	if !api.IsNotFoundError(err) {
		return api.Response{}, err
	}

	// make sure actual "id" field is set in payload
	if err := setIDField(id, &data); err != nil {
		return api.Response{}, fmt.Errorf("failed to create automation resource of type %v with id %s: unable to set the id field in order to create object: %w", resourceType, id, err)
	}

	return d.source.Create(ctx, resourceType, data)
}

func setIDField(id string, data *[]byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(*data, &m)
	if err != nil {
		return err
	}
	m["id"] = id
	*data, err = json.Marshal(m)
	if err != nil {
		return err
	}
	return nil
}
