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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/classic"
)

type purgeSourceStub struct {
	countDeleteCalled int
	delete            func(api api.API, id string) error
	list              func(api api.API) ([]dtclient.Value, error)
}

func (s *purgeSourceStub) List(_ context.Context, api api.API) ([]dtclient.Value, error) {
	return s.list(api)
}

func (s *purgeSourceStub) Delete(_ context.Context, api api.API, id string) error {
	s.countDeleteCalled++
	return s.delete(api, id)
}

func TestDeleteAll(t *testing.T) {
	responses := map[string][]dtclient.Value{
		api.Autotag: {
			{Id: "some-id-1"},
			{Id: "some-id-2"},
		},
		api.AppDetectionRule: {
			{Id: "some-id-3"},
			{Id: "some-id-4"},
		},
	}

	t.Run("simple case", func(t *testing.T) {
		c := purgeSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				if response, ok := responses[theApi.ID]; ok {
					return response, nil
				}

				return []dtclient.Value{}, nil
			},
			delete: func(theApi api.API, id string) error {
				assert.Contains(t, []string{"some-id-1", "some-id-2", "some-id-3", "some-id-4"}, id)
				assert.Contains(t, []string{api.Autotag, api.AppDetectionRule}, theApi.ID)
				return nil
			},
		}

		err := classic.NewPurger(&c, api.NewAPIs()).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, 4, c.countDeleteCalled,
			fmt.Sprintf("expected delete to be called 4 times but was called %d times", c.countDeleteCalled))
	})

	t.Run("only deletes configs of specific apis", func(t *testing.T) {
		c := purgeSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				if response, ok := responses[theApi.ID]; ok {
					return response, nil
				}

				return []dtclient.Value{}, nil
			},
			delete: func(theApi api.API, id string) error {
				assert.Contains(t, []string{"some-id-1", "some-id-2"}, id)
				assert.Contains(t, []string{api.Autotag}, theApi.ID)
				return nil
			},
		}

		filteredApis := api.NewAPIs().Filter(
			func(theApi api.API) bool {
				return theApi.ID != api.Autotag
			})

		err := classic.NewPurger(&c, filteredApis).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, 2, c.countDeleteCalled,
			fmt.Sprintf("expected delete to be called 2 times but was called %d times", c.countDeleteCalled))
	})

	t.Run("deletion continues even if error occurs during delete", func(t *testing.T) {
		c := purgeSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				if response, ok := responses[theApi.ID]; ok {
					return response, nil
				}

				return []dtclient.Value{}, nil
			},
			delete: func(theApi api.API, id string) error {
				assert.Contains(t, []string{"some-id-1", "some-id-2", "some-id-3", "some-id-4"}, id)
				assert.Contains(t, []string{api.Autotag, api.AppDetectionRule}, theApi.ID)

				if id == "some-id-2" {
					return errors.New("some unpredictable error")
				}

				return nil
			},
		}
		err := classic.NewPurger(&c, api.NewAPIs()).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.Equal(t, 4, c.countDeleteCalled,
			fmt.Sprintf("expected delete to be called 4 times but was called %d times", c.countDeleteCalled))
	})

	t.Run("deletion continues even if error occurs during list for one API", func(t *testing.T) {
		c := purgeSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				if theApi.ID == api.Autotag {
					return nil, errors.New("some unpredictable error")
				}

				if response, ok := responses[theApi.ID]; ok {
					return response, nil
				}

				return []dtclient.Value{}, nil
			},
			delete: func(theApi api.API, id string) error {
				assert.Contains(t, []string{"some-id-3", "some-id-4"}, id)
				assert.Contains(t, []string{api.AppDetectionRule}, theApi.ID)

				return nil
			},
		}
		err := classic.NewPurger(&c, api.NewAPIs()).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.Equal(t, 2, c.countDeleteCalled,
			fmt.Sprintf("expected delete to be called 2 times but was called %d times", c.countDeleteCalled))
	})

	t.Run("error if list fails", func(t *testing.T) {
		c := purgeSourceStub{
			list: func(theApi api.API) ([]dtclient.Value, error) {
				return nil, errors.New("some unpredictable error")
			},
		}

		err := classic.NewPurger(&c, api.NewAPIs()).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.Equal(t, 0, c.countDeleteCalled,
			fmt.Sprintf("expected delete to be called 0 times but was called %d times", c.countDeleteCalled))
	})
}
