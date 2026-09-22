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

	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/cloudconfiguration"
)

type deleteStubClient struct {
	deleteCalled bool
	delete       func(id string) (libAPI.Response, error)
	list         func() (libAPI.ListResponse, error)
}

func (s *deleteStubClient) List(context.Context) (libAPI.ListResponse, error) {
	return s.list()
}

func (s *deleteStubClient) Delete(_ context.Context, id string) (libAPI.Response, error) {
	s.deleteCalled = true
	return s.delete(id)
}

func TestDeleteByObjectId(t *testing.T) {
	t.Run("success if entry exists", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "cloud-configuration",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libAPI.Response{}, nil
			},
		}

		err := cloudconfiguration.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled)
	})

	t.Run("no error if entry doesn't exist", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "cloud-configuration",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound}
			},
		}

		err := cloudconfiguration.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("no error and no delete call if no originObjectId", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "cloud-configuration",
			Identifier: "some-config",
		}

		c := deleteStubClient{
			delete: func(_ string) (libAPI.Response, error) {
				t.Fatalf("should not be called")
				return libAPI.Response{}, nil
			},
		}

		err := cloudconfiguration.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.False(t, c.deleteCalled)
	})

	t.Run("error if delete fails", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "cloud-configuration",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(_ string) (libAPI.Response, error) {
				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := cloudconfiguration.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("deletion continues even if error occurs", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "cloud-configuration",
			OriginObjectId: "originObjectID",
		}
		failing := pointer.DeletePointer{
			Type:           "cloud-configuration",
			OriginObjectId: "bla",
		}

		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				if id == given.OriginObjectId {
					return libAPI.Response{}, nil
				}
				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := cloudconfiguration.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given, failing, given})
		assert.ErrorContains(t, err, "failed to delete 1 cloud configuration(s)")
	})
}

func TestDeleteAll(t *testing.T) {
	t.Run("simple case", func(t *testing.T) {
		c := deleteStubClient{
			list: func() (libAPI.ListResponse, error) {
				return libAPI.ListResponse{
					Objects: [][]byte{
						[]byte(`{"id": "uid_1"}`),
						[]byte(`{"id": "uid_2"}`),
						[]byte(`{"id": "uid_3"}`),
					},
				}, nil
			},
			delete: func(id string) (libAPI.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, id)
				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := cloudconfiguration.NewDeleter(&c).DeleteAll(t.Context())
		assert.NoError(t, err)
	})

	t.Run("deletion continues even if error occurs during delete", func(t *testing.T) {
		c := deleteStubClient{
			list: func() (libAPI.ListResponse, error) {
				return libAPI.ListResponse{
					Objects: [][]byte{
						[]byte(`{"id": "uid_1"}`),
						[]byte(`{"id": "uid_2"}`),
						[]byte(`{"id": "uid_3"}`),
					},
				}, nil
			},
			delete: func(id string) (libAPI.Response, error) {
				if id == "uid_2" {
					return libAPI.Response{}, errors.New("some unpredictable error")
				}
				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := cloudconfiguration.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
	})

	t.Run("error listing entries", func(t *testing.T) {
		c := deleteStubClient{
			list: func() (libAPI.ListResponse, error) {
				return libAPI.ListResponse{}, errors.New("some unexpected error")
			},
		}

		err := cloudconfiguration.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
	})
}
