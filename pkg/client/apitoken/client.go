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

package apitoken

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
)

const apiTokenPath = "/api/v2/apiTokens"

type Response struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Enabled             bool              `json:"enabled"`
	PersonalAccessToken bool              `json:"personalAccessToken"`
	Owner               string            `json:"owner"`
	CreationDate        string            `json:"creationDate"`
	Scopes              []string          `json:"scopes"`
	LastUsedDate        *string           `json:"lastUsedDate,omitempty"`
	LastUsedIpAddress   *string           `json:"lastUsedIpAddress,omitempty"`
	ExpirationDate      *string           `json:"expirationDate,omitempty"`
	ModifiedDate        *string           `json:"modifiedDate,omitempty"`
	AdditionalMetadata  map[string]string `json:"additionalMetadata,omitempty"`
}

type source interface {
	POST(ctx context.Context, endpoint string, body io.Reader, options corerest.RequestOptions) (*http.Response, error)
}

// GetApiTokenMetadata returns the metadata of a specified API token
//
// Required scope: Any API token scope
func GetApiTokenMetadata(ctx context.Context, client source, apiToken string) (Response, error) {
	type request struct {
		ApiToken string `json:"token"`
	}
	req := request{apiToken}
	body, err := json.Marshal(req)

	if err != nil {
		return Response{}, fmt.Errorf("unable to marshal API token request data: %w", err)
	}

	resp, err := client.POST(ctx, apiTokenPath+"/lookup", bytes.NewReader(body), corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests})

	if err != nil {
		return Response{}, fmt.Errorf("failed to query API token metadata: %w", err)
	}

	response, err := coreapi.NewResponseFromHTTPResponse(resp)

	if err != nil {
		return Response{}, fmt.Errorf("failed to query API token metadata: %w", err)
	}

	data := Response{}
	err = json.Unmarshal(response.Data, &data)

	if err != nil {
		return Response{}, fmt.Errorf("failed to unmarshal API token metadata: %w", err)
	}

	return data, nil
}
