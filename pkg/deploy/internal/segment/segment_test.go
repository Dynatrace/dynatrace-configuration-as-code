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

package segment_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/segment"
)

type testClient struct {
	updateStub func() (api.Response, error)
	createStub func() (api.Response, error)
	getAllStub func() ([]api.Response, error)
}

func (tc *testClient) Update(_ context.Context, _ string, _ []byte) (api.Response, error) {
	return tc.updateStub()
}

func (tc *testClient) GetAll(_ context.Context) ([]api.Response, error) {
	return tc.getAllStub()
}

func (tc *testClient) Create(_ context.Context, _ []byte) (api.Response, error) {
	return tc.createStub()
}

func TestDeploy(t *testing.T) {
	testCoordinate := coordinate.Coordinate{
		Project:  "my-project",
		Type:     "segment",
		ConfigId: "my-config-id",
	}
	tests := []struct {
		name        string
		inputConfig config.Config
		updateStub  func() (api.Response, error)
		createStub  func() (api.Response, error)
		getAllStub  func() ([]api.Response, error)
		expected    entities.ResolvedEntity
		expectErr   bool
	}{
		{
			name: "deploy with objectOriginId - success PUT",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-object-id",
				Type:           config.Segment{},
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			updateStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
			createStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
			getAllStub: func() ([]api.Response, error) {
				t.Fatalf("should not be called")
				return nil, nil
			},
			expected: entities.ResolvedEntity{
				Coordinate: testCoordinate,
				Properties: map[string]interface{}{
					"id": "my-object-id",
				},
				Skip: false,
			},
			expectErr: false,
		},
		{
			name: "deploy with objectOriginId - error PUT(error returned by upsert)",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-object-id",
				Type:           config.Segment{},
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			createStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
			updateStub: func() (api.Response, error) {
				return api.Response{}, fmt.Errorf("error")
			},
			getAllStub: func() ([]api.Response, error) {
				t.Fatalf("should not be called")
				return nil, nil
			},
			expectErr: true,
		},
		{
			name: "deploy with objectOriginId - success POST",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoordinate,
				Type:           config.Segment{},
				OriginObjectId: "my-object-id",
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			updateStub: func() (api.Response, error) {
				return api.Response{}, api.NewAPIErrorFromResponse(&http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("{}"))})
			},
			createStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusOK,
					Data: marshal(map[string]any{
						"uid":         "JMhNaJ0Zbf9",
						"name":        "no-match",
						"description": "post - update from monaco - change - 2",
						"isPublic":    false,
						"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
						"externalId":  "monaco-e2320031-d6c6-3c83-9706-b3e82b834129",
					}, t),
				}, nil
			},
			getAllStub: func() ([]api.Response, error) {
				var response []api.Response
				return response, nil
			},
			expected: entities.ResolvedEntity{
				Coordinate: testCoordinate,
				Properties: map[string]interface{}{
					"id": "JMhNaJ0Zbf9",
				},
				Skip: false,
			},
			expectErr: false,
		},
		{
			name: "deploy with externalId - success PUT",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.Segment{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
			getAllStub: func() ([]api.Response, error) {
				response := []api.Response{
					{
						StatusCode: http.StatusOK,
						Data: marshal(map[string]any{
							"uid":         "JMhNaJ0Zbf9",
							"name":        "no-match",
							"description": "post - update from monaco - change - 2",
							"isPublic":    false,
							"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
							"externalId":  "monaco-e2320031-d6c6-3c83-9706-b3e82b834129",
						}, t),
					},
					{
						StatusCode: http.StatusOK,
						Data: marshal(map[string]any{
							"uid":         "should-not-be-this-id",
							"name":        "match",
							"description": "post - update from monaco - change - 2",
							"isPublic":    false,
							"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
							"externalId":  "not-a-match",
						}, t),
					},
				}
				return response, nil
			},
			expected: entities.ResolvedEntity{
				Coordinate: testCoordinate,
				Properties: map[string]interface{}{
					"id": "JMhNaJ0Zbf9",
				},
				Skip: false,
			},
			expectErr: false,
		},
		{
			name: "deploy with externalId - error POST 400",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.Segment{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			createStub: func() (api.Response, error) {
				return api.Response{}, api.APIError{
					StatusCode: http.StatusBadRequest,
				}
			},
			getAllStub: func() ([]api.Response, error) {
				var response []api.Response
				return response, nil
			},
			expectErr: true,
		},
		{
			name: "deploy with externalId - error GET 400",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.Segment{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			getAllStub: func() ([]api.Response, error) {
				var response []api.Response
				return response, api.APIError{
					StatusCode: http.StatusBadRequest,
				}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testClient{updateStub: tt.updateStub, getAllStub: tt.getAllStub, createStub: tt.createStub}

			props, errs := tt.inputConfig.ResolveParameterValues(entities.New())
			assert.Empty(t, errs)

			renderedConfig, err := tt.inputConfig.Render(props)
			assert.NoError(t, err)

			resolvedEntity, err := segment.Deploy(t.Context(), &c, props, renderedConfig, &tt.inputConfig)
			if tt.expectErr {
				assert.Error(t, err)
			}
			if !tt.expectErr {
				assert.NoError(t, err)
				assert.Equal(t, resolvedEntity, tt.expected)
			}
		})
	}
}

func marshal(object map[string]any, t *testing.T) []byte {
	payload, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("error marshalling object: %v", err)
	}
	return payload
}
