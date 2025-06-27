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
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/slo"
)

type stubClient struct {
	list func() (api.PagedListResponse, error)
}

func (s stubClient) List(_ context.Context) (api.PagedListResponse, error) {
	return s.list()
}

func TestDownloader_Download(t *testing.T) {
	t.Run("download slo works", func(t *testing.T) {
		c := stubClient{list: func() (api.PagedListResponse, error) {
			return api.PagedListResponse{
				{Response: api.Response{
					StatusCode: http.StatusOK,
					Data:       nil,
				},
					Objects: [][]byte{
						[]byte(`{
							"id": "id",
							"externalId": "some_external_ID",
							"version": 1,
							"name": "slo_name"
						}`),
					},
				},
			}, nil
		}}

		sloApi := slo.NewDownloadAPI(c)
		result, err := sloApi.Download(context.TODO(), "project")

		assert.NoError(t, err)
		assert.Len(t, result, 1)

		require.Len(t, result[string(config.ServiceLevelObjectiveID)], 1, "all listed SLOs should be downloaded")

		actual := result[string(config.ServiceLevelObjectiveID)][0]

		assert.Equal(t, config.ServiceLevelObjective{}, actual.Type)
		assert.Equal(t, coordinate.Coordinate{Project: "project", Type: "slo-v2", ConfigId: "id"}, actual.Coordinate)
		assert.Equal(t, "id", actual.OriginObjectId)
		actualTemplate, err := actual.Template.Content()
		assert.NoError(t, err)
		assert.JSONEq(t, `{"name":"slo_name"}`, actualTemplate, "id, externalId and version must be deleted")

		assert.False(t, actual.Skip)
		assert.Empty(t, actual.Group)
		assert.Empty(t, actual.Environment)
		assert.Empty(t, actual.Parameters)
	})

	t.Run("slo without id is ignored", func(t *testing.T) {
		c := stubClient{list: func() (api.PagedListResponse, error) {
			return api.PagedListResponse{
				{Response: api.Response{
					StatusCode: http.StatusOK,
					Data:       nil,
				},
					Objects: [][]byte{[]byte(`{
						"externalId": "some_external_ID",
						"version": 1,
						"name": "slo_name"
					}`)},
				},
			}, nil
		}}

		sloApi := slo.NewDownloadAPI(c)
		result, err := sloApi.Download(context.TODO(), "project")

		assert.NoError(t, err)
		assert.Len(t, result, 1)

		assert.Empty(t, result[string(config.ServiceLevelObjectiveID)])
	})

	t.Run("Downloading multiple SLOs works", func(t *testing.T) {
		c := stubClient{list: func() (api.PagedListResponse, error) {
			return api.PagedListResponse{
				{
					Response: api.Response{StatusCode: http.StatusOK},
					Objects: [][]byte{
						[]byte(`{"id": "id1","externalId": "some_external_ID","version": 1,"name": "slo_name_1"}`),
						[]byte(`{"id": "id2","externalId": "some_external_ID","version": 1,"name": "slo_name_2"}`),
					},
				},
				{
					Response: api.Response{StatusCode: http.StatusOK},
					Objects:  [][]byte{[]byte(`{"id": "id3","externalId": "some_external_ID","version": 1,"name": "slo_name_3"}`)},
				},
			}, nil
		}}

		sloApi := slo.NewDownloadAPI(c)
		actual, err := sloApi.Download(context.TODO(), "project")

		assert.NoError(t, err)
		assert.Len(t, actual, 1)

		assert.Len(t, actual[string(config.ServiceLevelObjectiveID)], 3, "must contain all downloaded configs")
	})

	t.Run("no error downloading SLOs with faulty client", func(t *testing.T) {
		c := stubClient{list: func() (api.PagedListResponse, error) {
			return api.PagedListResponse{}, errors.New("some unexpected error")
		}}

		sloApi := slo.NewDownloadAPI(c)
		result, err := sloApi.Download(context.TODO(), "project")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("complete real payload", func(t *testing.T) {
		given := `{
			"id": "vu9U3hXa3",
			"externalId": "some_external_ID",
			"version": "vu9U3hXY3-71TeFdjerQ",
			"name": "New SLO",
			"description": "This is a description",
			"customSli": {
				"indicator": "timeseries sli=avg(dt.host.cpu.idle)"
			},
			"criteria": [
				{
				  "timeframeFrom": "now-7d",
				  "timeframeTo": "now",
				  "target": 99.8,
				  "warning": 99.9
				}
			],
			"tags": [
				"Stage:PROD"
			]
		}`
		expected := `{
			"name": "New SLO",
			"description": "This is a description",
			"customSli": {
				"indicator": "timeseries sli=avg(dt.host.cpu.idle)"
			},
			"criteria": [
				{
				  "timeframeFrom": "now-7d",
				  "timeframeTo": "now",
				  "target": 99.8,
				  "warning": 99.9
				}
			],
			"tags": [
				"Stage:PROD"
			]
		}`

		c := stubClient{list: func() (api.PagedListResponse, error) {
			return api.PagedListResponse{{
				Response: api.Response{
					StatusCode: http.StatusOK,
				},
				Objects: [][]byte{[]byte(given)}},
			}, nil
		}}

		sloApi := slo.NewDownloadAPI(c)
		result, err := sloApi.Download(context.TODO(), "project")
		assert.NoError(t, err)

		actual := result[string(config.ServiceLevelObjectiveID)][0].Template
		assert.Equal(t, "vu9U3hXa3", actual.ID())

		actualContent, err := actual.Content()
		assert.NoError(t, err)
		assert.JSONEq(t, expected, actualContent)
	})
}
