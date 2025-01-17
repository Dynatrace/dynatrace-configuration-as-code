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
	"net/http"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	segment "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/go-logr/logr"
)

type DeploySegmentClient interface {
	Upsert(ctx context.Context, id string, data []byte) (segment.Response, error)
	GetAll(ctx context.Context) ([]segment.Response, error)
}

type jsonResponse struct {
	UID        string `json:"uid"`
	Owner      string `json:"owner"`
	ExternalId string `json:"externalId"`
}

func Deploy(ctx context.Context, client DeploySegmentClient, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	externalId := idutils.GenerateUUIDFromCoordinate(c.Coordinate)
	requestPayload, err := addExternalId(externalId, renderedConfig)
	if err != nil {
		return entities.ResolvedEntity{}, fmt.Errorf("failed to add externalId to segments request payload: %w", err)
	}

	//Strategy 1 when OriginObjectId is set we try to get the object if it exists we update it else we create it.
	if c.OriginObjectId != "" {
		id, err := deployWithOriginObjectId(ctx, client, c, requestPayload)
		if err != nil {
			return entities.ResolvedEntity{}, fmt.Errorf("failed to deploy segment with externalId: %s : %w", externalId, err)
		}

		return createResolveEntity(id, properties, c), nil
	}

	//Strategy 2 is to try to find a match with external id and either update or create object if no match found.
	id, err := deployWithExternalID(ctx, client, externalId, requestPayload, c)
	if err != nil {
		return entities.ResolvedEntity{}, fmt.Errorf("failed to deploy segment with externalId: %s : %w", externalId, err)
	}

	return createResolveEntity(id, properties, c), nil
}

func addExternalId(externalId string, renderedConfig string) ([]byte, error) {
	var request map[string]any
	err := json.Unmarshal([]byte(renderedConfig), &request)
	if err != nil {
		return nil, err
	}
	request["externalId"] = externalId
	return json.Marshal(request)
}

func deployWithExternalID(ctx context.Context, client DeploySegmentClient, externalId string, requestPayload []byte, c *config.Config) (string, error) {
	id := ""
	responseData, match, err := findMatchOnRemote(ctx, client, externalId)
	if err != nil {
		return "", err
	}

	if match {
		id = responseData.UID
	}

	responseUpsert, err := deploy(ctx, client, id, requestPayload, c)
	if err != nil {
		return "", err
	}

	id, err = resolveIdFromResponse(responseUpsert, id)
	if err != nil {
		return "", err
	}

	return id, nil
}

func deployWithOriginObjectId(ctx context.Context, client DeploySegmentClient, c *config.Config, requestPayload []byte) (string, error) {
	responseUpsert, err := deploy(ctx, client, c.OriginObjectId, requestPayload, c)
	if err != nil {
		return "", err
	}

	return resolveIdFromResponse(responseUpsert, c.OriginObjectId)
}

func resolveIdFromResponse(responseUpsert segment.Response, id string) (string, error) {
	//For a POST we need to parse the response again to read out the ID
	if responseUpsert.StatusCode == http.StatusCreated {
		responseData, err := getJsonResponseFromSegmentsResponse(responseUpsert)
		if err != nil {
			return "", err
		}
		return responseData.UID, nil
	}
	return id, nil
}

func findMatchOnRemote(ctx context.Context, client DeploySegmentClient, externalId string) (jsonResponse, bool, error) {
	segmentsResponses, err := client.GetAll(ctx)
	if err != nil {
		return jsonResponse{}, false, fmt.Errorf("failed to GET segments: %w", err)
	}

	var responseData jsonResponse
	for _, segmentResponse := range segmentsResponses {
		responseData, err = getJsonResponseFromSegmentsResponse(segmentResponse)
		if err != nil {
			return jsonResponse{}, false, err
		}
		if responseData.ExternalId == externalId {
			return responseData, true, nil
		}
	}

	return jsonResponse{}, false, nil
}

func deploy(ctx context.Context, client DeploySegmentClient, id string, requestPayload []byte, c *config.Config) (segment.Response, error) {
	//create new context to carry logger
	ctx = logr.NewContext(ctx, log.WithCtxFields(ctx).GetLogr())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	responseUpsert, err := client.Upsert(ctx, id, requestPayload)
	if err != nil {
		var apiErr api.APIError
		if errors.As(err, &apiErr) {
			return api.Response{}, fmt.Errorf("failed to upsert segment with id %q: %w", id, err)
		}

		return api.Response{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert segment with id %q", id)).WithError(err)
	}

	return responseUpsert, nil
}

func createResolveEntity(id string, properties parameter.Properties, c *config.Config) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
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
