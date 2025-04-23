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

package templatetools_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/templatetools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

func TestNew(t *testing.T) {

	tests := []struct {
		name     string
		given    []byte
		expected templatetools.JSONObject
		wantErr  bool
	}{
		{
			name:     "simple case",
			given:    []byte(`{ "key1" : "value1", "key2" : "value2" }`),
			expected: templatetools.JSONObject{"key1": "value1", "key2": "value2"},
			wantErr:  false,
		}, {
			name:     "nil as argument returns an error",
			given:    nil,
			expected: nil,
			wantErr:  true,
		}, {
			name:     "empty JSON",
			given:    []byte(`{}`),
			expected: templatetools.JSONObject{},
			wantErr:  false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := templatetools.NewJSONObject(tc.given)

			assert.Equal(t, tc.expected, actual)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJSONObject_Parameterize(t *testing.T) {
	type (
		given struct {
			jsonObject templatetools.JSONObject
			key        string
		}
		expected struct {
			parameter *value.ValueParameter
			newValue  any
		}
		actual struct {
			parameter *value.ValueParameter
			value     any
		}
	)

	tests := []struct {
		name string
		given
		expected
	}{
		{
			name: "simple case",
			given: given{
				jsonObject: templatetools.JSONObject{"key1": "value1", "key2": "value2"},
				key:        "key1",
			},
			expected: expected{
				parameter: &value.ValueParameter{Value: "value1"},
				newValue:  "{{.key1}}",
			},
		}, {
			name: "an non-existent key",
			given: given{
				jsonObject: templatetools.JSONObject{"key1": "value1", "key2": 2},
				key:        "non-existent",
			},
			expected: expected{
				parameter: nil,
				newValue:  nil,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := actual{
				parameter: tc.given.jsonObject.Parameterize(tc.given.key),
				value:     tc.given.jsonObject.Get(tc.given.key),
			}

			assert.Equal(t, tc.expected.parameter, actual.parameter)
			assert.Equal(t, tc.expected.newValue, actual.value)
		})
	}
}

func TestJSONObject_ParameterizeAttribute(t *testing.T) {
	type (
		given struct {
			jsonObject         templatetools.JSONObject
			keyOfJSONAttribute string
			parameterName      string
		}
		expected struct {
			parameter *value.ValueParameter
			newValue  any
		}
		actual struct {
			parameter *value.ValueParameter
			value     any
		}
	)

	tests := []struct {
		name string
		given
		expected
	}{
		{
			name: "simple case - string",
			given: given{
				jsonObject:         templatetools.JSONObject{"key1": "value1", "key2": "value2"},
				keyOfJSONAttribute: "key1",
				parameterName:      "param1",
			},
			expected: expected{
				parameter: &value.ValueParameter{Value: "value1"},
				newValue:  "{{.param1}}",
			},
		}, {
			name: "simple case - integer",
			given: given{
				jsonObject:         templatetools.JSONObject{"key1": "value1", "key2": 2},
				keyOfJSONAttribute: "key2",
				parameterName:      "param2",
			},
			expected: expected{
				parameter: &value.ValueParameter{Value: 2},
				newValue:  "{{.param2}}",
			},
		}, {
			name: "an non-existent key",
			given: given{
				jsonObject:         templatetools.JSONObject{"key1": "value1", "key2": 2},
				keyOfJSONAttribute: "non-existent",
				parameterName:      "parameter",
			},
			expected: expected{
				parameter: nil,
				newValue:  nil,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := actual{
				parameter: tc.given.jsonObject.ParameterizeAttributeWith(tc.given.keyOfJSONAttribute, tc.given.parameterName),
				value:     tc.given.jsonObject.Get(tc.given.keyOfJSONAttribute),
			}

			assert.Equal(t, tc.expected.parameter, actual.parameter)
			assert.Equal(t, tc.expected.newValue, actual.value)
		})
	}
}

func TestJSONObject_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		given   templatetools.JSONObject
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "empty",
			given:   templatetools.JSONObject{},
			want:    []byte("{}"),
			wantErr: nil,
		},
		{
			name:    "simple case",
			given:   templatetools.JSONObject{"key1": "value1", "key2": 2},
			want:    []byte(`{"key1":"value1","key2":2}`),
			wantErr: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, _ := tc.given.ToJSON(false)

			assert.Equal(t, string(tc.want), string(actual))
		})
	}
}

func TestJSONObject_Delete(t *testing.T) {
	t.Run("delete single entry", func(t *testing.T) {
		given := json.RawMessage(`{"key1": "value1","key2": "value2","key3": "value3"}`)

		json, err := templatetools.NewJSONObject(given)
		require.NoError(t, err)

		json.Delete("key1")

		assert.Empty(t, json.Get("key1"))
		assert.Equal(t, "value2", json.Get("key2"))
		assert.Equal(t, "value3", json.Get("key3"))
	})

	t.Run("delete multiple entries", func(t *testing.T) {
		given := json.RawMessage(`{"key1": "value1","key2": "value2","key3": "value3"}`)

		json, err := templatetools.NewJSONObject(given)
		require.NoError(t, err)

		json.Delete("key1", "key2")

		assert.Empty(t, json.Get("key1"))
		assert.Empty(t, json.Get("key2"))
		assert.Equal(t, "value3", json.Get("key3"))
	})

	t.Run("try to delete unknown entry", func(t *testing.T) {
		given := json.RawMessage(`{"key1": "value1","key2": "value2","key3": "value3"}`)

		json, err := templatetools.NewJSONObject(given)
		require.NoError(t, err)

		json.Delete("unknown")

		assert.Equal(t, "value1", json.Get("key1"))
		assert.Equal(t, "value2", json.Get("key2"))
		assert.Equal(t, "value3", json.Get("key3"))
	})

	t.Run("try to delete nil", func(t *testing.T) {
		given := json.RawMessage(`{"key1": "value1","key2": "value2","key3": "value3"}`)

		json, err := templatetools.NewJSONObject(given)
		require.NoError(t, err)

		json.Delete(nil...)

		assert.Equal(t, "value1", json.Get("key1"))
		assert.Equal(t, "value2", json.Get("key2"))
		assert.Equal(t, "value3", json.Get("key3"))
	})
}
