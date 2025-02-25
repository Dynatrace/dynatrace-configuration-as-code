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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

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

func addExternalId(externalId string, payload []byte) ([]byte, error) {
	var request map[string]any
	err := json.Unmarshal(payload, &request)
	if err != nil {
		return nil, err
	}
	request["externalId"] = externalId
	return json.Marshal(request)
}
