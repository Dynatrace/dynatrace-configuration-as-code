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

package classic_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/classic"
)

type deleteSourceStub struct {
	countDeleteCalled int
	delete            func(api api.API, id string) error
	list              func(api api.API) ([]dtclient.Value, error)
}

func (s *deleteSourceStub) List(_ context.Context, api api.API) ([]dtclient.Value, error) {
	return s.list(api)
}

func (s *deleteSourceStub) Delete(_ context.Context, api api.API, id string) error {
	s.countDeleteCalled++
	return s.delete(api, id)
}

func TestDeleteByCoordinate(t *testing.T) {
	t.Run("no error when deleting nothing", func(t *testing.T) {
		err := classic.NewDeleter(&deleteSourceStub{}).Delete(t.Context(), []pointer.DeletePointer{})
		assert.NoError(t, err)
	})

	t.Run("no error if no matching config found", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       api.Autotag,
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				return []dtclient.Value{}, nil
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.Equal(t, 0, c.countDeleteCalled, "delete command was invoked")
	})

	t.Run("url for API with parent is handled correctly", func(t *testing.T) {
		responses := map[string][]dtclient.Value{
			api.DashboardShareSettings: {{Id: "some-id"}},
			api.Dashboard:              {{Id: "some-dashboard"}},
		}

		given := pointer.DeletePointer{
			Type:       api.DashboardShareSettings,
			Identifier: "some-id",
			Project:    "project",
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				if response, ok := responses[theApi.ID]; ok {
					return response, nil
				}

				return []dtclient.Value{}, nil
			},
			delete: func(theApi api.API, id string) error {
				assert.Equal(t, "/api/config/v1/dashboards/some-dashboard/shareSettings", theApi.URLPath)
				assert.Equal(t, api.DashboardShareSettings, theApi.ID)
				assert.Equal(t, "some-id", id)
				return nil
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.Equal(t, 1, c.countDeleteCalled, "delete command wasn't invoked")
	})

	t.Run("api with parent is skipped if parent is not found", func(t *testing.T) {
		responses := map[string][]dtclient.Value{
			api.DashboardShareSettings: {{Id: "some-id"}},
		}

		given := pointer.DeletePointer{
			Type:       api.DashboardShareSettings,
			Identifier: "some-id",
			Project:    "project",
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				if response, ok := responses[theApi.ID]; ok {
					return response, nil
				}

				return []dtclient.Value{}, nil
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.Equal(t, 0, c.countDeleteCalled, "delete command was invoked")
	})

	t.Run("error if resolving parent fails", func(t *testing.T) {
		responses := map[string][]dtclient.Value{
			api.DashboardShareSettings: {{Id: "some-id"}},
		}

		given := pointer.DeletePointer{
			Type:       api.DashboardShareSettings,
			Identifier: "some-id",
			Project:    "project",
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				if response, ok := responses[theApi.ID]; ok {
					return response, nil
				}

				return []dtclient.Value{}, fmt.Errorf("something went wrong")
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.Equal(t, 0, c.countDeleteCalled, "delete command was invoked")
	})

	t.Run("success if one classic config matches id", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       api.Autotag,
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				return []dtclient.Value{
					{Id: given.Identifier},
				}, nil
			},
			delete: func(theApi api.API, id string) error {
				assert.Equal(t, api.Autotag, theApi.ID)
				assert.Equal(t, given.Identifier, id)
				return nil
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.Equal(t, 1, c.countDeleteCalled, "delete command wasn't invoked")
	})

	t.Run("success if one classic config matches name", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       api.Autotag,
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				return []dtclient.Value{
					{Id: "some-id", Name: given.Identifier},
				}, nil
			},
			delete: func(theApi api.API, id string) error {
				assert.Equal(t, api.Autotag, theApi.ID)
				assert.Equal(t, "some-id", id)
				return nil
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.Equal(t, 1, c.countDeleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if multiple classic config match name", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       api.Autotag,
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				return []dtclient.Value{
					{Id: "some-id", Name: given.Identifier},
					{Id: "some-other-id", Name: given.Identifier},
				}, nil
			},
			delete: func(theApi api.API, id string) error {
				assert.Equal(t, api.Autotag, theApi.ID)
				assert.Equal(t, given.Identifier, id)
				return nil
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.ErrorContains(t, err, "unable to find unique config - matching IDs are [some-id some-other-id]")
		assert.Equal(t, 0, c.countDeleteCalled, "delete command was invoked")
	})

	t.Run("error if list fails", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       api.Autotag,
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				return nil, errors.New("some error")
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.Equal(t, 0, c.countDeleteCalled, "delete command was invoked")
	})

	t.Run("deletion continues even if error occurs", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       api.Autotag,
			Identifier: "monaco_identifier_1",
			Project:    "project",
		}
		other := pointer.DeletePointer{
			Type:       api.AppDetectionRule,
			Identifier: "monaco_identifier_2",
			Project:    "project",
		}

		responses := map[string][]dtclient.Value{
			api.Autotag: {
				{Id: "some-id", Name: given.Identifier},
			},
			api.AppDetectionRule: {
				{Id: "some-other-id", Name: other.Identifier},
			},
		}

		c := deleteSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				if response, ok := responses[theApi.ID]; ok {
					return response, nil
				}

				return []dtclient.Value{}, nil
			},
			delete: func(_ api.API, id string) error {
				if id == "some-id" {
					return nil
				}
				return errors.New("some unpredictable error")
			},
		}

		err := classic.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given, other, given}) // the pointer in the middle is to cause error behavior
		assert.Error(t, err)
		assert.Equal(t, 3, c.countDeleteCalled, fmt.Sprintf(
			"delete command was expected to be called 3 times, but was called %d times", c.countDeleteCalled))
	})
}
