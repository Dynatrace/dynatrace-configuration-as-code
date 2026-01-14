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
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

func TestLoadResources_Duplicates(t *testing.T) {
	testCases := []struct {
		description string
		content     string
	}{
		{
			description: "Load Resources - duplicate user",
			content: `
users:
- email: email@address.com
  groups:
  - Log viewer
`,
		},
		{
			description: "Load Resources - duplicate policy",
			content: `policies:
- name: My Policy
  id: my-policy
  level:
    type: account
  description: abcde
  policy: |-
    ALLOW automation:workflows:read;
`,
		},
		{
			description: "Load Resources - duplicate boundary",
			content: `boundaries:
- name: My Boundary
  id: my-boundary
  query: shared:app-id = "my-app"
`,
		},
		{
			description: "Load Resources - duplicate group",
			content: `groups:
- name: My Group
  id: my-group
  description: This is my group
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for _, folderName := range []string{"p1", "p2"} {
				afero.WriteFile(fs, fmt.Sprintf("%s/%s", folderName, "res.yaml"), []byte(tc.content), 0644)
			}

			_, err := LoadResources(fs, ".", manifest.ProjectDefinitionByProjectID{
				"p1": manifest.ProjectDefinition{Name: "p1", Group: "g1", Path: "p1"},
				"p2": manifest.ProjectDefinition{Name: "p2", Group: "g1", Path: "p2"},
			})

			assert.Error(t, err)
		})
	}
}

func TestLoadResources(t *testing.T) {
	fs := afero.NewMemMapFs()

	userContent := `
users:
- email: email@address.com
  groups:
  - Log viewer
`
	policyContent := `policies:
- name: My Policy
  id: my-policy
  level:
    type: account
  description: abcde
  policy: |-
    ALLOW automation:workflows:read;
`
	boundaryContent := `boundaries:
- name: My Boundary
  id: my-boundary
  query: shared:app-id = "my-app"
`
	groupContent := `groups:
- name: My Group
  id: my-group
  description: This is my group
`
	afero.WriteFile(fs, fmt.Sprintf("%s/%s", "p1", "user.yaml"), []byte(userContent), 0644)
	afero.WriteFile(fs, fmt.Sprintf("%s/%s", "p2", "boundary.yaml"), []byte(boundaryContent), 0644)
	afero.WriteFile(fs, fmt.Sprintf("%s/%s", "p3", "policy.yaml"), []byte(policyContent), 0644)
	afero.WriteFile(fs, fmt.Sprintf("%s/%s", "p4", "group.yaml"), []byte(groupContent), 0644)

	res, err := LoadResources(fs, ".", manifest.ProjectDefinitionByProjectID{
		"p1": manifest.ProjectDefinition{Name: "p1", Group: "g1", Path: "p1"},
		"p2": manifest.ProjectDefinition{Name: "p2", Group: "g1", Path: "p2"},
		"p3": manifest.ProjectDefinition{Name: "p3", Group: "g1", Path: "p3"},
		"p4": manifest.ProjectDefinition{Name: "p4", Group: "g1", Path: "p4"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.Len(t, res.Users, 1)
	assert.Len(t, res.Policies, 1)
	assert.Len(t, res.Boundaries, 1)
	assert.Len(t, res.Groups, 1)

}

func TestLoad(t *testing.T) {

	var assertGroupLoadedValidFunc = func(t *testing.T, g account.Group) {
		assert.Len(t, g.Account.Policies, 1)
		assert.Len(t, g.Account.Permissions, 1)
		assert.Len(t, g.Environment, 1)
		assert.Len(t, g.Environment[0].Policies, 2)
		assert.Len(t, g.Environment[0].Permissions, 1)
		assert.Len(t, g.ManagementZone, 1)
		assert.Len(t, g.ManagementZone[0].Permissions, 1)
	}

	t.Run("Load single file", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/valid.yaml")
		assert.NoError(t, err)

		assert.Len(t, loaded.Users, 1)
		assert.Contains(t, loaded.Users, "monaco@dynatrace.com", "expected user to exist: monaco@dynatrace.com")

		assert.Len(t, loaded.Groups, 1)
		g, exists := loaded.Groups["my-group"]
		assert.True(t, exists, "expected group to exist: my-group")
		assertGroupLoadedValidFunc(t, g)

		assert.Len(t, loaded.Policies, 1)
		assert.Contains(t, loaded.Policies, "my-policy", "expected policy to exist: my-policy")

		assert.Len(t, loaded.Boundaries, 1)
		assert.Contains(t, loaded.Boundaries, "my-boundary", "expected boundary to exist: my-boundary")

		require.Len(t, loaded.ServiceUsers, 1)
		assert.Equal(t, "Service User 1", loaded.ServiceUsers[0].Name, "expected service user to exist: Service User 1")
	})

	t.Run("Load single file - with refs", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/valid-with-refs.yaml")
		assert.NoError(t, err)
		assert.Len(t, loaded.Groups, 1)
		assert.NotNil(t, loaded.Groups["monaco-group"].Account)
		assert.Len(t, loaded.Groups["monaco-group"].Account.Policies, 2)
		assert.IsType(t, account.PolicyBinding{}, loaded.Groups["monaco-group"].Account.Policies[0])
		assert.IsType(t, account.Reference{}, loaded.Groups["monaco-group"].Account.Policies[0].Policy)
		assert.IsType(t, account.PolicyBinding{}, loaded.Groups["monaco-group"].Account.Policies[1])
		assert.IsType(t, account.StrReference(""), loaded.Groups["monaco-group"].Account.Policies[1].Policy)
		assert.NotNil(t, loaded.Groups["monaco-group"].Environment)
		assert.Len(t, loaded.Groups["monaco-group"].Environment, 1)
		assert.Equal(t, "vsy13800", loaded.Groups["monaco-group"].Environment[0].Name)
		assert.Len(t, loaded.Groups["monaco-group"].Environment[0].Policies, 2)
		assert.IsType(t, account.PolicyBinding{}, loaded.Groups["monaco-group"].Environment[0].Policies[0])
		assert.IsType(t, account.Reference{}, loaded.Groups["monaco-group"].Environment[0].Policies[0].Policy)
		assert.IsType(t, account.PolicyBinding{}, loaded.Groups["monaco-group"].Environment[0].Policies[1])
		assert.IsType(t, account.StrReference(""), loaded.Groups["monaco-group"].Environment[0].Policies[1].Policy)
		assert.Len(t, loaded.Policies, 2)
		assert.Len(t, loaded.Boundaries, 1)
	})

	t.Run("Load multiple files", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/multi")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
		assert.Len(t, loaded.ServiceUsers, 1)
		assert.Len(t, loaded.Boundaries, 1)
	})

	t.Run("Loads origin objectIDs", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/valid-origin-object-id.yaml")
		assert.NoError(t, err)
		assert.Contains(t, loaded.Users, "monaco@dynatrace.com", "expected user to exist: monaco@dynatrace.com")

		assert.Len(t, loaded.Groups, 1)
		g, exists := loaded.Groups["my-group"]
		assert.True(t, exists, "expected group to exist: my-group")
		assertGroupLoadedValidFunc(t, g)
		assert.Equal(t, "32952350-5e78-476d-ab1a-786dd9d4fe33", g.OriginObjectID, "expected group to be loaded with originObjectID")

		assert.Len(t, loaded.Policies, 1)
		p, exists := loaded.Policies["my-policy"]
		assert.Equal(t, "2338ebda-4aad-4911-96a2-6f60d7c3d2cb", p.OriginObjectID, "expected policy to be loaded with originObjectID")
		assert.True(t, exists, "expected policy to exist: my-policy")

		assert.Len(t, loaded.Boundaries, 1)
		b, exists := loaded.Boundaries["my-boundary"]
		assert.Equal(t, "0cd9c365-ed0e-4ef5-8440-46d02fceea3e", b.OriginObjectID, "expected boundary to be loaded with originObjectID")
		assert.True(t, exists, "expected boundary to exist: my-boundary")
	})

	t.Run("Load multiple files but ignore config files", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/multi-with-configs")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
		assert.Len(t, loaded.Boundaries, 1)
	})

	t.Run("Load multiple files and ignores any delete file", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/multi-with-delete-file")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
		assert.Len(t, loaded.Boundaries, 1)
	})

	t.Run("Loading a file with only configs does not lead to errors", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/no-accounts-but-configs.yaml")
		assert.NoError(t, err)
		assert.Empty(t, loaded.Users)
		assert.Empty(t, loaded.Groups)
		assert.Empty(t, loaded.Policies)
		assert.Empty(t, loaded.Boundaries)
	})

	t.Run("Load service users with same name but different originObjectIds", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/service-users-with-same-name-and-different-origin-object-ids.yaml")
		require.NoError(t, err)
		assert.Empty(t, loaded.Users)
		assert.Empty(t, loaded.Groups)
		assert.Empty(t, loaded.Policies)
		assert.Len(t, loaded.ServiceUsers, 2)
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

	t.Run("Duplicate boundary produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/duplicate-boundary.yaml")
		assert.Error(t, err)
	})

	t.Run("Service users with the same name and no origin object IDs produce error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/service-users-with-same-name-and-no-origin-object-ids.yaml")
		assert.Error(t, err)
	})

	t.Run("Service users with the same name and same origin object IDs produce error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/service-users-with-same-name-and-same-origin-object-ids.yaml")
		assert.Error(t, err)
	})

	t.Run("Service users with the same name and one missing origin object ID produce error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/service-users-with-same-name-and-missing-origin-object-id.yaml")
		assert.Error(t, err)
	})

	t.Run("Service users with the different name and same origin object IDs produce error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/service-users-with-different-name-and-same-origin-object-ids.yaml")
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

	t.Run("Partial boundary definition produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/partial-boundary.yaml")
		assert.Error(t, err)
	})

	t.Run("Partial user definition produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/partial-user.yaml")
		assert.Error(t, err)
	})

	t.Run("User definition with group reference with missing id field produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/no-id-field-group-ref.yaml")
		assert.Error(t, err)
		assert.ErrorContains(t, err, "missing required field 'id' for reference")
	})

	t.Run("Partial group definition produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/partial-group.yaml")
		assert.Error(t, err)
	})

	t.Run("Partial service user definition produces error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/partial-service-user.yaml")
		assert.Error(t, err)
	})

	t.Run("root folder not found", func(t *testing.T) {
		result, err := Load(afero.NewOsFs(), "testdata/non-existent-folder")
		assert.Equal(t, &account.Resources{
			Boundaries:   make(map[string]account.Boundary, 0),
			Policies:     make(map[string]account.Policy, 0),
			Groups:       make(map[string]account.Group, 0),
			Users:        make(map[string]account.User, 0),
			ServiceUsers: make([]account.ServiceUser, 0),
		}, result)
		assert.NoError(t, err)
	})

	t.Run("valid file produces no error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/valid.yaml")
		assert.NoError(t, err)
	})

	t.Run("referencing a missing environment level policy produces an error.", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/no-ref-group.yaml")
		assert.ErrorContains(t, err, "references missing group")
	})

	t.Run("environment level policy reference not found", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/no-ref-policy-env.yaml")
		assert.ErrorContains(t, err, "has an invalid policy reference for environment")
	})

	t.Run("referencing a missing account level policy produces an error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/no-ref-policy-account.yaml")
		assert.ErrorContains(t, err, "has an invalid account policy reference")
	})

	t.Run("mixed configs and account resources produces an error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/configs-accounts-mixed.yaml")
		assert.ErrorIs(t, err, ErrMixingConfigs)
	})

	t.Run("mixed delete and account resources produces an error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/deletefile-accounts-mixed.yaml")
		assert.ErrorIs(t, err, ErrMixingDelete)
	})

	t.Run("All different policy structure variants are supported when loading groups", func(t *testing.T) {
		data, err := Load(afero.NewOsFs(), "testdata/valid-policy-binding-variants.yaml")
		assert.NoError(t, err)
		assert.Len(t, data.Boundaries, 2)

		bindings := data.Groups["my-group"].Account.Policies
		assert.Len(t, bindings, 4)

		assert.Equal(t, account.PolicyBinding{Policy: account.StrReference("My Policy"), Boundaries: []account.Ref{}}, bindings[0])
		assert.Equal(t, account.PolicyBinding{Policy: account.Reference{Id: "my-policy2"}, Boundaries: []account.Ref{}}, bindings[1])
		assert.Equal(t, account.PolicyBinding{Policy: account.StrReference("My Policy3"), Boundaries: []account.Ref{}}, bindings[2])
		assert.Equal(t, account.PolicyBinding{Policy: account.Reference{Id: "my-policy4"},
			Boundaries: []account.Ref{account.Reference{Id: "my-boundary"}, account.StrReference("My Boundary2")}},
			bindings[3])
	})

	t.Run("Invalid policy binding leads to error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/invalid-policy-binding.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "boundaries are only supported when using the 'policy' key")
	})

	t.Run("Ambiguous policy binding leads to error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/ambiguous-policy-binding.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy definition is ambiguous")
	})

	t.Run("Loading a group with a missing environment name produces an error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/group-missing-env-name.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required field 'environment'")
	})
}
