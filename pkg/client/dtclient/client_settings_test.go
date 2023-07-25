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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	restCLient := rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy())

	d, _ := NewPlatformClient(server.URL, server.URL, restCLient, restCLient)

	t.Run("unmarshall data", func(t *testing.T) {
		expected := SchemaConstraints{SchemaId: "builtin:span-attribute", UniqueProperties: [][]string{{"key0", "key1"}, {"key2", "key3"}}}

		actual, err := d.fetchSchemasConstraints(context.TODO(), "builtin:span-attribute")

		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func Test_FetchSchemaConstraintsUsesCache(t *testing.T) {
	apiHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		apiHits++
		r := []byte(`{"schemaId": "builtin:span-attribute","schemaConstraints": []}`)
		rw.WriteHeader(http.StatusOK)
		rw.Write(r)

	}))
	defer server.Close()

	restClient := rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy())
	d, _ := NewPlatformClient(server.URL, server.URL, restClient, restClient)

	_, err := d.fetchSchemasConstraints(context.TODO(), "builtin:span-attribute")
	assert.NoError(t, err)
	assert.Equal(t, 1, apiHits)
	_, err = d.fetchSchemasConstraints(context.TODO(), "builtin:alerting.profile")
	assert.NoError(t, err)
	assert.Equal(t, 2, apiHits)
	_, err = d.fetchSchemasConstraints(context.TODO(), "builtin:span-attribute")
	assert.NoError(t, err)
	assert.Equal(t, 2, apiHits)
}

func Test_findObjectWithSameConstraints(t *testing.T) {
	type (
		given struct {
			schema  SchemaConstraints
			source  SettingsObject
			objects []DownloadSettingsObject
		}
	)

	t.Run("normal cases", func(t *testing.T) {
		tests := []struct {
			name     string
			given    given
			expected *DownloadSettingsObject
		}{
			{
				name: "single constraint - match",
				given: given{
					schema: SchemaConstraints{
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
				expected: &DownloadSettingsObject{Value: []byte(`{"A":"x"}`)},
			},
			{
				name: "single constraint - no match",
				given: given{
					schema: SchemaConstraints{
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
				name: "signe composite constraint - match",
				given: given{
					schema: SchemaConstraints{
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
				expected: &DownloadSettingsObject{Value: []byte(`{"A":"x", "B":"y"}`)},
			},
			{
				name: "signe composite constraint - no match",
				given: given{
					schema: SchemaConstraints{
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
					schema: SchemaConstraints{
						UniqueProperties: [][]string{
							{"A"},
							{"B"},
						},
					},
					source: SettingsObject{
						SchemaId: "schemaID", Content: []byte(`{"A":"x", "B":"y"}`),
					},
					objects: []DownloadSettingsObject{
						{Value: []byte(`{"A":"x", "B":"y"}`)},
						{Value: []byte(`{"A":"x2", "B":"y"}`)},
					},
				},
				expected: &DownloadSettingsObject{Value: []byte(`{"A":"x", "B":"y"}`)},
			},
			{
				name: "multiple simple constraints - one semi match",
				given: given{
					schema: SchemaConstraints{
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
				expected: &DownloadSettingsObject{Value: []byte(`{"A":"x", "B":"y1"}`)},
			},
			{
				name: "multiple simple constraints - no match",
				given: given{
					schema: SchemaConstraints{
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
				actual, err := findObjectWithSameConstraints(tc.given.schema, tc.given.source, tc.given.objects)

				fmt.Println(actual)
				assert.NoError(t, err)
				if tc.expected != nil {
					assert.NotNil(t, actual)
					assert.Equal(t, tc.expected, actual)
				} else {
					assert.Nil(t, actual)
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
					schema: SchemaConstraints{
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
				actual, err := findObjectWithSameConstraints(tc.given.schema, tc.given.source, tc.given.objects)

				assert.Nil(t, actual)
				assert.Error(t, err)
			})
		}

	})
}
