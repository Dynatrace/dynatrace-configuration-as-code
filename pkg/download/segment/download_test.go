/*
 * @license
 * Copyright 2024 Dynatrace LLC
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
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreLib "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/segment"
)

type stubClient struct {
	getAll func() ([]coreLib.Response, error)
}

func (s stubClient) GetAll(_ context.Context) ([]coreLib.Response, error) {
	return s.getAll()
}

func TestDownloader_Download(t *testing.T) {
	t.Run("download segments works", func(t *testing.T) {
		c := stubClient{getAll: func() ([]coreLib.Response, error) {
			return []coreLib.Response{
				{
					StatusCode: http.StatusOK,
					Data: []byte(`{
    "uid": "uid",
    "externalId": "some_external_ID",
    "version": 1,
    "name": "segment_name"
}`),
				},
			}, nil
		}}

		result, err := segment.Download(c, "project")

		assert.NoError(t, err)
		assert.Len(t, result, 1)

		require.Len(t, result[string(config.SegmentID)], 1, "all listed segments should be downloaded")

		actual := result[string(config.SegmentID)][0]

		assert.Equal(t, config.Segment{}, actual.Type)
		assert.Equal(t, coordinate.Coordinate{Project: "project", Type: "segment", ConfigId: "uid"}, actual.Coordinate)
		assert.Equal(t, "uid", actual.OriginObjectId)
		actualTemplate, err := actual.Template.Content()
		assert.NoError(t, err)
		assert.JSONEq(t, `{"name":"segment_name"}`, actualTemplate)

		assert.False(t, actual.Skip)
		assert.Empty(t, actual.Group)
		assert.Empty(t, actual.Environment)
		assert.Empty(t, actual.Parameters)
		assert.Empty(t, actual.SkipForConversion)
	})

	t.Run("segment without uio is ignored", func(t *testing.T) {
		c := stubClient{getAll: func() ([]coreLib.Response, error) {
			return []coreLib.Response{
				{
					StatusCode: http.StatusOK,
					Data: []byte(`{
    "externalId": "some_external_ID",
    "version": 1,
    "name": "segment_name"
}`),
				},
			}, nil
		}}

		result, err := segment.Download(c, "project")

		assert.NoError(t, err)
		assert.Len(t, result, 1)

		assert.Empty(t, result[string(config.SegmentID)])
	})

	t.Run("no error downloading segments with faulty client", func(t *testing.T) {
		c := stubClient{getAll: func() ([]coreLib.Response, error) {
			return []coreLib.Response{}, errors.New("some unexpected error")
		}}

		result, err := segment.Download(c, "project")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}
