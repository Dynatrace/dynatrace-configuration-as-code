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
package writer_test

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/writer"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestWriteAccountResources(t *testing.T) {
	type want struct {
		groups   string
		users    string
		policies string
	}
	tests := []struct {
		name           string
		givenResources account.Resources
		wantPersisted  want
	}{
		{
			"only users",
			account.Resources{
				Users: map[account.UserId]account.User{
					"monaco@dynatrace.com": account.User{
						Email: "monaco@dynatrace.com",
						Groups: []account.Ref{
							account.Reference{
								Type: "reference",
								Id:   "my-group",
							},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			want{
				users: `users:
- email: monaco@dynatrace.com
  groups:
  - type: reference
    id: my-group
  - Log viewer
`,
			},
		},
		{
			"groups are not written if user has none",
			account.Resources{
				Users: map[account.UserId]account.User{
					"monaco@dynatrace.com": {Email: "monaco@dynatrace.com"},
				},
			},
			want{
				users: `users:
- email: monaco@dynatrace.com
`,
			},
		},
		{
			"only groups",
			account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:          "my-group",
						Name:        "My Group",
						Description: "This is my group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
							Policies:    []account.Ref{account.StrReference("Request my Group Stuff")},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.Ref{
									account.StrReference("View environment"),
									account.Reference{
										Type: "reference",
										Id:   "my-policy",
									},
								},
							},
						},
						ManagementZone: []account.ManagementZone{
							{
								Environment:    "myenv123",
								ManagementZone: "My MZone",
								Permissions:    []string{"Do Stuff"},
							},
						},
					},
				},
			},
			want{
				groups: `groups:
- id: my-group
  name: My Group
  description: This is my group
  account:
    permissions:
    - View my Group Stuff
    policies:
    - Request my Group Stuff
  environment:
  - name: myenv123
    permissions:
    - View environment
    policies:
    - View environment
    - type: reference
      id: my-policy
  managementZone:
  - environment: myenv123
    managementZone: My MZone
    permissions:
    - Do Stuff
`,
			},
		},
		{
			"empty optional values are not included when writing groups",
			account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:   "my-group",
						Name: "My Group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
							Policies:    []account.Ref{account.StrReference("Request my Group Stuff")},
						},
					},
				},
			},
			want{
				groups: `groups:
- id: my-group
  name: My Group
  account:
    permissions:
    - View my Group Stuff
    policies:
    - Request my Group Stuff
`,
			},
		},
		{
			"group without any bindings is written correctly",
			account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:   "my-group",
						Name: "My Group",
					},
				},
			},
			want{
				groups: `groups:
- id: my-group
  name: My Group
`,
			},
		},
		{
			"group without any permissions is written correctly",
			account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:   "my-group",
						Name: "My Group",
						Account: &account.Account{
							Policies: []account.Ref{account.StrReference("Request my Group Stuff")},
						},
					},
				},
			},
			want{
				groups: `groups:
- id: my-group
  name: My Group
  account:
    policies:
    - Request my Group Stuff
`,
			},
		},
		{
			"group without any policies is written correctly",
			account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:   "my-group",
						Name: "My Group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
						},
					},
				},
			},
			want{
				groups: `groups:
- id: my-group
  name: My Group
  account:
    permissions:
    - View my Group Stuff
`,
			},
		},
		{
			"only policies",
			account.Resources{
				Policies: map[account.PolicyId]account.Policy{
					"my-policy": {
						ID:          "my-policy",
						Name:        "My Policy",
						Level:       account.PolicyLevelAccount{Type: "account"},
						Description: "This is my policy. There's many like it, but this one is mine.",
						Policy:      "ALLOW a:b:c;",
					},
				},
			},
			want{policies: `policies:
- id: my-policy
  name: My Policy
  level:
    type: account
  description: This is my policy. There's many like it, but this one is mine.
  policy: ALLOW a:b:c;
`,
			},
		},
		{
			"full resources",
			account.Resources{
				Users: map[account.UserId]account.User{
					"monaco@dynatrace.com": account.User{
						Email: "monaco@dynatrace.com",
						Groups: []account.Ref{
							account.Reference{
								Type: "reference",
								Id:   "my-group",
							},
							account.StrReference("Log viewer"),
						},
					},
				},
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:          "my-group",
						Name:        "My Group",
						Description: "This is my group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
							Policies:    []account.Ref{account.StrReference("Request my Group Stuff")},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.Ref{
									account.StrReference("View environment"),
									account.Reference{
										Type: "reference",
										Id:   "my-policy",
									},
								},
							},
						},
						ManagementZone: []account.ManagementZone{
							{
								Environment:    "myenv123",
								ManagementZone: "My MZone",
								Permissions:    []string{"Do Stuff"},
							},
						},
					},
				},
				Policies: map[account.PolicyId]account.Policy{
					"my-policy": {
						ID:          "my-policy",
						Name:        "My Policy",
						Level:       account.PolicyLevelAccount{Type: "account"},
						Description: "This is my policy. There's many like it, but this one is mine.",
						Policy:      "ALLOW a:b:c;",
					},
				},
			},
			want{
				users: `users:
- email: monaco@dynatrace.com
  groups:
  - type: reference
    id: my-group
  - Log viewer
`,
				groups: `groups:
- id: my-group
  name: My Group
  description: This is my group
  account:
    permissions:
    - View my Group Stuff
    policies:
    - Request my Group Stuff
  environment:
  - name: myenv123
    permissions:
    - View environment
    policies:
    - View environment
    - type: reference
      id: my-policy
  managementZone:
  - environment: myenv123
    managementZone: My MZone
    permissions:
    - Do Stuff
`,
				policies: `policies:
- id: my-policy
  name: My Policy
  level:
    type: account
  description: This is my policy. There's many like it, but this one is mine.
  policy: ALLOW a:b:c;
`,
			},
		},
		{
			"with origin objectIDs",
			account.Resources{
				Users: map[account.UserId]account.User{
					"monaco@dynatrace.com": account.User{
						Email: "monaco@dynatrace.com",
						Groups: []account.Ref{
							account.Reference{
								Type: "reference",
								Id:   "my-group",
							},
							account.StrReference("Log viewer"),
						},
					},
				},
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:             "my-group",
						OriginObjectID: "ObjectID-123",
						Name:           "My Group",
						Description:    "This is my group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
							Policies:    []account.Ref{account.StrReference("Request my Group Stuff")},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.Ref{
									account.StrReference("View environment"),
									account.Reference{
										Type: "reference",
										Id:   "my-policy",
									},
								},
							},
						},
						ManagementZone: []account.ManagementZone{
							{
								Environment:    "myenv123",
								ManagementZone: "My MZone",
								Permissions:    []string{"Do Stuff"},
							},
						},
					},
				},
				Policies: map[account.PolicyId]account.Policy{
					"my-policy": {
						ID:             "my-policy",
						OriginObjectID: "ObjectID-456",
						Name:           "My Policy",
						Level:          account.PolicyLevelAccount{Type: "account"},
						Description:    "This is my policy. There's many like it, but this one is mine.",
						Policy:         "ALLOW a:b:c;",
					},
				},
			},
			want{
				users: `users:
- email: monaco@dynatrace.com
  groups:
  - type: reference
    id: my-group
  - Log viewer
`,
				groups: `groups:
- id: my-group
  name: My Group
  description: This is my group
  account:
    permissions:
    - View my Group Stuff
    policies:
    - Request my Group Stuff
  environment:
  - name: myenv123
    permissions:
    - View environment
    policies:
    - View environment
    - type: reference
      id: my-policy
  managementZone:
  - environment: myenv123
    managementZone: My MZone
    permissions:
    - Do Stuff
  originObjectId: ObjectID-123
`,
				policies: `policies:
- id: my-policy
  name: My Policy
  level:
    type: account
  description: This is my policy. There's many like it, but this one is mine.
  policy: ALLOW a:b:c;
  originObjectId: ObjectID-456
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := writer.Context{
				Fs:            afero.NewMemMapFs(),
				OutputFolder:  "test/",
				ProjectFolder: "project/",
			}
			err := writer.Write(c, tt.givenResources)
			assert.NoError(t, err)

			expectedFolder := c.OutputFolder

			users := filepath.Join(expectedFolder, c.ProjectFolder, "users.yaml")
			if tt.wantPersisted.users == "" {
				exists, err := afero.Exists(c.Fs, users)
				assert.NoError(t, err)
				assert.False(t, exists, "expected no users file to be created")
			} else {
				assertFile(t, c.Fs, users, tt.wantPersisted.users)
			}

			groups := filepath.Join(expectedFolder, c.ProjectFolder, "groups.yaml")
			if tt.wantPersisted.groups == "" {
				exists, err := afero.Exists(c.Fs, groups)
				assert.NoError(t, err)
				assert.False(t, exists, "expected no groups file to be created")
			} else {
				assertFile(t, c.Fs, groups, tt.wantPersisted.groups)
			}

			policies := filepath.Join(expectedFolder, c.ProjectFolder, "policies.yaml")
			if tt.wantPersisted.policies == "" {
				exists, err := afero.Exists(c.Fs, policies)
				assert.NoError(t, err)
				assert.False(t, exists, "expected no policies file to be created")
			} else {
				assertFile(t, c.Fs, policies, tt.wantPersisted.policies)
			}

		})
	}
}

func assertFile(t *testing.T, fs afero.Fs, expectedPath, expectedContent string) {
	exists, err := afero.Exists(fs, expectedPath)
	assert.True(t, exists)
	assert.NoError(t, err)
	got, err := afero.ReadFile(fs, expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, string(got))
}
