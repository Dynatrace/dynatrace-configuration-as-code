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

package account

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testCase struct {
	description string
	content     string
}

func TestLoadResources_Duplicates(t *testing.T) {
	testCases := []testCase{
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

			_, err := loadResources(fs, ".", manifest.ProjectDefinitionByProjectID{
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

	res, err := loadResources(fs, ".", manifest.ProjectDefinitionByProjectID{
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
