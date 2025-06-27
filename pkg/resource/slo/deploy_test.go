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

package slo_test

import (
	"bytes"
	"context"
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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/slo"
)

type testClient struct {
	expectedPayload []byte
	listStub        func() (api.PagedListResponse, error)
	updateStub      func() (api.Response, error)
	createStub      func() (api.Response, error)
}

func (tc *testClient) List(_ context.Context) (api.PagedListResponse, error) {
	return tc.listStub()
}

func (tc *testClient) Update(_ context.Context, _ string, actualPayload []byte) (api.Response, error) {
	if tc.expectedPayload != nil {
		if !bytes.Equal(tc.expectedPayload, actualPayload) {
			return api.Response{}, fmt.Errorf("expected payload '%s' but got '%s'", string(tc.expectedPayload), string(actualPayload))
		}
	}

	return tc.updateStub()
}

func (tc *testClient) Create(_ context.Context, actualPayload []byte) (api.Response, error) {
	if tc.expectedPayload != nil {
		if !bytes.Equal(tc.expectedPayload, actualPayload) {
			return api.Response{}, fmt.Errorf("expected payload '%s' but got '%s'", string(tc.expectedPayload), string(actualPayload))
		}
	}

	return tc.createStub()
}

func TestDeploySuccess(t *testing.T) {
	testCoordinate := coordinate.Coordinate{
		Project:  "project",
		Type:     "slo-v2",
		ConfigId: "config-id",
	}
	tests := []struct {
		name                   string
		inputConfig            config.Config
		updateStub             func() (api.Response, error)
		createStub             func() (api.Response, error)
		listStub               func() (api.PagedListResponse, error)
		expected               entities.ResolvedEntity
		expectedRequestPayload []byte
	}{
		{
			name: "deploy with objectOriginId",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-object-id",
				Type:           config.ServiceLevelObjective{},
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			updateStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
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
			expectedRequestPayload: []byte("{\"externalId\":\"monaco-614c832a-b2c4-30c0-8e5b-f017366a4b1a\"}"),
		},
		{
			name: "deploy with externalId",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.ServiceLevelObjective{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				response := api.Response{
					StatusCode: http.StatusOK,
				}
				return response, nil
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				list := api.PagedListResponse{
					api.ListResponse{
						Response: api.Response{
							StatusCode: 200,
							Data:       []byte(`totalCount: 1, slos: [{"name": "some-name", "customSli": {"indicator": "some-query"}, "criteria": [{"warning": 95}], "tags": ["latency:500ms"], "id": "some-id", "version": "some-version"}]"`),
						},
						Objects: [][]byte{
							[]byte(`{"name": "some-name", "customSli": {"indicator": "some-query"}, "criteria": [{"warning": 95}], "tags": ["latency:500ms"], "id": "some-id", "version": "some-version", "externalId": "monaco-614c832a-b2c4-30c0-8e5b-f017366a4b1a"}`),
						},
					},
				}
				return list, nil
			},
			expected: entities.ResolvedEntity{
				Coordinate: testCoordinate,
				Properties: map[string]interface{}{
					"id": "some-id",
				},
				Skip: false,
			},
			expectedRequestPayload: []byte("{\"externalId\":\"monaco-614c832a-b2c4-30c0-8e5b-f017366a4b1a\"}"),
		},
		{
			name: "create new object on remote",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.ServiceLevelObjective{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusCreated,
					Data:       []byte(`{"name": "some-name", "customSli": {"indicator": "some-query"}, "criteria": [{"warning": 95}], "tags": ["latency:500ms"], "id": "some-id", "version": "some-version", "externalId": "external-id"}`),
				}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				list := api.PagedListResponse{
					api.ListResponse{
						Response: api.Response{
							StatusCode: 200,
							Data:       []byte(`totalCount: 0, slos: []"`),
						},
						Objects: [][]byte{
							[]byte(`{}`),
						},
					},
				}
				return list, nil
			},
			expected: entities.ResolvedEntity{
				Coordinate: testCoordinate,
				Properties: map[string]interface{}{
					"id": "some-id",
				},
				Skip: false,
			},
			expectedRequestPayload: []byte("{\"externalId\":\"monaco-614c832a-b2c4-30c0-8e5b-f017366a4b1a\"}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testClient{updateStub: tt.updateStub, listStub: tt.listStub, createStub: tt.createStub, expectedPayload: tt.expectedRequestPayload}

			props, errs := tt.inputConfig.ResolveParameterValues(entities.New())
			assert.Empty(t, errs)
			deployable := slo.NewDeployAPI(&c)

			resolvedEntity, err := deployable.Deploy(context.TODO(), props, "{}", &tt.inputConfig)

			assert.NoError(t, err)
			assert.Equal(t, resolvedEntity, tt.expected)
		})
	}
}

func TestDeployErrors(t *testing.T) {
	testCoordinate := coordinate.Coordinate{
		Project:  "project",
		Type:     "slo-v2",
		ConfigId: "config-id",
	}
	tests := []struct {
		name        string
		inputConfig config.Config
		updateStub  func() (api.Response, error)
		createStub  func() (api.Response, error)
		listStub    func() (api.PagedListResponse, error)
	}{
		{
			name: "deploy with objectOriginId - error at Update",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-object-id",
				Type:           config.ServiceLevelObjective{},
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			updateStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusBadRequest,
				}, api.NewAPIErrorFromResponse(&http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("{}"))})
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				t.Fatalf("should not be called")
				return nil, nil
			},
		},
		{
			name: "deploy with externalId - error at list",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.ServiceLevelObjective{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				return nil, api.NewAPIErrorFromResponse(&http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("{}"))})
			},
		},
		{
			name: "deploy with externalId find match - error on update",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.ServiceLevelObjective{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				response := api.Response{
					StatusCode: http.StatusBadRequest,
				}
				return response, api.NewAPIErrorFromResponse(&http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("{}"))})
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				list := api.PagedListResponse{
					api.ListResponse{
						Response: api.Response{
							StatusCode: 200,
							Data:       []byte(`totalCount: 1, slos: [{"name": "some-name", "customSli": {"indicator": "some-query"}, "criteria": [{"warning": 95}], "tags": ["latency:500ms"], "id": "some-id", "version": "some-version"}]"`),
						},
						Objects: [][]byte{
							[]byte(`{"name": "some-name", "customSli": {"indicator": "some-query"}, "criteria": [{"warning": 95}], "tags": ["latency:500ms"], "id": "some-id", "version": "some-version", "externalId": "monaco-614c832a-b2c4-30c0-8e5b-f017366a4b1a"}`),
						},
					},
				}
				return list, nil
			},
		},
		{
			name: "deploy with externalId find match - error at unmarshalling from list",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.ServiceLevelObjective{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				list := api.PagedListResponse{
					api.ListResponse{
						Response: api.Response{
							StatusCode: 200,
							Data:       []byte(`totalCoun`),
						},
						Objects: [][]byte{
							[]byte(`{"n`),
						},
					},
				}
				return list, nil
			},
		},
		{
			name: "create new remote object - error at create",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.ServiceLevelObjective{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				return api.Response{}, api.NewAPIErrorFromResponse(&http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("{}"))})
			},
			listStub: func() (api.PagedListResponse, error) {
				list := api.PagedListResponse{
					api.ListResponse{
						Response: api.Response{
							StatusCode: 200,
							Data:       []byte(`totalCount: 0, slos: []"`),
						},
						Objects: [][]byte{
							[]byte(`{}`),
						},
					},
				}
				return list, nil
			},
		},
		{
			name: "create new remote object - error at unmarshalling payload",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.ServiceLevelObjective{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			updateStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusCreated,
					Data:       []byte(`{"name": "some broken json`),
				}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				list := api.PagedListResponse{
					api.ListResponse{
						Response: api.Response{
							StatusCode: 200,
							Data:       []byte(`totalCount: 0, slos: []"`),
						},
						Objects: [][]byte{
							[]byte(`{}`),
						},
					},
				}
				return list, nil
			},
		},
		{
			name: "fails if payload is slo-v1",
			inputConfig: config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", `{"evaluationType": "AGGREGATE"}`),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-object-id",
				Type:           config.ServiceLevelObjective{},
				Parameters:     config.Parameters{},
				Skip:           false,
			},
			updateStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				t.Fatalf("should not be called")
				return nil, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testClient{updateStub: tt.updateStub, listStub: tt.listStub, createStub: tt.createStub}

			props, errs := tt.inputConfig.ResolveParameterValues(entities.New())
			assert.Empty(t, errs)

			templateContent, contentErr := tt.inputConfig.Template.Content()
			assert.NoError(t, contentErr)

			deployable := slo.NewDeployAPI(&c)

			_, err := deployable.Deploy(context.TODO(), props, templateContent, &tt.inputConfig)
			assert.Error(t, err)
		})
	}
}
