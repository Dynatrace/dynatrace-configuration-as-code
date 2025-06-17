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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
)

type assertAndRespond func(t *testing.T, bucketName string, data []byte) (api.Response, error)

type testClient struct {
	t                    *testing.T
	assertAndRespondFunc assertAndRespond
}

func (c testClient) Upsert(_ context.Context, bucketName string, data []byte) (api.Response, error) {
	return c.assertAndRespondFunc(c.t, bucketName, data)
}

func TestDeploy(t *testing.T) {

	testCoord := coordinate.Coordinate{
		Project:  "proj",
		Type:     "bucket",
		ConfigId: "my-bucket",
	}

	tests := []struct {
		name             string
		givenConfig      config.Config
		assertAndRespond assertAndRespond
		want             entities.ResolvedEntity
		wantErr          bool
	}{
		{
			"upserts by generated coordinate ID",
			config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoord,
				Type:       config.BucketType{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			func(t *testing.T, bucketName string, data []byte) (api.Response, error) {
				expectedName := "proj_my-bucket"
				assert.Equal(t, expectedName, bucketName)
				return api.Response{
					StatusCode: 200,
					Data:       data,
				}, nil
			},
			entities.ResolvedEntity{
				Coordinate: testCoord,
				Properties: parameter.Properties{
					config.IdParameter: "proj_my-bucket",
				},
			},
			false,
		},
		{
			"upserts by OriginObjectId if set",
			config.Config{
				Template:       template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate:     testCoord,
				Type:           config.BucketType{},
				Parameters:     config.Parameters{},
				OriginObjectId: "PreExistingBucket",
				Skip:           false,
			},
			func(t *testing.T, bucketName string, data []byte) (api.Response, error) {
				assert.Equal(t, "PreExistingBucket", bucketName)
				return api.Response{
					StatusCode: 200,
					Data:       data,
				}, nil
			},
			entities.ResolvedEntity{
				Coordinate: testCoord,
				Properties: parameter.Properties{
					config.IdParameter: "PreExistingBucket",
				},
			},
			false,
		},
		{
			"returns error on upsert error",
			config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoord,
				Type:       config.BucketType{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			func(t *testing.T, bucketName string, data []byte) (api.Response, error) {
				return api.Response{}, errors.New("fail")
			},
			entities.ResolvedEntity{},
			true,
		},
		{
			"returns error if HTTP request for upsert failed",
			config.Config{
				Template:   template.NewInMemoryTemplate("path/file.json", "{}"),
				Coordinate: testCoord,
				Type:       config.BucketType{},
				Parameters: config.Parameters{},
				Skip:       false,
			},
			func(t *testing.T, bucketName string, data []byte) (api.Response, error) {
				return api.Response{}, &api.APIError{
					StatusCode: 400,
					Body:       []byte("Your request is bad and you should feel bad"),
				}
			},
			entities.ResolvedEntity{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			c := testClient{
				t,
				tt.assertAndRespond,
			}

			props, errs := tt.givenConfig.ResolveParameterValues(entities.New())
			assert.Empty(t, errs)
			templ, err := tt.givenConfig.Render(props)
			assert.NoError(t, err)

			got, err := bucket.NewDeployAPI(c).Deploy(t.Context(), props, templ, &tt.givenConfig)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
