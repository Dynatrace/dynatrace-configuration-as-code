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
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/segment"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

type testClient struct {
	upsertStub func() (segments.Response, error)
	getStub    func() (segments.Response, error)
	getAllStub func() ([]segments.Response, error)
}

func (tc *testClient) Upsert(_ context.Context, _ string, _ []byte) (segments.Response, error) {
	return tc.upsertStub()
}

func (tc *testClient) Get(_ context.Context, _ string) (segments.Response, error) {
	return tc.getStub()
}

func (tc *testClient) GetAll(_ context.Context) ([]segments.Response, error) {
	return tc.getAllStub()
}

func TestDeploy(t *testing.T) {
	testCoordinate := coordinate.Coordinate{
		Project:  "my-project",
		Type:     "segment",
		ConfigId: "my-config-id",
	}
	tests := []struct {
		name           string
		inputConfig    config.Config
		upsertStub     func() (segments.Response, error)
		getStub        func() (segments.Response, error)
		getAllStub     func() ([]segments.Response, error)
		expected       entities.ResolvedEntity
		expectErr      bool
		expectedErrMsg string
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
			upsertStub: func() (segments.Response, error) {
				return segments.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
			getStub: func() (segments.Response, error) {
				return segments.Response{
					StatusCode: http.StatusOK,
					Data: marshal(map[string]any{
						"uid":         "JMhNaJ0Zbf9",
						"name":        "test-segment-post-match",
						"description": "post - update from monaco - change - 2",
						"isPublic":    false,
						"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
						"externalId":  "project_segment:segment:some-id",
					}),
				}, nil
			},
			getAllStub: func() ([]segments.Response, error) {
				panic("should not be called")
			},
			expected: entities.ResolvedEntity{
				EntityName: testCoordinate.String(),
				Coordinate: testCoordinate,
				Properties: map[string]interface{}{
					"id": "my-object-id",
				},
				Skip: false,
			},
			expectErr: false,
		},
		{
			name: "deploy with objectOriginId, no object found on remote - success POST with externalId",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-object-id",
				Type:           config.Segment{},
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			upsertStub: func() (segments.Response, error) {
				return segments.Response{
					StatusCode: http.StatusCreated,
					Data: marshal(map[string]any{
						"uid":         "JMhNaJ0Zbf9",
						"name":        "test-segment-post-match",
						"description": "post - update from monaco - change - 2",
						"isPublic":    false,
						"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
						"externalId":  "project_segment:segment:some-id",
					}),
				}, nil
			},
			getStub: func() (segments.Response, error) {
				return segments.Response{}, api.APIError{
					StatusCode: http.StatusNotFound,
				}
			},
			getAllStub: func() ([]segments.Response, error) {
				var response []segments.Response
				return response, nil
			},
			expected: entities.ResolvedEntity{
				EntityName: testCoordinate.String(),
				Coordinate: testCoordinate,
				Properties: map[string]interface{}{
					"id": "JMhNaJ0Zbf9",
				},
				Skip: false,
			},
			expectErr: false,
		},
		{
			name: "deploy with objectOriginId - error PUT",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-object-id",
				Type:           config.Segment{},
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			upsertStub: func() (segments.Response, error) {
				return segments.Response{}, fmt.Errorf("error")
			},
			getStub: func() (segments.Response, error) {
				return segments.Response{
					StatusCode: http.StatusOK,
					Data: marshal(map[string]any{
						"uid":         "JMhNaJ0Zbf9",
						"name":        "test-segment-post-match",
						"description": "post - update from monaco - change - 2",
						"isPublic":    false,
						"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
						"externalId":  "project_segment:segment:some-id",
					}),
				}, nil
			},
			getAllStub: func() ([]segments.Response, error) {
				panic("should not be called")
			},
			expectErr:      true,
			expectedErrMsg: "failed to deploy segment with externalId",
		},
		{
			name: "deploy with objectOriginId - error PUT",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-object-id",
				Type:           config.Segment{},
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			upsertStub: func() (segments.Response, error) {
				return segments.Response{}, fmt.Errorf("error")
			},
			getStub: func() (segments.Response, error) {
				return segments.Response{
					StatusCode: http.StatusOK,
					Data: marshal(map[string]any{
						"uid":         "JMhNaJ0Zbf9",
						"name":        "test-segment-post-match",
						"description": "post - update from monaco - change - 2",
						"isPublic":    false,
						"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
						"externalId":  "project_segment:segment:some-id",
					}),
				}, nil
			},
			getAllStub: func() ([]segments.Response, error) {
				panic("should not be called")
			},
			expectErr:      true,
			expectedErrMsg: "failed to deploy segment with externalId",
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
			upsertStub: func() (segments.Response, error) {
				return segments.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
			getStub: func() (segments.Response, error) {
				panic("should not be called")
			},
			getAllStub: func() ([]segments.Response, error) {
				response := []segments.Response{
					{
						StatusCode: http.StatusOK,
						Data: marshal(map[string]any{
							"uid":         "JMhNaJ0Zbf9",
							"name":        "no-match",
							"description": "post - update from monaco - change - 2",
							"isPublic":    false,
							"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
							"externalId":  "project_segment:segment:some-id-no-match",
						}),
					},
					{
						StatusCode: http.StatusOK,
						Data: marshal(map[string]any{
							"uid":         "JMhNaJ0Zbf9",
							"name":        "match",
							"description": "post - update from monaco - change - 2",
							"isPublic":    false,
							"owner":       "79a4c92e-379b-4cd7-96a3-78a601b6a69b",
							"externalId":  "my-project:segment:my-config-id",
						}),
					},
				}
				return response, nil
			},
			expected: entities.ResolvedEntity{
				EntityName: testCoordinate.String(),
				Coordinate: testCoordinate,
				Properties: map[string]interface{}{
					"id": "JMhNaJ0Zbf9",
				},
				Skip: false,
			},
			expectErr: false,
		},
		{
			name: "deploy with externalId - error PUT 400",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.Segment{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			upsertStub: func() (segments.Response, error) {
				return segments.Response{}, api.APIError{
					StatusCode: http.StatusBadRequest,
				}
			},
			getStub: func() (segments.Response, error) {
				panic("should not be called")
			},
			getAllStub: func() ([]segments.Response, error) {
				var response []segments.Response
				return response, nil
			},
			expectErr:      true,
			expectedErrMsg: "failed to deploy segment with externalId",
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
			upsertStub: func() (segments.Response, error) {
				panic("should not be called")
			},
			getStub: func() (segments.Response, error) {
				panic("should not be called")
			},
			getAllStub: func() ([]segments.Response, error) {
				var response []segments.Response
				return response, api.APIError{
					StatusCode: http.StatusBadRequest,
				}
			},
			expectErr:      true,
			expectedErrMsg: "failed to deploy segment with externalId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testClient{upsertStub: tt.upsertStub, getStub: tt.getStub, getAllStub: tt.getAllStub}

			props, errs := tt.inputConfig.ResolveParameterValues(entities.New())
			assert.Empty(t, errs)

			renderedConfig, err := tt.inputConfig.Render(props)
			assert.NoError(t, err)

			resolvedEntity, err := segment.Deploy(context.Background(), &c, props, renderedConfig, &tt.inputConfig)
			if tt.expectErr {
				assert.ErrorContains(t, err, tt.expectedErrMsg)
			}
			if !tt.expectErr {
				assert.NoError(t, err)
				assert.Equal(t, resolvedEntity, tt.expected)
			}
		})
	}
}

func marshal(object map[string]any) []byte {
	payload, err := json.Marshal(object)
	if err != nil {
		panic(err)
	}
	return payload
}
