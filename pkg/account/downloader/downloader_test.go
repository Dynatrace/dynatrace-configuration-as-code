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

package downloader_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	stringutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/downloader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/downloader/internal/http"
)

func TestDownloader_DownloadConfiguration(t *testing.T) {
	uuidVar := "27dde8b6-2ed3-48f1-90b5-e4c0eae8b9bd"
	uuidVar2 := "3c345885-ff01-428b-ba49-b3381819f6dd"
	toID := stringutils.Sanitize
	tests := []struct {
		name      string
		given     mockData
		expected  account.Resources
		expectErr bool
	}{
		{
			name:  "empty account",
			given: mockData{},
			expected: account.Resources{
				Policies: make(map[account.PolicyId]account.Policy),
				Groups:   make(map[account.GroupId]account.Group),
				Users:    make(map[account.UserId]account.User),
			},
		},
		{
			name: "account policy",
			given: mockData{
				policies: []accountmanagement.PolicyOverview{
					{
						Uuid:        "2ff9314d-3c97-4607-bd49-460a53de1390",
						Name:        "test policy - tenant",
						Description: "some description",
						LevelType:   "account",
					}},
				policieDef: &accountmanagement.LevelPolicyDto{
					StatementQuery: "THIS IS statement",
				},
			},
			expected: account.Resources{
				Policies: map[account.PolicyId]account.Policy{
					toID("test policy - tenant"): {
						ID:             toID("test policy - tenant"),
						Name:           "test policy - tenant",
						Level:          account.PolicyLevelAccount{Type: "account"},
						Description:    "some description",
						Policy:         "THIS IS statement",
						OriginObjectID: "2ff9314d-3c97-4607-bd49-460a53de1390",
					},
				},
				Groups: make(map[account.GroupId]account.Group),
				Users:  make(map[account.UserId]account.User),
			},
		},
		{
			name: "environment policy",
			given: mockData{
				policies: []accountmanagement.PolicyOverview{
					{
						Uuid:        "2ff9314d-3c97-4607-bd49-460a53de1390",
						Name:        "test policy - tenant",
						Description: "some description",
						LevelId:     "abc12345",
						LevelType:   "environment",
					}},
				policieDef: &accountmanagement.LevelPolicyDto{
					Uuid:           "07beda6d-6a02-4827-9c1c-49037c96f176",
					Name:           "test policy",
					Description:    "user friendly description",
					StatementQuery: "THIS IS statement",
				},
			},
			expected: account.Resources{
				Policies: map[account.PolicyId]account.Policy{
					toID("test policy - tenant"): {
						ID:   toID("test policy - tenant"),
						Name: "test policy - tenant",
						Level: account.PolicyLevelEnvironment{
							Type:        "environment",
							Environment: "abc12345",
						},
						Description:    "some description",
						Policy:         "THIS IS statement",
						OriginObjectID: "2ff9314d-3c97-4607-bd49-460a53de1390",
					},
				},
				Groups: make(map[account.GroupId]account.Group),
				Users:  make(map[account.UserId]account.User),
			},
		},
		{
			name: "global policy",
			given: mockData{
				policies: []accountmanagement.PolicyOverview{{
					Uuid:        "07beda6d-6a02-4827-9c1c-49037c96f176",
					Name:        "test global policy",
					Description: "user friendly description",
					LevelId:     "",
					LevelType:   "global",
				}},
			},
			expected: account.Resources{
				Policies: map[account.PolicyId]account.Policy{},
				Groups:   make(map[account.GroupId]account.Group),
				Users:    make(map[account.UserId]account.User),
			},
		},
		{
			name: "no policy details (GetPolicyDefinition returns nil)",
			given: mockData{
				policies: []accountmanagement.PolicyOverview{{
					Uuid:        uuid.New().String(),
					Name:        "test policy",
					Description: "",
					LevelId:     "",
					LevelType:   "account",
				}},
			},
			expectErr: true,
		},
		{
			name: "only user",
			given: mockData{
				users:      []accountmanagement.UsersDto{{Email: "usert@some.org"}},
				userGroups: &accountmanagement.GroupUserDto{Email: "usert@some.org"},
			},
			expected: account.Resources{
				Policies: make(map[account.PolicyId]account.Policy),
				Groups:   make(map[account.GroupId]account.Group),
				Users: map[account.UserId]account.User{
					"usert@some.org": {Email: "usert@some.org"},
				},
			},
		},
		{
			name: "user with one group",
			given: mockData{
				groups: []accountmanagement.GetGroupDto{{
					Uuid: &uuidVar,
					Name: "test group",
				}},
				users: []accountmanagement.UsersDto{{Email: "usert@some.org"}},
				userGroups: &accountmanagement.GroupUserDto{
					Email:  "usert@some.org",
					Groups: []accountmanagement.AccountGroupDto{{Uuid: uuidVar}},
				},
			},
			expected: account.Resources{
				Policies: map[string]account.Policy{},
				Groups: map[account.GroupId]account.Group{
					toID("test group"): {
						ID:             toID("test group"),
						Name:           "test group",
						OriginObjectID: uuidVar,
					},
				},
				Users: map[account.UserId]account.User{
					"usert@some.org": {Email: "usert@some.org",
						Groups: []account.Ref{account.Reference{Id: toID("test group")}},
					},
				},
			},
		},
		{
			name: "no requested user details (GetGroupsForUser returns nil) ",
			given: mockData{
				users:      []accountmanagement.UsersDto{{Email: "usert@some.org"}},
				userGroups: nil,
			},
			expectErr: true,
		},
		{
			name: "empty group",
			given: mockData{
				groups: []accountmanagement.GetGroupDto{{
					Uuid: &uuidVar,
					Name: "test group",
				}},
			},
			expected: account.Resources{
				Policies: map[account.PolicyId]account.Policy{},
				Groups: map[account.GroupId]account.Group{
					toID("test group"): {
						ID:             toID("test group"),
						Name:           "test group",
						OriginObjectID: uuidVar,
					},
				},
				Users: map[account.UserId]account.User{},
			},
		},
		{
			name: "group with federated attribute values",
			given: mockData{
				groups: []accountmanagement.GetGroupDto{{
					Uuid:                     &uuidVar,
					Name:                     "test group",
					FederatedAttributeValues: []string{"firstName", "lastName", "memberOf"},
				}},
			},
			expected: account.Resources{
				Policies: map[account.PolicyId]account.Policy{},
				Groups: map[account.GroupId]account.Group{
					toID("test group"): {
						ID:                       toID("test group"),
						Name:                     "test group",
						FederatedAttributeValues: []string{"firstName", "lastName", "memberOf"},
						OriginObjectID:           uuidVar,
					},
				},
				Users: map[account.UserId]account.User{},
			},
		},
		{
			name: "groups with policies (account and environment)",
			given: mockData{
				ai:   &account.AccountInfo{AccountUUID: "e34fa4d6-b53a-43e0-9be0-cccca1a4da44"},
				envs: []accountmanagement.TenantResourceDto{{Id: "abc12345"}},
				policies: []accountmanagement.PolicyOverview{
					{
						Uuid:      "2ff9314d-3c97-4607-bd49-460a53de1390",
						Name:      "account policy",
						LevelType: "account",
					},
					{
						Uuid:      "bc7df7b7-9387-45ff-974f-56573c072e4c",
						Name:      "environment policy",
						LevelId:   "abc12345",
						LevelType: "environment",
					},
				},
				policieDef: &accountmanagement.LevelPolicyDto{},
				groups: []accountmanagement.GetGroupDto{
					{
						Uuid: &uuidVar,
						Name: "test group",
					},
					{
						Uuid: &uuidVar2,
						Name: "second test group",
					},
				},
				policyGroupBindings: []policyGroupBindings{
					{
						levelType: "account",
						levelId:   "e34fa4d6-b53a-43e0-9be0-cccca1a4da44",
						bindings: &accountmanagement.LevelPolicyBindingDto{
							PolicyBindings: []accountmanagement.Binding{{
								PolicyUuid: "2ff9314d-3c97-4607-bd49-460a53de1390",
								Groups:     []string{uuidVar, uuidVar2},
							}},
						},
						err: nil,
					},
					{
						levelType: "environment",
						levelId:   "abc12345",
						bindings: &accountmanagement.LevelPolicyBindingDto{
							PolicyBindings: []accountmanagement.Binding{{
								PolicyUuid: "bc7df7b7-9387-45ff-974f-56573c072e4c",
								Groups:     []string{uuidVar},
							}},
						},
						err: nil,
					},
				},
			},
			expected: account.Resources{
				Policies: map[account.PolicyId]account.Policy{
					toID("account policy"): {
						ID:             toID("account policy"),
						Name:           "account policy",
						Level:          account.PolicyLevelAccount{Type: "account"},
						OriginObjectID: "2ff9314d-3c97-4607-bd49-460a53de1390",
					},
					toID("environment policy"): account.Policy{
						ID:             toID("environment policy"),
						Name:           "environment policy",
						Level:          account.PolicyLevelEnvironment{Type: "environment", Environment: "abc12345"},
						OriginObjectID: "bc7df7b7-9387-45ff-974f-56573c072e4c",
					},
				},
				Groups: map[account.GroupId]account.Group{
					toID("test group"): {
						ID:             toID("test group"),
						Name:           "test group",
						OriginObjectID: uuidVar,
						Account: &account.Account{
							Policies: []account.Ref{account.Reference{Id: toID("account policy")}},
						},
						Environment: []account.Environment{
							{
								Name:     "abc12345",
								Policies: []account.Ref{account.Reference{Id: toID("environment policy")}},
							},
						},
					},
					toID("second test group"): {
						ID:             toID("second test group"),
						Name:           "second test group",
						OriginObjectID: uuidVar2,
						Account: &account.Account{
							Policies: []account.Ref{account.Reference{Id: toID("account policy")}},
						},
					},
				},
				Users: map[account.UserId]account.User{},
			},
		},
		{
			name: "groups with permissions",
			given: mockData{
				ai: &account.AccountInfo{AccountUUID: "e34fa4d6-b53a-43e0-9be0-cccca1a4da44"},
				envs: []accountmanagement.TenantResourceDto{{
					Name: "tenant1",
					Id:   "abc12345",
				}},
				mzones: []accountmanagement.ManagementZoneResourceDto{{
					Parent: "abc12345",
					Name:   "managementZone",
					Id:     "2698219524301731104",
				}},
				groups: []accountmanagement.GetGroupDto{
					{
						Uuid: &uuidVar,
						Name: "test group",
					},
					{
						Uuid: &uuidVar2,
						Name: "second test group",
					},
				},
				permissionsBindings: []permissionGroupBindings{
					{
						groupUUID: uuidVar,
						bindings: &accountmanagement.PermissionsGroupDto{
							Permissions: []accountmanagement.PermissionsDto{
								{
									PermissionName: "account-viewer",
									Scope:          "e34fa4d6-b53a-43e0-9be0-cccca1a4da44",
									ScopeType:      "account",
								},
								{
									PermissionName: "account-editor",
									Scope:          "e34fa4d6-b53a-43e0-9be0-cccca1a4da44",
									ScopeType:      "account",
								},
								{
									PermissionName: "tenant-logviewer",
									Scope:          "abc12345",
									ScopeType:      "tenant",
								},
								{
									PermissionName: "tenant-viewer",
									Scope:          "abc12345",
									ScopeType:      "tenant",
								},
							},
						},
						err: nil,
					},
					{
						groupUUID: uuidVar2,
						bindings: &accountmanagement.PermissionsGroupDto{
							Permissions: []accountmanagement.PermissionsDto{
								{
									PermissionName: "account-viewer",
									Scope:          "e34fa4d6-b53a-43e0-9be0-cccca1a4da44",
									ScopeType:      "account",
								},
								{
									PermissionName: "tenant-view-security-problems",
									Scope:          "abc12345:2698219524301731104",
									ScopeType:      "management-zone",
								},
								{
									PermissionName: "tenant-viewer",
									Scope:          "abc12345:2698219524301731104",
									ScopeType:      "management-zone",
								},
							},
						},
						err: nil,
					},
				},
			},
			expected: account.Resources{
				Policies: map[account.PolicyId]account.Policy{},
				Groups: map[account.GroupId]account.Group{
					toID("test group"): {
						ID:             toID("test group"),
						Name:           "test group",
						OriginObjectID: uuidVar,
						Account: &account.Account{
							Permissions: []string{"account-viewer", "account-editor"},
						},
						Environment: []account.Environment{
							{
								Name:        "abc12345",
								Permissions: []string{"tenant-logviewer", "tenant-viewer"},
							},
						},
					},
					toID("second test group"): {
						ID:             toID("second test group"),
						Name:           "second test group",
						OriginObjectID: uuidVar2,
						Account: &account.Account{
							Permissions: []string{"account-viewer"},
						},
						ManagementZone: []account.ManagementZone{{
							Environment:    "abc12345",
							ManagementZone: "managementZone",
							Permissions:    []string{"tenant-view-security-problems", "tenant-viewer"},
						}},
					},
				},
				Users: map[account.UserId]account.User{},
			},
		},
		{
			name: "no group details (GetGroupsForUser returns nil)",
			given: mockData{
				users: []accountmanagement.UsersDto{{Email: "test.user@some.org"}},
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := newMockDownloader(tc.given, t).DownloadResources(t.Context())

			if !tc.expectErr {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, *actual)
			} else {
				assert.Error(t, err)
			}
		})
	}

	t.Run("http client error handling", func(t *testing.T) {
		givenErr := errors.New("given error")
		tests := []struct {
			name            string
			given           mockData
			expectedMessage string
		}{
			{
				name:            "GetEnvironmentsAndMZones returns error",
				given:           mockData{environmentsAndMZonesError: givenErr},
				expectedMessage: "failed to get a list of environments and management zones for account ",
			},
			{
				name:            "GetPoliciesForAccount returns error",
				given:           mockData{policiesError: givenErr},
				expectedMessage: "failed to get a list of policies for account",
			},
			{
				name:            "GetUsers returns error",
				given:           mockData{usersError: givenErr},
				expectedMessage: "failed to get a list of users for account",
			},
			{
				name:            "GetGroups returns error",
				given:           mockData{groupsError: givenErr},
				expectedMessage: "failed to get a list of groups for account",
			},
			{
				name: "GetGroupsForUser returns error",
				given: mockData{
					users:              []accountmanagement.UsersDto{{Email: "test.user@some.org"}},
					groupsForUserError: givenErr,
				},
				expectedMessage: "failed to get a list of bind groups for user",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				d := newMockDownloader(tc.given, t)

				_, actualErr := d.DownloadResources(t.Context())

				assert.Error(t, actualErr)
				assert.ErrorContains(t, actualErr, givenErr.Error(), "Returned error must contain original error")
				assert.ErrorContains(t, actualErr, tc.expectedMessage, "Return error must contain additional information")
			})
		}
	})
}

type (
	policyGroupBindings struct {
		levelType, levelId string
		bindings           *accountmanagement.LevelPolicyBindingDto
		err                error
	}
	permissionGroupBindings struct {
		groupUUID string
		bindings  *accountmanagement.PermissionsGroupDto
		err       error
	}

	mockData struct {
		ai                  *account.AccountInfo
		envs                []accountmanagement.TenantResourceDto
		mzones              []accountmanagement.ManagementZoneResourceDto
		policies            []accountmanagement.PolicyOverview
		policieDef          *accountmanagement.LevelPolicyDto
		policyGroupBindings []policyGroupBindings
		permissionsBindings []permissionGroupBindings
		groups              []accountmanagement.GetGroupDto
		users               []accountmanagement.UsersDto
		userGroups          *accountmanagement.GroupUserDto

		environmentsAndMZonesError,
		policiesError,
		policyDefinitionError,
		groupsError,
		usersError,
		groupsForUserError error
	}
)

func newMockDownloader(d mockData, t *testing.T) *downloader.Downloader {
	if d.ai == nil {
		d.ai = &account.AccountInfo{
			Name:        "test",
			AccountUUID: uuid.New().String(),
		}
	}
	client := http.NewMockhttpClient(gomock.NewController(t))

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), d.ai.AccountUUID).Return(d.envs, d.mzones, d.environmentsAndMZonesError).MinTimes(0).MaxTimes(1)
	client.EXPECT().GetPolicies(gomock.Any(), d.ai.AccountUUID).Return(d.policies, d.policiesError).MinTimes(0).MaxTimes(1)
	client.EXPECT().GetPolicyDefinition(gomock.Any(), gomock.AnyOf(toSliceOfAny(d.policies)...)).Return(d.policieDef, d.policyDefinitionError).AnyTimes()
	if len(d.policyGroupBindings) == 0 {
		client.EXPECT().GetPolicyGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(&accountmanagement.LevelPolicyBindingDto{}, nil).AnyTimes()
	} else {
		for _, b := range d.policyGroupBindings {
			client.EXPECT().GetPolicyGroupBindings(gomock.Any(), b.levelType, b.levelId).Return(b.bindings, b.err).MinTimes(1)
		}
	}
	if len(d.permissionsBindings) == 0 {
		client.EXPECT().GetPermissionFor(gomock.Any(), d.ai.AccountUUID, gomock.Any()).Return(&accountmanagement.PermissionsGroupDto{}, nil).AnyTimes()
	} else {
		for _, b := range d.permissionsBindings {
			client.EXPECT().GetPermissionFor(gomock.Any(), d.ai.AccountUUID, b.groupUUID).Return(b.bindings, b.err).AnyTimes()
		}
	}
	client.EXPECT().GetGroups(gomock.Any(), d.ai.AccountUUID).Return(d.groups, d.groupsError).MinTimes(0).MaxTimes(1)
	client.EXPECT().GetUsers(gomock.Any(), d.ai.AccountUUID).Return(d.users, d.usersError).MinTimes(0).MaxTimes(1)
	client.EXPECT().GetGroupsForUser(gomock.Any(), userEmail(d.users), d.ai.AccountUUID).Return(d.userGroups, d.groupsForUserError).AnyTimes()

	return downloader.New4Test(d.ai, client)
}

func userEmail(u []accountmanagement.UsersDto) string {
	if u == nil {
		return ""
	}
	return u[0].Email
}

func toSliceOfAny[T any](s []T) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
