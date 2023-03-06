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
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeleteSettings(t *testing.T) {
	t.Run("TestDeleteSettings", func(t *testing.T) {
		c := client.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).DoAndReturn(func(schemaID string, listOpts client.ListSettingsOptions) ([]client.DownloadSettingsObject, error) {
			assert.True(t, listOpts.Filter(client.DownloadSettingsObject{ExternalId: "monaco:YnVpbHRpbjphbGVydGluZy5wcm9maWxlJGlkMQ=="}))
			return []client.DownloadSettingsObject{
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
					Type:     "builtin:alerting.profile",
					ConfigId: "id1",
				},
			},
		}
		errs := DeleteConfigs(c, api.NewV1Apis(), entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteSettings - List settings with external ID fails", func(t *testing.T) {
		c := client.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]client.DownloadSettingsObject{}, client.RespError{Err: fmt.Errorf("WHOPS"), StatusCode: 0})
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:     "builtin:alerting.profile",
					ConfigId: "id1",
				},
			},
		}
		errs := DeleteConfigs(c, api.NewV1Apis(), entriesToDelete)
		assert.Len(t, errs, 1, "errors should have len 1")
	})

	t.Run("TestDeleteSettings - List settings returns no objects", func(t *testing.T) {
		c := client.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]client.DownloadSettingsObject{}, nil)
		entriesToDelete := map[string][]DeletePointer{
			"builtin:alerting.profile": {
				{
					Type:     "builtin:alerting.profile",
					ConfigId: "id1",
				},
			},
		}
		errs := DeleteConfigs(c, api.NewV1Apis(), entriesToDelete)
		assert.Len(t, errs, 0)
	})

	t.Run("TestDeleteSettings - Delete settings based on object ID fails", func(t *testing.T) {
		c := client.NewMockClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Return([]client.DownloadSettingsObject{
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
					Type:     "builtin:alerting.profile",
					ConfigId: "id1",
				},
			},
		}
		errs := DeleteConfigs(c, api.NewV1Apis(), entriesToDelete)
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
		values  []api.Value
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
				entries: []DeletePointer{{ConfigId: "d1"}, {ConfigId: "d2"}, {ConfigId: "d3"}},
				values:  []api.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
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
				values:  []api.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
			},
		},
		{
			name: "More deletes",
			args: args{
				entries: []DeletePointer{{ConfigId: "d1"}, {ConfigId: "d2"}, {ConfigId: "d3"}},
				values:  []api.Value{{Name: "d1", Id: "id1"}},
			},
			expect: expect{
				ids:     []string{"id1"},
				numErrs: 0,
			},
		},
		{
			name: "More values",
			args: args{
				entries: []DeletePointer{{ConfigId: "d1"}},
				values:  []api.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
			},
			expect: expect{
				ids:     []string{"id1"},
				numErrs: 0,
			},
		},
		{
			name: "Id-fallback",
			args: args{
				entries: []DeletePointer{{ConfigId: "d1"}, {ConfigId: "d2-id"}},
				values:  []api.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "d2-id"}, {Name: "d3", Id: "id3"}},
			},
			expect: expect{
				ids:     []string{"id1", "d2-id"},
				numErrs: 0,
			},
		},
		{
			name: "Duplicate names",
			args: args{
				entries: []DeletePointer{{ConfigId: "d1"}, {ConfigId: "d2"}},
				values:  []api.Value{{Name: "d1"}, {Name: "d1"}, {Name: "d2"}, {Name: "d2"}},
			},
			expect: expect{
				ids:     []string{},
				numErrs: 2,
			},
		},
		{
			name: "Combined",
			args: args{
				entries: []DeletePointer{{ConfigId: "d1"}, {ConfigId: "d2"}, {ConfigId: "d3"}, {ConfigId: "d4-id"}},
				values:  []api.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d2", Id: "id-something"}, {Name: "d3", Id: "id3"}, {Id: "d4-id"}},
			},
			expect: expect{
				ids:     []string{"id1", "id3", "d4-id"},
				numErrs: 1,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := api.NewApi("some-id", "", "", false, false, "", false)

			apiMap := api.APIs{a.GetId(): a}
			entriesToDelete := map[string][]DeletePointer{a.GetId(): tc.args.entries}

			client := client.NewMockClient(gomock.NewController(t))
			client.EXPECT().ListConfigs(a).Return(tc.args.values, nil)

			for _, id := range tc.expect.ids {
				client.EXPECT().DeleteConfigById(a, id)
			}

			errs := DeleteConfigs(client, apiMap, entriesToDelete)

			assert.Equal(t, len(errs), tc.expect.numErrs)
		})
	}
}

func TestSplitConfigsForDeletionClientReturnsError(t *testing.T) {
	a := api.NewApi("some-id", "", "", false, false, "", false)

	apiMap := api.APIs{a.GetId(): a}
	entriesToDelete := map[string][]DeletePointer{a.GetId(): {{}}}

	client := client.NewMockClient(gomock.NewController(t))
	client.EXPECT().ListConfigs(a).Return(nil, errors.New("error"))

	errs := DeleteConfigs(client, apiMap, entriesToDelete)

	assert.NotEmpty(t, errs, "an error should be returned")
}
