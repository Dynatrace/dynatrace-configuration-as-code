/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package automationutils

import (
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
)

// Response is a "general" Response type holding the ID and the response payload
type Response struct {
	// ID is the identifier that will be used when creating a new automation object
	ID string `json:"id"`
	// Data is the whole body of an automation object
	Data []byte `json:"-"`
}

func DecodeResponse(r automation.Response) (Response, error) {
	d, err := api.DecodeJSON[Response](r)
	if err != nil {
		return Response{}, err
	}
	if d.ID == "" {
		return Response{}, fmt.Errorf("failed to decode response - id field missing")
	}

	d.Data = r.Data
	return d, nil
}

func DecodeListResponse(r automation.ListResponse) ([]Response, error) {
	rawResponses := r.All()
	res := make([]Response, len(rawResponses))
	for i, raw := range rawResponses {
		var v Response
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		if v.ID == "" {
			return nil, fmt.Errorf("failed to decode response - id field missing")
		}

		v.Data = raw
		res[i] = v
	}

	return res, nil
}
