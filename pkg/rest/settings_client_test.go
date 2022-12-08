// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"encoding/json"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpsert(t *testing.T) {

	tests := []struct {
		name            string
		content         string
		expectError     bool
		expectEntity    api.DynatraceEntity
		responseCode    int
		responseContent string
	}{
		{
			name:         "Invalid json returns an error",
			content:      "{",
			expectError:  true,
			expectEntity: api.DynatraceEntity{},
		},
		{
			name:        "Simple valid call with valid response",
			content:     "{}",
			expectError: false,
			expectEntity: api.DynatraceEntity{
				Id:   "entity-id",
				Name: "entity-id",
			},
			responseContent: `[{"objectId": "entity-id"}]`,
		},
		{
			name:        "Simple valid call with valid response",
			content:     "{}",
			expectError: false,
			expectEntity: api.DynatraceEntity{
				Id:   "entity-id",
				Name: "entity-id",
			},
			responseContent: `[{"objectId": "entity-id"}]`,
		},
		{
			name:            "Valid request, invalid response",
			content:         "{}",
			expectError:     true,
			expectEntity:    api.DynatraceEntity{},
			responseContent: `{`,
		},
		{
			name:         "Valid request, 400 return",
			content:      "{}",
			expectError:  true,
			expectEntity: api.DynatraceEntity{},
			responseCode: 400,
		},
		{
			name:            "Valid request, but empty response",
			content:         "{}",
			expectError:     true,
			expectEntity:    api.DynatraceEntity{},
			responseContent: `[]`,
		},
		{
			name:            "Valid request, but multiple responses",
			content:         "{}",
			expectError:     true,
			expectEntity:    api.DynatraceEntity{},
			responseContent: `[{"objectId": "entity-id"},{"objectId": "entity-id"}]`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {

				// Build  & assert object we expect Dynatrace to receive
				var expectedSettingsObject any
				err := json.Unmarshal([]byte(test.content), &expectedSettingsObject)
				assert.NilError(t, err)

				expectedRequestPayload := settingsRequest{
					ExternalId:    util.GenerateExternalId("builtin:alerting.profile", "user-provided-id"),
					Scope:         "tenant",
					Value:         expectedSettingsObject,
					SchemaId:      "builtin:alerting.profile",
					SchemaVersion: "",
				}

				var obj []settingsRequest
				err = json.NewDecoder(r.Body).Decode(&obj)
				assert.NilError(t, err)

				assert.DeepEqual(t, obj, []settingsRequest{expectedRequestPayload})

				// response to client
				if test.responseCode != 0 {
					http.Error(writer, test.responseContent, test.responseCode)
				} else {
					_, err := writer.Write([]byte(test.responseContent))
					assert.NilError(t, err)
				}
			}))

			c, err := newDynatraceClient(server.URL, "token", server.Client(), defaultRetrySettings)
			assert.NilError(t, err)

			resp, err := c.UpsertSettings(nil, SettingsObject{
				Id:       "user-provided-id",
				SchemaID: "builtin:alerting.profile",
				Scope:    "tenant",
				Content:  []byte(test.content),
			})

			assert.Equal(t, err != nil, test.expectError)
			assert.DeepEqual(t, resp, test.expectEntity)
		})
	}
}
