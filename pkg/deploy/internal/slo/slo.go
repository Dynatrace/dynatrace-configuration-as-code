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

package slo

import (
	"context"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/deployhandler"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/go-logr/logr"
)

type deployServiceLevelObjectiveClient interface {
	List(ctx context.Context) (api.PagedListResponse, error)
	Update(ctx context.Context, id string, data []byte) (api.Response, error)
	Create(ctx context.Context, data []byte) (api.Response, error)
}

func Deploy(ctx context.Context, client deployServiceLevelObjectiveClient, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	ctx = logr.NewContext(ctx, log.WithCtxFields(ctx).GetLogr())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	data := deployhandler.NewHandlerData(ctx, client, properties, []byte(renderedConfig), c)
	addExternalIdHandler := deployhandler.AddExternalIDHandler{}
	deployWithOriginObjectID := deployhandler.OriginObjectIDHandler{}
	matchWithExternalIDHandler := deployhandler.MatchWithExternalIDHandler{
		ExternalIDKey: "externalId",
		IDKey:         "id",
		RemoteCall: func() ([][]byte, error) {
			apiResponse, err := client.List(ctx)
			if err != nil {
				return nil, err
			}

			return apiResponse.All(), nil
		},
	}
	createHandler := deployhandler.CreateHandler{IDKey: "id"}
	addExternalIdHandler.Next(&deployWithOriginObjectID).Next(&matchWithExternalIDHandler).Next(&createHandler)

	return addExternalIdHandler.Handle(data)
}
