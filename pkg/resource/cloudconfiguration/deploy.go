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

package cloudconfiguration

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

//go:generate mockgen -source=deploy.go -destination=cloudconfiguration_deploy_mock.go -package=cloudconfiguration DeploySource
type DeploySource interface {
	Get(ctx context.Context, id string) (api.Response, error)
	Create(ctx context.Context, data []byte) (api.Response, error)
	Update(ctx context.Context, id string, data []byte) (api.Response, error)
}

type DeployAPI struct {
	source DeploySource
}

func NewDeployAPI(source DeploySource) *DeployAPI {
	return &DeployAPI{source}
}

func (d DeployAPI) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	data := []byte(renderedConfig)

	if c.OriginObjectId != "" {
		_, err := d.source.Update(ctx, c.OriginObjectId, data)
		if err == nil {
			return resolvedEntity(c.OriginObjectId, c, properties), nil
		}
		if !api.IsNotFoundError(err) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update cloud configuration '%s'", c.OriginObjectId)).WithError(err)
		}
	}

	resp, err := d.source.Create(ctx, data)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "failed to create cloud configuration").WithError(err)
	}

	id, err := idFromResponse(resp)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "failed to read ID from create response").WithError(err)
	}

	return resolvedEntity(id, c, properties), nil
}

func idFromResponse(resp api.Response) (string, error) {
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Data, &body); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if body.ID == "" {
		return "", fmt.Errorf("response missing 'id' field")
	}
	return body.ID, nil
}

func resolvedEntity(id string, c *config.Config, properties parameter.Properties) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
	}
}
