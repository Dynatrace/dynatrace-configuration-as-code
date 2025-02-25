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
	"errors"
	"fmt"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

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

func isAPIErrorStatusNotFound(err error) bool {
	var apiErr api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.StatusCode == http.StatusNotFound
}
