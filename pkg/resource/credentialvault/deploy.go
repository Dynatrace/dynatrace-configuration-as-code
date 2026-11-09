/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package credentialvault

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/extract"
)

type DeploySource interface {
	List(ctx context.Context) (api.PagedListResponse, error)
	Update(ctx context.Context, id string, data []byte) (api.Response, error)
	Create(ctx context.Context, data []byte) (api.Response, error)
}

type DeployAPI struct {
	source DeploySource
}

func NewDeployAPI(source DeploySource) *DeployAPI {
	return &DeployAPI{source}
}

type credentialVaultResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (d DeployAPI) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	requestPayload := []byte(renderedConfig)

	//Strategy 1 when OriginObjectId is set we update the object
	if c.OriginObjectId != "" {
		_, err := d.source.Update(ctx, c.OriginObjectId, requestPayload)
		if err == nil {
			return createResolveEntity(c.OriginObjectId, properties, c), nil
		}

		if !api.IsNotFoundError(err) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to deploy credential vault entry: %s", c.OriginObjectId)).WithError(err)
		}
	}

	//Strategy 2 is to try to find a match with name and update it
	name, err := extract.ConfigName(c, properties)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "failed to get credential vault name").WithError(err)
	}
	id, err := findEntryByName(ctx, d.source, name)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("error finding credential vault entry with name: %s", name)).WithError(err)
	}

	if id != "" {
		_, err := d.source.Update(ctx, id, requestPayload)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update credential vault entry with id: %s", id)).WithError(err)
		}
		return createResolveEntity(id, properties, c), nil
	}

	//Strategy 3 is to create a new credential vault entry
	createResponse, err := d.source.Create(ctx, requestPayload)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to deploy credential vault entry with name: %s", name)).WithError(err)
	}

	response, err := responseFromHttpData(createResponse)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to unmarshal credential vault entry with name: %s", name)).WithError(err)
	}

	return createResolveEntity(response.ID, properties, c), nil
}

func responseFromHttpData(rawResponse api.Response) (credentialVaultResponse, error) {
	var response credentialVaultResponse
	err := json.Unmarshal(rawResponse.Data, &response)
	if err != nil {
		return credentialVaultResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

func createResolveEntity(id string, properties parameter.Properties, c *config.Config) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
	}
}
