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

package automation_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	coreautomation "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/automation"
)

type deleteStubClient struct {
	deleteCalled bool
	delete       func(resourceType coreautomation.ResourceType, id string) (libAPI.Response, error)
	list         func(resourceType coreautomation.ResourceType) (libAPI.PagedListResponse, error)
}

func (s *deleteStubClient) List(_ context.Context, resourceType coreautomation.ResourceType) (libAPI.PagedListResponse, error) {
	return s.list(resourceType)
}

func (s *deleteStubClient) Delete(_ context.Context, resourceType coreautomation.ResourceType, id string) (libAPI.Response, error) {
	s.deleteCalled = true
	return s.delete(resourceType, id)
}

func TestDeleteByCoordinate(t *testing.T) {
	t.Run("success if one automation matches generated UUID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       string(config.Workflow),
			Identifier: "monaco_identifier",
			Project:    "project",
		}
		uuid := idutils.GenerateUUIDFromCoordinate(given.AsCoordinate())

		c := deleteStubClient{
			delete: func(resourceType coreautomation.ResourceType, id string) (libAPI.Response, error) {
				assert.Equal(t, uuid, id)
				assert.Equal(t, coreautomation.Workflows, resourceType)
				return libAPI.Response{}, nil
			},
		}

		err := automation.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("no error if delete pointers are empty", func(t *testing.T) {
		err := automation.NewDeleter(&deleteStubClient{}).Delete(t.Context(), []pointer.DeletePointer{})
		assert.NoError(t, err)
	})

	t.Run("no error if delete returns 404", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       string(config.SchedulingRule),
			Identifier: "monaco_identifier",
			Project:    "project",
		}
		uuid := idutils.GenerateUUIDFromCoordinate(given.AsCoordinate())

		c := deleteStubClient{
			delete: func(resourceType coreautomation.ResourceType, id string) (libAPI.Response, error) {
				assert.Equal(t, uuid, id)
				assert.Equal(t, coreautomation.SchedulingRules, resourceType)
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusNotFound}
			},
		}

		err := automation.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if delete returns 500", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       string(config.BusinessCalendar),
			Identifier: "monaco_identifier",
			Project:    "project",
		}
		uuid := idutils.GenerateUUIDFromCoordinate(given.AsCoordinate())

		c := deleteStubClient{
			delete: func(resourceType coreautomation.ResourceType, id string) (libAPI.Response, error) {
				assert.Equal(t, uuid, id)
				assert.Equal(t, coreautomation.BusinessCalendars, resourceType)
				return libAPI.Response{}, libAPI.APIError{StatusCode: http.StatusInternalServerError}
			},
		}

		err := automation.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if automation delete is deleteCalled with wrong type", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       api.AlertingProfile,
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteStubClient{}

		err := automation.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.False(t, c.deleteCalled, "delete command was invoked")
	})
}

func TestDeleteByObjectId(t *testing.T) {
	t.Run("success if automation exists", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           string(config.Workflow),
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(resourceType coreautomation.ResourceType, id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				assert.Equal(t, coreautomation.Workflows, resourceType)
				return libAPI.Response{}, nil
			},
		}

		err := automation.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if delete fails", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           string(config.Workflow),
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			delete: func(resourceType coreautomation.ResourceType, id string) (libAPI.Response, error) {
				assert.Equal(t, given.OriginObjectId, id)
				assert.Equal(t, coreautomation.Workflows, resourceType)
				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := automation.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
	})

	t.Run("deletion continues even if error occurs", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           string(config.Workflow),
			OriginObjectId: "originObjectID",
		}
		other := pointer.DeletePointer{
			Type:           string(config.Workflow),
			OriginObjectId: "other",
		}

		c := deleteStubClient{
			delete: func(resourceType coreautomation.ResourceType, uid string) (libAPI.Response, error) {
				assert.Equal(t, coreautomation.Workflows, resourceType)
				if uid == given.OriginObjectId {
					return libAPI.Response{}, nil
				}

				return libAPI.Response{}, errors.New("some unpredictable error")
			},
		}

		err := automation.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given, other, given}) // the pointer in the middle is to cause error behavior
		assert.ErrorContains(t, err, "failed to delete 1 automation object(s)")
	})
}

func TestDeleteAll(t *testing.T) {
	var responses = map[coreautomation.ResourceType]libAPI.PagedListResponse{
		coreautomation.Workflows: {
			{
				Response: libAPI.Response{},
				Objects: [][]byte{
					[]byte(`{"id": "uid_1"}`),
				},
			},
		},
		coreautomation.SchedulingRules: {
			{
				Response: libAPI.Response{},
				Objects: [][]byte{
					[]byte(`{"id": "uid_2"}`),
				},
			},
		},
		coreautomation.BusinessCalendars: {
			{
				Response: libAPI.Response{},
				Objects: [][]byte{
					[]byte(`{"id": "uid_3"}`),
				},
			},
		},
	}

	t.Run("simple case", func(t *testing.T) {
		c := deleteStubClient{
			list: func(resourceType coreautomation.ResourceType) (libAPI.PagedListResponse, error) {
				if value, ok := responses[resourceType]; ok {
					return value, nil
				}

				return libAPI.PagedListResponse{}, nil
			},
			delete: func(resourceType coreautomation.ResourceType, uid string) (libAPI.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, uid)
				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := automation.NewDeleter(&c).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, c.deleteCalled, true)
	})

	t.Run("deletion continues on errors with malformed json", func(t *testing.T) {
		c := deleteStubClient{
			list: func(resourceType coreautomation.ResourceType) (libAPI.PagedListResponse, error) {
				if resourceType == coreautomation.Workflows {
					return libAPI.PagedListResponse{
						{Response: libAPI.Response{}, Objects: [][]byte{[]byte(`{malformed-json}`)}},
					}, nil
				}

				if value, ok := responses[resourceType]; ok {
					return value, nil
				}

				return libAPI.PagedListResponse{}, nil
			},
			delete: func(resourceType coreautomation.ResourceType, uid string) (libAPI.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, uid)
				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := automation.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.Equal(t, c.deleteCalled, true)
	})

	t.Run("error if list fails", func(t *testing.T) {
		c := deleteStubClient{
			list: func(resourceType coreautomation.ResourceType) (libAPI.PagedListResponse, error) {
				return libAPI.PagedListResponse{}, fmt.Errorf("some unpredictable error")
			},
			delete: func(resourceType coreautomation.ResourceType, uid string) (libAPI.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2"}, uid)
				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := automation.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
	})

	t.Run("deletion continues even if error occurs during delete", func(t *testing.T) {
		c := deleteStubClient{
			list: func(resourceType coreautomation.ResourceType) (libAPI.PagedListResponse, error) {
				if value, ok := responses[resourceType]; ok {
					return value, nil
				}

				return libAPI.PagedListResponse{}, nil
			},
			delete: func(resourceType coreautomation.ResourceType, uid string) (libAPI.Response, error) {
				assert.Contains(t, []string{"uid_1", "uid_2", "uid_3"}, uid)
				if uid == "uid_1" {
					return libAPI.Response{}, fmt.Errorf("some unpredictable error")
				}

				return libAPI.Response{StatusCode: http.StatusOK}, nil
			},
		}

		err := automation.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.Equal(t, c.deleteCalled, true)
	})
}
