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

package delete_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils/matcher"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

func TestDeleteSettings_LegacyExternalID(t *testing.T) {
	t.Run("TestDeleteSettings_LegacyExternalID", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
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
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("12345")).Return(nil)
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		errs := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteSettings_LegacyExternalID - List settings with external ID fails", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, coreapi.APIError{StatusCode: 0})
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.Error(t, err)
	})

	t.Run("TestDeleteSettings_LegacyExternalID - List settings returns no objects", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, nil)
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("TestDeleteSettings_LegacyExternalID - Delete settings based on object ID fails", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{
			{
				ExternalId:    "externalID",
				SchemaVersion: "v1",
				SchemaId:      "builtin:alerting.profile",
				ObjectId:      "12345",
				Scope:         "tenant",
				Value:         nil,
			},
		}, nil)
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("12345")).Return(fmt.Errorf("WHOPS"))
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.Error(t, err)
	})
}

func TestDeleteSettings(t *testing.T) {
	t.Run("TestDeleteSettings", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Eq("builtin:alerting.profile"), gomock.Any()).DoAndReturn(func(ctx context.Context, schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
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
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("12345")).Return(nil)
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("TestDeleteSettings - List settings with external ID fails", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, coreapi.APIError{StatusCode: 0})
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.Error(t, err)
	})

	t.Run("TestDeleteSettings - List settings returns no objects", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, nil)
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("TestDeleteSettings - Delete settings based on object ID fails", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{
			{
				ExternalId:    "externalID",
				SchemaVersion: "v1",
				SchemaId:      "builtin:alerting.profile",
				ObjectId:      "12345",
				Scope:         "tenant",
				Value:         nil,
			},
		}, nil)
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("12345")).Return(fmt.Errorf("WHOPS"))
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.Error(t, err)
	})

	t.Run("TestDeleteSettings - Skips non-deletable Objects", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
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
					ModificationInfo: &dtclient.SettingsModificationInfo{
						Deletable:  false, // can not be deleted and should be skipped early
						Modifiable: true,
					},
				},
			}, nil

		})
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("12345")).Times(0) // deletion should not be attempted for non-deletable objects
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("identification via 'objectId'", func(t *testing.T) {
		c := client.NewMockSettingsClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Eq("builtin:alerting.profile"), gomock.Any()).DoAndReturn(func(ctx context.Context, schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
			assert.True(t, listOpts.Filter(dtclient.DownloadSettingsObject{ObjectId: "DT-original-object-ID"}), "Expected request filtering for objectId %q", "DT-original-object-ID")
			return []dtclient.DownloadSettingsObject{
				{
					ExternalId:    "externalID",
					SchemaVersion: "v1",
					SchemaId:      "builtin:alerting.profile",
					ObjectId:      "DT-original-object-ID",
					Scope:         "tenant",
					Value:         nil,
				},
			}, nil

		})
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("DT-original-object-ID")).Return(nil)
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:           "builtin:alerting.profile",
					OriginObjectId: "DT-original-object-ID",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{SettingsClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

}

func TestDelete_Automations(t *testing.T) {
	t.Run("TestDeleteAutomations", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "workflows") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/e8fd06bf-08ab-3a2f-9d3f-1fd66ea870a2"))
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		c := automation.NewClient(rest.NewClient(serverURL, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"workflow": {
				{
					Type:       "workflow",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		errs := delete.Configs(t.Context(), client.ClientSet{AutClient: c}, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteAutomations - Several Types", func(t *testing.T) {

		var workflowDeleted, calendarDeleted, scheduleDeleted bool

		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "workflows") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/e8fd06bf-08ab-3a2f-9d3f-1fd66ea870a2"))
				rw.WriteHeader(http.StatusOK)
				workflowDeleted = true
				return
			}
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "business-calendars") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/0d17aa4d-9502-3fea-aa90-4e9529b3f199"))
				rw.WriteHeader(http.StatusOK)
				calendarDeleted = true
				return
			}
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "scheduling-rules") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/e8f508f5-ff81-32a5-be6d-5d6c6295dabb"))
				rw.WriteHeader(http.StatusOK)
				scheduleDeleted = true
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		c := automation.NewClient(rest.NewClient(serverURL, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"workflow": {
				{
					Type:       "workflow",
					Project:    "project",
					Identifier: "id1",
				},
			},
			"business-calendar": {
				{
					Type:       "business-calendar",
					Project:    "project",
					Identifier: "id2",
				},
			},
			"scheduling-rule": {
				{
					Type:       "scheduling-rule",
					Project:    "project",
					Identifier: "id3",
				},
			},
		}
		errs := delete.Configs(t.Context(), client.ClientSet{AutClient: c}, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
		assert.True(t, workflowDeleted, "expected workflow to be deleted but it was not")
		assert.True(t, calendarDeleted, "expected business-calendar to be deleted but it was not")
		assert.True(t, scheduleDeleted, "expected scheduling-rule to be deleted but it was not")
	})

	t.Run("TestDeleteAutomations - No Error if object does not exist", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "workflows") {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		c := automation.NewClient(rest.NewClient(serverURL, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"workflow": {
				{
					Type:       "workflow",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		errs := delete.Configs(t.Context(), client.ClientSet{AutClient: c}, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteAutomations - Returns Error on HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "workflows") {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		c := automation.NewClient(rest.NewClient(serverURL, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"workflow": {
				{
					Type:       "workflow",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err = delete.Configs(t.Context(), client.ClientSet{AutClient: c}, entriesToDelete)
		assert.Error(t, err)
	})

	t.Run("identification via 'objectId'", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "workflows") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/DT-original-object-ID"))
				rw.WriteHeader(http.StatusOK)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		assert.NoError(t, err)
		c := automation.NewClient(rest.NewClient(serverURL, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"workflow": {
				{
					Type:           "workflow",
					OriginObjectId: "DT-original-object-ID",
				},
			},
		}
		errs := delete.Configs(t.Context(), client.ClientSet{AutClient: c}, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})
}

func TestDeleteBuckets(t *testing.T) {
	t.Run("should call delete of bucket", func(t *testing.T) {
		mux := http.NewServeMux()
		apiCalls := 0
		mux.HandleFunc("GET /platform/storage/management/v1/bucket-definitions/{bucketName}", func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusNotFound)
			apiCalls++
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"bucket": {
				{
					Type:           "bucket",
					OriginObjectId: "origin_object_ID",
				},
			},
		}
		errs := delete.Configs(t.Context(), client.ClientSet{BucketClient: c}, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
		// to be sure that the bucket client was actually used
		assert.Equal(t, apiCalls, 1)
	})

	t.Run("should print a log if bucket client is not defined", func(t *testing.T) {
		builder := strings.Builder{}
		log.PrepareLogging(t.Context(), afero.NewOsFs(), false, &builder, false, false)

		entriesToDelete := delete.DeleteEntries{
			"bucket": {
				{
					Type:           "bucket",
					OriginObjectId: "origin_object_ID",
				},
			},
		}
		errs := delete.Configs(t.Context(), client.ClientSet{}, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
		assert.Contains(t, builder.String(), "Skipped deletion")
	})
}

func TestSplitConfigsForDeletion(t *testing.T) {
	a := api.NewAPIs()[api.ApplicationWeb]
	type (
		expect struct {
			ids []string
			err bool
		}

		args struct {
			entries []pointer.DeletePointer
			values  []dtclient.Value
		}
	)

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
				entries: []pointer.DeletePointer{{Identifier: "d1", Type: a.ID}, {Identifier: "d2", Type: a.ID}, {Identifier: "d3", Type: a.ID}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
			},
			expect: expect{
				ids: []string{"id1", "id2", "id3"},
				err: false,
			},
		},
		{
			name: "Empty entries, nothing deleted",
			args: args{
				entries: []pointer.DeletePointer{},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
			},
		},
		{
			name: "More deletes",
			args: args{
				entries: []pointer.DeletePointer{{Identifier: "d1", Type: a.ID}, {Identifier: "d2", Type: a.ID}, {Identifier: "d3", Type: a.ID}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}},
			},
			expect: expect{
				ids: []string{"id1"},
				err: false,
			},
		},
		{
			name: "More values",
			args: args{
				entries: []pointer.DeletePointer{{Identifier: "d1", Type: a.ID}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d3", Id: "id3"}},
			},
			expect: expect{
				ids: []string{"id1"},
				err: false,
			},
		},
		{
			name: "ID-fallback",
			args: args{
				entries: []pointer.DeletePointer{{Identifier: "d1", Type: a.ID}, {Identifier: "d2-id", Type: a.ID}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "d2-id"}, {Name: "d3", Id: "id3"}},
			},
			expect: expect{
				ids: []string{"id1", "d2-id"},
				err: false,
			},
		},
		{
			name: "Duplicate names",
			args: args{
				entries: []pointer.DeletePointer{{Identifier: "d1", Type: a.ID}, {Identifier: "d2", Type: a.ID}},
				values:  []dtclient.Value{{Name: "d1", Id: "1"}, {Name: "d1", Id: "2"}, {Name: "d2", Id: "3"}, {Name: "d2", Id: "4"}},
			},
			expect: expect{
				ids: []string{},
				err: true,
			},
		},
		{
			name: "Combined",
			args: args{
				entries: []pointer.DeletePointer{{Identifier: "d1", Type: a.ID}, {Identifier: "d2", Type: a.ID}, {Identifier: "d3", Type: a.ID}, {Identifier: "d4-id", Type: a.ID}},
				values:  []dtclient.Value{{Name: "d1", Id: "id1"}, {Name: "d2", Id: "id2"}, {Name: "d2", Id: "id-something"}, {Name: "d3", Id: "id3"}, {Id: "d4-id"}},
			},
			expect: expect{
				ids: []string{"id1", "id3", "d4-id"},
				err: true,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			entriesToDelete := delete.DeleteEntries{a.ID: tc.args.entries}

			c := client.NewMockConfigClient(gomock.NewController(t))
			if len(tc.args.entries) > 0 {
				c.EXPECT().List(gomock.Any(), matcher.EqAPI(a)).Return(tc.args.values, nil).Times(len(tc.args.entries))
			}
			for _, id := range tc.expect.ids {
				c.EXPECT().Delete(gomock.Any(), matcher.EqAPI(a), id).Times(1)
			}

			err := delete.Configs(t.Context(), client.ClientSet{ConfigClient: c}, entriesToDelete)
			if tc.expect.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigsWithParent(t *testing.T) {
	theAPI := api.NewAPIs()[api.KeyUserActionsMobile]

	type (
		listMock struct {
			api      api.API
			response []dtclient.Value
			err      error
		}
		delMock struct {
			api api.API
			id  string
			err error
		}
		mockData struct {
			parentList, list *listMock
			del              *delMock
		}
	)

	tests := []struct {
		name      string
		mock      mockData
		forDelete delete.DeleteEntries
		wantErr   bool
	}{
		{
			name: "simple case",
			mock: mockData{
				parentList: &listMock{
					api:      *theAPI.Parent,
					response: []dtclient.Value{{Id: "APP-ID", Name: "application name"}},
				},
				list: &listMock{
					api:      theAPI.ApplyParentObjectID("APP-ID"),
					response: []dtclient.Value{{Id: "DT-id-of-app", Name: "test"}},
				},
				del: &delMock{
					api: theAPI.ApplyParentObjectID("APP-ID"),
					id:  "DT-id-of-app",
				},
			},
			forDelete: delete.DeleteEntries{
				"key-user-actions-mobile": {
					{
						Type:       "key-user-actions-mobile",
						Identifier: "test",
						Scope:      "application name",
					},
				},
			},
		},
		{
			name: "can't get list - error",
			mock: mockData{
				parentList: &listMock{
					api:      *theAPI.Parent,
					response: []dtclient.Value{{Id: "APP-ID", Name: "application name"}},
				},
				list: &listMock{
					api: theAPI.ApplyParentObjectID("APP-ID"),
					err: coreapi.APIError{StatusCode: http.StatusBadRequest},
				}},
			forDelete: delete.DeleteEntries{
				"key-user-actions-mobile": {
					{
						Type:       "key-user-actions-mobile",
						Identifier: "test",
						Scope:      "application name",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "can't get parent - error",
			mock: mockData{
				parentList: &listMock{
					api: *theAPI.Parent,
					err: coreapi.APIError{StatusCode: http.StatusBadRequest},
				}},
			forDelete: delete.DeleteEntries{
				"key-user-actions-mobile": {
					{
						Type:       "key-user-actions-mobile",
						Identifier: "test",
						Scope:      "application name",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "parent doesn't exist - no error",
			mock: mockData{
				parentList: &listMock{
					api:      *theAPI.Parent,
					response: []dtclient.Value{{Id: "APP-ID", Name: "not my app"}},
				},
			},
			forDelete: delete.DeleteEntries{
				"key-user-actions-mobile": {
					{
						Type:       "key-user-actions-mobile",
						Identifier: "test",
						Scope:      "application name",
					},
				},
			},
		},
		{
			name: "object doesn't exist - no error",
			mock: mockData{
				parentList: &listMock{
					api:      *theAPI.Parent,
					response: []dtclient.Value{{Id: "APP-ID", Name: "application name"}},
				},
				list: &listMock{
					api:      theAPI.ApplyParentObjectID("APP-ID"),
					response: []dtclient.Value{{Id: "12345", Name: "your princess is in another castle"}},
				},
			},
			forDelete: delete.DeleteEntries{
				"key-user-actions-mobile": {
					{
						Type:       "key-user-actions-mobile",
						Identifier: "test",
						Scope:      "application name",
					},
				},
			},
		},
		{
			name: "delete object doesn't exist (e.g. already deleted) - no error",
			mock: mockData{
				parentList: &listMock{
					api:      *theAPI.Parent,
					response: []dtclient.Value{{Id: "APP-ID", Name: "application name"}},
				},
				list: &listMock{
					api:      theAPI.ApplyParentObjectID("APP-ID"),
					response: []dtclient.Value{{Id: "DT-id-of-app", Name: "test"}},
				},
				del: &delMock{
					api: theAPI.ApplyParentObjectID("APP-ID"),
					id:  "DT-id-of-app",
					err: coreapi.APIError{StatusCode: http.StatusNotFound},
				},
			},
			forDelete: delete.DeleteEntries{
				"key-user-actions-mobile": {
					{
						Type:       "key-user-actions-mobile",
						Identifier: "test",
						Scope:      "application name",
					},
				},
			},
		},
		{
			name: "delete fails",
			mock: mockData{
				parentList: &listMock{
					api:      *theAPI.Parent,
					response: []dtclient.Value{{Id: "APP-ID", Name: "application name"}},
				},
				list: &listMock{
					api:      theAPI.ApplyParentObjectID("APP-ID"),
					response: []dtclient.Value{{Id: "DT-id-of-app", Name: "test"}},
				},
				del: &delMock{
					api: theAPI.ApplyParentObjectID("APP-ID"),
					id:  "DT-id-of-app",
					err: fmt.Errorf("FAILED"),
				},
			},
			forDelete: delete.DeleteEntries{
				"key-user-actions-mobile": {
					{
						Type:       "key-user-actions-mobile",
						Identifier: "test",
						Scope:      "application name",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := client.NewMockConfigClient(gomock.NewController(t))
			if tc.mock.parentList != nil {
				c.EXPECT().List(gomock.Any(), matcher.EqAPI(tc.mock.parentList.api)).Return(tc.mock.parentList.response, tc.mock.parentList.err).Times(1)
			}
			if tc.mock.list != nil {
				c.EXPECT().List(gomock.Any(), matcher.EqAPI(tc.mock.list.api)).Return(tc.mock.list.response, tc.mock.list.err).Times(1)
			}
			if tc.mock.del != nil {
				c.EXPECT().Delete(gomock.Any(), matcher.EqAPI(tc.mock.del.api), tc.mock.del.id).Return(tc.mock.del.err).Times(1)
			}

			err := delete.Configs(t.Context(), client.ClientSet{ConfigClient: c}, tc.forDelete)
			if !tc.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDelete_Classic(t *testing.T) {
	t.Run("identification via 'name'", func(t *testing.T) {
		c := client.NewMockConfigClient(gomock.NewController(t))
		theAPI := api.NewAPIs()[api.ApplicationWeb]
		c.EXPECT().List(gomock.Any(), matcher.EqAPI(theAPI)).Return([]dtclient.Value{{Id: "DT-id-of-app", Name: "application name"}}, nil).Times(1)
		c.EXPECT().Delete(gomock.Any(), matcher.EqAPI(theAPI.ApplyParentObjectID("APP-ID")), "DT-id-of-app").Return(nil).Times(1)

		given := delete.DeleteEntries{
			"application-web": {
				{
					Type:       "application-web",
					Identifier: "application name",
				},
			},
		}

		err := delete.Configs(t.Context(), client.ClientSet{ConfigClient: c}, given)
		require.NoError(t, err)
	})

	t.Run("identification via 'objectId'", func(t *testing.T) {
		c := client.NewMockConfigClient(gomock.NewController(t))
		c.EXPECT().Delete(gomock.Any(), matcher.EqAPI(api.NewAPIs()["application-web"]), "DT-id-of-app").Return(nil).Times(1)

		given := delete.DeleteEntries{
			"application-web": {
				{
					Type:           "application-web",
					OriginObjectId: "DT-id-of-app",
				},
			},
		}

		err := delete.Configs(t.Context(), client.ClientSet{ConfigClient: c}, given)
		require.NoError(t, err)
	})

	t.Run("skip delete of DashboardShareSettings", func(t *testing.T) {
		c := client.NewMockConfigClient(gomock.NewController(t))
		given := delete.DeleteEntries{
			"dashboard-share-settings": {
				{
					Type:           "dashboard-share-settings",
					OriginObjectId: "this isn't relevant",
				},
			},
		}

		err := delete.Configs(t.Context(), client.ClientSet{ConfigClient: c}, given)
		require.NoError(t, err)
	})
}

func TestDeleteClassicKeyUserActionsWeb(t *testing.T) {
	t.Run("uniqueness", func(t *testing.T) {
		theAPI := api.NewAPIs()[api.KeyUserActionsWeb]
		c := client.NewMockConfigClient(gomock.NewController(t))

		c.EXPECT().List(gomock.Any(), matcher.EqAPI(*theAPI.Parent)).Return([]dtclient.Value{{Id: "APP-ID", Name: "application name"}}, nil).Times(1)
		c.EXPECT().List(gomock.Any(), matcher.EqAPI(theAPI.ApplyParentObjectID("APP-ID"))).Return([]dtclient.Value{
			{Id: "DT-id-of-app", Name: "test", CustomFields: map[string]any{"name": "test", "domain": "test.com", "actionType": "Load"}},
			{Id: "DT-id-of-app2", Name: "test", CustomFields: map[string]any{"name": "test", "domain": "test2.com", "actionType": "Load"}},
		}, nil).Times(1)
		c.EXPECT().Delete(gomock.Any(), matcher.EqAPI(theAPI.ApplyParentObjectID("APP-ID")), "DT-id-of-app").Return(nil).Times(1)

		de := delete.DeleteEntries{
			"key-user-actions-web": {
				{
					Type:       "key-user-actions-web",
					Identifier: "test",
					Scope:      "application name",
					ActionType: "Load",
					Domain:     "test.com",
				},
			},
		}

		err := delete.Configs(t.Context(), client.ClientSet{ConfigClient: c}, de)
		assert.NoError(t, err)
	})

	t.Run("identification via 'objectId'", func(t *testing.T) {
		theAPI := api.NewAPIs()[api.KeyUserActionsWeb]
		c := client.NewMockConfigClient(gomock.NewController(t))

		c.EXPECT().List(gomock.Any(), matcher.EqAPI(*theAPI.Parent)).Return([]dtclient.Value{{Id: "APP-ID", Name: "application name"}}, nil).Times(1)
		c.EXPECT().Delete(gomock.Any(), matcher.EqAPI(theAPI.ApplyParentObjectID("APP-ID")), "DT-id-of-app").Return(nil).Times(1)

		de := delete.DeleteEntries{
			"key-user-actions-web": {
				{
					Type:           "key-user-actions-web",
					OriginObjectId: "DT-id-of-app",
					Scope:          "application name",
					ActionType:     "Load",
					Domain:         "test.com",
				},
			},
		}

		err := delete.Configs(t.Context(), client.ClientSet{ConfigClient: c}, de)
		assert.NoError(t, err)
	})
}

func TestDelete_Documents(t *testing.T) {
	t.Run("delete via coordinate", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "document",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID := idutils.GenerateExternalID(given.AsCoordinate())
		c := client.NewMockDocumentClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Eq(fmt.Sprintf("externalId=='%s'", externalID))).
			Times(1).
			Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: "originObjectID"}}}}, nil)
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("originObjectID")).Times(1)

		entriesToDelete := delete.DeleteEntries{given.Type: {given}}
		err := delete.Configs(t.Context(), client.ClientSet{DocumentClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("config declared via coordinate doesn't exists - no error", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "document",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID := idutils.GenerateExternalID(given.AsCoordinate())
		c := client.NewMockDocumentClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Eq(fmt.Sprintf("externalId=='%s'", externalID))).
			Times(1).
			Return(documents.ListResponse{Responses: []documents.Response{}}, nil)

		entriesToDelete := delete.DeleteEntries{given.Type: {given}}
		err := delete.Configs(t.Context(), client.ClientSet{DocumentClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("config declared via coordinate have multiple match - no delete, no error", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:       "document",
			Identifier: "monaco_identifier",
			Project:    "project",
		}

		externalID := idutils.GenerateExternalID(given.AsCoordinate())
		c := client.NewMockDocumentClient(gomock.NewController(t))
		c.EXPECT().List(gomock.Any(), gomock.Eq(fmt.Sprintf("externalId=='%s'", externalID))).
			Times(1).
			Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: "originObjectID_1"}}, {Metadata: documents.Metadata{ID: "originObjectID_2"}}}}, nil)

		entriesToDelete := delete.DeleteEntries{given.Type: {given}}
		err := delete.Configs(t.Context(), client.ClientSet{DocumentClient: c}, entriesToDelete)
		assert.Error(t, err)
	})

	t.Run("delete via originID", func(t *testing.T) {
		c := client.NewMockDocumentClient(gomock.NewController(t))
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("originObjectID")).Times(1)

		entriesToDelete := delete.DeleteEntries{
			"document": {
				{
					Type:           "document",
					OriginObjectId: "originObjectID",
				},
			},
		}
		err := delete.Configs(t.Context(), client.ClientSet{DocumentClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("config declared via originID doesn't exists - no error", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "document",
			OriginObjectId: "originObjectID",
		}

		c := client.NewMockDocumentClient(gomock.NewController(t))
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("originObjectID")).Times(1).Return(libAPI.Response{}, coreapi.APIError{StatusCode: http.StatusNotFound})

		entriesToDelete := delete.DeleteEntries{given.Type: {given}}
		err := delete.Configs(t.Context(), client.ClientSet{DocumentClient: c}, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("error during delete - skip the entry", func(t *testing.T) {
		given := pointer.DeletePointer{
			Type:           "document",
			OriginObjectId: "originObjectID"}

		c := client.NewMockDocumentClient(gomock.NewController(t))
		c.EXPECT().Delete(gomock.Any(), gomock.Eq("originObjectID")).Times(1).Return(libAPI.Response{}, coreapi.APIError{StatusCode: http.StatusInternalServerError}) // the error can be any kind except 404

		entriesToDelete := delete.DeleteEntries{given.Type: {given}}
		err := delete.Configs(t.Context(), client.ClientSet{DocumentClient: c}, entriesToDelete)
		assert.Error(t, err)
	})
}

func TestDelete_Segments(t *testing.T) {
	c := client.TestSegmentsClient{}
	given := delete.DeleteEntries{
		"segment": {
			{
				Type:           "segment",
				OriginObjectId: "originObjectID",
			},
		},
	}

	err := delete.Configs(t.Context(), client.ClientSet{SegmentClient: &c}, given)
	// DummyClient returns unimplemented error on every execution of any method
	assert.Error(t, err, "unimplemented")
}

func TestDelete_SLOv2(t *testing.T) {
	c := client.TestServiceLevelObjectiveClient{}
	given := delete.DeleteEntries{
		"slo-v2": {
			{
				Type:           "slo-v2",
				OriginObjectId: "originObjectID",
			},
		},
	}

	err := delete.Configs(t.Context(), client.ClientSet{ServiceLevelObjectiveClient: &c}, given)
	// DummyClient returns unimplemented error on every execution of any method
	assert.Error(t, err, "unimplemented")
}

// TestDelete_EmptyDeleteEntriesShouldNotLog tests that no logs are produced when no configs are requested to be deleted.
func TestDelete_EmptyDeleteEntriesShouldNotLog(t *testing.T) {
	logSpy := bytes.Buffer{}
	log.PrepareLogging(t.Context(), afero.NewOsFs(), false, &logSpy, false, false)

	c := client.NewMockConfigClient(gomock.NewController(t))
	given := delete.DeleteEntries{}

	err := delete.Configs(t.Context(), client.ClientSet{ConfigClient: c}, given)
	require.NoError(t, err)
	assert.Empty(t, logSpy.String())
}
