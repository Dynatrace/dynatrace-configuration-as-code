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
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/cloudconfiguration"
)

type downloadStubClient struct {
	list func() (api.ListResponse, error)
}

func (s downloadStubClient) List(_ context.Context) (api.ListResponse, error) {
	return s.list()
}

func TestDownloader_Download(t *testing.T) {
	t.Run("download works", func(t *testing.T) {
		c := downloadStubClient{list: func() (api.ListResponse, error) {
			return api.ListResponse{
				Response: api.Response{StatusCode: http.StatusOK},
				Objects: [][]byte{
					[]byte(`{
						"id": "id-1",
						"tenantUuid": "tenant-1",
						"connectionId": "conn-1",
						"cloudProvider": "aws",
						"displayName": "my-config",
						"region": "eu-west-1",
						"bucket": "my-bucket",
						"path": "some/path",
						"createdAt": "2026-01-01T00:00:00Z"
					}`),
				},
			}, nil
		}}

		downloadAPI := cloudconfiguration.NewDownloadAPI(c)
		result, err := downloadAPI.Download(t.Context(), "project")

		assert.NoError(t, err)
		require.Len(t, result, 1)
		require.Len(t, result[string(config.CloudConfigurationID)], 1)

		actual := result[string(config.CloudConfigurationID)][0]

		assert.Equal(t, config.CloudConfiguration{}, actual.Type)
		assert.Equal(t, coordinate.Coordinate{Project: "project", Type: "cloud-configuration", ConfigId: "id-1"}, actual.Coordinate)
		assert.Equal(t, "id-1", actual.OriginObjectId)

		actualContent, err := actual.Template.Content()
		assert.NoError(t, err)
		assert.JSONEq(t, `{
			"connectionId": "conn-1",
			"cloudProvider": "aws",
			"displayName": "my-config",
			"region": "eu-west-1",
			"bucket": "my-bucket",
			"path": "some/path"
		}`, actualContent, "id, tenantUuid, createdAt and updatedAt must be removed")
	})

	t.Run("entry without id is ignored", func(t *testing.T) {
		c := downloadStubClient{list: func() (api.ListResponse, error) {
			return api.ListResponse{
				Response: api.Response{StatusCode: http.StatusOK},
				Objects:  [][]byte{[]byte(`{"displayName": "no-id-here"}`)},
			}, nil
		}}

		downloadAPI := cloudconfiguration.NewDownloadAPI(c)
		result, err := downloadAPI.Download(t.Context(), "project")

		assert.NoError(t, err)
		assert.Empty(t, result[string(config.CloudConfigurationID)])
	})

	t.Run("downloading multiple entries works", func(t *testing.T) {
		c := downloadStubClient{list: func() (api.ListResponse, error) {
			return api.ListResponse{
				Response: api.Response{StatusCode: http.StatusOK},
				Objects: [][]byte{
					[]byte(`{"id": "id-1", "displayName": "config-1"}`),
					[]byte(`{"id": "id-2", "displayName": "config-2"}`),
				},
			}, nil
		}}

		downloadAPI := cloudconfiguration.NewDownloadAPI(c)
		result, err := downloadAPI.Download(t.Context(), "project")

		assert.NoError(t, err)
		assert.Len(t, result[string(config.CloudConfigurationID)], 2)
	})

	t.Run("no error downloading with faulty client", func(t *testing.T) {
		c := downloadStubClient{list: func() (api.ListResponse, error) {
			return api.ListResponse{}, errors.New("some unexpected error")
		}}

		downloadAPI := cloudconfiguration.NewDownloadAPI(c)
		result, err := downloadAPI.Download(t.Context(), "project")

		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}
