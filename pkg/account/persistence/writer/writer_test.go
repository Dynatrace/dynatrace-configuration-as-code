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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/writer"
)

func TestWriteAccountResources(t *testing.T) {
	type want struct {
		groups       string
		users        string
		policies     string
		serviceUsers string
		boundaries   string
	}
	type testCase struct {
		name              string
		givenResources    account.Resources
		wantPersisted     want
		boundariesEnabled bool
	}

	tests := []testCase{
		{
			name: "only users",
			givenResources: account.Resources{
				Users: map[account.UserId]account.User{
					"monaco@dynatrace.com": {
						Email: "monaco@dynatrace.com",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			wantPersisted: want{
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
			name: "groups are not written if user has none",
			givenResources: account.Resources{
				Users: map[account.UserId]account.User{
					"monaco@dynatrace.com": {Email: "monaco@dynatrace.com"},
				},
			},
			wantPersisted: want{
				users: `users:
- email: monaco@dynatrace.com
`,
			},
		},
		{
			name: "only groups",
			givenResources: account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:          "my-group",
						Name:        "My Group",
						Description: "This is my group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
							Policies:    []account.PolicyBinding{{Policy: account.StrReference("Request my Group Stuff")}},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.PolicyBinding{
									{Policy: account.StrReference("View environment")},
									{Policy: account.Reference{Id: "my-policy"}},
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
			wantPersisted: want{
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
			name: "only groups with boundaries serialize to new structure with `policy` nesting and boundaries",
			givenResources: account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:          "my-group",
						Name:        "My Group",
						Description: "This is my group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
							Policies:    []account.PolicyBinding{{Policy: account.StrReference("Request my Group Stuff")}},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.PolicyBinding{
									{
										Policy: account.StrReference("View environment"),
										Boundaries: []account.Ref{
											account.StrReference("My boundary"),
											account.Reference{Id: "my-boundary"},
										},
									},
									{Policy: account.Reference{Id: "my-policy"}},
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
			wantPersisted: want{
				groups: `groups:
- id: my-group
  name: My Group
  description: This is my group
  account:
    permissions:
    - View my Group Stuff
    policies:
    - policy: Request my Group Stuff
  environments:
  - environment: myenv123
    permissions:
    - View environment
    policies:
    - policy:
        type: reference
        id: my-policy
    - policy: View environment
      boundaries:
      - My boundary
      - type: reference
        id: my-boundary
  managementZones:
  - environment: myenv123
    managementZone: My MZone
    permissions:
    - Do Stuff
`,
			},
			boundariesEnabled: true,
		},
		{
			name: "empty optional values are not included when writing groups",
			givenResources: account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:   "my-group",
						Name: "My Group",
						Account: &account.Account{
							Permissions: []string{"View my Group Stuff"},
							Policies:    []account.PolicyBinding{{Policy: account.StrReference("Request my Group Stuff")}},
						},
					},
				},
			},
			wantPersisted: want{
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
			name: "group without any bindings is written correctly",
			givenResources: account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:   "my-group",
						Name: "My Group",
					},
				},
			},
			wantPersisted: want{
				groups: `groups:
- id: my-group
  name: My Group
`,
			},
		},
		{
			name: "group without any permissions is written correctly",
			givenResources: account.Resources{
				Groups: map[account.GroupId]account.Group{
					"my-group": {
						ID:   "my-group",
						Name: "My Group",
						Account: &account.Account{
							Policies: []account.PolicyBinding{{Policy: account.StrReference("Request my Group Stuff")}},
						},
					},
				},
			},
			wantPersisted: want{
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
			name: "group without any policies is written correctly",
			givenResources: account.Resources{
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
			wantPersisted: want{
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
			name: "only policies",
			givenResources: account.Resources{
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
			wantPersisted: want{policies: `policies:
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
			name:              "only boundaries",
			boundariesEnabled: true,
			givenResources: account.Resources{
				Boundaries: map[account.BoundaryId]account.Boundary{
					"my-boundary": {
						ID:             "my-boundary",
						Name:           "My Boundary",
						Query:          "Some query here",
						OriginObjectID: "some-id",
					},
				},
			},
			wantPersisted: want{boundaries: `boundaries:
- id: my-boundary
  name: My Boundary
  query: Some query here
  originObjectId: some-id
`,
			},
		},
		{
			name:              "only boundaries - FF disabled",
			boundariesEnabled: false,
			givenResources: account.Resources{
				Boundaries: map[account.BoundaryId]account.Boundary{
					"my-boundary": {
						ID:             "my-boundary",
						Name:           "My Boundary",
						Query:          "Some query here",
						OriginObjectID: "some-id",
					},
				},
			},
			wantPersisted: want{boundaries: ``},
		},
		{
			name: "only service users",
			givenResources: account.Resources{
				ServiceUsers: []account.ServiceUser{
					{
						Name:        "Service User 1",
						Description: "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			wantPersisted: want{serviceUsers: `serviceUsers:
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
			name: "two service users with same name but different origin object IDs",
			givenResources: account.Resources{
				ServiceUsers: []account.ServiceUser{
					{
						Name:           "Service User",
						OriginObjectID: "abc1",
						Description:    "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
					{
						Name:           "Service User",
						OriginObjectID: "abc2",
						Description:    "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			wantPersisted: want{serviceUsers: `serviceUsers:
- name: Service User
  description: Description of service user
  groups:
  - Log viewer
  - type: reference
    id: my-group
  originObjectId: abc1
- name: Service User
  description: Description of service user
  groups:
  - Log viewer
  - type: reference
    id: my-group
  originObjectId: abc2
`,
			},
		},
		{
			name: "groups are not written if service user has none",
			givenResources: account.Resources{
				ServiceUsers: []account.ServiceUser{
					{
						Name:        "Service User 1",
						Description: "Description of service user",
					},
				},
			},
			wantPersisted: want{
				serviceUsers: `serviceUsers:
- name: Service User 1
  description: Description of service user
`,
			},
		},
		{
			name: "full resources",
			givenResources: account.Resources{
				Users: map[account.UserId]account.User{
					"monaco@dynatrace.com": {
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
							Policies:    []account.PolicyBinding{{Policy: account.StrReference("Request my Group Stuff")}},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.PolicyBinding{
									{Policy: account.StrReference("View environment")},
									{Policy: account.Reference{Id: "my-policy"}},
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
				ServiceUsers: []account.ServiceUser{
					{
						Name:        "Service User 1",
						Description: "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			wantPersisted: want{
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
				serviceUsers: `serviceUsers:
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
			name: "with origin objectIDs",
			givenResources: account.Resources{
				Users: map[account.UserId]account.User{
					"monaco@dynatrace.com": {
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
							Policies:    []account.PolicyBinding{{Policy: account.StrReference("Request my Group Stuff")}},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.PolicyBinding{
									{Policy: account.StrReference("View environment")},
									{Policy: account.Reference{Id: "my-policy"}},
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
				ServiceUsers: []account.ServiceUser{
					{
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
			wantPersisted: want{
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
				serviceUsers: `serviceUsers:
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
			name: "file contents are sorted",
			givenResources: account.Resources{
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
							Policies:    []account.PolicyBinding{{Policy: account.StrReference("Request my Group Stuff")}},
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
							Policies:    []account.PolicyBinding{{Policy: account.StrReference("Request my Group Stuff")}},
						},
						Environment: []account.Environment{
							{
								Name:        "myenv456",
								Permissions: []string{"View environment"},
								Policies: []account.PolicyBinding{
									{Policy: account.StrReference("View environment")},
									{Policy: account.Reference{Id: "second-policy"}},
								},
							},
							{
								Name:        "myenv123",
								Permissions: []string{"View environment"},
								Policies: []account.PolicyBinding{
									{Policy: account.StrReference("View environment")},
									{Policy: account.Reference{Id: "first-policy"}},
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
				ServiceUsers: []account.ServiceUser{
					{
						Name:        "Second service user",
						Description: "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
					{
						Name:        "First service user",
						Description: "Description of service user",
						Groups: []account.Ref{
							account.Reference{Id: "my-group"},
							account.StrReference("Log viewer"),
						},
					},
				},
			},
			wantPersisted: want{
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
				serviceUsers: `serviceUsers:
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
			if tt.boundariesEnabled {
				t.Setenv(featureflags.Boundaries.EnvName(), "true")
			} else {
				t.Setenv(featureflags.Boundaries.EnvName(), "false")
			}

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

			boundariesFileName := filepath.Join(expectedFolder, c.ProjectFolder, "boundaries.yaml")
			if tt.wantPersisted.boundaries == "" {
				assertNoFile(t, c.Fs, boundariesFileName)
			} else {
				assertFile(t, c.Fs, boundariesFileName, tt.wantPersisted.boundaries)
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

func assertNoFile(t *testing.T, fs afero.Fs, expectedPath string) {
	exists, err := afero.Exists(fs, expectedPath)
	assert.NoError(t, err)
	assert.False(t, exists, "expected file not to exist")
}
