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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

func TestNewClassicSettingsClient(t *testing.T) {
	t.Run("Client has correct URLs and settings API path", func(t *testing.T) {
		server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{})
		defer server.Close()

		client, err := NewClassicSettingsClient(corerest.NewClient(server.URL(), server.Client()))
		assert.NoError(t, err)
		assert.Equal(t, settingsSchemaAPIPathClassic, client.settingsSchemaAPIPath)
		assert.Equal(t, settingsObjectAPIPathClassic, client.settingsObjectAPIPath)
	})
}

func TestNewClassicSettingsClientWithAutoServerVersion(t *testing.T) {
	t.Run("Valid server version is parsed correctly", func(t *testing.T) {

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

		dcl, err := NewClassicSettingsClient(corerest.NewClient(server.URL(), server.Client()), WithAutoServerVersion())

		server.Close()
		assert.NoError(t, err)
		assert.Equal(t, version.Version{Major: 1, Minor: 262}, dcl.serverVersion)
	})

	t.Run("Invalid server version is parsed to unknown", func(t *testing.T) {
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

		dcl, err := NewClassicSettingsClient(corerest.NewClient(server.URL(), server.Client()), WithAutoServerVersion())
		assert.NoError(t, err)
		assert.Equal(t, version.UnknownVersion, dcl.serverVersion)
	})
}

func Test_schemaDetails(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case settingsSchemaAPIPathPlatform + "/builtin:span-attribute":
			r := []byte(`
{
    "schemaId": "builtin:span-attribute",
    "schemaConstraints": [
        {
            "type": "some another type",
            "customMessage": "Attribute keys must be unique.",
            "something": "example"
        },
        {
            "type": "UNIQUE",
            "customMessage": "Attribute keys must be unique.",
            "uniqueProperties": [
                "key0",
                "key1"
            ]
        },
        {
            "type": "UNIQUE",
            "customMessage": "Attribute keys must be unique.",
            "uniqueProperties": [
                "key2",
                "key3"
            ]
        }
    ]
}`)
			rw.WriteHeader(http.StatusOK)
			rw.Write(r)
		default:
			rw.WriteHeader(http.StatusNotFound)

		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter())

	d, err := NewPlatformSettingsClient(restClient)
	require.NoError(t, err)

	t.Run("unmarshall data", func(t *testing.T) {
		expected := Schema{SchemaId: "builtin:span-attribute", UniqueProperties: [][]string{{"key0", "key1"}, {"key2", "key3"}}}

		actual, err := d.GetSchemaById(context.TODO(), "builtin:span-attribute")

		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func Test_GetSchemaUsesCache(t *testing.T) {
	apiHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		apiHits++
		r := []byte(`{"schemaId": "builtin:span-attribute","schemaConstraints": []}`)
		rw.WriteHeader(http.StatusOK)
		rw.Write(r)

	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter())

	d, err := NewPlatformSettingsClient(restClient)
	require.NoError(t, err)

	_, err = d.GetSchemaById(context.TODO(), "builtin:span-attribute")
	assert.NoError(t, err)
	assert.Equal(t, 1, apiHits)
	_, err = d.GetSchemaById(context.TODO(), "builtin:alerting.profile")
	assert.NoError(t, err)
	assert.Equal(t, 2, apiHits)
	_, err = d.GetSchemaById(context.TODO(), "builtin:span-attribute")
	assert.NoError(t, err)
	assert.Equal(t, 2, apiHits)
}

func Test_findObjectWithSameConstraints(t *testing.T) {
	type (
		given struct {
			schema  Schema
			source  SettingsObject
			objects []DownloadSettingsObject
		}
	)

	t.Run("normal cases", func(t *testing.T) {
		tests := []struct {
			name     string
			given    given
			expected *match
		}{
			{
				name: "single constraint with boolean values- match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":true}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":true}`)},
						{Value: []byte(`{"A":false}`)},
					},
				},
				expected: &match{
					object: DownloadSettingsObject{Value: []byte(`{"A":true}`)},
					matches: constraintMatch{
						"A": true,
					},
				},
			},
			{
				name: "single constraint with int values - no match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":2}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":3}`)},
						{Value: []byte(`{"A":"x2"}`)},
					},
				},
				expected: nil,
			},
			{
				name: "single constraint - match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x"}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":"x"}`)},
						{Value: []byte(`{"A":"x1"}`)},
					},
				},
				expected: &match{
					object: DownloadSettingsObject{Value: []byte(`{"A":"x"}`)},
					matches: constraintMatch{
						"A": "x",
					},
				},
			},
			{
				name: "single constraint - no match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x"}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":"x1"}`)},
						{Value: []byte(`{"A":"x2"}`)},
					},
				},
				expected: nil,
			},
			{
				name: "single complex object constraint - match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A": {"key":"x", "val":"y"}}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A": {"key":"x", "val":"y"}}`)},
						{Value: []byte(`{"A": {"key":"x1", "val":"y"}}`)},
					},
				},
				expected: &match{
					object: DownloadSettingsObject{Value: []byte(`{"A": {"key":"x", "val":"y"}}`)},
					matches: constraintMatch{
						"A": map[string]any{
							"key": "x",
							"val": "y",
						},
					},
				},
			},
			{
				name: "single complex object constraint - no match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A": {"key":"x", "val":"y"}}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A": {"key":"x1", "val":"y"}}`)},
						{Value: []byte(`{"A": {"key":"x", "val":"y1"}}`)},
					},
				},
				expected: nil,
			},
			{
				name: "single list value constraint - match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A": [1,2,3]}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A": [1,2,3]}`)},
						{Value: []byte(`{"A": [3,2,1]}`)},
					},
				},
				expected: &match{
					object: DownloadSettingsObject{Value: []byte(`{"A": [1,2,3]}`)},
					matches: constraintMatch{
						"A": []interface{}{float64(1), float64(2), float64(3)},
					},
				},
			},
			{
				name: "single list value constraint - no match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A": [1,2,3]}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A": []}`)},
						{Value: []byte(`{"A": [3,2,1]}`)},
					},
				},
				expected: nil,
			},
			{
				name: "signe composite constraint - match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A", "B"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x", "B":"y"}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":"x", "B":"y"}`)},
						{Value: []byte(`{"A":"x", "B":"y1"}`)},
					},
				},
				expected: &match{
					object: DownloadSettingsObject{Value: []byte(`{"A":"x", "B":"y"}`)},
					matches: constraintMatch{
						"A": "x",
						"B": "y",
					},
				},
			},
			{
				name: "signe composite constraint - no match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A", "B"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x", "B":"y"}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":"x", "B":"y1"}`)},
						{Value: []byte(`{"A":"x", "B":"y2"}`)},
					},
				},
				expected: nil,
			},
			{
				name: "multiple simple constraints - one perfect match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
							{"A", "B"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x", "B":"y"}`),
					},
					objects: []DownloadSettingsObject{
						{ObjectId: "obj_1", Value: []byte(`{"A":"x", "B":"y"}`)},
						{ObjectId: "obj_2", Value: []byte(`{"A":"x2", "B":"y"}`)},
					},
				},
				expected: &match{
					object: DownloadSettingsObject{ObjectId: "obj_1", Value: []byte(`{"A":"x", "B":"y"}`)},
					matches: constraintMatch{
						"A": "x",
						"B": "y",
					},
				},
			},
			{
				name: "multiple simple constraints - one semi match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
							{"B"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x", "B":"y"}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":"x", "B":"y1"}`)},
						{Value: []byte(`{"A":"x2", "B":"y2"}`)},
					},
				},
				expected: &match{
					object: DownloadSettingsObject{Value: []byte(`{"A":"x", "B":"y1"}`)},
					matches: constraintMatch{
						"A": "x",
					},
				},
			},
			{
				name: "multiple simple constraints - no match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
							{"B"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x", "B":"y"}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":"x1", "B":"y1"}`)},
						{Value: []byte(`{"A":"x2", "B":"y2"}`)},
					},
				},
				expected: nil,
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				actual, found, err := findObjectWithSameConstraints(tc.given.schema, tc.given.source, tc.given.objects)

				fmt.Println(actual)
				assert.NoError(t, err)
				if tc.expected != nil {
					assert.True(t, found)
					assert.Equal(t, *tc.expected, actual)
				} else {
					assert.False(t, found)
				}
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name  string
			given given
		}{
			{
				name: "multiple simple constraints - multiple match",
				given: given{
					schema: Schema{
						UniqueProperties: [][]string{
							{"A"},
							{"B"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x", "B":"y"}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":"x", "B":"y1"}`)},
						{Value: []byte(`{"A":"x2", "B":"y"}`)},
					},
				},
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := findObjectWithSameConstraints(tc.given.schema, tc.given.source, tc.given.objects)
				assert.Error(t, err)
			})
		}

	})
}

func TestUpsertSettings(t *testing.T) {
	coord := coordinate.Coordinate{Project: "my-project", ConfigId: "user-provided-id", Type: "builtin:alerting.profile"}
	exId, err := idutils.GenerateExternalIDForSettingsObject(coord)
	assert.NoError(t, err)

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
			name:                       "Updating an object, where there is a conflict on the remote system works",
			serverVersion:              version.Version{Major: 1, Minor: 262, Patch: 0},
			expectSettingsRequestValue: "{}",
			expectOriginObjectID:       "", // no origin object id is important for this test
			expectError:                false,
			expectEntity: DynatraceEntity{
				Id:   "entity-id",
				Name: "entity-id",
			},
			postSettingsResponseContent: `[{"objectId": "entity-id"}]`,
			listSettingsResponseCode:    http.StatusOK,
			listSettingsResponseContent: fmt.Sprintf(`{"items":[`+
				`{"externalId":"","objectId":"anObjectID","scope":"tenant"},`+ // setting with originObjectId to be updated
				`{"externalId":"%s","objectId":"ORIGIN_OBJECT_ID","scope":"tenant"}`+ // setting with externalId to be updated
				`]}`, exId),
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
				if r.URL.Path == settingsSchemaAPIPathClassic+"/builtin:alerting.profile" {
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
				assert.NoError(t, err)
				extId, _ := idutils.GenerateExternalIDForSettingsObject(coordinate.Coordinate{
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
				assert.NoError(t, err)
				assert.Equal(t, obj, expectedRequestPayload)

				// response to client
				if test.postSettingsResponseCode != 0 {
					http.Error(writer, test.postSettingsResponseContent, test.postSettingsResponseCode)
				} else {
					_, err := writer.Write([]byte(test.postSettingsResponseContent))
					assert.NoError(t, err)
				}
			}))

			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)
			restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter(), corerest.WithConcurrentRequestLimit(5))

			c, err := NewClassicSettingsClient(restClient,
				WithServerVersion(test.serverVersion),
				WithRetrySettings(testRetrySettings),
				WithExternalIDGenerator(idutils.GenerateExternalIDForSettingsObject))
			require.NoError(t, err)

			resp, err := c.UpsertSettings(context.TODO(), SettingsObject{
				OriginObjectId: "anObjectID",
				Coordinate:     coord,
				SchemaId:       "builtin:alerting.profile",
				Scope:          "tenant",
				Content:        []byte(test.expectSettingsRequestValue),
			}, UpsertSettingsOptions{})

			assert.Equal(t, err != nil, test.expectError)
			assert.Equal(t, resp, test.expectEntity)
		})
	}
}

func TestUpsertSettingsRetries(t *testing.T) {
	numAPICalls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("{}"))
			return
		}

		numAPICalls++
		if numAPICalls < 3 {
			rw.WriteHeader(409)
			return
		}
		rw.WriteHeader(200)
		_, _ = rw.Write([]byte(`[{"objectId": "abcdefg"}]`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter(), corerest.WithConcurrentRequestLimit(5))

	client, err := NewClassicSettingsClient(restClient,
		WithRetrySettings(testRetrySettings),
		WithExternalIDGenerator(idutils.GenerateExternalIDForSettingsObject))
	require.NoError(t, err)

	_, err = client.UpsertSettings(context.TODO(), SettingsObject{
		Coordinate: coordinate.Coordinate{Type: "some:schema", ConfigId: "id"},
		SchemaId:   "some:schema",
		Content:    []byte("{}"),
	}, UpsertSettingsOptions{})

	assert.NoError(t, err)
	assert.Equal(t, numAPICalls, 3)
}

func TestUpsertSettingsFromCache(t *testing.T) {
	numAPIGetCalls := 0
	numAPIPostCalls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == settingsSchemaAPIPathClassic+"/some:schema" {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("{}"))
			return
		}
		if req.Method == http.MethodGet {
			numAPIGetCalls++
			rw.WriteHeader(200)
			rw.Write([]byte("{}"))
			return
		}

		numAPIPostCalls++
		rw.WriteHeader(200)
		rw.Write([]byte(`[{"objectId": "abcdefg"}]`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter(), corerest.WithConcurrentRequestLimit(5))

	client, err := NewClassicSettingsClient(restClient,
		WithRetrySettings(testRetrySettings),
		WithExternalIDGenerator(idutils.GenerateExternalIDForSettingsObject))
	require.NoError(t, err)

	_, err = client.UpsertSettings(context.TODO(), SettingsObject{
		Coordinate: coordinate.Coordinate{Type: "some:schema", ConfigId: "id"},
		SchemaId:   "some:schema",
		Content:    []byte("{}"),
	}, UpsertSettingsOptions{})

	assert.NoError(t, err)
	assert.Equal(t, 1, numAPIGetCalls)
	assert.Equal(t, 1, numAPIPostCalls)

	_, err = client.UpsertSettings(context.TODO(), SettingsObject{
		Coordinate: coordinate.Coordinate{Type: "some:schema", ConfigId: "id"},
		SchemaId:   "some:schema",
		Content:    []byte("{}"),
	}, UpsertSettingsOptions{})

	assert.NoError(t, err)
	assert.Equal(t, 1, numAPIGetCalls) // still one
	assert.Equal(t, 2, numAPIPostCalls)
}

func TestUpsertSettingsFromCache_CacheInvalidated(t *testing.T) {
	numGetAPICalls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == settingsSchemaAPIPathClassic+"/some:schema" {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("{}"))
			return
		}
		if req.Method == http.MethodGet {
			numGetAPICalls++
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("{}"))
			return
		}

		rw.WriteHeader(409)
		rw.Write([]byte(`{}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter(), corerest.WithConcurrentRequestLimit(5))

	client, err := NewClassicSettingsClient(restClient,
		WithRetrySettings(testRetrySettings),
		WithExternalIDGenerator(idutils.GenerateExternalIDForSettingsObject))
	require.NoError(t, err)

	client.UpsertSettings(context.TODO(), SettingsObject{
		Coordinate: coordinate.Coordinate{Type: "some:schema", ConfigId: "id"},
		SchemaId:   "some:schema",
		Content:    []byte("{}"),
	}, UpsertSettingsOptions{})
	assert.Equal(t, 1, numGetAPICalls)

	client.UpsertSettings(context.TODO(), SettingsObject{
		Coordinate: coordinate.Coordinate{Type: "some:schema", ConfigId: "id"},
		SchemaId:   "some:schema",
		Content:    []byte("{}"),
	}, UpsertSettingsOptions{})
	assert.Equal(t, 2, numGetAPICalls)

	client.UpsertSettings(context.TODO(), SettingsObject{
		Coordinate: coordinate.Coordinate{Type: "some:schema", ConfigId: "id"},
		SchemaId:   "some:schema",
		Content:    []byte("{}"),
	}, UpsertSettingsOptions{})
	assert.Equal(t, 3, numGetAPICalls)

}

func TestUpsertSettingsConsidersUniqueKeyConstraints(t *testing.T) {

	type given struct {
		schemaDetailsResponse schemaDetailsResponse
		listSettingsResponse  []DownloadSettingsObject
		settingsObject        SettingsObject
	}
	type want struct {
		error               bool
		postSettingsRequest settingsRequest
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			"Creates new object if none exists",
			given{
				schemaDetailsResponse: schemaDetailsResponse{
					SchemaId: "builtin:alerting.profile",
					SchemaConstraints: []schemaConstraint{
						{
							Type:             "UNIQUE",
							UniqueProperties: []string{"key_2"},
						},
					},
				},
				listSettingsResponse: []DownloadSettingsObject{},
				settingsObject: SettingsObject{
					Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:alerting.profile", ConfigId: "id"},
					SchemaId:   "builtin:alerting.profile",
					Content:    []byte(`{ "key_1": "a", "key_2": 42 }`),
				},
			},
			want{
				error: false,
				postSettingsRequest: settingsRequest{
					SchemaId:   "builtin:alerting.profile",
					ExternalId: "monaco:cCRidWlsdGluOmFsZXJ0aW5nLnByb2ZpbGUkaWQ=",
					Value: map[string]interface{}{
						"key_1": "a",
						"key_2": float64(42),
					},
				},
			},
		},
		{
			"Creates new object if no matching unique key is found",
			given{
				schemaDetailsResponse: schemaDetailsResponse{
					SchemaId: "builtin:alerting.profile",
					SchemaConstraints: []schemaConstraint{
						{
							Type:             "UNIQUE",
							UniqueProperties: []string{"key_1"},
						},
					},
				},
				listSettingsResponse: []DownloadSettingsObject{
					{
						ExternalId: "externalID--1",
						SchemaId:   "builtin:alerting.profile",
						ObjectId:   "objectID--1",
						Value:      []byte(`{ "key_1": "NOT A MATCH", "key_2": "dont-care" }`),
					},
					{
						ExternalId: "externalID--2",
						SchemaId:   "builtin:alerting.profile",
						ObjectId:   "objectID--2",
						Value:      []byte(`{ "key_1": "NOT A MATCH EITHER", "key_2": "dont-care" }`),
					},
				},
				settingsObject: SettingsObject{
					Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:alerting.profile", ConfigId: "id"},
					SchemaId:   "builtin:alerting.profile",
					Content:    []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
				},
			},
			want{
				error: false,
				postSettingsRequest: settingsRequest{
					SchemaId:   "builtin:alerting.profile",
					ExternalId: "monaco:cCRidWlsdGluOmFsZXJ0aW5nLnByb2ZpbGUkaWQ=",
					Value: map[string]interface{}{
						"key_1": "MATCH",
						"key_2": "dont-care",
					},
				},
			},
		},
		{
			"Updates object if matching unique key is found",
			given{
				schemaDetailsResponse: schemaDetailsResponse{
					SchemaId: "builtin:alerting.profile",
					SchemaConstraints: []schemaConstraint{
						{
							Type:             "UNIQUE",
							UniqueProperties: []string{"key_1"},
						},
					},
				},
				listSettingsResponse: []DownloadSettingsObject{
					{
						ExternalId: "externalID--1",
						SchemaId:   "builtin:alerting.profile",
						ObjectId:   "objectID--1",
						Value:      []byte(`{ "key_1": "NOT A MATCH", "key_2": "dont-care" }`),
					},
					{
						ExternalId: "externalID--2",
						SchemaId:   "builtin:alerting.profile",
						ObjectId:   "objectID--2",
						Value:      []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
					},
				},
				settingsObject: SettingsObject{
					Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:alerting.profile", ConfigId: "id"},
					SchemaId:   "builtin:alerting.profile",
					Content:    []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
				},
			},
			want{
				error: false,
				postSettingsRequest: settingsRequest{
					SchemaId:   "builtin:alerting.profile",
					ObjectId:   "objectID--2", // object ID of matching object
					ExternalId: "monaco:cCRidWlsdGluOmFsZXJ0aW5nLnByb2ZpbGUkaWQ=",
					Value: map[string]interface{}{
						"key_1": "MATCH",
						"key_2": "dont-care",
					},
				},
			},
		},
		{
			"Updates object if matching unique key is found - complex key object",
			given{
				schemaDetailsResponse: schemaDetailsResponse{
					SchemaId: "builtin:alerting.profile",
					SchemaConstraints: []schemaConstraint{
						{
							Type:             "UNIQUE",
							UniqueProperties: []string{"key_1"},
						},
					},
				},
				listSettingsResponse: []DownloadSettingsObject{
					{
						ExternalId: "externalID--1",
						SchemaId:   "builtin:alerting.profile",
						ObjectId:   "objectID--1",
						Value:      []byte(`{ "key_1": { "a": [false,true,false], "b": 21.0, "c": { "cK": "cV" } }, "key_2": "dont-care" }`),
					},
					{
						ExternalId: "externalID--2",
						SchemaId:   "builtin:alerting.profile",
						ObjectId:   "objectID--2",
						Value:      []byte(`{ "key_1": { "a": [false,true,false], "b": 42.0, "c": { "cK": "cV" } }, "key_2": "dont-care" }`),
					},
				},
				settingsObject: SettingsObject{
					Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:alerting.profile", ConfigId: "id"},
					SchemaId:   "builtin:alerting.profile",
					Content:    []byte(`{ "key_1": { "a": [false,true,false], "b": 42.0, "c": { "cK": "cV" } }, "key_2": "new value" }`),
				},
			},
			want{
				error: false,
				postSettingsRequest: settingsRequest{
					SchemaId:   "builtin:alerting.profile",
					ObjectId:   "objectID--2", // object ID of matching object
					ExternalId: "monaco:cCRidWlsdGluOmFsZXJ0aW5nLnByb2ZpbGUkaWQ=",
					Value: map[string]interface{}{
						"key_1": map[string]interface{}{
							"a": []interface{}{false, true, false},
							"b": 42.0,
							"c": map[string]interface{}{
								"cK": "cV",
							},
						},
						"key_2": "new value",
					},
				},
			},
		},
		{
			"Returns error if several matching objects are found",
			given{
				schemaDetailsResponse: schemaDetailsResponse{
					SchemaId: "builtin:alerting.profile",
					SchemaConstraints: []schemaConstraint{
						{
							Type:             "UNIQUE",
							UniqueProperties: []string{"key_1"},
						},
					},
				},
				listSettingsResponse: []DownloadSettingsObject{
					{
						ExternalId: "externalID--1",
						SchemaId:   "builtin:alerting.profile",
						ObjectId:   "objectID--1",
						Value:      []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
					},
					{
						ExternalId: "externalID--2",
						SchemaId:   "builtin:alerting.profile",
						ObjectId:   "objectID--2",
						Value:      []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
					},
				},
				settingsObject: SettingsObject{
					Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:alerting.profile", ConfigId: "id"},
					SchemaId:   "builtin:alerting.profile",
					Content:    []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
				},
			},
			want{
				error:               true,
				postSettingsRequest: settingsRequest{},
			},
		},
		{
			"Considers Scope when looking for matching objects",
			given{
				schemaDetailsResponse: schemaDetailsResponse{
					SchemaId: "builtin:alerting.profile",
					SchemaConstraints: []schemaConstraint{
						{
							Type:             "UNIQUE",
							UniqueProperties: []string{"key_1"},
						},
					},
				},
				listSettingsResponse: []DownloadSettingsObject{
					{
						ExternalId: "externalID--1",
						SchemaId:   "builtin:alerting.profile",
						Scope:      "HOST-1", // same scope, but no match
						ObjectId:   "objectID--1",
						Value:      []byte(`{ "key_1": "NOT A MATCH", "key_2": "dont-care" }`),
					},
					{
						ExternalId: "externalID--2",
						SchemaId:   "builtin:alerting.profile",
						Scope:      "HOST-1", // match in same scope
						ObjectId:   "objectID--2",
						Value:      []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
					},
					{
						ExternalId: "externalID--3",
						SchemaId:   "builtin:alerting.profile",
						Scope:      "HOST-2", // match but in different scope
						ObjectId:   "objectID--3",
						Value:      []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
					},
				},
				settingsObject: SettingsObject{
					Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:alerting.profile", ConfigId: "id"},
					SchemaId:   "builtin:alerting.profile",
					Scope:      "HOST-1",
					Content:    []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
				},
			},
			want{
				error: false,
				postSettingsRequest: settingsRequest{
					SchemaId:   "builtin:alerting.profile",
					Scope:      "HOST-1",
					ObjectId:   "objectID--2", // object ID of matching object
					ExternalId: "monaco:cCRidWlsdGluOmFsZXJ0aW5nLnByb2ZpbGUkaWQ=",
					Value: map[string]interface{}{
						"key_1": "MATCH",
						"key_2": "dont-care",
					},
				},
			},
		},

		{
			"Matching keys in different scopes do not produce a match - new object is created",
			given{
				schemaDetailsResponse: schemaDetailsResponse{
					SchemaId: "builtin:alerting.profile",
					SchemaConstraints: []schemaConstraint{
						{
							Type:             "UNIQUE",
							UniqueProperties: []string{"key_1"},
						},
					},
				},
				listSettingsResponse: []DownloadSettingsObject{
					{
						ExternalId: "externalID--2",
						SchemaId:   "builtin:alerting.profile",
						Scope:      "HOST-2",
						ObjectId:   "objectID--2",
						Value:      []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
					},
					{
						ExternalId: "externalID--3",
						SchemaId:   "builtin:alerting.profile",
						Scope:      "HOST-3",
						ObjectId:   "objectID--3",
						Value:      []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
					},
					{
						ExternalId: "externalID--4",
						SchemaId:   "builtin:alerting.profile",
						Scope:      "HOST-4",
						ObjectId:   "objectID--4",
						Value:      []byte(`{ "key_1": "MATCH", "key_2": "dont-care" }`),
					},
				},
				settingsObject: SettingsObject{
					Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:alerting.profile", ConfigId: "id"},
					SchemaId:   "builtin:alerting.profile",
					Scope:      "HOST-1",
					Content:    []byte(`{ "key_1": "a", "key_2": 42 }`),
				},
			},
			want{
				error: false,
				postSettingsRequest: settingsRequest{
					SchemaId:   "builtin:alerting.profile",
					Scope:      "HOST-1",
					ExternalId: "monaco:cCRidWlsdGluOmFsZXJ0aW5nLnByb2ZpbGUkaWQ=",
					Value: map[string]interface{}{
						"key_1": "a",
						"key_2": float64(42),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {

				// GET schema details
				if r.URL.Path == settingsSchemaAPIPathClassic+"/builtin:alerting.profile" {
					writer.WriteHeader(http.StatusOK)
					b, err := json.Marshal(tt.given.schemaDetailsResponse)
					assert.NoError(t, err)
					_, _ = writer.Write(b)
					return
				}

				// GET settings objects
				if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, settingsObjectAPIPathClassic) {
					// response to client
					writer.WriteHeader(http.StatusOK)
					l := struct {
						Items []DownloadSettingsObject `json:"items"`
					}{
						tt.given.listSettingsResponse,
					}
					b, err := json.Marshal(l)
					assert.NoError(t, err)
					_, _ = writer.Write(b)
					return
				}

				// ASSERT expected object creation POST request
				assert.Equal(t, http.MethodPost, r.Method)
				var obj []settingsRequest
				err := json.NewDecoder(r.Body).Decode(&obj)
				assert.NoError(t, err)
				assert.Len(t, obj, 1)
				assert.Equal(t, tt.want.postSettingsRequest, obj[0])

				writer.WriteHeader(200)
				writer.Write([]byte(`[ { "objectId": "abcsd423==" } ]`))
			}))
			defer server.Close()

			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter(), corerest.WithConcurrentRequestLimit(5))

			c, err := NewClassicSettingsClient(restClient,
				WithRetrySettings(testRetrySettings),
				WithExternalIDGenerator(idutils.GenerateExternalIDForSettingsObject))
			require.NoError(t, err)

			_, err = c.UpsertSettings(context.TODO(), tt.given.settingsObject, UpsertSettingsOptions{})
			if tt.want.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListKnownSettings(t *testing.T) {

	tests := []struct {
		name                      string
		givenSchemaID             string
		givenListSettingsOpts     ListSettingsOptions
		givenServerResponses      []testServerResponse
		want                      []DownloadSettingsObject
		wantQueryParamsPerAPICall [][]testQueryParams
		wantNumberOfAPICalls      int
		wantError                 bool
	}{
		{
			name:          "Lists Settings objects as expected",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:                  "Lists Settings objects without value field as expected",
			givenSchemaID:         "builtin:something",
			givenListSettingsOpts: ListSettingsOptions{DiscardValue: true},
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", reducedListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:          "Lists Settings objects with filter as expected",
			givenSchemaID: "builtin:something",
			givenListSettingsOpts: ListSettingsOptions{Filter: func(o DownloadSettingsObject) bool {
				return o.ExternalId == "RG9jdG9yIFdobwo="
			}},
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ] }`},
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4641", "externalId": "RG9jdG9yIabcdef="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:          "Handles Pagination when listing settings objects",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{200, `{ "items": [ {"objectId": "b1d4c623-25e0-4b54-9eb5-6734f1a72041", "externalId": "VGhlIE1hc3Rlcgo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
				{
					ExternalId: "VGhlIE1hc3Rlcgo=",
					ObjectId:   "b1d4c623-25e0-4b54-9eb5-6734f1a72041",
				},
			},

			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfAPICalls: 2,
			wantError:            false,
		},
		{
			name:          "Returns empty if list if no items exist",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ ] }`},
			},
			want: []DownloadSettingsObject{},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:          "Returns error if HTTP error is encountered - 400",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{400, `epic fail`},
			},
			want: nil,
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            true,
		},
		{
			name:          "Returns error if HTTP error is encountered - 403",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{403, `epic fail`},
			},
			want: nil,
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            true,
		},
		{
			name:          "Retries on HTTP error on paginated request and returns eventual success",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{400, `get next page fail`},
				{400, `retry fail`},
				{200, `{ "items": [ {"objectId": "b1d4c623-25e0-4b54-9eb5-6734f1a72041", "externalId": "VGhlIE1hc3Rlcgo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
				{
					ExternalId: "VGhlIE1hc3Rlcgo=",
					ObjectId:   "b1d4c623-25e0-4b54-9eb5-6734f1a72041",
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfAPICalls: 4,
			wantError:            false,
		},
		{
			name:          "Returns error if HTTP error is encountered getting further paginated responses",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{400, `get next page fail`},
				{400, `retry fail 1`},
				{400, `retry fail 2`},
				{400, `retry fail 3`},
			},
			want: nil,
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfAPICalls: 5,
			wantError:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.wantQueryParamsPerAPICall) > 0 {
					params := tt.wantQueryParamsPerAPICall[apiCalls]
					for _, param := range params {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.NotNil(t, addedQueryParameter)
						assert.NotEmpty(t, addedQueryParameter)
						assert.Equal(t, addedQueryParameter[0], param.value)
					}
				} else {
					assert.Equal(t, "", req.URL.RawQuery, "expected no query params - but '%s' was sent", req.URL.RawQuery)
				}

				resp := tt.givenServerResponses[apiCalls]
				if resp.statusCode != 200 {
					http.Error(rw, resp.body, resp.statusCode)
				} else {
					_, _ = rw.Write([]byte(resp.body))
				}

				apiCalls++
				assert.LessOrEqualf(t, apiCalls, tt.wantNumberOfAPICalls, "expected at most %d API calls to happen, but encountered call %d", tt.wantNumberOfAPICalls, apiCalls)
			}))
			defer server.Close()

			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter(), corerest.WithConcurrentRequestLimit(5))

			client, err := NewClassicSettingsClient(restClient,
				WithRetrySettings(testRetrySettings),
				WithExternalIDGenerator(idutils.GenerateExternalIDForSettingsObject))
			require.NoError(t, err)

			res, err1 := client.List(context.TODO(), tt.givenSchemaID, tt.givenListSettingsOpts)

			if tt.wantError {
				assert.Error(t, err1)
			} else {
				assert.NoError(t, err1)
			}

			assert.Equal(t, tt.want, res)

			assert.Equal(t, apiCalls, tt.wantNumberOfAPICalls, "expected exactly %d API calls to happen but %d calls where made", tt.wantNumberOfAPICalls, apiCalls)
		})
	}
}

func TestSettingsClientGet(t *testing.T) {
	type fields struct {
		environmentURL string
		retrySettings  RetrySettings
	}
	type args struct {
		objectID string
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		givenTestServerResp *testServerResponse
		wantURLPath         string
		wantResult          *DownloadSettingsObject
		wantErr             bool
	}{
		{
			name:   "Get Setting by ID - server response != 2xx",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 500,
				body:       "{}",
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantResult:  nil,
			wantErr:     true,
		},
		{
			name:   "Get Setting by ID - invalid server response",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 200,
				body:       `{bs}`,
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantResult:  nil,
			wantErr:     true,
		},
		{
			name:   "Get Setting by ID",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 200,
				body:       `{"objectId":"12345","externalId":"54321", "schemaVersion":"1.0","schemaId":"builtin:bla","scope":"tenant"}`,
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantResult: &DownloadSettingsObject{
				ExternalId:    "54321",
				SchemaVersion: "1.0",
				SchemaId:      "builtin:bla",
				ObjectId:      "12345",
				Scope:         "tenant",
				Value:         nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				assert.Equal(t, tt.wantURLPath, req.URL.Path)
				if resp := tt.givenTestServerResp; resp != nil {
					if resp.statusCode != 200 {
						http.Error(rw, resp.body, resp.statusCode)
					} else {
						_, _ = rw.Write([]byte(resp.body))
					}
				}

			}))
			defer server.Close()

			var serverURL *url.URL
			var err error
			if tt.fields.environmentURL != "" {
				serverURL, err = url.Parse(tt.fields.environmentURL)
				require.NoError(t, err)
			} else {
				serverURL, err = url.Parse(server.URL)
				require.NoError(t, err)
			}

			restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter(), corerest.WithConcurrentRequestLimit(5))

			client, err := NewClassicSettingsClient(restClient,
				WithRetrySettings(tt.fields.retrySettings),
				WithExternalIDGenerator(idutils.GenerateExternalIDForSettingsObject))
			require.NoError(t, err)

			settingsObj, err := client.Get(context.TODO(), tt.args.objectID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantResult, settingsObj)

		})
	}

}

func TestDeleteSettings(t *testing.T) {
	type fields struct {
		environmentURL string
		retrySettings  RetrySettings
	}
	type args struct {
		objectID string
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		givenTestServerResp *testServerResponse
		wantURLPath         string
		wantErr             bool
	}{
		{
			name:   "Delete Settings - server response != 2xx",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 500,
				body:       "{}",
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantErr:     true,
		},
		{
			name:   "Delete Settings - server response 404 does not result in an err",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 404,
				body:       "{}",
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantErr:     false,
		},
		{
			name:   "Delete Settings - object ID is passed",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				assert.Equal(t, tt.wantURLPath, req.URL.Path)
				if resp := tt.givenTestServerResp; resp != nil {
					if resp.statusCode != 200 {
						http.Error(rw, resp.body, resp.statusCode)
					} else {
						_, _ = rw.Write([]byte(resp.body))
					}
				}

			}))
			defer server.Close()

			var serverURL *url.URL
			var err error
			if tt.fields.environmentURL != "" {
				serverURL, err = url.Parse(tt.fields.environmentURL)
				require.NoError(t, err)
			} else {
				serverURL, err = url.Parse(server.URL)
				require.NoError(t, err)
			}

			restClient := corerest.NewClient(serverURL, server.Client(), corerest.WithRateLimiter(), corerest.WithConcurrentRequestLimit(5))

			client, err := NewClassicSettingsClient(restClient,
				WithRetrySettings(tt.fields.retrySettings),
				WithExternalIDGenerator(idutils.GenerateExternalIDForSettingsObject))
			require.NoError(t, err)

			if err := client.DeleteSettings(context.TODO(), tt.args.objectID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteSettings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
