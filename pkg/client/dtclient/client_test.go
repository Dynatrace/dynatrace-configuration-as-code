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

package dtclient

import (
	"net/http"
	"testing"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/stretchr/testify/assert"
)

var mockAPI = api.API{ID: "mock-api", SingleConfiguration: true}
var mockAPINotSingle = api.API{ID: "mock-api", SingleConfiguration: false}

func TestNewClassicClient(t *testing.T) {
	t.Run("Client has correct urls and settings api path", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{})
		defer server.Close()

		client, err := NewClassicClient(corerest.NewClient(server.URL(), server.Client()))
		assert.NoError(t, err)
		assert.Equal(t, settingsSchemaAPIPathClassic, client.settingsSchemaAPIPath)
		assert.Equal(t, settingsObjectAPIPathClassic, client.settingsObjectAPIPath)
	})
}

func TestCreateDynatraceClientWithAutoServerVersion(t *testing.T) {
	t.Run("Server version is correctly set to determined value", func(t *testing.T) {

		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{"version" : "1.262.0.20230214-193525"}`,
						ContentType:  "application/json",
					}
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		dcl, err := NewClassicClient(corerest.NewClient(server.URL(), server.Client()), WithAutoServerVersion())

		server.Close()
		assert.NoError(t, err)
		assert.Equal(t, version.Version{Major: 1, Minor: 262}, dcl.serverVersion)
	})

	t.Run("Server version is correctly set to unknown", func(t *testing.T) {
		responses := []testutils.ResponseDef{
			{
				GET: func(t *testing.T, req *http.Request) testutils.Response {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{}`,
						ContentType:  "application/json",
					}
				},
			},
		}

		server := testutils.NewHTTPTestServer(t, responses)
		defer server.Close()

		dcl, err := NewClassicClient(corerest.NewClient(server.URL(), server.Client()), WithAutoServerVersion())
		assert.NoError(t, err)
		assert.Equal(t, version.UnknownVersion, dcl.serverVersion)
	})
}

func TestResourceContextWorksAsModificationInfo(t *testing.T) {
	tests := []struct {
		name             string
		modificationInfo SettingsModificationInfo
		resourceContext  SettingsResourceContext
	}{
		{
			name: "only read item",
			modificationInfo: SettingsModificationInfo{
				Deletable:       false,
				Modifiable:      false,
				Movable:         false,
				ModifiablePaths: []interface{}{"apiColor", "thirdPartyApi"},
			},
			resourceContext: SettingsResourceContext{
				Operations:      []string{"read"},
				Movable:         false,
				ModifiablePaths: []interface{}{"apiColor", "thirdPartyApi"},
			},
		},
		{
			name: "deletable, modifiable, movable item - no movable param provided in resource context",
			modificationInfo: SettingsModificationInfo{
				Deletable:       true,
				Modifiable:      true,
				Movable:         true,
				ModifiablePaths: []interface{}{},
			},
			resourceContext: SettingsResourceContext{
				Operations:      []string{"read", "write", "delete"},
				ModifiablePaths: []interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settingObjectWithModificationInfo := DownloadSettingsObject{
				ModificationInfo: &tt.modificationInfo,
			}
			settingObjectWithResourceContext := DownloadSettingsObject{
				ResourceContext: &tt.resourceContext,
			}
			assert.Equal(t, settingObjectWithModificationInfo.IsDeletable(), settingObjectWithResourceContext.IsDeletable())
			assert.Equal(t, settingObjectWithModificationInfo.IsMovable(), settingObjectWithResourceContext.IsMovable())
			assert.Equal(t, settingObjectWithModificationInfo.IsModifiable(), settingObjectWithResourceContext.IsModifiable())

			assert.Equal(t, settingObjectWithModificationInfo.GetModifiablePaths(), settingObjectWithResourceContext.GetModifiablePaths())
		})
	}
}
