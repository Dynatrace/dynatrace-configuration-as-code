/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package credentialvault_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/credentialvault"
)

type testClient struct {
	expectedPayload []byte
	listStub        func() (api.PagedListResponse, error)
	updateStub      func() (api.Response, error)
	createStub      func() (api.Response, error)
}

const payload = `{"name":"my-name"}`

var createResponse = []byte(`{"id": "my-id"}`)
var listItemResponse = []byte(`{"id": "my-id", "name": "my-name"}`)

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
		Type:     string(config.CredentialVaultID),
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
				Template:       template.NewInMemoryTemplate("path/file.json", payload),
				Coordinate:     testCoordinate,
				OriginObjectId: "my-id",
				Type:           config.CredentialVault{},
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
				Properties: map[string]any{
					"id": "my-id",
				},
				Skip: false,
			},
			expectedRequestPayload: []byte(payload),
		},
		{
			name: "deploy with name",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", payload),
				Coordinate: testCoordinate,
				Type:       config.CredentialVault{},
				Parameters: config.Parameters{
					config.NameParameter: &value.ValueParameter{Value: "my-name"},
				},
				Skip: false,
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
						StatusCode: 200,
						Objects: [][]byte{
							listItemResponse,
						},
					},
				}
				return list, nil
			},
			expected: entities.ResolvedEntity{
				Coordinate: testCoordinate,
				Properties: map[string]any{
					"id":   "my-id",
					"name": "my-name",
				},
				Skip: false,
			},
			expectedRequestPayload: []byte(payload),
		},
		{
			name: "create new object on remote",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", payload),
				Coordinate: testCoordinate,
				Type:       config.CredentialVault{},
				Parameters: config.Parameters{
					config.NameParameter: &value.ValueParameter{Value: "my-name"},
				},
				Skip: false,
			},
			updateStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusCreated,
					Data:       createResponse,
				}, nil
			},
			listStub: func() (api.PagedListResponse, error) {
				list := api.PagedListResponse{
					api.ListResponse{
						StatusCode: 200,
						Objects: [][]byte{
							[]byte(`{}`),
						},
					},
				}
				return list, nil
			},
			expected: entities.ResolvedEntity{
				Coordinate: testCoordinate,
				Properties: map[string]any{
					"id":   "my-id",
					"name": "my-name",
				},
				Skip: false,
			},
			expectedRequestPayload: []byte(payload),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testClient{updateStub: tt.updateStub, listStub: tt.listStub, createStub: tt.createStub, expectedPayload: tt.expectedRequestPayload}

			props, errs := tt.inputConfig.ResolveParameterValues(entities.New())
			require.Empty(t, errs)
			deployable := credentialvault.NewDeployAPI(&c)

			resolvedEntity, err := deployable.Deploy(t.Context(), props, payload, &tt.inputConfig)

			require.NoError(t, err)
			assert.Equal(t, resolvedEntity, tt.expected)
		})
	}
}

func TestDeployErrors(t *testing.T) {
	testCoordinate := coordinate.Coordinate{
		Project:  "project",
		Type:     string(config.CredentialVaultID),
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
				OriginObjectId: "my-id",
				Type:           config.CredentialVault{},
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
			name: "deploy with name - error at list",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.CredentialVault{},
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
			name: "deploy with name find match - error on update",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.CredentialVault{},
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
						StatusCode: 200,
						Objects: [][]byte{
							listItemResponse,
						},
					},
				}
				return list, nil
			},
		},
		{
			name: "deploy with name find match - error at unmarshalling from list",
			inputConfig: config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoordinate,
				Type:       config.CredentialVault{},
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
						StatusCode: 200,
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
				Type:       config.CredentialVault{},
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
						StatusCode: 200,
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
				Type:       config.CredentialVault{},
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
						StatusCode: 200,
						Objects: [][]byte{
							[]byte(`{}`),
						},
					},
				}
				return list, nil
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

			deployable := credentialvault.NewDeployAPI(&c)

			_, err := deployable.Deploy(t.Context(), props, templateContent, &tt.inputConfig)
			assert.Error(t, err)
		})
	}
}
