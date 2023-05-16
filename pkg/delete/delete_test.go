//go:build unit

/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package delete

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeleteSettings_LegacyExternalID(t *testing.T) {
	t.Run("TestDeleteSettings_LegacyExternalID", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).DoAndReturn(func(schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
			assert.True(t, listOpts.Filter(dtclient.DownloadSettingsObject{ExternalId: "monaco:YnVpbHRpbjphbGVydGluZy5wcm9maWxlJGlkMQ=="}))
			return []dtclient.DownloadSettingsObject{
				{
					ExternalId:    "externalID",
					SchemaVersion: "v1",
					SchemaId:      "builtin:alerting.profile",
					ObjectId:      "12345",
					Scope:         "tenant",
					Value:         nil,
				},
			}, nil

		})
		c.EXPECT().DeleteSettings(gomock.Eq("12345")).Return(nil)
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		errs := Configs(c, api.NewAPIs(), entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteSettings_LegacyExternalID - List settings with external ID fails", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, client.RespError{Err: fmt.Errorf("WHOPS"), StatusCode: 0})
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		errs := Configs(c, api.NewAPIs(), entriesToDelete)
		assert.Len(t, errs, 1, "errors should have len 1")
	})

	t.Run("TestDeleteSettings_LegacyExternalID - List settings returns no objects", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, nil)
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		errs := Configs(c, api.NewAPIs(), entriesToDelete)
		assert.Len(t, errs, 0)
	})

	t.Run("TestDeleteSettings_LegacyExternalID - Delete settings based on object ID fails", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{
			{
				ExternalId:    "externalID",
				SchemaVersion: "v1",
				SchemaId:      "builtin:alerting.profile",
				ObjectId:      "12345",
				Scope:         "tenant",
				Value:         nil,
			},
		}, nil)
		c.EXPECT().DeleteSettings(gomock.Eq("12345")).Return(fmt.Errorf("WHOPS"))
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		errs := Configs(c, api.NewAPIs(), entriesToDelete)
		assert.Len(t, errs, 1, "errors should have len 1")
	})

}

func TestDeleteSettings(t *testing.T) {
	t.Run("TestDeleteSettings", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).DoAndReturn(func(schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
			expectedExtID := "monaco:cHJvamVjdCRidWlsdGluOmFsZXJ0aW5nLnByb2ZpbGUkaWQx"
			assert.True(t, listOpts.Filter(dtclient.DownloadSettingsObject{ExternalId: expectedExtID}), "Expected request filtering for externalID %q", expectedExtID)
			return []dtclient.DownloadSettingsObject{
				{
					ExternalId:    "externalID",
					SchemaVersion: "v1",
					SchemaId:      "builtin:alerting.profile",
					ObjectId:      "12345",
					Scope:         "tenant",
					Value:         nil,
				},
			}, nil

		})
		c.EXPECT().DeleteSettings(gomock.Eq("12345")).Return(nil)
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		errs := Configs(c, api.NewAPIs(), entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteSettings - List settings with external ID fails", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, client.RespError{Err: fmt.Errorf("WHOPS"), StatusCode: 0})
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		errs := Configs(c, api.NewAPIs(), entriesToDelete)
		assert.Len(t, errs, 1, "errors should have len 1")
	})

	t.Run("TestDeleteSettings - List settings returns no objects", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, nil)
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		errs := Configs(c, api.NewAPIs(), entriesToDelete)
		assert.Len(t, errs, 0)
	})

	t.Run("TestDeleteSettings - Delete settings based on object ID fails", func(t *testing.T) {
		c := dtclient.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{
			{
				ExternalId:    "externalID",
				SchemaVersion: "v1",
				SchemaId:      "builtin:alerting.profile",
				ObjectId:      "12345",
				Scope:         "tenant",
				Value:         nil,
			},
		}, nil)
		c.EXPECT().DeleteSettings(gomock.Eq("12345")).Return(fmt.Errorf("WHOPS"))
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		errs := Configs(c, api.NewAPIs(), entriesToDelete)
		assert.Len(t, errs, 1, "errors should have len 1")
	})

}

func TestSplitConfigsForDeletion(t *testing.T) {
	type expect struct {
		ids     []string
		numErrs int
	}

	type args struct {
		entries []DeletePointer
		values  []dtclient.Value
	}

	tests := []struct {
		name   string
		args   args
		expect expect
	}{
		{
			name: "Empty everything",
		},
		{
			name: "Full overlap",
			args: args{
				entries: []DeletePointer{{Identifier: "d1"}, {Identifier: "d2"}, {Identifier: "d3"}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
			},
			expect: expect{
				ids:     []string{"id1", "id2", "id3"},
				numErrs: 0,
			},
		},
		{
			name: "Empty entries, nothing deleted",
			args: args{
				entries: []DeletePointer{},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
			},
		},
		{
			name: "More deletes",
			args: args{
				entries: []DeletePointer{{Identifier: "d1"}, {Identifier: "d2"}, {Identifier: "d3"}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}},
			},
			expect: expect{
				ids:     []string{"id1"},
				numErrs: 0,
			},
		},
		{
			name: "More values",
			args: args{
				entries: []DeletePointer{{Identifier: "d1"}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
			},
			expect: expect{
				ids:     []string{"id1"},
				numErrs: 0,
			},
		},
		{
			name: "Id-fallback",
			args: args{
				entries: []DeletePointer{{Identifier: "d1"}, {Identifier: "d2-id"}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "d2-id"}, {Name: "d3", Id: "id3"}},
			},
			expect: expect{
				ids:     []string{"id1", "d2-id"},
				numErrs: 0,
			},
		},
		{
			name: "Duplicate names",
			args: args{
				entries: []DeletePointer{{Identifier: "d1"}, {Identifier: "d2"}},
				values:  []dtclient.Value{{Name: "d1"}, {Name: "d1"}, {Name: "d2"}, {Name: "d2"}},
			},
			expect: expect{
				ids:     []string{},
				numErrs: 2,
			},
		},
		{
			name: "Combined",
			args: args{
				entries: []DeletePointer{{Identifier: "d1"}, {Identifier: "d2"}, {Identifier: "d3"}, {Identifier: "d4-id"}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d2", Id: "id-something"}, {Name: "d3", Id: "id3"}, {Id: "d4-id"}},
			},
			expect: expect{
				ids:     []string{"id1", "id3", "d4-id"},
				numErrs: 1,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := api.API{ID: "some-id"}

			apiMap := api.APIs{a.ID: a}
			entriesToDelete := map[string][]DeletePointer{a.ID: tc.args.entries}

			client := dtclient.NewMockClient(gomock.NewController(t))
			client.EXPECT().ListConfigs(a).Return(tc.args.values, nil)

			for _, id := range tc.expect.ids {
				client.EXPECT().DeleteConfigById(a, id)
			}

			errs := Configs(client, apiMap, entriesToDelete)

			assert.Equal(t, len(errs), tc.expect.numErrs)
		})
	}
}

func TestSplitConfigsForDeletionClientReturnsError(t *testing.T) {
	a := api.API{ID: "some-id"}

	apiMap := api.APIs{a.ID: a}
	entriesToDelete := map[string][]DeletePointer{a.ID: {{}}}

	client := dtclient.NewMockClient(gomock.NewController(t))
	client.EXPECT().ListConfigs(a).Return(nil, errors.New("error"))

	errs := Configs(client, apiMap, entriesToDelete)

	assert.NotEmpty(t, errs, "an error should be returned")
}
