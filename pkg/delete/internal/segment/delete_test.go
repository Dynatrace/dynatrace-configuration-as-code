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

func TestDeleteByCoordinate(t *testing.T) {

	t.Run("success if one segment matches generated external ID", func(t *testing.T) {
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

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.called, "delete command wasn't invoked")
	})

	t.Run("no error if no segment matches generated external ID", func(t *testing.T) {
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

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("error if multiple segments match generated external ID", func(t *testing.T) {
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

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.False(t, c.called, "it's not known what needs to be deleted")
	})

	t.Run("error if list fails", func(t *testing.T) {
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

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
	})
}

func TestDeleteByObjectId(t *testing.T) {

	t.Run("sucess if segment exists", func(t *testing.T) {
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

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.called)
	})

	t.Run("no error if segment doesn't exist", func(t *testing.T) {
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

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("error if delete fails", func(t *testing.T) {
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

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("error if server error during delete", func(t *testing.T) {
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

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("deletion continues even if error occurs", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "segment",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := stubClient{
			delete: func(uid string) (libSegment.Response, error) {
				if uid == given.OriginObjectId {
					return libSegment.Response{}, nil
				}
				return libSegment.Response{}, errors.New("some unpredictable error")
			},
		}

		err := segment.Delete(t.Context(), &c, []pointer.DeletePointer{given, {OriginObjectId: "bla"}, given}) // the pointer in the middle is to cause error behavior
		assert.ErrorContains(t, err, "failed to delete 1 segment objects(s)")
	})
}

func TestDeleteAll(t *testing.T) {
	t.Run("simple case", func(t *testing.T) {
		c := stubClient{
			list: func() (libSegment.Response, error) {
				return libSegment.Response{Data: []byte(`[{"uid": "uid_1"},{"uid": "uid_2"},{"uid": "uid_3"}]`)}, nil
			},
			delete: func(uid string) (libSegment.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, uid)
				return libSegment.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := segment.DeleteAll(t.Context(), &c)
		assert.NoError(t, err)
	})

	t.Run("deletion continues even if error occurs during delete", func(t *testing.T) {
		c := stubClient{
			list: func() (libSegment.Response, error) {
				return libSegment.Response{Data: []byte(`[{"uid": "uid_1"},{"uid": "uid_2"},{"uid": "uid_3"}]`)}, nil
			},
			delete: func(uid string) (libSegment.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, uid)
				if uid == "uid_2" {
					return libSegment.Response{}, errors.New("some unpredictable error")
				}
				return libSegment.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := segment.DeleteAll(t.Context(), &c)
		assert.Error(t, err)
	})
}
