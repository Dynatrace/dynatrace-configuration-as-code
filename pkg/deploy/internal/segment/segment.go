/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package segment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	segment "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/go-logr/logr"
	"time"
)

type DeploySegmentClient interface {
	Upsert(ctx context.Context, id string, data []byte) (segment.Response, error)
	GetAll(ctx context.Context) ([]segment.Response, error)
}

func Deploy(ctx context.Context, client DeploySegmentClient, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	//create new context to carry logger
	ctx = logr.NewContext(ctx, log.WithCtxFields(ctx).GetLogr())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// external id is generated from project-configType-configId
	externalId := c.Coordinate.String()

	var request map[string]any
	err := json.Unmarshal([]byte(renderedConfig), &request)
	request["externalId"] = externalId

	//Strategy 1 if OriginObjectId is set check on remote if an object can be found and update it,
	//update the externalId to the computed one we generate
	if c.OriginObjectId != "" {
		request["uid"] = c.OriginObjectId
		payload, _ := json.Marshal(request)
		responseUpsert, err := client.Upsert(ctx, c.OriginObjectId, payload)
		if err != nil {
			return entities.ResolvedEntity{}, err
		}
		if responseUpsert.Data == nil {
			properties[config.IdParameter] = c.OriginObjectId
		}

		return entities.ResolvedEntity{
			EntityName: externalId,
			Coordinate: c.Coordinate,
			Properties: properties,
		}, nil
	}

	//Strategy 2 is to generate external id and either update or create object
	segmentsResponses, err := client.GetAll(ctx)
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	var response map[string]interface{}
	for _, segmentResponse := range segmentsResponses {
		err = json.Unmarshal(segmentResponse.Data, &response)
		if err != nil {
			return entities.ResolvedEntity{}, err
		}
		//In case of a match, the put needs additional fields
		if response["externalId"] == externalId {
			request["uid"] = response["uid"].(string)
			request["owner"] = response["owner"].(string)
			externalId = response["uid"].(string)
		}
	}
	payload, _ := json.Marshal(request)
	_, err = client.Upsert(ctx, externalId, payload)
	if err != nil {
		var apiErr api.APIError
		if errors.As(err, &apiErr) {
			return entities.ResolvedEntity{}, fmt.Errorf("failed to upsert segment with segmentName %q: %w", externalId, err)
		}

		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert segment with segmentName %q", externalId)).WithError(err)
	}

	properties[config.IdParameter] = externalId

	return entities.ResolvedEntity{
		EntityName: externalId,
		Coordinate: c.Coordinate,
		Properties: properties,
	}, nil
}
