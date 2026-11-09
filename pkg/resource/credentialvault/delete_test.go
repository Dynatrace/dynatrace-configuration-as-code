//go:build unit

/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package credentialvault_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/credentialvault"
)

type deleteStubClient struct {
	deleteCalled bool
	delete       func(id string) (libAPI.Response, error)
	list         func() (libAPI.PagedListResponse, error)
}

func (s *deleteStubClient) List(context.Context) (libAPI.PagedListResponse, error) {
	return s.list()
}

func (s *deleteStubClient) Delete(_ context.Context, id string) (libAPI.Response, error) {
	s.deleteCalled = true
	return s.delete(id)
}

func TestDeleteWithCoordinate(t *testing.T) {
	t.Run("success if one credential vault entry matches the identifier", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       string(config.CredentialVaultID),
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteStubClient{
			list: func() (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{
					libAPI.ListResponse{
						Objects: [][]byte{
							fmt.Appendf(nil, `{"id": "uid_1", "name":"%s"}`, given.Identifier),
							[]byte(`{"id": "uid_2", "name":"wrong"}`),
						},
					}}, nil
			},
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, "uid_1", id)
				return libAPI.Response{}, nil
			},
		}

		err := credentialvault.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})

		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("no error if no credential vault entry matches identifier", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       string(config.CredentialVaultID),
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteStubClient{
			list: func() (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{
					libAPI.ListResponse{Objects: [][]byte{[]byte(`{"uid": "uid_2", "name":"wrong"}`)}},
				}, nil
			},
		}

		err := credentialvault.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("error if multiple credential vault entries match the identifier", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       string(config.CredentialVaultID),
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteStubClient{
			list: func() (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{
					libAPI.ListResponse{Objects: [][]byte{
						fmt.Appendf(nil, `{"id": "uid_1", "name":"%s"}`, given.Identifier),
						fmt.Appendf(nil, `{"id": "uid_2", "name":"%s"}`, given.Identifier),
					}},
				}, nil
			},
		}

		err := credentialvault.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.False(t, c.deleteCalled, "it's not known what needs to be deleted")
	})
}

func TestDeleteByObjectId(t *testing.T) {
	t.Run("success if the credential vault entry exists", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           string(config.CredentialVaultID),
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libAPI.Response{}, nil
			},
		}

		err := credentialvault.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled)
	})

	t.Run("no error if the credential vault entry doesn't exist", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           string(config.CredentialVaultID),
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound}
			},
		}

		err := credentialvault.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("error if delete fails", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           string(config.CredentialVaultID),
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := deleteStubClient{
			delete: func(_ string) (libAPI.Response, error) {
				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := credentialvault.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("error if server error during delete", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           string(config.CredentialVaultID),
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := deleteStubClient{
			delete: func(_ string) (libAPI.Response, error) {
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusInternalServerError}
			},
		}

		err := credentialvault.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("deletion continues even if error occurs", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           string(config.CredentialVaultID),
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

		err := credentialvault.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given, {OriginObjectId: "bla"}, given}) // the pointer in the middle is to cause error behavior
		assert.ErrorContains(t, err, "failed to delete 1")
	})
}

func TestDeleteAll(t *testing.T) {
	t.Run("simple case", func(t *testing.T) {
		c := deleteStubClient{
			list: func() (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{
					libAPI.ListResponse{
						Objects: [][]byte{
							[]byte(`{"id": "uid_1"}`),
							[]byte(`{"id": "uid_2"}`),
							[]byte(`{"id": "uid_3"}`),
						}},
				}, nil
			},
			delete: func(uid string) (libAPI.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, uid)
				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := credentialvault.NewDeleter(&c).DeleteAll(t.Context())
		assert.NoError(t, err)
	})

	t.Run("deletion continues even if error occurs during delete", func(t *testing.T) {
		c := deleteStubClient{
			list: func() (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{
					libAPI.ListResponse{
						Objects: [][]byte{
							[]byte(`{"id": "uid_1"}`),
							[]byte(`{"id": "uid_2"}`),
							[]byte(`{"id": "uid_3"}`),
						}},
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

		err := credentialvault.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
	})
}
