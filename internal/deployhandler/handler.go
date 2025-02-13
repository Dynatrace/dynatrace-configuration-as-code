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

package deployhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

type HandlerData struct {
	ctx        context.Context
	client     deploymentClient
	properties parameter.Properties
	payload    []byte
	c          *config.Config
	externalID *string
}

func NewHandlerData(ctx context.Context, client deploymentClient, properties parameter.Properties, payload []byte, c *config.Config) *HandlerData {
	return &HandlerData{ctx: ctx, client: client, properties: properties, payload: payload, c: c}
}

type deploymentClient interface {
	Update(ctx context.Context, id string, body []byte) (coreapi.Response, error)
	Create(ctx context.Context, body []byte) (coreapi.Response, error)
}

type Handler interface {
	Next(handler Handler) Handler
	Handle(data *HandlerData) (entities.ResolvedEntity, error)
}

type BaseHandler struct {
	next Handler
}

func (b *BaseHandler) Next(handler Handler) Handler {
	b.next = handler
	return handler
}

type AddExternalIDHandler struct {
	BaseHandler
}

func (h *AddExternalIDHandler) Handle(data *HandlerData) (entities.ResolvedEntity, error) {
	externalID, err := idutils.GenerateExternalIDForDocument(data.c.Coordinate)
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	data.payload, err = addExternalId(externalID, data.payload)
	data.externalID = &externalID

	if h.next != nil {
		return h.next.Handle(data)
	}

	return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(data.c, "OriginObjectIDHandler no next handler found")
}

// OriginObjectIDHandler will check if originObjectID is set and try to update the remote object matching the ID
type OriginObjectIDHandler struct {
	BaseHandler
}

func (o *OriginObjectIDHandler) Handle(data *HandlerData) (entities.ResolvedEntity, error) {
	if data.c.OriginObjectId != "" {
		_, err := data.client.Update(data.ctx, data.c.OriginObjectId, data.payload)
		if err == nil {
			return createResolveEntity(data.c.OriginObjectId, data.properties, data.c), nil
		}

		if !isAPIErrorStatusNotFound(err) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(data.c, fmt.Sprintf("failed to deploy slo: %s", data.c.OriginObjectId)).WithError(err)
		}
	}

	//This is temporary till all handlers are implemented
	if o.next != nil {
		return o.next.Handle(data)
	}

	return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(data.c, "OriginObjectIDHandler no next handler found")
}

type MatchWithExternalIDHandler struct {
	BaseHandler
	IDKey         string
	ExternalIDKey string
	RemoteCall    func() ([][]byte, error)
}

func (h *MatchWithExternalIDHandler) Handle(data *HandlerData) (entities.ResolvedEntity, error) {
	payloadList, err := h.RemoteCall() //@TODO List interface in client lib needs to be standardized
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	var response map[string]any
	var id string
	for _, payload := range payloadList {
		if err := json.Unmarshal(payload, &response); err != nil {
			return entities.ResolvedEntity{}, err
		}
		value, ok := response[h.ExternalIDKey].(string)
		if ok && value == *data.externalID {
			id = response[h.IDKey].(string)
			break
		}
	}
	//If no match is found we call the next handler
	if id == "" {
		return h.next.Handle(data)
	}

	_, err = data.client.Update(data.ctx, id, data.payload)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(data.c, fmt.Sprintf("error finding object with externalID: %s", *data.externalID)).WithError(err)
	}

	return createResolveEntity(id, data.properties, data.c), nil
}

type CreateHandler struct {
	BaseHandler
	IDKey string
}

func (h *CreateHandler) Handle(data *HandlerData) (entities.ResolvedEntity, error) {
	createResponse, err := data.client.Create(data.ctx, data.payload)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(data.c, fmt.Sprintf("failed to create object with externalID: %s", *data.externalID)).WithError(err)
	}

	id, err := getIDFromResponse(createResponse, h.IDKey)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(data.c, fmt.Sprintf("failed to unmarshal object with externalID: %s", *data.externalID)).WithError(err)
	}

	return createResolveEntity(id, data.properties, data.c), nil
}

func addExternalId(externalId string, payload []byte) ([]byte, error) {
	var request map[string]any
	err := json.Unmarshal(payload, &request)
	if err != nil {
		return nil, err
	}
	request["externalId"] = externalId
	return json.Marshal(request)
}

func createResolveEntity(id string, properties parameter.Properties, c *config.Config) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
	}
}

func isAPIErrorStatusNotFound(err error) bool {
	var apiErr api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.StatusCode == http.StatusNotFound
}

func getIDFromResponse(rawResponse api.Response, field string) (string, error) {
	var response map[string]any
	err := json.Unmarshal(rawResponse.Data, &response)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response[field].(string), nil
}
