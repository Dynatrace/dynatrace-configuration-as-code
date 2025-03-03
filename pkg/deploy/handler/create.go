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
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	deployErr "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

var (
	ErrUnmarshal = errors.New("error unmarshalling payload")
	ErrCreate    = errors.New("error creating remote resource")
)

type CreateHandler struct {
	BaseHandler
	IDKey string
}

func (h *CreateHandler) Handle(data *HandlerData) (entities.ResolvedEntity, error) {
	createResponse, err := data.client.Create(data.ctx, data.payload)
	if err != nil {
		return entities.ResolvedEntity{}, deployErr.NewFromErr(data.c,
			ErrDeployFailed{configID: data.c.Type.ID(), externalId: *data.externalID},
			ErrCreate,
			err,
		)
	}

	id, err := getIDFromResponse(createResponse, h.IDKey)
	if err != nil {
		return entities.ResolvedEntity{}, deployErr.NewFromErr(data.c,
			ErrDeployFailed{configID: data.c.Type.ID(), externalId: *data.externalID},
			ErrUnmarshal,
			err,
		)
	}

	return createResolveEntity(id, data.properties, data.c), nil
}

func getIDFromResponse(rawResponse api.Response, field string) (string, error) {
	var response map[string]any
	err := json.Unmarshal(rawResponse.Data, &response)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response[field].(string), nil
}
