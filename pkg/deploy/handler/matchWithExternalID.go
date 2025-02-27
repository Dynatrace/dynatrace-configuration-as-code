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
	"encoding/json"
	"errors"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	deployErr "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

type MatchWithExternalIDHandler struct {
	BaseHandler
	IDKey         string
	ExternalIDKey string
	RemoteCall    func() ([][]byte, error)
}

func (h *MatchWithExternalIDHandler) Handle(data *HandlerData) (entities.ResolvedEntity, error) {
	//When List interface in client lib needs to be standardized, this can be removed
	payloadList, err := h.RemoteCall()
	if err != nil {
		return entities.ResolvedEntity{}, deployErr.NewFromErr(data.c, errors.Join(
			ErrDeployFailed{configID: data.c.Type.ID(), externalId: *data.externalID},
			err,
		))
	}

	var response map[string]any
	var id string
	for _, payload := range payloadList {
		if err := json.Unmarshal(payload, &response); err != nil {
			return entities.ResolvedEntity{}, deployErr.NewFromErr(data.c, errors.Join(
				ErrDeployFailed{configID: data.c.Type.ID(), externalId: *data.externalID},
				err,
			))
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
		return entities.ResolvedEntity{}, deployErr.NewFromErr(data.c, errors.Join(
			ErrDeployFailed{configID: data.c.Type.ID(), externalId: *data.externalID},
			err,
		))
	}

	return createResolveEntity(id, data.properties, data.c), nil
}
