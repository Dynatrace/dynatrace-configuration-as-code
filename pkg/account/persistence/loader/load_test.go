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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
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
	groupContent := `groups:
- name: My Group
  id: my-group
  description: This is my group
`
	afero.WriteFile(fs, fmt.Sprintf("%s/%s", "p1", "user.yaml"), []byte(userContent), 0644)
	afero.WriteFile(fs, fmt.Sprintf("%s/%s", "p2", "policy.yaml"), []byte(policyContent), 0644)
	afero.WriteFile(fs, fmt.Sprintf("%s/%s", "p3", "group.yaml"), []byte(groupContent), 0644)

	res, err := LoadResources(fs, ".", manifest.ProjectDefinitionByProjectID{
		"p1": manifest.ProjectDefinition{Name: "p1", Group: "g1", Path: "p1"},
		"p2": manifest.ProjectDefinition{Name: "p2", Group: "g1", Path: "p2"},
		"p3": manifest.ProjectDefinition{Name: "p3", Group: "g1", Path: "p3"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.Len(t, res.Users, 1)
	assert.Len(t, res.Policies, 1)
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
		t.Setenv(featureflags.ServiceUsers.EnvName(), "true")
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

		assert.Len(t, loaded.ServiceUsers, 1)
		assert.Contains(t, loaded.ServiceUsers, "Service User 1", "expected service user to exist: Service User 1")
	})

	t.Run("Load single file - service user feature flag disabled", func(t *testing.T) {
		t.Setenv(featureflags.ServiceUsers.EnvName(), "false")
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
		assert.Len(t, loaded.ServiceUsers, 0)
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
		t.Setenv(featureflags.ServiceUsers.EnvName(), "true")
		loaded, err := Load(afero.NewOsFs(), "testdata/multi")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
		assert.Len(t, loaded.ServiceUsers, 1)
	})

	t.Run("Loads origin objectIDs", func(t *testing.T) {
		t.Setenv(featureflags.ServiceUsers.EnvName(), "true")
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
	})

	t.Run("Load multiple files but ignore config files", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/multi-with-configs")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
	})

	t.Run("Load multiple files and ignores any delete file", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "testdata/multi-with-delete-file")
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

	t.Run("Duplicate service user produces error", func(t *testing.T) {
		t.Setenv(featureflags.ServiceUsers.EnvName(), "true")
		_, err := Load(afero.NewOsFs(), "testdata/duplicate-service-user.yaml")
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
		t.Setenv(featureflags.ServiceUsers.EnvName(), "true")
		_, err := Load(afero.NewOsFs(), "testdata/partial-service-user.yaml")
		assert.Error(t, err)
	})

	t.Run("root folder not found", func(t *testing.T) {
		result, err := Load(afero.NewOsFs(), "testdata/non-existent-folder")
		assert.Equal(t, &account.Resources{
			Policies:     make(map[string]account.Policy, 0),
			Groups:       make(map[string]account.Group, 0),
			Users:        make(map[string]account.User, 0),
			ServiceUsers: make(map[string]account.ServiceUser, 0),
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
		assert.ErrorContains(t, err, "references missing policy")
	})

	t.Run("referencing a missing account level policy produces an error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/no-ref-policy-account.yaml")
		assert.ErrorContains(t, err, "references missing policy")
	})

	t.Run("mixed configs and account resources produces an error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/configs-accounts-mixed.yaml")
		assert.ErrorIs(t, err, ErrMixingConfigs)
	})

	t.Run("mixed delete and account resources produces an error", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "testdata/deletefile-accounts-mixed.yaml")
		assert.ErrorIs(t, err, ErrMixingDelete)
	})
}
