//go:build unit

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

package document_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/document"
)

type deleteStubClient struct {
	deleteCalled bool
	delete       func(id string) (libAPI.Response, error)
	list         func(externalID string) (documents.ListResponse, error)
}

func (s *deleteStubClient) List(_ context.Context, filter string) (documents.ListResponse, error) {
	return s.list(filter)
}

func (s *deleteStubClient) Delete(_ context.Context, id string) (libAPI.Response, error) {
	s.deleteCalled = true
	return s.delete(id)
}

func TestDeleteByCoordinate(t *testing.T) {
	t.Run("success if one document matches generated external ID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "document",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID := idutils.GenerateExternalID(given.AsCoordinate())
		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, externalID, id)
				return libAPI.Response{}, nil
			},
		}

		err := document.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("no error if no document matches generated external ID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "document",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID := idutils.GenerateExternalID(given.AsCoordinate())
		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, externalID, id)
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound}
			},
		}

		err := document.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command was invoked")
	})
}

func TestDeleteByObjectId(t *testing.T) {
	t.Run("success if document exists", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "document",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libAPI.Response{}, nil
			},
		}

		err := document.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("no error if document doesn't exist", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "document",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound}
			},
		}

		err := document.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if delete fails", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "document",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := deleteStubClient{
			delete: func(_ string) (libAPI.Response, error) {
				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := document.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if server error during delete", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "document",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := deleteStubClient{
			delete: func(_ string) (libAPI.Response, error) {
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusInternalServerError}
			},
		}

		err := document.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("deletion continues even if error occurs", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "document",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := deleteStubClient{
			delete: func(uid string) (libAPI.Response, error) {
				if uid == given.OriginObjectId {
					return libAPI.Response{}, nil
				}
				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := document.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given, {OriginObjectId: "bla"}, given}) // the pointer in the middle is to cause error behavior
		assert.ErrorContains(t, err, "failed to delete 1 document(s)")
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})
}

func TestDeleteAll(t *testing.T) {

	t.Run("filter string is as expected", func(t *testing.T) {
		expectedFilter := "type='dashboard' or type='notebook' or type='launchpad'"
		c := deleteStubClient{
			list: func(filter string) (documents.ListResponse, error) {
				assert.Equal(t, expectedFilter, filter)
				return documents.ListResponse{}, nil
			},
		}

		err := document.NewDeleter(&c).DeleteAll(t.Context())
		assert.NoError(t, err)
	})

	t.Run("no delete called if there is nothing to delete", func(t *testing.T) {
		c := deleteStubClient{
			list: func(filter string) (documents.ListResponse, error) {
				return documents.ListResponse{}, nil
			},
		}

		err := document.NewDeleter(&c).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.False(t, c.deleteCalled, "delete command was invoked")
	})

	t.Run("simple case", func(t *testing.T) {
		expectedFilter := "type='dashboard' or type='notebook' or type='launchpad'"
		c := deleteStubClient{
			list: func(filter string) (documents.ListResponse, error) {
				assert.Equal(t, expectedFilter, filter)
				return documents.ListResponse{
					libAPI.Response{},
					[]documents.Response{
						{
							libAPI.Response{},
							documents.Metadata{ID: "uid_1"},
						},
						{
							libAPI.Response{},
							documents.Metadata{ID: "uid_2"},
						},
						{
							libAPI.Response{},
							documents.Metadata{ID: "uid_3"},
						},
					},
				}, nil
			},
			delete: func(uid string) (libAPI.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, uid)
				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := document.NewDeleter(&c).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("deletion continues even if error occurs during delete", func(t *testing.T) {
		c := deleteStubClient{
			list: func(filter string) (documents.ListResponse, error) {
				return documents.ListResponse{
					libAPI.Response{},
					[]documents.Response{
						{
							libAPI.Response{},
							documents.Metadata{ID: "uid_1"},
						},
						{
							libAPI.Response{},
							documents.Metadata{ID: "uid_2"},
						},
						{
							libAPI.Response{},
							documents.Metadata{ID: "uid_3"},
						},
					},
				}, nil
			},
			delete: func(uid string) (libAPI.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, uid)
				if uid == "uid_2" {
					return libAPI.Response{}, errors.New("some unpredictable error")
				}
				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := document.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if list fails", func(t *testing.T) {
		c := deleteStubClient{
			list: func(filter string) (documents.ListResponse, error) {
				return documents.ListResponse{
					libAPI.Response{},
					[]documents.Response{},
				}, errors.New("some unpredictable error")
			},
		}

		err := document.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.False(t, c.deleteCalled, "delete command was invoked")
	})
}
