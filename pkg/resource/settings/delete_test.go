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

package settings_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/settings"
)

type deleteStubClient struct {
	deleteCalled bool
	delete       func(id string) error
	list         func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error)
	listSchemas  func() (dtclient.SchemaList, error)
}

func (s *deleteStubClient) ListSchemas(_ context.Context) (dtclient.SchemaList, error) {
	return s.listSchemas()
}

func (s *deleteStubClient) List(_ context.Context, schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
	return s.list(schema, options)
}

func (s *deleteStubClient) Delete(_ context.Context, id string) error {
	s.deleteCalled = true
	return s.delete(id)
}

func TestDeleteByCoordinate(t *testing.T) {
	t.Run("no error when deleting nothing", func(t *testing.T) {
		err := settings.NewDeleter(&deleteStubClient{}).Delete(t.Context(), []pointer.DeletePointer{})
		assert.NoError(t, err)
	})

	t.Run("success if one settings object matches generated external ID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "builtin:some-settings-schema",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID, _ := idutils.GenerateExternalIDForSettingsObject(given.AsCoordinate())
		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				returnedObject := dtclient.DownloadSettingsObject{
					SchemaId:   given.Type,
					ExternalId: externalID,
					ObjectId:   "objectId",
				}

				assert.Equal(t, "builtin:some-settings-schema", schema)
				assert.True(t, options.Filter(returnedObject))

				return []dtclient.DownloadSettingsObject{returnedObject}, nil
			},
			delete: func(id string) error {
				assert.Equal(t, "objectId", id)
				return nil
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if external ID cannot be generated", func(t *testing.T) {
		given := pointer.DeletePointer{
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				returnedObject := dtclient.DownloadSettingsObject{
					ExternalId: "invalid",
					ObjectId:   "objectId",
				}

				return []dtclient.DownloadSettingsObject{returnedObject}, nil
			},
			delete: func(id string) error {
				assert.Equal(t, "objectId", id)
				return nil
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.False(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("no error if no settings object matches generated external ID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "builtin:some-settings-schema",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				return []dtclient.DownloadSettingsObject{}, nil
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.False(t, c.deleteCalled, "delete command was invoked")
	})

	t.Run("no error if multiple settings object match generated external ID", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "builtin:some-settings-schema",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID, _ := idutils.GenerateExternalIDForSettingsObject(given.AsCoordinate())
		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				returnedObject := dtclient.DownloadSettingsObject{
					SchemaId:   given.Type,
					ExternalId: externalID,
					ObjectId:   "objectId1",
				}
				returnedObject2 := dtclient.DownloadSettingsObject{
					SchemaId:   given.Type,
					ExternalId: externalID,
					ObjectId:   "objectId2",
				}

				assert.Equal(t, "builtin:some-settings-schema", schema)
				assert.True(t, options.Filter(returnedObject))
				assert.True(t, options.Filter(returnedObject2))

				return []dtclient.DownloadSettingsObject{returnedObject, returnedObject2}, nil
			},
			delete: func(id string) error {
				assert.Contains(t, []string{"objectId1", "objectId2"}, id)
				return nil
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled, "delete command wasn't invoked")
	})

	t.Run("error if list fails", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "builtin:some-settings-schema",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				return nil, errors.New("some unpredictable error")
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
	})
}

func TestDeleteByObjectId(t *testing.T) {
	t.Run("success if settings object exists", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "builtin:some-settings-schema",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				returnedObject := dtclient.DownloadSettingsObject{
					SchemaId: given.Type,
					ObjectId: given.OriginObjectId,
				}

				assert.Equal(t, "builtin:some-settings-schema", schema)
				assert.True(t, options.Filter(returnedObject))

				return []dtclient.DownloadSettingsObject{returnedObject}, nil
			},
			delete: func(id string) error {
				assert.Equal(t, given.OriginObjectId, id)
				return nil
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled)
	})

	t.Run("no error if settings object doesn't exist", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "builtin:some-settings-schema",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				return []dtclient.DownloadSettingsObject{}, nil
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.False(t, c.deleteCalled)
	})

	t.Run("non-deletable settings object is skipped", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "builtin:some-settings-schema",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				returnedObject := dtclient.DownloadSettingsObject{
					SchemaId: given.Type,
					ObjectId: given.OriginObjectId,
					ModificationInfo: &dtclient.SettingsModificationInfo{
						Deletable: false,
					},
				}

				assert.Equal(t, "builtin:some-settings-schema", schema)
				assert.True(t, options.Filter(returnedObject))

				return []dtclient.DownloadSettingsObject{returnedObject}, nil
			},
			delete: func(id string) error {
				assert.Fail(t, "should not be called")
				return nil
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.NoError(t, err)
		assert.False(t, c.deleteCalled)
	})

	t.Run("error if delete fails", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "builtin:some-settings-schema",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				returnedObject := dtclient.DownloadSettingsObject{
					SchemaId: given.Type,
					ObjectId: given.OriginObjectId,
				}

				assert.Equal(t, "builtin:some-settings-schema", schema)
				assert.True(t, options.Filter(returnedObject))

				return []dtclient.DownloadSettingsObject{returnedObject}, nil
			},
			delete: func(_ string) error {
				return errors.New("some unpredictable error")
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.True(t, c.deleteCalled)
	})

	t.Run("error if server error during delete", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "builtin:some-settings-schema",
			OriginObjectId: "originObjectID",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				returnedObject := dtclient.DownloadSettingsObject{
					SchemaId: given.Type,
					ObjectId: given.OriginObjectId,
				}

				assert.Equal(t, "builtin:some-settings-schema", schema)
				assert.True(t, options.Filter(returnedObject))

				return []dtclient.DownloadSettingsObject{returnedObject}, nil
			},
			delete: func(_ string) error {
				return libAPI.APIError{StatusCode: http.StatusInternalServerError}
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given})
		assert.Error(t, err)
		assert.True(t, c.deleteCalled)
	})

	t.Run("deletion continues even if error occurs", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "builtin:some-settings-schema",
			OriginObjectId: "originObjectID",
		}
		other := pointer.DeletePointer{
			Type:           "builtin:some-other-schema",
			OriginObjectId: "some-other-id",
		}

		c := deleteStubClient{
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				var givenObject = dtclient.DownloadSettingsObject{
					SchemaId: given.Type,
					ObjectId: given.OriginObjectId,
				}

				var otherObject = dtclient.DownloadSettingsObject{
					SchemaId: other.Type,
					ObjectId: other.OriginObjectId,
				}

				if options.Filter(givenObject) {
					return []dtclient.DownloadSettingsObject{givenObject}, nil
				} else if options.Filter(otherObject) {
					return []dtclient.DownloadSettingsObject{otherObject}, nil
				}

				return []dtclient.DownloadSettingsObject{}, nil
			},
			delete: func(id string) error {
				if id == given.OriginObjectId {
					return nil
				}
				return errors.New("some unpredictable error")
			},
		}

		err := settings.NewDeleter(&c).Delete(t.Context(), []pointer.DeletePointer{given, other, given}) // the pointer in the middle is to cause error behavior
		assert.ErrorContains(t, err, "failed to delete 1 settings object(s)")
		assert.True(t, c.deleteCalled)
	})
}

func TestDeleteAll(t *testing.T) {
	responses := map[string][]dtclient.DownloadSettingsObject{
		"builtin:some-schema": {
			{SchemaId: "builtin:some-schema", ObjectId: "some-object-1"},
			{SchemaId: "builtin:some-schema", ObjectId: "some-object-2"},
		},
		"builtin:some-other-schema": {
			{SchemaId: "builtin:some-other-schema", ObjectId: "some-object-3"},
		},
	}

	t.Run("simple case", func(t *testing.T) {
		c := deleteStubClient{
			listSchemas: func() (dtclient.SchemaList, error) {
				return []dtclient.SchemaItem{
					{SchemaId: "builtin:some-schema"},
					{SchemaId: "builtin:some-other-schema"},
				}, nil
			},
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				if response, ok := responses[schema]; ok {
					return response, nil
				}

				return []dtclient.DownloadSettingsObject{}, nil
			},
			delete: func(uid string) error {
				assert.Contains(t, []string{"some-object-1", "some-object-2", "some-object-3"}, uid)
				return nil
			},
		}

		err := settings.NewDeleter(&c).DeleteAll(t.Context())
		assert.NoError(t, err)
		assert.True(t, c.deleteCalled)
	})

	t.Run("deletion continues even if error occurs during delete", func(t *testing.T) {
		c := deleteStubClient{
			listSchemas: func() (dtclient.SchemaList, error) {
				return []dtclient.SchemaItem{
					{SchemaId: "builtin:some-schema"},
					{SchemaId: "builtin:some-other-schema"},
				}, nil
			},
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				if response, ok := responses[schema]; ok {
					return response, nil
				}

				return []dtclient.DownloadSettingsObject{}, nil
			},
			delete: func(uid string) error {
				assert.Contains(t, []string{"some-object-1", "some-object-2", "some-object-3"}, uid)
				if uid == "some-object-2" {
					return errors.New("some unpredictable error")
				}

				return nil
			},
		}
		err := settings.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
		assert.True(t, c.deleteCalled)
	})

	t.Run("error if list fails", func(t *testing.T) {
		c := deleteStubClient{
			listSchemas: func() (dtclient.SchemaList, error) {
				return []dtclient.SchemaItem{
					{SchemaId: "builtin:some-schema"},
					{SchemaId: "builtin:some-other-schema"},
				}, nil
			},
			list: func(schema string, options dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
				return nil, errors.New("some unpredictable error")
			},
		}

		err := settings.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
	})

	t.Run("error if listSchemas fails", func(t *testing.T) {
		c := deleteStubClient{
			listSchemas: func() (dtclient.SchemaList, error) {
				return nil, errors.New("some unpredictable error")
			},
		}

		err := settings.NewDeleter(&c).DeleteAll(t.Context())
		assert.Error(t, err)
	})
}
