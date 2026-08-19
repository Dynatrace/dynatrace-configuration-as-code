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

package cloudconfiguration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
)

const endpointPath = "platform-reserved/business-analytics/v1/cloud-configurations"

type Client struct {
	restClient *rest.Client
}

func NewClient(client *rest.Client) *Client {
	return &Client{restClient: client}
}

func (c *Client) List(ctx context.Context) (api.ListResponse, error) {
	resp, err := c.restClient.GET(ctx, endpointPath, rest.RequestOptions{})
	if err != nil {
		return api.ListResponse{}, fmt.Errorf("failed to list cloud configurations: %w", err)
	}
	return processListResponse(resp)
}

func (c *Client) Get(ctx context.Context, id string) (api.Response, error) {
	if id == "" {
		return api.Response{}, fmt.Errorf("failed to get cloud configuration: %w", errors.New(`argument "id" is empty`))
	}
	path, err := url.JoinPath(endpointPath, id)
	if err != nil {
		return api.Response{}, fmt.Errorf("failed to get cloud configuration with id %s: %w", id, err)
	}
	resp, err := c.restClient.GET(ctx, path, rest.RequestOptions{})
	if err != nil {
		return api.Response{}, fmt.Errorf("failed to get cloud configuration with id %s: %w", id, err)
	}
	return api.NewResponseFromHTTPResponse(resp)
}

func (c *Client) Create(ctx context.Context, data []byte) (api.Response, error) {
	resp, err := c.restClient.POST(ctx, endpointPath, bytes.NewReader(data), rest.RequestOptions{})
	if err != nil {
		return api.Response{}, fmt.Errorf("failed to create cloud configuration: %w", err)
	}
	return api.NewResponseFromHTTPResponse(resp)
}

func (c *Client) Update(ctx context.Context, id string, data []byte) (api.Response, error) {
	if id == "" {
		return api.Response{}, fmt.Errorf("failed to update cloud configuration: %w", errors.New(`argument "id" is empty`))
	}
	path, err := url.JoinPath(endpointPath, id)
	if err != nil {
		return api.Response{}, fmt.Errorf("failed to update cloud configuration with id %s: %w", id, err)
	}
	resp, err := c.restClient.PUT(ctx, path, bytes.NewReader(data), rest.RequestOptions{})
	if err != nil {
		return api.Response{}, fmt.Errorf("failed to update cloud configuration with id %s: %w", id, err)
	}
	return api.NewResponseFromHTTPResponse(resp)
}

func (c *Client) Delete(ctx context.Context, id string) (api.Response, error) {
	if id == "" {
		return api.Response{}, fmt.Errorf("failed to delete cloud configuration: %w", errors.New(`argument "id" is empty`))
	}
	path, err := url.JoinPath(endpointPath, id)
	if err != nil {
		return api.Response{}, fmt.Errorf("failed to delete cloud configuration with id %s: %w", id, err)
	}
	resp, err := c.restClient.DELETE(ctx, path, rest.RequestOptions{})
	if err != nil {
		return api.Response{}, fmt.Errorf("failed to delete cloud configuration with id %s: %w", id, err)
	}
	return api.NewResponseFromHTTPResponse(resp)
}

func processListResponse(httpResp *http.Response) (api.ListResponse, error) {
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return api.ListResponse{}, api.NewAPIErrorFromResponse(httpResp)
	}

	if !rest.IsSuccess(httpResp) {
		return api.ListResponse{}, api.NewAPIErrorFromResponseAndBody(httpResp, body)
	}

	var s struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(body, &s); err != nil {
		return api.ListResponse{}, api.NewAPIErrorFromResponseAndBody(httpResp, body)
	}

	var objects [][]byte
	for _, item := range s.Items {
		objects = append(objects, item)
	}

	return api.ListResponse{
		Response: api.Response{
			StatusCode: httpResp.StatusCode,
			Header:     httpResp.Header,
			Request:    api.NewRequestInfoFromRequest(httpResp.Request),
		},
		Objects: objects,
	}, nil
}
