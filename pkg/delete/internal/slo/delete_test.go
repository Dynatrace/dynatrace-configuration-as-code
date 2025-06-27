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

package slo_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/slo"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

func TestDeleteWithCoordinate(t *testing.T) {
	t.Run("success if one slo-v2 matches generated external ID", func(t *testing.T) {

		given := pointer.DeletePointer{
			Type:       "slo-v2",
			Identifier: "monaco_identifier",
			Project:    "project",
		}
		externalID := idutils.GenerateExternalID(given.AsCoordinate())

		c := stubClient{
			list: func() (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{
					libAPI.ListResponse{
						Objects: [][]byte{
							[]byte(fmt.Sprintf(`{"id": "uid_1", "externalId":"%s"}`, externalID)),
							[]byte(`{"id": "uid_2", "externalId":"wrong"}`),
						},
					}}, nil
			},
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, "uid_1", id)
				return libAPI.Response{}, nil
			},
		}

		err := slo.Delete(context.TODO(), &c, []pointer.DeletePointer{given})

		assert.NoError(t, err)
		assert.True(t, c.called, "delete command wasn't invoked")
	})

	t.Run("no error if no slo-v2 matches generated external ID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "slo-v2",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := stubClient{
			list: func() (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{
					libAPI.ListResponse{Objects: [][]byte{[]byte(`{"uid": "uid_2", "externalId":"wrong"}`)}},
				}, nil
			},
		}

		err := slo.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("error if multiple slo-v2 match generated external ID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "slo-v2",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID := idutils.GenerateExternalID(given.AsCoordinate())
		c := stubClient{
			list: func() (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{
					libAPI.ListResponse{Objects: [][]byte{
						[]byte(fmt.Sprintf(`{"id": "uid_1", "externalId":"%s"}`, externalID)),
						[]byte(fmt.Sprintf(`{"id": "uid_2", "externalId":"%s"}`, externalID)),
					}},
				}, nil
			},
		}

		err := slo.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.False(t, c.called, "it's not known what needs to be deleted")
	})
}

func TestDeleteByObjectId(t *testing.T) {
	t.Run("success if SLOv2 exists", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "slo-v2",
			OriginObjectId: "originObjectID",
		}

		c := stubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libAPI.Response{}, nil
			},
		}

		err := slo.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.called)
	})

	t.Run("no error if SLOv2 doesn't exist", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "slo-v2",
			OriginObjectId: "originObjectID",
		}

		c := stubClient{
			delete: func(id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound}
			},
		}

		err := slo.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.NoError(t, err)
	})

	t.Run("error if delete fails", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "slo-v2",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := stubClient{
			delete: func(_ string) (libAPI.Response, error) {
				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := slo.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("error if server error during delete", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "slo-v2",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := stubClient{
			delete: func(_ string) (libAPI.Response, error) {
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusInternalServerError}
			},
		}

		err := slo.Delete(context.TODO(), &c, []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("deletion continues even if error occurs", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "slo-v2",
			OriginObjectId: "originObjectID",
			Project:        "project",
		}

		c := stubClient{
			delete: func(uid string) (libAPI.Response, error) {
				if uid == given.OriginObjectId {
					return libAPI.Response{}, nil
				}
				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := slo.Delete(context.TODO(), &c, []pointer.DeletePointer{given, {OriginObjectId: "bla"}, given}) // the pointer in the middle is to cause error behavior
		assert.ErrorContains(t, err, "failed to delete 1 slo-v2 objects(s)")
	})
}

type stubClient struct {
	called bool
	delete func(id string) (libAPI.Response, error)
	list   func() (libAPI.PagedListResponse, error)
}

func (s *stubClient) List(context.Context) (libAPI.PagedListResponse, error) {
	return s.list()
}

func (s *stubClient) Delete(_ context.Context, id string) (libAPI.Response, error) {
	s.called = true
	return s.delete(id)
}

func TestDeleteAll(t *testing.T) {
	t.Run("simple case", func(t *testing.T) {
		c := stubClient{
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

		err := slo.DeleteAll(context.TODO(), &c)
		assert.NoError(t, err)
	})

	t.Run("deletion continues even if error occurs during delete", func(t *testing.T) {
		c := stubClient{
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

		err := slo.DeleteAll(context.TODO(), &c)
		assert.Error(t, err)
	})
}
