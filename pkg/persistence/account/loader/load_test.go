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
		_, exists := loaded.Users["monaco@dynatrace.com"]
		assert.True(t, exists, "expected user to exist: monaco@dynatrace.com")
		assert.Len(t, loaded.Groups, 1)
		_, exists = loaded.Groups["my-group"]
		assert.True(t, exists, "expected group to exist: my-group")
		assert.Len(t, loaded.Policies, 1)
		_, exists = loaded.Policies["my-policy"]
		assert.True(t, exists, "expected policy to exist: my-policy")
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

	t.Run("Loads origin objectIDs", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/valid-origin-object-id.yaml")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		_, exists := loaded.Users["monaco@dynatrace.com"]
		assert.True(t, exists, "expected user to exist: monaco@dynatrace.com")
		assert.Len(t, loaded.Groups, 1)
		g, exists := loaded.Groups["my-group"]
		assert.True(t, exists, "expected group to exist: my-group")
		assert.Equal(t, "32952350-5e78-476d-ab1a-786dd9d4fe33", g.OriginObjectID, "expected group to be loaded with originObjectID")
		assert.Len(t, loaded.Policies, 1)
		p, exists := loaded.Policies["my-policy"]
		assert.Equal(t, "2338ebda-4aad-4911-96a2-6f60d7c3d2cb", p.OriginObjectID, "expected policy to be loaded with originObjectID")
		assert.True(t, exists, "expected policy to exist: my-policy")
		assert.Len(t, maps.Values(loaded.Groups)[0].Account.Policies, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Account.Permissions, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment[0].Policies, 2)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment[0].Permissions, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].ManagementZone, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].ManagementZone[0].Permissions, 1)

	})

	t.Run("Load multiple files but ignore config files", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/multi-with-configs")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
	})

	t.Run("Loading a file with only configs does not lead to errors", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/no-accounts-but-configs.yaml")
		assert.NoError(t, err)
		assert.Empty(t, loaded.Users)
		assert.Empty(t, loaded.Groups)
		assert.Empty(t, loaded.Policies)
	})

	t.Run("Duplicate group produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/duplicate-group.yaml")
		assert.Error(t, err)
	})

	t.Run("Duplicate user produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/duplicate-user.yaml")
		assert.Error(t, err)
	})

	t.Run("Duplicate policy produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/duplicate-policy.yaml")
		assert.Error(t, err)
	})

	t.Run("Missing environment ID for env-level policy produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/policy-missing-env-id.yaml")
		assert.Error(t, err)
	})

	t.Run("Partial policy definition produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/partial-policy.yaml")
		assert.Error(t, err)
	})

	t.Run("Partial user definition produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/partial-user.yaml")
		assert.Error(t, err)
	})

	t.Run("Partial group definition produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/partial-group.yaml")
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

func TestValidateReferences(t *testing.T) {
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
			name:     "mixing configs and account resources",
			path:     "testdata/configs-accounts-mixed.yaml",
			expected: ErrMixingConfigs,
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

func TestIsAccountConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			`neither "users", "groups", nor "policies is given"`,
			"configs: []",
			false,
		},
		{
			`"users" is given`,
			"users: []",
			true,
		},
		{
			`"groups" is given`,
			"groups: []",
			true,
		},
		{
			`"policies" is given`,
			"policies: []",
			true,
		},
		{
			`some other invalid config is given - not relevant for AM resource check`,
			"today: [isANiceDay]",
			false,
		},
		{
			`empty file`,
			"",
			false,
		},
		{
			"some completely wrong file content should still not fail",
			"<!DOCTYPE html>",
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
			err := afero.WriteFile(fs, "file.yaml", []byte(tt.content), 0644)
			assert.NoError(t, err)

			res := IsAccountConfigFile(fs, "file.yaml")
			assert.Equal(t, tt.expected, res)
		})
	}

}
