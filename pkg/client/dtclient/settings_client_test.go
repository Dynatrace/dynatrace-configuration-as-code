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
	"context"
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpsertSettings(t *testing.T) {
	tests := []struct {
		name                        string
		expectSettingsRequestValue  string
		expectOriginObjectID        string
		serverVersion               version.Version
		expectError                 bool
		expectEntity                DynatraceEntity
		postSettingsResponseCode    int
		postSettingsResponseContent string
		getSettingsResponseCode     int
		getSettingsResponseContent  string
		listSettingsResponseCode    int
		listSettingsResponseContent string
	}{
		{
			name:                        "Invalid json returns an error",
			expectSettingsRequestValue:  "{",
			expectError:                 true,
			expectEntity:                DynatraceEntity{},
			listSettingsResponseCode:    http.StatusOK,
			listSettingsResponseContent: `{"items":[{"externalId":"","objectId":"ORIGIN_OBJECT_ID","scope":"tenant"}]}`,
		},
		{
			name: "Valid call with valid response",
			serverVersion: version.Version{
				Major: 1,
				Minor: 262,
				Patch: 0,
			},
			expectSettingsRequestValue: "{}",
			expectOriginObjectID:       "anObjectID",
			expectError:                false,
			expectEntity: DynatraceEntity{
				Id:   "entity-id",
				Name: "entity-id",
			},
			postSettingsResponseContent: `[{"objectId": "entity-id"}]`,
			listSettingsResponseCode:    http.StatusOK,
			listSettingsResponseContent: `{"items":[{"externalId":"","objectId":"ORIGIN_OBJECT_ID","scope":"tenant"}]}`,
		},
		{
			name: "Valid call with valid response - Object with external ID already exists",
			serverVersion: version.Version{
				Major: 1,
				Minor: 262,
				Patch: 0,
			},
			expectSettingsRequestValue: "{}",
			expectOriginObjectID:       "ORIGIN_OBJECT_ID",
			expectError:                false,
			expectEntity: DynatraceEntity{
				Id:   "entity-id",
				Name: "entity-id",
			},
			postSettingsResponseContent: `[{"objectId": "entity-id"}]`,
			listSettingsResponseCode:    http.StatusOK,
			listSettingsResponseContent: `{"items":[{"externalId":"monaco:YnVpbHRpbjphbGVydGluZy5wcm9maWxlJHVzZXItcHJvdmlkZWQtaWQ=","objectId":"ORIGIN_OBJECT_ID","scope":"tenant"}]}`,
		},
		{
			name:                        "Valid request, invalid response",
			expectSettingsRequestValue:  "{}",
			expectOriginObjectID:        "anObjectID",
			expectError:                 true,
			postSettingsResponseContent: `{`,
			listSettingsResponseCode:    http.StatusOK,
			listSettingsResponseContent: `{"items":[{"externalId":"","objectId":"ORIGIN_OBJECT_ID","scope":"tenant"}]}`,
		},
		{
			name:                        "Valid request, 400 return",
			expectSettingsRequestValue:  "{}",
			expectOriginObjectID:        "anObjectID",
			expectError:                 true,
			postSettingsResponseCode:    400,
			listSettingsResponseCode:    http.StatusOK,
			listSettingsResponseContent: `{"items":[{"externalId":"","objectId":"ORIGIN_OBJECT_ID","scope":"tenant"}]}`,
		},
		{
			name:                        "Valid request, but empty response",
			expectSettingsRequestValue:  "{}",
			expectOriginObjectID:        "anObjectID",
			expectError:                 true,
			postSettingsResponseContent: `[]`,
			listSettingsResponseCode:    http.StatusOK,
			listSettingsResponseContent: `{"items":[{"externalId":"","objectId":"ORIGIN_OBJECT_ID","scope":"tenant"}]}`,
		},
		{
			name:                        "Valid request, but multiple responses",
			expectSettingsRequestValue:  "{}",
			expectOriginObjectID:        "anObjectID",
			expectError:                 true,
			expectEntity:                DynatraceEntity{},
			postSettingsResponseContent: `[{"objectId": "entity-id"},{"objectId": "entity-id"}]`,
			listSettingsResponseCode:    http.StatusOK,
			listSettingsResponseContent: `{"items":[{"externalId":"","objectId":"ORIGIN_OBJECT_ID","scope":"tenant"}]}`,
		},
		{
			name:                       "Upsert existing settings 2.0 object on tenant < 1.262.0",
			expectSettingsRequestValue: "{}",
			serverVersion: version.Version{
				Major: 1,
				Minor: 260,
				Patch: 0,
			},
			expectError: false,
			expectEntity: DynatraceEntity{
				Id:   "anObjectID",
				Name: "anObjectID",
			},
			getSettingsResponseCode:     200,
			postSettingsResponseContent: `{"objectId": "entity-id"}`,
			getSettingsResponseContent:  `{"externalId": "monaco:YnVpbHRpbjphbGVydGluZy5wcm9maWxlJHVzZXItcHJvdmlkZWQtaWQ=","objectId": "anObjectID","scope": "tenant"}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/builtin:alerting.profile" {
					writer.WriteHeader(http.StatusOK)
					writer.Write([]byte("{}"))
					return
				}
				// GET settings requests
				if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/settings/objects") {
					// GET single settings obj request
					if len(strings.TrimPrefix(r.URL.Path, "/api/v2/settings/objects")) > 0 {
						writer.WriteHeader(test.getSettingsResponseCode)
						writer.Write([]byte(test.getSettingsResponseContent))
						return
					}
					// response to client
					writer.WriteHeader(test.listSettingsResponseCode)
					writer.Write([]byte(test.listSettingsResponseContent))
					return
				}

				// Build  & assert object we expect Dynatrace to receive
				var expectedSettingsObject any
				err := json.Unmarshal([]byte(test.expectSettingsRequestValue), &expectedSettingsObject)
				assert.NilError(t, err)
				extId, _ := idutils.GenerateExternalID(coordinate.Coordinate{
					Project:  "my-project",
					Type:     "builtin:alerting.profile",
					ConfigId: "user-provided-id",
				})
				expectedRequestPayload := []settingsRequest{{
					ExternalId: extId,
					Scope:      "tenant",
					Value:      expectedSettingsObject,
					SchemaId:   "builtin:alerting.profile",
					ObjectId:   test.expectOriginObjectID,
				},
				}

				var obj []settingsRequest
				err = json.NewDecoder(r.Body).Decode(&obj)
				assert.NilError(t, err)
				assert.DeepEqual(t, obj, expectedRequestPayload)

				// response to client
				if test.postSettingsResponseCode != 0 {
					http.Error(writer, test.postSettingsResponseContent, test.postSettingsResponseCode)
				} else {
					_, err := writer.Write([]byte(test.postSettingsResponseContent))
					assert.NilError(t, err)
				}
			}))

			restClient := rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy())
			c, _ := NewClassicClient(server.URL, restClient,
				WithServerVersion(test.serverVersion),
				WithRetrySettings(testRetrySettings),
				WithClientRequestLimiter(concurrency.NewLimiter(5)),
				WithExternalIDGenerator(idutils.GenerateExternalID))

			resp, err := c.UpsertSettings(context.TODO(), SettingsObject{
				OriginObjectId: "anObjectID",
				Coordinate:     coordinate.Coordinate{Project: "my-project", ConfigId: "user-provided-id", Type: "builtin:alerting.profile"},
				SchemaId:       "builtin:alerting.profile",
				Scope:          "tenant",
				Content:        []byte(test.expectSettingsRequestValue),
			})

			assert.Equal(t, err != nil, test.expectError)
			assert.DeepEqual(t, resp, test.expectEntity)
		})
	}
}
