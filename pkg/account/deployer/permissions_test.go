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

package deployer

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSchema(t *testing.T) {
	t.Run("Successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{}"))
		}))
		defer server.Close()

		client := server.Client()

		schema, err := getSchema(context.TODO(), client, server.URL)
		require.NoError(t, err)
		require.Equal(t, []byte("{}"), schema)
	})

	t.Run("Error on client.Get", func(t *testing.T) {
		client := &http.Client{}
		schema, err := getSchema(context.TODO(), client, "invalid-url")
		require.Error(t, err)
		require.Empty(t, schema)
	})
}

func TestGetSupportedPermissionIds(t *testing.T) {
	testCases := []struct {
		name        string
		input       []byte
		expected    []string
		expectError bool
	}{
		{
			name:     "Valid input",
			input:    []byte(`{"components":{"schemas":{"PermissionsDto":{"properties":{"permissionName":{"enum":["id1","id2"]}}}}}}`),
			expected: []string{"id1", "id2"},
		},
		{
			name:        "Invalid JSON",
			input:       []byte(`invalid json`),
			expectError: true,
		},
		{
			name:        "Missing 'enum' field",
			input:       []byte(`{"components":{"schemas":{"PermissionsDto":{"properties":{"permissionName":{}}}}}}`),
			expectError: true,
		},
		{
			name:        "Invalid 'enum' field type",
			input:       []byte(`{"components":{"schemas":{"PermissionsDto":{"properties":{"permissionName":{"enum":"invalid"}}}}}}`),
			expectError: true,
		},
		{
			name:        "Missing 'permissionName' field",
			input:       []byte(`{"components":{"schemas":{"PermissionsDto":{"properties":{}}}}}`),
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			permissionIds, err := parseSupportedPermissionIds(testCase.input)

			if testCase.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, permissionIds)
			}
		})
	}
}
