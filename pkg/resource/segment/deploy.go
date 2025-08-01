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

package segment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

type DeploySource interface {
	Update(ctx context.Context, id string, data []byte) (api.Response, error)
	Create(ctx context.Context, data []byte) (api.Response, error)
	GetAll(ctx context.Context) ([]api.Response, error)
}
type jsonResponse struct {
	UID        string `json:"uid"`
	ExternalId string `json:"externalId"`
}

type DeployAPI struct {
	source DeploySource
}

func NewDeployAPI(source DeploySource) *DeployAPI {
	return &DeployAPI{source: source}
}

func (d DeployAPI) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	ctx = logr.NewContextWithSlogLogger(ctx, slog.Default())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	externalId := idutils.GenerateExternalID(c.Coordinate)
	requestPayload, err := addExternalId(externalId, renderedConfig)
	if err != nil {
		return entities.ResolvedEntity{}, fmt.Errorf("failed to add externalId to segments request payload: %w", err)
	}

	//Strategy 1 when OriginObjectId is set we update the object
	if c.OriginObjectId != "" {
		_, err := d.source.Update(ctx, c.OriginObjectId, requestPayload)
		if err == nil {
			return createResolveEntity(c.OriginObjectId, properties, c), nil
		}

		if !api.IsNotFoundError(err) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to deploy segment: %s", c.OriginObjectId)).WithError(err)
		}
	}

	//Strategy 2 is to try to find a match with external id and update it
	matchData, match, err := d.findMatchOnRemote(ctx, externalId)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("error finding segment with externalId: %s", externalId)).WithError(err)
	}

	if match {
		_, err := d.source.Update(ctx, matchData.UID, requestPayload)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update segment with externalId: %s", externalId)).WithError(err)
		}
		return createResolveEntity(matchData.UID, properties, c), nil
	}

	//Strategy 3 is to create a new segment object
	createResponse, err := d.source.Create(ctx, requestPayload)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to create segment with externalId: %s", externalId)).WithError(err)
	}

	responseData, err := getJsonResponseFromSegmentsResponse(createResponse)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to unmarshal segment with externalId: %s", externalId)).WithError(err)
	}

	return createResolveEntity(responseData.UID, properties, c), nil
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

func (d DeployAPI) findMatchOnRemote(ctx context.Context, externalId string) (jsonResponse, bool, error) {
	segmentsResponses, err := d.source.GetAll(ctx)
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

func createResolveEntity(id string, properties parameter.Properties, c *config.Config) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
	}
}

func getJsonResponseFromSegmentsResponse(rawResponse api.Response) (jsonResponse, error) {
	var response jsonResponse
	err := json.Unmarshal(rawResponse.Data, &response)
	if err != nil {
		return jsonResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}
