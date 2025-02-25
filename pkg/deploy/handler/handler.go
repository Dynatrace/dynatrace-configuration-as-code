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

package handler

import (
	"context"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
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

func createResolveEntity(id string, properties parameter.Properties, c *config.Config) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
	}
}
