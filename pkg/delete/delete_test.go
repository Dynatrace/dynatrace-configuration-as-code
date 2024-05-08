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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils/matcher"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	monacoREST "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var automationTypes = map[string]config.AutomationResource{
	string(config.Workflow):         config.Workflow,
	string(config.BusinessCalendar): config.BusinessCalendar,
	string(config.SchedulingRule):   config.SchedulingRule,
}

func TestDeleteSettings_LegacyExternalID(t *testing.T) {
	t.Run("TestDeleteSettings_LegacyExternalID", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
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
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		errs := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteSettings_LegacyExternalID - List settings with external ID fails", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, monacoREST.RespError{Err: fmt.Errorf("WHOPS"), StatusCode: 0})
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Error(t, err)
	})

	t.Run("TestDeleteSettings_LegacyExternalID - List settings returns no objects", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, nil)
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("TestDeleteSettings_LegacyExternalID - Delete settings based on object ID fails", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{
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
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Error(t, err)
	})
}

func TestDeleteSettings(t *testing.T) {
	t.Run("TestDeleteSettings", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Eq("builtin:alerting.profile"), gomock.Any()).DoAndReturn(func(ctx context.Context, schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
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
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("TestDeleteSettings - List settings with external ID fails", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, monacoREST.RespError{Err: fmt.Errorf("WHOPS"), StatusCode: 0})
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Error(t, err)
	})

	t.Run("TestDeleteSettings - List settings returns no objects", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{}, nil)
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("TestDeleteSettings - Delete settings based on object ID fails", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dtclient.DownloadSettingsObject{
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
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Error(t, err)
	})

	t.Run("TestDeleteSettings - Skips non-deletable Objects", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
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
		c.EXPECT().DeleteSettings(gomock.Eq("12345")).Times(0) // deletion should not be attempted for non-deletable objects
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:       "builtin:alerting.profile",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.NoError(t, err)
	})

	t.Run("identification via 'objectId'", func(t *testing.T) {
		c := client.NewMockDynatraceClient(gomock.NewController(t))
		c.EXPECT().ListSettings(gomock.Any(), gomock.Eq("builtin:alerting.profile"), gomock.Any()).DoAndReturn(func(ctx context.Context, schemaID string, listOpts dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error) {
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
		c.EXPECT().DeleteSettings(gomock.Eq("DT-original-object-ID")).Return(nil)
		entriesToDelete := delete.DeleteEntries{
			"builtin:alerting.profile": {
				{
					Type:           "builtin:alerting.profile",
					OriginObjectId: "DT-original-object-ID",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Settings: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.NoError(t, err)
	})

}

func TestDeleteAutomations(t *testing.T) {
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
		errs := delete.Configs(context.TODO(), delete.ClientSet{Automation: c}, api.NewAPIs(), automationTypes, entriesToDelete)
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
		errs := delete.Configs(context.TODO(), delete.ClientSet{Automation: c}, api.NewAPIs(), automationTypes, entriesToDelete)
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
		errs := delete.Configs(context.TODO(), delete.ClientSet{Automation: c}, api.NewAPIs(), automationTypes, entriesToDelete)
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
		err = delete.Configs(context.TODO(), delete.ClientSet{Automation: c}, api.NewAPIs(), automationTypes, entriesToDelete)
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
		errs := delete.Configs(context.TODO(), delete.ClientSet{Automation: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})
}

func TestDeleteBuckets(t *testing.T) {
	t.Run("TestDeleteBuckets", func(t *testing.T) {
		deletingBucketResponse := []byte(`{
 "bucketName": "bucket name",
 "table": "metrics",
 "displayName": "Default metrics (15 months)",
 "status": "deleting",
 "retentionDays": 462,
 "metricInterval": "PT1M",
 "version": 1
}`)

		getCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "bucket-definitions") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/project_id1"))
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				return
			}
			if req.Method == http.MethodGet && getCalls < 5 {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/project_id1"))
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				getCalls++
				return
			} else if req.Method == http.MethodGet {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"bucket": {
				{
					Type:       "bucket",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		errs := delete.Configs(context.TODO(), delete.ClientSet{Buckets: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteBuckets - No Error if object does not exist", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "bucket-definitions") {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"bucket": {
				{
					Type:       "bucket",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		errs := delete.Configs(context.TODO(), delete.ClientSet{Buckets: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

	t.Run("TestDeleteAutomations - Returns Error on HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "bucket-definitions") {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		c := buckets.NewClient(rest.NewClient(u, server.Client()))

		entriesToDelete := delete.DeleteEntries{
			"bucket": {
				{
					Type:       "bucket",
					Project:    "project",
					Identifier: "id1",
				},
			},
		}
		err := delete.Configs(context.TODO(), delete.ClientSet{Buckets: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Error(t, err, "there should be one delete error")
	})

	t.Run("identification via 'objectId'", func(t *testing.T) {
		deletingBucketResponse := []byte(`{
 "bucketName": "bucket name",
 "table": "metrics",
 "displayName": "Default metrics (15 months)",
 "status": "deleting",
 "retentionDays": 462,
 "metricInterval": "PT1M",
 "version": 1
}`)

		getCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete && strings.Contains(req.RequestURI, "bucket-definitions") {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/origin_object_ID"))
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				return
			}
			if req.Method == http.MethodGet && getCalls < 5 {
				assert.True(t, strings.HasSuffix(req.URL.Path, "/origin_object_ID"))
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(deletingBucketResponse)
				getCalls++
				return
			} else if req.Method == http.MethodGet {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			assert.Fail(t, "unexpected HTTP call")
		}))
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
		errs := delete.Configs(context.TODO(), delete.ClientSet{Buckets: c}, api.NewAPIs(), automationTypes, entriesToDelete)
		assert.Empty(t, errs, "errors should be empty")
	})

}

func TestSplitConfigsForDeletion(t *testing.T) {
	a := api.API{ID: "some-id", URLPath: "url"}
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
			apiMap := api.APIs{a.ID: a}
			entriesToDelete := delete.DeleteEntries{a.ID: tc.args.entries}

			c := client.NewMockDynatraceClient(gomock.NewController(t))
			if len(tc.args.entries) > 0 {
				c.EXPECT().ListConfigs(gomock.Any(), a).Return(tc.args.values, nil).Times(len(tc.args.entries))
			}
			for _, id := range tc.expect.ids {
				c.EXPECT().DeleteConfigById(a, id).Times(1)
			}

			err := delete.Configs(context.TODO(), delete.ClientSet{Classic: c}, apiMap, automationTypes, entriesToDelete)
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
					err: monacoREST.RespError{Err: fmt.Errorf("FAIL"), StatusCode: http.StatusBadRequest},
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
					err: monacoREST.RespError{Err: fmt.Errorf("FAIL"), StatusCode: http.StatusBadRequest},
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
					err: monacoREST.RespError{Err: fmt.Errorf("GONE ALREADY"), StatusCode: http.StatusNotFound},
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
			client := client.NewMockDynatraceClient(gomock.NewController(t))
			if tc.mock.parentList != nil {
				client.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(tc.mock.parentList.api)).Return(tc.mock.parentList.response, tc.mock.parentList.err).Times(1)
			}
			if tc.mock.list != nil {
				client.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(tc.mock.list.api)).Return(tc.mock.list.response, tc.mock.list.err).Times(1)
			}
			if tc.mock.del != nil {
				client.EXPECT().DeleteConfigById(matcher.EqAPI(tc.mock.del.api), tc.mock.del.id).Return(tc.mock.del.err).Times(1)
			}

			err := delete.Configs(context.TODO(), delete.ClientSet{Classic: client}, api.NewAPIs(), automationTypes, tc.forDelete)
			if !tc.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDeleteClassic(t *testing.T) {
	t.Run("identification via 'name'", func(t *testing.T) {
		client := client.NewMockDynatraceClient(gomock.NewController(t))
		theAPI := api.NewAPIs()[api.ApplicationWeb]
		client.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(theAPI)).Return([]dtclient.Value{{Id: "DT-id-of-app", Name: "application name"}}, nil).Times(1)
		client.EXPECT().DeleteConfigById(matcher.EqAPI(theAPI.ApplyParentObjectID("APP-ID")), "DT-id-of-app").Return(nil).Times(1)

		given := delete.DeleteEntries{
			"application-web": {
				{
					Type:       "application-web",
					Identifier: "application name",
				},
			},
		}

		err := delete.Configs(context.TODO(), delete.ClientSet{Classic: client}, api.NewAPIs(), automationTypes, given)
		require.NoError(t, err)
	})

	t.Run("identification via 'objectId'", func(t *testing.T) {
		client := client.NewMockDynatraceClient(gomock.NewController(t))
		client.EXPECT().DeleteConfigById(matcher.EqAPI(api.NewAPIs()["application-web"]), "DT-id-of-app").Return(nil).Times(1)

		given := delete.DeleteEntries{
			"application-web": {
				{
					Type:           "application-web",
					OriginObjectId: "DT-id-of-app",
				},
			},
		}

		err := delete.Configs(context.TODO(), delete.ClientSet{Classic: client}, api.NewAPIs(), automationTypes, given)
		require.NoError(t, err)
	})
}

func TestDeleteClassicKeyUserActionsWeb(t *testing.T) {
	t.Run("uniqueness", func(t *testing.T) {
		theAPI := api.NewAPIs()[api.KeyUserActionsWeb]
		client := client.NewMockDynatraceClient(gomock.NewController(t))

		client.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(*theAPI.Parent)).Return([]dtclient.Value{{Id: "APP-ID", Name: "application name"}}, nil).Times(1)
		client.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(theAPI.ApplyParentObjectID("APP-ID"))).Return([]dtclient.Value{
			{Id: "DT-id-of-app", Name: "test", CustomFields: map[string]any{"name": "test", "domain": "test.com", "actionType": "Load"}},
			{Id: "DT-id-of-app2", Name: "test", CustomFields: map[string]any{"name": "test", "domain": "test2.com", "actionType": "Load"}},
		}, nil).Times(1)
		client.EXPECT().DeleteConfigById(matcher.EqAPI(theAPI.ApplyParentObjectID("APP-ID")), "DT-id-of-app").Return(nil).Times(1)

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

		err := delete.Configs(context.TODO(), delete.ClientSet{Classic: client}, api.NewAPIs(), automationTypes, de)
		assert.NoError(t, err)
	})

	t.Run("identification via 'objectId'", func(t *testing.T) {
		theAPI := api.NewAPIs()[api.KeyUserActionsWeb]
		client := client.NewMockDynatraceClient(gomock.NewController(t))

		client.EXPECT().ListConfigs(gomock.Any(), matcher.EqAPI(*theAPI.Parent)).Return([]dtclient.Value{{Id: "APP-ID", Name: "application name"}}, nil).Times(1)
		client.EXPECT().DeleteConfigById(matcher.EqAPI(theAPI.ApplyParentObjectID("APP-ID")), "DT-id-of-app").Return(nil).Times(1)

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

		err := delete.Configs(context.TODO(), delete.ClientSet{Classic: client}, api.NewAPIs(), automationTypes, de)
		assert.NoError(t, err)
	})

}
