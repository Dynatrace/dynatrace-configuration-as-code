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
package loader

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("Load single file", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/valid.yaml")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Account.Policies, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Account.Permissions, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment[0].Policies, 2)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment[0].Permissions, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].ManagementZone, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].ManagementZone[0].Permissions, 1)

	})

	t.Run("Load single file - with refs", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/valid-with-refs.yaml")
		assert.NoError(t, err)
		assert.Len(t, loaded.Groups, 1)
		assert.NotNil(t, loaded.Groups["monaco-group"].Account)
		assert.Len(t, loaded.Groups["monaco-group"].Account.Policies, 2)
		assert.IsType(t, account.Reference{}, loaded.Groups["monaco-group"].Account.Policies[0])
		assert.IsType(t, account.StrReference(""), loaded.Groups["monaco-group"].Account.Policies[1])
		assert.NotNil(t, loaded.Groups["monaco-group"].Environment)
		assert.Len(t, loaded.Groups["monaco-group"].Environment, 1)
		assert.Equal(t, "vsy13800", loaded.Groups["monaco-group"].Environment[0].Name)
		assert.Len(t, loaded.Groups["monaco-group"].Environment[0].Policies, 2)
		assert.IsType(t, account.Reference{}, loaded.Groups["monaco-group"].Environment[0].Policies[0])
		assert.IsType(t, account.StrReference(""), loaded.Groups["monaco-group"].Environment[0].Policies[1])
		assert.Len(t, loaded.Policies, 2)
	})

	t.Run("Load multiple files", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/multi")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
	})

	t.Run("Duplicate group", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/duplicate-group.yaml")
		assert.Error(t, err)
	})

	t.Run("Duplicate user", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/duplicate-user.yaml")
		assert.Error(t, err)
	})

	t.Run("Duplicate policy", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/duplicate-policy.yaml")
		assert.Error(t, err)
	})

	t.Run("root folder not found", func(t *testing.T) {
		result, err := Load(afero.NewOsFs(), "testdata/non-existent-folder")
		assert.Equal(t, &account.Resources{
			Policies: make(map[string]account.Policy, 0),
			Groups:   make(map[string]account.Group, 0),
			Users:    make(map[string]account.User, 0),
		}, result)
		assert.NoError(t, err)
	})
}

func TestValidateT(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		expected       error
		expectedErrMsg string
	}{
		{

			name:           "group reference not found",
			path:           "testdata/no-ref-group.yaml",
			expected:       ErrRefMissing,
			expectedErrMsg: `error validating account resources with id "non-existing-group-ref": no referenced target found`,
		},
		{
			name:           "environment level policy reference not found",
			path:           "testdata/no-ref-policy-env.yaml",
			expected:       ErrRefMissing,
			expectedErrMsg: `error validating account resources with id "non-existing-policy-ref": no referenced target found`,
		},
		{
			name:           "account level policy reference not found",
			path:           "testdata/no-ref-policy-account.yaml",
			expected:       ErrRefMissing,
			expectedErrMsg: `error validating account resources with id "non-existing-policy-ref": no referenced target found`,
		},
		{
			name:           "group reference with missing id field",
			path:           "testdata/no-id-field-group-ref.yaml",
			expected:       ErrIdFieldMissing,
			expectedErrMsg: `error validating account resources: no ref id field found`,
		},
		{
			name:     "valid",
			path:     "testdata/valid.yaml",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(afero.NewOsFs(), tc.path)
			if tc.expected != nil {
				assert.ErrorIs(t, err, tc.expected)
				assert.ErrorContains(t, err, tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
