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
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/writer"
)

func TestWriteAccountResources(t *testing.T) {
	t.Setenv(featureflags.ServiceUsers.EnvName(), "true")
	type want struct {
		groups       string
		users        string
		policies     string
		serviceUsers string
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
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			want{
				users: `users:
- email: monaco@dynatrace.com
  groups:
  - Log viewer
  - type: reference
    id: my-group
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
									account.Reference{Id: "my-policy"},
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
  environments:
  - environment: myenv123
    permissions:
    - View environment
    policies:
    - type: reference
      id: my-policy
    - View environment
  managementZones:
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
			"only service users",
			account.Resources{
				ServiceUsers: map[account.ServiceUserId]account.ServiceUser{
					"Service User 1": {
						Name:        "Service User 1",
						Description: "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			want{serviceUsers: `service-users:
- name: Service User 1
  description: Description of service user
  groups:
  - Log viewer
  - type: reference
    id: my-group
`,
			},
		},
		{
			"groups are not written if service user has none",
			account.Resources{
				ServiceUsers: map[account.ServiceUserId]account.ServiceUser{
					"Service User 1": {
						Name:        "Service User 1",
						Description: "Description of service user",
					},
				},
			},
			want{
				serviceUsers: `service-users:
- name: Service User 1
  description: Description of service user
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
							account.Reference{Id: "my-group"},
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
									account.Reference{Id: "my-policy"},
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
				ServiceUsers: map[account.ServiceUserId]account.ServiceUser{
					"Service User 1": {
						Name:        "Service User 1",
						Description: "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			want{
				users: `users:
- email: monaco@dynatrace.com
  groups:
  - Log viewer
  - type: reference
    id: my-group
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
  environments:
  - environment: myenv123
    permissions:
    - View environment
    policies:
    - type: reference
      id: my-policy
    - View environment
  managementZones:
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
				serviceUsers: `service-users:
- name: Service User 1
  description: Description of service user
  groups:
  - Log viewer
  - type: reference
    id: my-group
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
							account.Reference{Id: "my-group"},
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
									account.Reference{Id: "my-policy"},
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
				ServiceUsers: map[account.ServiceUserId]account.ServiceUser{
					"Service User 1": {
						Name:           "Service User 1",
						OriginObjectID: "ObjectID-789",
						Description:    "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			want{
				users: `users:
- email: monaco@dynatrace.com
  groups:
  - Log viewer
  - type: reference
    id: my-group
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
  environments:
  - environment: myenv123
    permissions:
    - View environment
    policies:
    - type: reference
      id: my-policy
    - View environment
  managementZones:
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
				serviceUsers: `service-users:
- name: Service User 1
  description: Description of service user
  groups:
  - Log viewer
  - type: reference
    id: my-group
  originObjectId: ObjectID-789
`,
			},
		},
		{
			"file contents are sorted",
			account.Resources{
				Users: map[account.UserId]account.User{
					"first@dynatrace.com": account.User{
						Email: "first@dynatrace.com",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
					"second@dynatrace.com": account.User{
						Email: "second@dynatrace.com",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
				Groups: map[account.GroupId]account.Group{
					"second-group": {
						ID:          "second-group",
						Name:        "My other Group",
						Description: "This is my group",
						Account: &account.Account{
							Permissions: []string{},
							Policies:    []account.Ref{account.StrReference("Request my Group Stuff")},
						},
						Environment:    []account.Environment{},
						ManagementZone: []account.ManagementZone{},
					},
					"first-group": {
						ID:          "first-group",
						Name:        "My Group",
						Description: "This is my group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
							Policies:    []account.Ref{account.StrReference("Request my Group Stuff")},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv456",
								Permissions: []string{"View environment"},
								Policies: []account.Ref{
									account.StrReference("View environment"),
									account.Reference{Id: "second-policy"},
								},
							},
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.Ref{
									account.StrReference("View environment"),
									account.Reference{Id: "first-policy"},
								},
							},
						},
						ManagementZone: []account.ManagementZone{
							{
								Environment:    "myenv123",
								ManagementZone: "Second MZone",
								Permissions:    []string{"Do Stuff"},
							},
							{
								Environment:    "myenv123",
								ManagementZone: "First MZone",
								Permissions:    []string{"C", "B", "A"},
							},
							{
								Environment:    "myenv456",
								ManagementZone: "First MZone",
								Permissions:    []string{"C", "B", "A"},
							},
						},
					},
				},
				Policies: map[account.PolicyId]account.Policy{
					"second-policy": {
						ID:          "second-policy",
						Name:        "My other Policy",
						Level:       account.PolicyLevelAccount{Type: "account"},
						Description: "This is my policy. There's many like it, but this one is mine.",
						Policy:      "ALLOW a:b:c;",
					},
					"first-policy": {
						ID:          "first-policy",
						Name:        "My Policy",
						Level:       account.PolicyLevelAccount{Type: "account"},
						Description: "This is my policy. There's many like it, but this one is mine.",
						Policy:      "ALLOW a:b:c;",
					},
				},
				ServiceUsers: map[account.ServiceUserId]account.ServiceUser{
					"Second service user": {
						Name:        "Second service user",
						Description: "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
					"First service user": {
						Name:        "First service user",
						Description: "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			want{
				users: `users:
- email: first@dynatrace.com
  groups:
  - Log viewer
  - type: reference
    id: my-group
- email: second@dynatrace.com
  groups:
  - Log viewer
  - type: reference
    id: my-group
`,
				groups: `groups:
- id: first-group
  name: My Group
  description: This is my group
  account:
    permissions:
    - View my Group Stuff
    policies:
    - Request my Group Stuff
  environments:
  - environment: myenv123
    permissions:
    - View environment
    policies:
    - type: reference
      id: first-policy
    - View environment
  - environment: myenv456
    permissions:
    - View environment
    policies:
    - type: reference
      id: second-policy
    - View environment
  managementZones:
  - environment: myenv123
    managementZone: First MZone
    permissions:
    - A
    - B
    - C
  - environment: myenv123
    managementZone: Second MZone
    permissions:
    - Do Stuff
  - environment: myenv456
    managementZone: First MZone
    permissions:
    - A
    - B
    - C
- id: second-group
  name: My other Group
  description: This is my group
  account:
    policies:
    - Request my Group Stuff
`,
				policies: `policies:
- id: first-policy
  name: My Policy
  level:
    type: account
  description: This is my policy. There's many like it, but this one is mine.
  policy: ALLOW a:b:c;
- id: second-policy
  name: My other Policy
  level:
    type: account
  description: This is my policy. There's many like it, but this one is mine.
  policy: ALLOW a:b:c;
`,
				serviceUsers: `service-users:
- name: First service user
  description: Description of service user
  groups:
  - Log viewer
  - type: reference
    id: my-group
- name: Second service user
  description: Description of service user
  groups:
  - Log viewer
  - type: reference
    id: my-group
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := writer.Context{
				Fs:            afero.NewMemMapFs(),
				OutputFolder:  "test",
				ProjectFolder: "project",
			}
			err := writer.Write(c, tt.givenResources)
			assert.NoError(t, err)

			expectedFolder := c.OutputFolder

			usersFilename := filepath.Join(expectedFolder, c.ProjectFolder, "users.yaml")
			if tt.wantPersisted.users == "" {
				assertNoFile(t, c.Fs, usersFilename)
			} else {
				assertFile(t, c.Fs, usersFilename, tt.wantPersisted.users)
			}

			groupsFilename := filepath.Join(expectedFolder, c.ProjectFolder, "groups.yaml")
			if tt.wantPersisted.groups == "" {
				assertNoFile(t, c.Fs, groupsFilename)
			} else {
				assertFile(t, c.Fs, groupsFilename, tt.wantPersisted.groups)
			}

			policiesFilename := filepath.Join(expectedFolder, c.ProjectFolder, "policies.yaml")
			if tt.wantPersisted.policies == "" {
				assertNoFile(t, c.Fs, policiesFilename)
			} else {
				assertFile(t, c.Fs, policiesFilename, tt.wantPersisted.policies)
			}

			serviceUsersFilename := filepath.Join(expectedFolder, c.ProjectFolder, "service-users.yaml")
			if tt.wantPersisted.serviceUsers == "" {
				assertNoFile(t, c.Fs, serviceUsersFilename)
			} else {
				assertFile(t, c.Fs, serviceUsersFilename, tt.wantPersisted.serviceUsers)
			}

		})
	}
}

func TestServiceUsersNotPersistedIfFeatureFlagDisabled(t *testing.T) {
	t.Setenv(featureflags.ServiceUsers.EnvName(), "false")
	resources :=
		account.Resources{
			Users: map[account.UserId]account.User{
				"monaco@dynatrace.com": account.User{
					Email: "monaco@dynatrace.com",
					Groups: []account.Ref{
						account.Reference{Id: "my-group"},
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
								account.Reference{Id: "my-policy"},
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
			ServiceUsers: map[account.ServiceUserId]account.ServiceUser{
				"Service User 1": {
					Name:        "Service User 1",
					Description: "Description of service user",
					Groups: []account.Ref{
						account.Reference{Id: "my-group"},
						account.StrReference("Log viewer"),
					},
				},
			},
		}
	expectedUsers := `users:
- email: monaco@dynatrace.com
  groups:
  - Log viewer
  - type: reference
    id: my-group
`
	expectedGroups := `groups:
- id: my-group
  name: My Group
  description: This is my group
  account:
    permissions:
    - View my Group Stuff
    policies:
    - Request my Group Stuff
  environments:
  - environment: myenv123
    permissions:
    - View environment
    policies:
    - type: reference
      id: my-policy
    - View environment
  managementZones:
  - environment: myenv123
    managementZone: My MZone
    permissions:
    - Do Stuff
`
	expectedPolicies := `policies:
- id: my-policy
  name: My Policy
  level:
    type: account
  description: This is my policy. There's many like it, but this one is mine.
  policy: ALLOW a:b:c;
`

	c := writer.Context{
		Fs:            afero.NewMemMapFs(),
		OutputFolder:  "test",
		ProjectFolder: "project",
	}
	err := writer.Write(c, resources)
	assert.NoError(t, err)
	assertFile(t, c.Fs, filepath.Join(c.OutputFolder, c.ProjectFolder, "users.yaml"), expectedUsers)
	assertFile(t, c.Fs, filepath.Join(c.OutputFolder, c.ProjectFolder, "groups.yaml"), expectedGroups)
	assertFile(t, c.Fs, filepath.Join(c.OutputFolder, c.ProjectFolder, "policies.yaml"), expectedPolicies)
	assertNoFile(t, c.Fs, filepath.Join(c.OutputFolder, c.ProjectFolder, "service-users.yaml"))
}

func assertFile(t *testing.T, fs afero.Fs, expectedPath, expectedContent string) {
	exists, err := afero.Exists(fs, expectedPath)
	assert.True(t, exists)
	assert.NoError(t, err)
	got, err := afero.ReadFile(fs, expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, string(got))
}

func assertNoFile(t *testing.T, fs afero.Fs, expectedPath string) {
	exists, err := afero.Exists(fs, expectedPath)
	assert.NoError(t, err)
	assert.False(t, exists, "expected file not to exist")
}
