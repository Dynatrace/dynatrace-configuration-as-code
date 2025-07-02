//go:build unit

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

package bucket_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
)

func getBucketActiveResponse(bucketName string) []byte {
	return []byte(fmt.Sprintf(`{
		 "bucketName": "%s",
		 "table": "metrics",
		 "displayName": "Default metrics (15 months)",
		 "status": "active",
		 "retentionDays": 462,
		 "metricInterval": "PT1M",
		 "version": 1
	}`, bucketName))
}

type testClient struct {
	get    func(_ context.Context, bucketName string) (api.Response, error)
	update func(_ context.Context, bucketName string, data []byte) (api.Response, error)
	create func(_ context.Context, bucketName string, data []byte) (api.Response, error)
}

func (c testClient) Get(ctx context.Context, bucketName string) (api.Response, error) {
	if c.get != nil {
		return c.get(ctx, bucketName)
	}
	return api.Response{}, api.APIError{StatusCode: http.StatusNotFound}
}

func (c testClient) Update(ctx context.Context, bucketName string, data []byte) (api.Response, error) {
	return c.update(ctx, bucketName, data)
}

func (c testClient) Create(ctx context.Context, bucketName string, data []byte) (api.Response, error) {
	return c.create(ctx, bucketName, data)
}

func TestDeploy(t *testing.T) {

	testCoord := coordinate.Coordinate{
		Project:  "proj",
		Type:     "bucket",
		ConfigId: "my-bucket",
	}

	t.Run("creates by generated coordinate ID", func(t *testing.T) {
		client := testClient{
			get: func(_ context.Context, bucketName string) (api.Response, error) {
				return api.Response{}, api.APIError{StatusCode: http.StatusNotFound}
			},
			create: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				expectedName := "proj_my-bucket"
				require.Equal(t, expectedName, bucketName)
				return api.Response{
					StatusCode: 200,
					Data:       data,
				}, nil
			},
		}
		cfg := config.Config{
			Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
			Coordinate: testCoord,
			Type:       config.BucketType{},
			Parameters: config.Parameters{},
			Skip:       false,
		}
		props, errs := cfg.ResolveParameterValues(entities.New())
		require.Empty(t, errs)
		templ, err := cfg.Render(props)
		require.NoError(t, err)
		got, err := bucket.NewDeployAPI(client).Deploy(t.Context(), props, templ, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, got, entities.ResolvedEntity{
			Coordinate: testCoord,
			Properties: parameter.Properties{
				config.IdParameter: "proj_my-bucket",
			},
		})
	})

	t.Run("creates by OriginObjectId if set", func(t *testing.T) {
		client := testClient{
			get: func(_ context.Context, bucketName string) (api.Response, error) {
				return api.Response{}, api.APIError{StatusCode: http.StatusNotFound}
			},
			create: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				assert.Equal(t, "PreExistingBucket", bucketName)
				return api.Response{
					StatusCode: 200,
					Data:       data,
				}, nil
			},
		}
		cfg := config.Config{
			Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
			Coordinate:     testCoord,
			Type:           config.BucketType{},
			Parameters:     config.Parameters{},
			OriginObjectId: "PreExistingBucket",
			Skip:           false,
		}
		props, errs := cfg.ResolveParameterValues(entities.New())
		require.Empty(t, errs)
		templ, err := cfg.Render(props)
		require.NoError(t, err)
		got, err := bucket.NewDeployAPI(client).Deploy(t.Context(), props, templ, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, got, entities.ResolvedEntity{
			Coordinate: testCoord,
			Properties: parameter.Properties{
				config.IdParameter: "PreExistingBucket",
			},
		})
	})

	t.Run("returns error on create error", func(t *testing.T) {
		client := testClient{
			create: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				return api.Response{}, errors.New("fail")
			},
		}
		cfg := config.Config{
			Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
			Coordinate: testCoord,
			Type:       config.BucketType{},
			Parameters: config.Parameters{},
			Skip:       false,
		}
		props, errs := cfg.ResolveParameterValues(entities.New())
		require.Empty(t, errs)
		templ, err := cfg.Render(props)
		require.NoError(t, err)
		_, err = bucket.NewDeployAPI(client).Deploy(t.Context(), props, templ, &cfg)
		assert.Error(t, err)
	})

	t.Run("returns error if HTTP request for create failed", func(t *testing.T) {
		client := testClient{
			create: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				return api.Response{}, api.APIError{
					StatusCode: 400,
					Body:       []byte("Your request is bad and you should feel bad"),
				}
			},
		}
		cfg := config.Config{
			Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
			Coordinate: testCoord,
			Type:       config.BucketType{},
			Parameters: config.Parameters{},
			Skip:       false,
		}
		props, errs := cfg.ResolveParameterValues(entities.New())
		require.Empty(t, errs)
		templ, err := cfg.Render(props)
		require.NoError(t, err)
		_, err = bucket.NewDeployAPI(client).Deploy(t.Context(), props, templ, &cfg)
		assert.Error(t, err)
	})

	t.Run("calls update if bucket already exists", func(t *testing.T) {
		client := testClient{
			get: func(_ context.Context, bucketName string) (api.Response, error) {
				return api.Response{Data: getBucketActiveResponse(bucketName)}, nil
			},
			create: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				t.Error("create should not be called")
				return api.Response{}, errors.New("fail")
			},
			update: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				return api.Response{}, nil
			},
		}
		cfg := config.Config{
			Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
			Coordinate: testCoord,
			Type:       config.BucketType{},
			Parameters: config.Parameters{},
			Skip:       false,
		}
		props, errs := cfg.ResolveParameterValues(entities.New())
		require.Empty(t, errs)
		templ, err := cfg.Render(props)
		require.NoError(t, err)
		got, err := bucket.NewDeployAPI(client).Deploy(t.Context(), props, templ, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, got, entities.ResolvedEntity{
			Coordinate: testCoord,
			Properties: parameter.Properties{
				config.IdParameter: "proj_my-bucket",
			},
		})
	})

	t.Run("returns error if stable check failed", func(t *testing.T) {
		customErr := errors.New("custom error")
		client := testClient{
			get: func(_ context.Context, bucketName string) (api.Response, error) {
				return api.Response{}, customErr
			},
			create: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				t.Error("create should not be called")
				return api.Response{}, errors.New("fail")
			},
		}
		cfg := config.Config{
			Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
			Coordinate: testCoord,
			Type:       config.BucketType{},
			Parameters: config.Parameters{},
			Skip:       false,
		}
		props, errs := cfg.ResolveParameterValues(entities.New())
		require.Empty(t, errs)
		templ, err := cfg.Render(props)
		require.NoError(t, err)
		_, err = bucket.NewDeployAPI(client).Deploy(t.Context(), props, templ, &cfg)
		assert.ErrorIs(t, err, customErr)
	})

	t.Run("returns error if stable check after create failed", func(t *testing.T) {
		customErr := errors.New("custom error")
		getCount := 0
		getResponses := []struct {
			api.Response
			error
		}{
			{
				api.Response{}, api.APIError{StatusCode: http.StatusNotFound},
			},
			{
				api.Response{}, customErr,
			},
		}
		createCalled := false
		client := testClient{
			create: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				createCalled = true
				return api.Response{
					StatusCode: 200,
					Data:       data,
				}, nil
			},
			get: func(_ context.Context, bucketName string) (api.Response, error) {
				response := getResponses[getCount]
				getCount++
				return response.Response, response.error
			},
		}
		cfg := config.Config{
			Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
			Coordinate: testCoord,
			Type:       config.BucketType{},
			Parameters: config.Parameters{},
			Skip:       false,
		}
		props, errs := cfg.ResolveParameterValues(entities.New())
		require.Empty(t, errs)
		templ, err := cfg.Render(props)
		require.NoError(t, err)
		_, err = bucket.NewDeployAPI(client).Deploy(t.Context(), props, templ, &cfg)
		assert.ErrorIs(t, err, customErr)
		assert.True(t, createCalled)
	})

	t.Run("logs if the bucket is active after creation", func(t *testing.T) {
		getCount := 0
		getResponses := []struct {
			api.Response
			error
		}{
			{
				api.Response{}, api.APIError{StatusCode: http.StatusNotFound},
			},
			{
				api.Response{Data: activeBucketResponse}, nil,
			},
		}
		client := testClient{
			create: func(_ context.Context, bucketName string, data []byte) (api.Response, error) {
				return api.Response{
					StatusCode: 200,
					Data:       data,
				}, nil
			},
			get: func(_ context.Context, bucketName string) (api.Response, error) {
				response := getResponses[getCount]
				getCount++
				return response.Response, response.error
			},
		}
		cfg := config.Config{
			Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
			Coordinate: testCoord,
			Type:       config.BucketType{},
			Parameters: config.Parameters{},
			Skip:       false,
		}
		builder := strings.Builder{}
		log.PrepareLogging(t.Context(), afero.NewOsFs(), true, &builder, false, false)
		props, errs := cfg.ResolveParameterValues(entities.New())
		require.Empty(t, errs)
		templ, err := cfg.Render(props)
		require.NoError(t, err)
		_, err = bucket.NewDeployAPI(client).Deploy(t.Context(), props, templ, &cfg)
		require.NoError(t, err)
		assert.Contains(t, builder.String(), "ready to use")
	})
}
