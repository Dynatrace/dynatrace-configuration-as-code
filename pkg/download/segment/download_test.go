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

		result, err := segment.Download(t.Context(), c, "project")

		assert.NoError(t, err)
		assert.Len(t, result, 1)

		require.Len(t, result[string(config.SegmentID)], 1, "all listed segments should be downloaded")

		actual := result[string(config.SegmentID)][0]

		assert.Equal(t, config.Segment{}, actual.Type)
		assert.Equal(t, coordinate.Coordinate{Project: "project", Type: "segment", ConfigId: "uid"}, actual.Coordinate)
		assert.Equal(t, "uid", actual.OriginObjectId)
		actualTemplate, err := actual.Template.Content()
		assert.NoError(t, err)
		assert.JSONEq(t, `{"name":"segment_name"}`, actualTemplate, "uid, externalId and version must be deleted")

		assert.False(t, actual.Skip)
		assert.Empty(t, actual.Group)
		assert.Empty(t, actual.Environment)
		assert.Empty(t, actual.Parameters)
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

		result, err := segment.Download(t.Context(), c, "project")

		assert.NoError(t, err)
		assert.Len(t, result, 1)

		assert.Empty(t, result[string(config.SegmentID)])
	})

	t.Run("Downloading multiple segments works", func(t *testing.T) {
		c := stubClient{getAll: func() ([]coreLib.Response, error) {
			return []coreLib.Response{
				{Data: []byte(`{"uid": "uid1","externalId": "some_external_ID","version": 1,"name": "segment_name"}`), StatusCode: http.StatusOK},
				{Data: []byte(`{"uid": "uid2","externalId": "some_external_ID","version": 1,"name": "segment_name"}`), StatusCode: http.StatusOK},
				{Data: []byte(`{"uid": "uid3","externalId": "some_external_ID","version": 1,"name": "segment_name"}`), StatusCode: http.StatusOK},
			}, nil
		}}

		actual, err := segment.Download(t.Context(), c, "project")

		assert.NoError(t, err)
		assert.Len(t, actual, 1)

		assert.Len(t, actual[string(config.SegmentID)], 3, "must contain all downloaded configs")
	})

	t.Run("no error downloading segments with faulty client", func(t *testing.T) {
		c := stubClient{getAll: func() ([]coreLib.Response, error) {
			return []coreLib.Response{}, errors.New("some unexpected error")
		}}

		result, err := segment.Download(t.Context(), c, "project")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("complete real payload", func(t *testing.T) {
		given := `{
  "uid": "PfdP4Qp1IJG",
  "externalId": "some_external_ID",
  "name": "Host based logs",
  "variables": {
    "type": "query",
    "value": "fetch dt.entity.host | fields id, entity.name"
  },
  "isPublic": true,
  "owner": "cd3fc936-5b1a-4d6c-b1b6-f1025dbde7d5",
  "allowedOperations": [
    "READ"
  ],
  "includes": [
    {
      "filter": "{\"type\":\"Group\",\"range\":{\"from\":0,\"to\":58},\"logicalOperator\":\"OR\",\"explicit\":false,\"children\":[{\"type\":\"Statement\",\"range\":{\"from\":0,\"to\":22},\"key\":{\"type\":\"Key\",\"textValue\":\"dt.entity.host\",\"value\":\"dt.entity.host\",\"range\":{\"from\":0,\"to\":14}},\"operator\":{\"type\":\"ComparisonOperator\",\"textValue\":\"=\",\"value\":\"=\",\"range\":{\"from\":15,\"to\":16}},\"value\":{\"type\":\"String\",\"textValue\":\"\\\"$id\\\"\",\"value\":\"$id\",\"range\":{\"from\":17,\"to\":22},\"isEscaped\":true}},{\"type\":\"LogicalOperator\",\"textValue\":\"OR\",\"value\":\"OR\",\"range\":{\"from\":23,\"to\":25}},{\"type\":\"Statement\",\"range\":{\"from\":26,\"to\":57},\"key\":{\"type\":\"Key\",\"textValue\":\"dt.entity.host\",\"value\":\"dt.entity.host\",\"range\":{\"from\":26,\"to\":40}},\"operator\":{\"type\":\"ComparisonOperator\",\"textValue\":\"=\",\"value\":\"=\",\"range\":{\"from\":41,\"to\":42}},\"value\":{\"type\":\"String\",\"textValue\":\"\\\"$entity.name\\\"\",\"value\":\"$entity.name\",\"range\":{\"from\":43,\"to\":57},\"isEscaped\":true}}]}",
      "dataObject": "logs",
      "applyTo": []
    },
    {
      "filter": "{\"type\":\"Group\",\"range\":{\"from\":0,\"to\":11},\"logicalOperator\":\"AND\",\"explicit\":false,\"children\":[{\"type\":\"Statement\",\"range\":{\"from\":0,\"to\":10},\"key\":{\"type\":\"Key\",\"textValue\":\"id\",\"value\":\"id\",\"range\":{\"from\":0,\"to\":2}},\"operator\":{\"type\":\"ComparisonOperator\",\"textValue\":\"=\",\"value\":\"=\",\"range\":{\"from\":3,\"to\":4}},\"value\":{\"type\":\"String\",\"textValue\":\"\\\"$id\\\"\",\"value\":\"$id\",\"range\":{\"from\":5,\"to\":10},\"isEscaped\":true}}]}",
      "dataObject": "dt.entity.host",
      "applyTo": []
    }
  ],
  "version": 16
}`
		expected := `{
  "name": "Host based logs",
  "variables": {
    "type": "query",
    "value": "fetch dt.entity.host | fields id, entity.name"
  },
  "isPublic": true,
  "owner": "cd3fc936-5b1a-4d6c-b1b6-f1025dbde7d5",
  "allowedOperations": [
    "READ"
  ],
  "includes": [
    {
      "filter": "{\"type\":\"Group\",\"range\":{\"from\":0,\"to\":58},\"logicalOperator\":\"OR\",\"explicit\":false,\"children\":[{\"type\":\"Statement\",\"range\":{\"from\":0,\"to\":22},\"key\":{\"type\":\"Key\",\"textValue\":\"dt.entity.host\",\"value\":\"dt.entity.host\",\"range\":{\"from\":0,\"to\":14}},\"operator\":{\"type\":\"ComparisonOperator\",\"textValue\":\"=\",\"value\":\"=\",\"range\":{\"from\":15,\"to\":16}},\"value\":{\"type\":\"String\",\"textValue\":\"\\\"$id\\\"\",\"value\":\"$id\",\"range\":{\"from\":17,\"to\":22},\"isEscaped\":true}},{\"type\":\"LogicalOperator\",\"textValue\":\"OR\",\"value\":\"OR\",\"range\":{\"from\":23,\"to\":25}},{\"type\":\"Statement\",\"range\":{\"from\":26,\"to\":57},\"key\":{\"type\":\"Key\",\"textValue\":\"dt.entity.host\",\"value\":\"dt.entity.host\",\"range\":{\"from\":26,\"to\":40}},\"operator\":{\"type\":\"ComparisonOperator\",\"textValue\":\"=\",\"value\":\"=\",\"range\":{\"from\":41,\"to\":42}},\"value\":{\"type\":\"String\",\"textValue\":\"\\\"$entity.name\\\"\",\"value\":\"$entity.name\",\"range\":{\"from\":43,\"to\":57},\"isEscaped\":true}}]}",
      "dataObject": "logs",
      "applyTo": []
    },
    {
      "filter": "{\"type\":\"Group\",\"range\":{\"from\":0,\"to\":11},\"logicalOperator\":\"AND\",\"explicit\":false,\"children\":[{\"type\":\"Statement\",\"range\":{\"from\":0,\"to\":10},\"key\":{\"type\":\"Key\",\"textValue\":\"id\",\"value\":\"id\",\"range\":{\"from\":0,\"to\":2}},\"operator\":{\"type\":\"ComparisonOperator\",\"textValue\":\"=\",\"value\":\"=\",\"range\":{\"from\":3,\"to\":4}},\"value\":{\"type\":\"String\",\"textValue\":\"\\\"$id\\\"\",\"value\":\"$id\",\"range\":{\"from\":5,\"to\":10},\"isEscaped\":true}}]}",
      "dataObject": "dt.entity.host",
      "applyTo": []
    }
  ]
}`

		c := stubClient{getAll: func() ([]coreLib.Response, error) {
			return []coreLib.Response{{StatusCode: http.StatusOK, Data: []byte(given)}}, nil
		}}

		result, err := segment.Download(t.Context(), c, "project")
		assert.NoError(t, err)

		actual := result[string(config.SegmentID)][0].Template
		assert.Equal(t, "PfdP4Qp1IJG", actual.ID())

		actualContent, err := actual.Content()
		assert.NoError(t, err)
		assert.JSONEq(t, expected, actualContent)
	})
}
