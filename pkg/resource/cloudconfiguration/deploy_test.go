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

package cloudconfiguration_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/cloudconfiguration"
)

type testClient struct {
	updateStub func(id string) (api.Response, error)
	createStub func() (api.Response, error)
}

func (tc *testClient) Update(_ context.Context, id string, _ []byte) (api.Response, error) {
	return tc.updateStub(id)
}

func (tc *testClient) Create(_ context.Context, _ []byte) (api.Response, error) {
	return tc.createStub()
}

func testConfig(originObjectID string) config.Config {
	return config.Config{
		Template: template.NewInMemoryTemplate("path/file.json", "{}"),
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "cloud-configuration",
			ConfigId: "config-id",
		},
		OriginObjectId: originObjectID,
		Type:           config.CloudConfiguration{},
		Parameters:     config.Parameters{},
	}
}

func TestDeploySuccess(t *testing.T) {
	t.Run("update by originObjectId", func(t *testing.T) {
		c := testClient{
			updateStub: func(id string) (api.Response, error) {
				assert.Equal(t, "my-object-id", id)
				return api.Response{StatusCode: http.StatusOK}, nil
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
		}

		inputConfig := testConfig("my-object-id")
		props, errs := inputConfig.ResolveParameterValues(entities.New())
		assert.Empty(t, errs)

		resolvedEntity, err := cloudconfiguration.NewDeployAPI(&c).Deploy(t.Context(), props, "{}", &inputConfig)

		assert.NoError(t, err)
		assert.Equal(t, entities.ResolvedEntity{
			Coordinate: inputConfig.Coordinate,
			Properties: map[string]any{"id": "my-object-id"},
		}, resolvedEntity)
	})

	t.Run("create when no originObjectId", func(t *testing.T) {
		c := testClient{
			updateStub: func(_ string) (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusCreated,
					Data:       []byte(`{"id": "new-id", "displayName": "my-config"}`),
				}, nil
			},
		}

		inputConfig := testConfig("")
		props, errs := inputConfig.ResolveParameterValues(entities.New())
		assert.Empty(t, errs)

		resolvedEntity, err := cloudconfiguration.NewDeployAPI(&c).Deploy(t.Context(), props, "{}", &inputConfig)

		assert.NoError(t, err)
		assert.Equal(t, entities.ResolvedEntity{
			Coordinate: inputConfig.Coordinate,
			Properties: map[string]any{"id": "new-id"},
		}, resolvedEntity)
	})

	t.Run("create when update returns not found", func(t *testing.T) {
		c := testClient{
			updateStub: func(_ string) (api.Response, error) {
				return api.Response{}, api.APIError{StatusCode: http.StatusNotFound}
			},
			createStub: func() (api.Response, error) {
				return api.Response{
					StatusCode: http.StatusCreated,
					Data:       []byte(`{"id": "new-id"}`),
				}, nil
			},
		}

		inputConfig := testConfig("stale-object-id")
		props, errs := inputConfig.ResolveParameterValues(entities.New())
		assert.Empty(t, errs)

		resolvedEntity, err := cloudconfiguration.NewDeployAPI(&c).Deploy(t.Context(), props, "{}", &inputConfig)

		assert.NoError(t, err)
		assert.Equal(t, "new-id", resolvedEntity.Properties[config.IdParameter])
	})
}

func TestDeployErrors(t *testing.T) {
	t.Run("update fails with non-404 error", func(t *testing.T) {
		c := testClient{
			updateStub: func(_ string) (api.Response, error) {
				return api.Response{}, errors.New("some unexpected error")
			},
			createStub: func() (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
		}

		inputConfig := testConfig("my-object-id")
		props, errs := inputConfig.ResolveParameterValues(entities.New())
		assert.Empty(t, errs)

		_, err := cloudconfiguration.NewDeployAPI(&c).Deploy(t.Context(), props, "{}", &inputConfig)
		assert.Error(t, err)
	})

	t.Run("create fails", func(t *testing.T) {
		c := testClient{
			updateStub: func(_ string) (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				return api.Response{}, errors.New("some unexpected error")
			},
		}

		inputConfig := testConfig("")
		props, errs := inputConfig.ResolveParameterValues(entities.New())
		assert.Empty(t, errs)

		_, err := cloudconfiguration.NewDeployAPI(&c).Deploy(t.Context(), props, "{}", &inputConfig)
		assert.Error(t, err)
	})

	t.Run("create response missing id", func(t *testing.T) {
		c := testClient{
			updateStub: func(_ string) (api.Response, error) {
				t.Fatalf("should not be called")
				return api.Response{}, nil
			},
			createStub: func() (api.Response, error) {
				return api.Response{StatusCode: http.StatusCreated, Data: []byte(`{}`)}, nil
			},
		}

		inputConfig := testConfig("")
		props, errs := inputConfig.ResolveParameterValues(entities.New())
		assert.Empty(t, errs)

		_, err := cloudconfiguration.NewDeployAPI(&c).Deploy(t.Context(), props, "{}", &inputConfig)
		assert.Error(t, err)
	})
}
