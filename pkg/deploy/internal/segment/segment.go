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
	"net/http"
	"time"
)

type DeploySegmentClient interface {
	Upsert(ctx context.Context, id string, data []byte) (segment.Response, error)
	GetAll(ctx context.Context) ([]segment.Response, error)
	Get(ctx context.Context, id string) (segment.Response, error)
}

type jsonResponse struct {
	UID        string `json:"uid"`
	Owner      string `json:"owner"`
	ExternalId string `json:"externalId"`
}

func Deploy(ctx context.Context, client DeploySegmentClient, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	var request map[string]any
	err := json.Unmarshal([]byte(renderedConfig), &request)
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	//externalId is generated from [project-configType-configId]
	externalId := c.Coordinate.String()
	request["externalId"] = externalId

	//Strategy 1 when OriginObjectId is set we try to get the object if it exists we update it.
	if c.OriginObjectId != "" {
		id, err := deployWithOriginObjectId(ctx, client, request, c)
		if err != nil {
			return entities.ResolvedEntity{}, fmt.Errorf("failed to deploy segment with externalId: %s, with error: %w", externalId, err)
		}
		if id != "" {
			return createResolveEntity(id, externalId, properties, c), nil
		}
	}

	//Strategy 2 is to try to find a match with external id and either update or create object if no match found.
	id, err := deployWithExternalId(ctx, client, request, c, externalId)
	if err != nil {
		return entities.ResolvedEntity{}, fmt.Errorf("failed to deploy segment with externalId: %s, with error: %w", externalId, err)
	}

	return createResolveEntity(id, externalId, properties, c), nil
}

func deployWithOriginObjectId(ctx context.Context, client DeploySegmentClient, request map[string]any, c *config.Config) (string, error) {
	_, err := client.Get(ctx, c.OriginObjectId)
	if err != nil {
		apiError := api.APIError{}
		if errors.As(err, &apiError) && apiError.StatusCode == http.StatusNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to fetch segment object: %w", err)
	}

	request["uid"] = c.OriginObjectId
	payload, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal segment request: %w", err)
	}

	_, err = deploy(ctx, client, c.OriginObjectId, payload, c)
	if err != nil {
		return "", fmt.Errorf("failed API request: %w", err)
	}

	return c.OriginObjectId, nil
}

func deployWithExternalId(ctx context.Context, client DeploySegmentClient, request map[string]any, c *config.Config, externalId string) (string, error) {
	segmentsResponses, err := client.GetAll(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to GET segments: %w", err)
	}

	var responseData jsonResponse
	for _, segmentResponse := range segmentsResponses {
		responseData, err = getJsonResponseFromSegmentsResponse(segmentResponse)
		if err != nil {
			return "", err
		}
		//In case of a match, the put needs additional fields
		if responseData.ExternalId == request["externalId"] {
			request["uid"] = responseData.UID
			request["owner"] = responseData.Owner
			break
		}
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal segment request: %w", err)
	}

	responseUpsert, err := deploy(ctx, client, responseData.UID, payload, c)
	if err != nil {
		return "", fmt.Errorf("failed API request: %w", err)
	}

	//For a POST we need to parse the response again to read out the ID
	if responseUpsert.StatusCode == http.StatusCreated {
		responseData, err = getJsonResponseFromSegmentsResponse(responseUpsert)
		if err != nil {
			return "", err
		}
	}

	return responseData.UID, nil
}

func deploy(ctx context.Context, client DeploySegmentClient, id string, payload []byte, c *config.Config) (segment.Response, error) {
	//create new context to carry logger
	ctx = logr.NewContext(ctx, log.WithCtxFields(ctx).GetLogr())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	responseUpsert, err := client.Upsert(ctx, id, payload)
	if err != nil {
		var apiErr api.APIError
		if errors.As(err, &apiErr) {
			return api.Response{}, fmt.Errorf("failed to upsert segment with id %q: %w", id, err)
		}

		return api.Response{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert segkent with id %q", id)).WithError(err)
	}

	return responseUpsert, nil
}

func createResolveEntity(id string, externalId string, properties parameter.Properties, c *config.Config) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
		EntityName: externalId,
		Coordinate: c.Coordinate,
		Properties: properties,
	}
}

func getJsonResponseFromSegmentsResponse(rawResponse segment.Response) (jsonResponse, error) {
	var response jsonResponse
	err := json.Unmarshal(rawResponse.Data, &response)
	if err != nil {
		return jsonResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}
