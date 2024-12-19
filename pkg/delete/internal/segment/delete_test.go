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
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	libSegment "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/segment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type stubClient struct {
	called bool
	delete func(id string) (libSegment.Response, error)
	list   func() (libSegment.Response, error)
}

func (s *stubClient) List(_ context.Context) (libSegment.Response, error) {
	return s.list()
}

func (s *stubClient) Delete(_ context.Context, id string) (libSegment.Response, error) {
	s.called = true
	return s.delete(id)
}

func TestDelete(t *testing.T) {

	t.Run("delete via coordinate", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "segment",
			Identifier: "monaco_identifier",
			Project:    "project",
		}
		externalID, _ := idutils.GenerateExternalIDForDocument(given.AsCoordinate())

		c := stubClient{
			list: func() (libSegment.Response, error) {
				return libSegment.Response{Data: []byte(fmt.Sprintf(`[{"uid": "uid_1", "externalId":"%s"},{"uid": "uid_2", "externalId":"wrong"}]`, externalID))}, nil
			},
			delete: func(id string) (libSegment.Response, error) {
				assert.Equal(t, "uid_1", id)
				return libSegment.Response{}, nil
			},
		}

		err := segment.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.called, "delete command wasn't invoked")
	})

	t.Run("config declared via coordinate doesn't exists - no error (wanted state achieved)", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "segment",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := stubClient{
			list: func() (libSegment.Response, error) {
				return libSegment.Response{Data: []byte(`[{"uid": "uid_2", "externalId":"wrong"}]`)}, nil
			},
		}

		err := segment.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("config declared via coordinate have multiple match - an error", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "segment",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID, _ := idutils.GenerateExternalIDForDocument(given.AsCoordinate())
		c := stubClient{
			list: func() (libSegment.Response, error) {
				return libSegment.Response{Data: []byte(fmt.Sprintf(`[{"uid": "uid_1", "externalId":"%s"},{"uid": "uid_2", "externalId":"%s"}]`, externalID, externalID))}, nil
			},
		}

		err := segment.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.False(t, c.called, "it's not known what needs to be deleted")
	})

	t.Run("config declared via coordinate failed to get externalId - an error", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "segment",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := stubClient{
			list: func() (libSegment.Response, error) {
				return libSegment.Response{}, errors.New("some unpredictable error")
			},
		}

		err := segment.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("delete via originID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "segment",
			OriginObjectId: "originObjectID",
		}

		c := stubClient{
			delete: func(id string) (libSegment.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libSegment.Response{}, nil
			},
		}

		err := segment.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.called)
	})

	t.Run("config declared via originID doesn't exists - no error (wanted state achieved)", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "segment",
			OriginObjectId: "originObjectID",
		}

		c := stubClient{
			delete: func(id string) (libSegment.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound}
			},
		}

		err := segment.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("error during delete - an error", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "segment",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := stubClient{
			delete: func(_ string) (libSegment.Response, error) {
				return libSegment.Response{}, errors.New("some unpredictable error")
			},
		}

		err := segment.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("server error during delete - an error", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "segment",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := stubClient{
			delete: func(_ string) (libSegment.Response, error) {
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusInternalServerError}
			},
		}

		err := segment.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	// t.Run("error during delete - continue to delete, an error", func(t *testing.T) {
	// 	given := pointer.DeletePointer{
	// 		Type:           "segment",
	// 		OriginObjectId: "originObjectID",
	// 		Project:        "project",
	// 	}
	//
	// 	c := client.NewMockSegmentClient(gomock.NewController(t))
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("originObjectID")).Times(1).Return(libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound})
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("originObjectID")).Times(1).Return(libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusInternalServerError}) // the error can be any kind except 404
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("originObjectID")).Times(1).Return(libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound})
	//
	// 	err := segment.Delete(context.TODO(), c, []pointer.DeletePointer{given, given, given})
	// 	assert.Error(t, err)
	// })
}

func TestDeleteAll(t *testing.T) {
	// t.Run("simple case", func(t *testing.T) {
	// 	c := client.NewMockSegmentClient(gomock.NewController(t))
	// 	c.EXPECT().List(gomock.Any()).Times(1).
	// 		Return(libSegment.Response{Data: []byte(`[{"uid": "uid_1"},{"uid": "uid_2"},{"uid": "uid_3"}]`)}, nil)
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("uid_1")).Times(1)
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("uid_2")).Times(1)
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("uid_3")).Times(1)
	//
	// 	err := segment.DeleteAll(context.TODO(), c)
	// 	assert.NoError(t, err)
	// })

	// t.Run("error during delete - continue to delete, an error", func(t *testing.T) {
	// 	c := client.NewMockSegmentClient(gomock.NewController(t))
	// 	c.EXPECT().List(gomock.Any()).Times(1).
	// 		Return(libSegment.Response{Data: []byte(`[{"uid": "uid_1"},{"uid": "uid_2"},{"uid": "uid_3"}]`)}, nil)
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("uid_1")).Times(1)
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("uid_2")).Times(1).Return(libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusInternalServerError}) // the error can be any kind except 404
	// 	c.EXPECT().Delete(gomock.Any(), gomock.Eq("uid_3")).Times(1)
	//
	// 	err := segment.DeleteAll(context.TODO(), c)
	// 	assert.Error(t, err)
	// })
}
