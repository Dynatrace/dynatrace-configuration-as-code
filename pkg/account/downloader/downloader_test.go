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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	stringutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/downloader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/downloader/internal/http"
)

var (
	accountUUID   = "32ee817e-0db4-4f28-b1d1-a2b7032cdf29"
	groupUUID1    = "27dde8b6-2ed3-48f1-90b5-e4c0eae8b9bd"
	groupUUID2    = "3c345885-ff01-428b-ba49-b3381819f6dd"
	boundaryUUID1 = "2e3a1a18-6803-4742-aca2-70fbe312bd18"
	toID          = stringutils.Sanitize
	originalErr   = errors.New("original error")
)

// TestDownloader_EmptyAccount tests that downloading an empty account succeeds.
func TestDownloader_EmptyAccount(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Empty(t, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_AccountPolicy tests that downloading an account-level policy succeeds.
func TestDownloader_AccountPolicy(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)

	policyOverview := accountmanagement.PolicyOverview{
		Uuid:        "2ff9314d-3c97-4607-bd49-460a53de1390",
		Name:        "test policy - tenant",
		Description: "some description",
		LevelType:   "account",
	}
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{policyOverview}, nil)
	client.EXPECT().GetPolicyDefinition(gomock.Any(), policyOverview).Return(&accountmanagement.LevelPolicyDto{
		StatementQuery: "THIS IS statement",
	}, nil)

	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, map[account.PolicyId]account.Policy{
		toID("test policy - tenant"): {
			ID:             toID("test policy - tenant"),
			Name:           "test policy - tenant",
			Level:          account.PolicyLevelAccount{Type: "account"},
			Description:    "some description",
			Policy:         "THIS IS statement",
			OriginObjectID: "2ff9314d-3c97-4607-bd49-460a53de1390",
		},
	}, result.Policies)
	assert.Empty(t, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_EnvironmentPolicy tests that downloading an environment-level policy succeeds.
func TestDownloader_EnvironmentPolicy(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)

	policyOverview := accountmanagement.PolicyOverview{
		Uuid:        "2ff9314d-3c97-4607-bd49-460a53de1390",
		Name:        "test policy - tenant",
		Description: "some description",
		LevelId:     "abc12345",
		LevelType:   "environment",
	}
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{policyOverview}, nil)
	client.EXPECT().GetPolicyDefinition(gomock.Any(), policyOverview).Return(&accountmanagement.LevelPolicyDto{
		Uuid:           "07beda6d-6a02-4827-9c1c-49037c96f176",
		Name:           "test policy",
		Description:    "user friendly description",
		StatementQuery: "THIS IS statement",
	}, nil)

	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, map[account.PolicyId]account.Policy{
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
	}, result.Policies)
	assert.Empty(t, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_GlobalPolicy tests that downloading a global policy succeeds.
func TestDownloader_GlobalPolicy(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)

	policyOverview := accountmanagement.PolicyOverview{
		Uuid:        "07beda6d-6a02-4827-9c1c-49037c96f176",
		Name:        "test global policy",
		Description: "user friendly description",
		LevelType:   "global",
	}
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{policyOverview}, nil)

	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Empty(t, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_NoPolicyDetails tests that downloading a policy without details fails as expected.
func TestDownloader_NoPolicyDetails(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)

	policyOverview := accountmanagement.PolicyOverview{
		Uuid:        "2ff9314d-3c97-4607-bd49-460a53de1390",
		Name:        "test policy",
		Description: "",
		LevelId:     "",
		LevelType:   "account",
	}
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{policyOverview}, nil)
	client.EXPECT().GetPolicyDefinition(gomock.Any(), policyOverview).Return(nil, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Error(t, err)
	require.Nil(t, result)
}

// TestDownloader_OnlyUser tests that downloading an account with a single user succeeds.
func TestDownloader_OnlyUser(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{{Email: "usert@some.org"}}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "usert@some.org", accountUUID).Return(&accountmanagement.GroupUserDto{Email: "usert@some.org"}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Empty(t, result.Groups)
	assert.Equal(t, map[account.UserId]account.User{
		"usert@some.org": {Email: "usert@some.org"},
	}, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_UserWithOneGroup tests that downloading an account with a user that belongs to one group succeeds.
func TestDownloader_UserWithOneGroup(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{{
		Uuid: &groupUUID1,
		Name: "test group",
	}}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(&accountmanagement.LevelPolicyBindingDto{}, nil)
	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, gomock.Any()).Return(&accountmanagement.PermissionsGroupDto{}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{{Email: "usert@some.org"}}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "usert@some.org", accountUUID).Return(&accountmanagement.GroupUserDto{
		Email:  "usert@some.org",
		Groups: []accountmanagement.AccountGroupDto{{Uuid: groupUUID1}},
	}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Equal(t, map[account.GroupId]account.Group{
		toID("test group"): {
			ID:             toID("test group"),
			Name:           "test group",
			OriginObjectID: groupUUID1,
		}}, result.Groups)
	assert.Equal(t, map[account.UserId]account.User{
		"usert@some.org": {Email: "usert@some.org",
			Groups: []account.Ref{account.Reference{Id: toID("test group")}},
		},
	}, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_NoRequestedUserDetails tests that downloading a user without user details fails as expected.
func TestDownloader_NoRequestedUserDetails(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{{Email: "usert@some.org"}}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "usert@some.org", accountUUID).Return(nil, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestDownloader_EmptyGroup tests that downloading an account with an empty group succeeds.
func TestDownloader_EmptyGroup(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{{
		Uuid: &groupUUID1,
		Name: "test group",
	}}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(&accountmanagement.LevelPolicyBindingDto{}, nil)
	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, gomock.Any()).Return(&accountmanagement.PermissionsGroupDto{}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Equal(t, map[account.GroupId]account.Group{
		toID("test group"): {
			ID:             toID("test group"),
			Name:           "test group",
			OriginObjectID: groupUUID1,
		},
	}, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_GroupWithFederatedAttributeValues tests that downloading a group with federated attribute values succeeds.
func TestDownloader_GroupWithFederatedAttributeValues(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{{
		Uuid:                     &groupUUID1,
		Name:                     "test group",
		FederatedAttributeValues: []string{"firstName", "lastName", "memberOf"},
	}}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(&accountmanagement.LevelPolicyBindingDto{}, nil)
	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, gomock.Any()).Return(&accountmanagement.PermissionsGroupDto{}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Equal(t, map[account.GroupId]account.Group{
		toID("test group"): {
			ID:                       toID("test group"),
			Name:                     "test group",
			FederatedAttributeValues: []string{"firstName", "lastName", "memberOf"},
			OriginObjectID:           groupUUID1,
		},
	}, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_GroupsWithPolicies tests that downloading groups with policies succeeds.
func TestDownloader_GroupsWithPolicies(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	accountPolicyOverview := accountmanagement.PolicyOverview{
		Uuid:      "2ff9314d-3c97-4607-bd49-460a53de1390",
		Name:      "account policy",
		LevelType: "account",
	}

	environmentPolicyOverview := accountmanagement.PolicyOverview{
		Uuid:      "bc7df7b7-9387-45ff-974f-56573c072e4c",
		Name:      "environment policy",
		LevelId:   "abc12345",
		LevelType: "environment",
	}

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{{Id: "abc12345"}}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{accountPolicyOverview, environmentPolicyOverview}, nil)

	client.EXPECT().GetPolicyDefinition(gomock.Any(), accountPolicyOverview).Return(&accountmanagement.LevelPolicyDto{}, nil)
	client.EXPECT().GetPolicyDefinition(gomock.Any(), environmentPolicyOverview).Return(&accountmanagement.LevelPolicyDto{}, nil)

	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{
		{
			Uuid: &groupUUID1,
			Name: "test group",
		},
		{
			Uuid: &groupUUID2,
			Name: "second test group",
		}}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "account", accountUUID).Return(&accountmanagement.LevelPolicyBindingDto{
		PolicyBindings: []accountmanagement.Binding{{
			PolicyUuid: "2ff9314d-3c97-4607-bd49-460a53de1390",
			Groups:     []string{groupUUID1, groupUUID2},
		}}}, nil)

	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, groupUUID1).Return(&accountmanagement.PermissionsGroupDto{}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "environment", "abc12345").Return(&accountmanagement.LevelPolicyBindingDto{
		PolicyBindings: []accountmanagement.Binding{{
			PolicyUuid: "bc7df7b7-9387-45ff-974f-56573c072e4c",
			Groups:     []string{groupUUID1},
		}},
	}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "account", accountUUID).Return(&accountmanagement.LevelPolicyBindingDto{
		PolicyBindings: []accountmanagement.Binding{{
			PolicyUuid: "2ff9314d-3c97-4607-bd49-460a53de1390",
			Groups:     []string{groupUUID1, groupUUID2},
		}}}, nil)

	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, groupUUID2).Return(&accountmanagement.PermissionsGroupDto{}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "environment", "abc12345").Return(&accountmanagement.LevelPolicyBindingDto{
		PolicyBindings: []accountmanagement.Binding{{
			PolicyUuid: "bc7df7b7-9387-45ff-974f-56573c072e4c",
			Groups:     []string{groupUUID1},
		}},
	}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, map[account.PolicyId]account.Policy{
		toID("account policy"): {
			ID:             toID("account policy"),
			Name:           "account policy",
			Level:          account.PolicyLevelAccount{Type: "account"},
			OriginObjectID: "2ff9314d-3c97-4607-bd49-460a53de1390",
		},
		toID("environment policy"): {
			ID:             toID("environment policy"),
			Name:           "environment policy",
			Level:          account.PolicyLevelEnvironment{Type: "environment", Environment: "abc12345"},
			OriginObjectID: "bc7df7b7-9387-45ff-974f-56573c072e4c",
		},
	}, result.Policies)
	assert.Equal(t, map[account.GroupId]account.Group{
		toID("test group"): {
			ID:             toID("test group"),
			Name:           "test group",
			OriginObjectID: groupUUID1,
			Account: &account.Account{
				Policies: []account.PolicyBinding{{Policy: account.Reference{Id: toID("account policy")}}},
			},
			Environment: []account.Environment{
				{
					Name:     "abc12345",
					Policies: []account.PolicyBinding{{Policy: account.Reference{Id: toID("environment policy")}}},
				},
			},
		},
		toID("second test group"): {
			ID:             toID("second test group"),
			Name:           "second test group",
			OriginObjectID: groupUUID2,
			Account: &account.Account{
				Policies: []account.PolicyBinding{{Policy: account.Reference{Id: toID("account policy")}}},
			},
		},
	}, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_GroupsWithPoliciesAndBoundaries tests that downloading groups with policies and boundaries succeeds.
func TestDownloader_GroupsWithPoliciesAndBoundaries(t *testing.T) {
	t.Setenv(featureflags.Boundaries.EnvName(), "true")
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	accountPolicyOverview := accountmanagement.PolicyOverview{
		Uuid:      "2ff9314d-3c97-4607-bd49-460a53de1390",
		Name:      "account policy",
		LevelType: "account",
	}

	environmentPolicyOverview := accountmanagement.PolicyOverview{
		Uuid:      "bc7df7b7-9387-45ff-974f-56573c072e4c",
		Name:      "environment policy",
		LevelId:   "abc12345",
		LevelType: "environment",
	}

	policyBoundaryOverview := accountmanagement.PolicyBoundaryOverview{
		Uuid:          boundaryUUID1,
		Name:          "boundary name",
		BoundaryQuery: "some boundary query",
	}

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{{Id: "abc12345"}}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{accountPolicyOverview, environmentPolicyOverview}, nil)

	client.EXPECT().GetPolicyDefinition(gomock.Any(), accountPolicyOverview).Return(&accountmanagement.LevelPolicyDto{}, nil)
	client.EXPECT().GetPolicyDefinition(gomock.Any(), environmentPolicyOverview).Return(&accountmanagement.LevelPolicyDto{}, nil)

	client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{policyBoundaryOverview}, nil)

	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{
		{
			Uuid: &groupUUID1,
			Name: "test group",
		},
		{
			Uuid: &groupUUID2,
			Name: "second test group",
		}}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "account", accountUUID).Return(&accountmanagement.LevelPolicyBindingDto{
		PolicyBindings: []accountmanagement.Binding{{
			PolicyUuid: "2ff9314d-3c97-4607-bd49-460a53de1390",
			Groups:     []string{groupUUID1, groupUUID2},
			Boundaries: []string{boundaryUUID1},
		}}}, nil)

	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, groupUUID1).Return(&accountmanagement.PermissionsGroupDto{}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "environment", "abc12345").Return(&accountmanagement.LevelPolicyBindingDto{
		PolicyBindings: []accountmanagement.Binding{{
			PolicyUuid: "bc7df7b7-9387-45ff-974f-56573c072e4c",
			Groups:     []string{groupUUID1},
			Boundaries: []string{boundaryUUID1},
		}},
	}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "account", accountUUID).Return(&accountmanagement.LevelPolicyBindingDto{
		PolicyBindings: []accountmanagement.Binding{{
			PolicyUuid: "2ff9314d-3c97-4607-bd49-460a53de1390",
			Groups:     []string{groupUUID1, groupUUID2},
			Boundaries: []string{boundaryUUID1},
		}}}, nil)

	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, groupUUID2).Return(&accountmanagement.PermissionsGroupDto{}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "environment", "abc12345").Return(&accountmanagement.LevelPolicyBindingDto{
		PolicyBindings: []accountmanagement.Binding{{
			PolicyUuid: "bc7df7b7-9387-45ff-974f-56573c072e4c",
			Groups:     []string{groupUUID1},
			Boundaries: []string{boundaryUUID1},
		}},
	}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	expectedBoundaries := map[account.BoundaryId]account.Boundary{
		toID("boundary name"): {
			ID:             toID("boundary name"),
			Name:           "boundary name",
			Query:          "some boundary query",
			OriginObjectID: boundaryUUID1,
		},
	}
	assert.Equal(t, expectedBoundaries, result.Boundaries)
	assert.Equal(t, map[account.PolicyId]account.Policy{
		toID("account policy"): {
			ID:             toID("account policy"),
			Name:           "account policy",
			Level:          account.PolicyLevelAccount{Type: "account"},
			OriginObjectID: "2ff9314d-3c97-4607-bd49-460a53de1390",
		},
		toID("environment policy"): {
			ID:             toID("environment policy"),
			Name:           "environment policy",
			Level:          account.PolicyLevelEnvironment{Type: "environment", Environment: "abc12345"},
			OriginObjectID: "bc7df7b7-9387-45ff-974f-56573c072e4c",
		},
	}, result.Policies)
	assert.Equal(t, map[account.GroupId]account.Group{
		toID("test group"): {
			ID:             toID("test group"),
			Name:           "test group",
			OriginObjectID: groupUUID1,
			Account: &account.Account{
				Policies: []account.PolicyBinding{
					{
						Policy:     account.Reference{Id: toID("account policy")},
						Boundaries: []account.Ref{account.Reference{Id: toID("boundary name")}},
					},
				},
			},
			Environment: []account.Environment{
				{
					Name: "abc12345",
					Policies: []account.PolicyBinding{
						{
							Policy:     account.Reference{Id: toID("environment policy")},
							Boundaries: []account.Ref{account.Reference{Id: toID("boundary name")}},
						},
					},
				},
			},
		},
		toID("second test group"): {
			ID:             toID("second test group"),
			Name:           "second test group",
			OriginObjectID: groupUUID2,
			Account: &account.Account{
				Policies: []account.PolicyBinding{
					{
						Policy:     account.Reference{Id: toID("account policy")},
						Boundaries: []account.Ref{account.Reference{Id: toID("boundary name")}},
					},
				},
			},
		},
	}, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_NoGroupDetails tests that downloading a group without details fails as expected.
func TestDownloader_NoGroupDetails(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{{Email: "test.user@some.org"}}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "test.user@some.org", accountUUID).Return(nil, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestDownloader_GroupsWithPermissions tests that downloading a group with permissions succeeds.
func TestDownloader_GroupsWithPermissions(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return(
		[]accountmanagement.TenantResourceDto{{
			Name: "tenant1",
			Id:   "abc12345",
		}}, []accountmanagement.ManagementZoneResourceDto{{
			Parent: "abc12345",
			Name:   "managementZone",
			Id:     "2698219524301731104",
		}}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)

	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{
		{
			Uuid: &groupUUID1,
			Name: "test group",
		},
		{
			Uuid: &groupUUID2,
			Name: "second test group",
		}}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "account", accountUUID).Return(&accountmanagement.LevelPolicyBindingDto{}, nil)

	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, groupUUID1).Return(&accountmanagement.PermissionsGroupDto{
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
	}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "environment", "abc12345").Return(&accountmanagement.LevelPolicyBindingDto{}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "account", accountUUID).Return(&accountmanagement.LevelPolicyBindingDto{}, nil)

	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, groupUUID2).Return(&accountmanagement.PermissionsGroupDto{
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
	}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), "environment", "abc12345").Return(&accountmanagement.LevelPolicyBindingDto{}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, map[account.PolicyId]account.Policy{}, result.Policies)
	assert.Equal(t, map[account.GroupId]account.Group{
		toID("test group"): {
			ID:             toID("test group"),
			Name:           "test group",
			OriginObjectID: groupUUID1,
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
			OriginObjectID: groupUUID2,
			Account: &account.Account{
				Permissions: []string{"account-viewer"},
			},
			ManagementZone: []account.ManagementZone{{
				Environment:    "abc12345",
				ManagementZone: "managementZone",
				Permissions:    []string{"tenant-view-security-problems", "tenant-viewer"},
			}},
		},
	}, result.Groups)
	assert.Empty(t, result.Users)
	assert.Empty(t, result.ServiceUsers)
}

// TestDownloader_OnlyServiceUser tests that downloading an account with a single service user succeeds.
func TestDownloader_OnlyServiceUser(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{{Uid: "abc1", Email: "service.user@some.org", Name: "service_user", Description: accountmanagement.PtrString("A service user")}}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "service.user@some.org", accountUUID).Return(&accountmanagement.GroupUserDto{Email: "service.user@some.org"}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Empty(t, result.Groups)
	assert.Empty(t, result.Users)
	assert.Equal(t, []account.ServiceUser{
		{OriginObjectID: "abc1", Name: "service_user", Description: "A service user"},
	}, result.ServiceUsers)
}

// TestDownloader_TwoServiceUsersWithSameName tests that downloading an account with two service users with the same name succeeds.
func TestDownloader_TwoServiceUsersWithSameName(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{
		{Uid: "abc1", Email: "abc1@some.org", Name: "service_user", Description: accountmanagement.PtrString("A service user")},
		{Uid: "abc2", Email: "abc2@some.org", Name: "service_user", Description: accountmanagement.PtrString("A service user")},
	}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "abc1@some.org", accountUUID).Return(&accountmanagement.GroupUserDto{Email: "abc1@some.org"}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "abc2@some.org", accountUUID).Return(&accountmanagement.GroupUserDto{Email: "abc2@some.org"}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Empty(t, result.Groups)
	assert.Empty(t, result.Users)
	assert.Equal(t, []account.ServiceUser{
		{OriginObjectID: "abc1", Name: "service_user", Description: "A service user"},
		{OriginObjectID: "abc2", Name: "service_user", Description: "A service user"},
	}, result.ServiceUsers)
}

// TestDownloader_ServiceUserWithOneGroup tests that downloading a service user belonging to one group succeeds.
func TestDownloader_ServiceUserWithOneGroup(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{{
		Uuid: &groupUUID1,
		Name: "test_group",
	}}, nil)

	client.EXPECT().GetPolicyGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(&accountmanagement.LevelPolicyBindingDto{}, nil)
	client.EXPECT().GetPermissionFor(gomock.Any(), accountUUID, gomock.Any()).Return(&accountmanagement.PermissionsGroupDto{}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return(nil, nil)

	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{{Email: "service.user@some.org", Name: "service_user", Description: accountmanagement.PtrString("A service user")}}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "service.user@some.org", accountUUID).Return(&accountmanagement.GroupUserDto{
		Email:  "service.user@some.org",
		Groups: []accountmanagement.AccountGroupDto{{Uuid: groupUUID1}},
	}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Policies)
	assert.Equal(t, map[account.GroupId]account.Group{
		toID("test_group"): {
			ID:             toID("test_group"),
			Name:           "test_group",
			OriginObjectID: groupUUID1,
		}}, result.Groups)
	assert.Empty(t, result.Users)

	assert.Equal(t, []account.ServiceUser{
		{Name: "service_user", Description: "A service user",
			Groups: []account.Ref{account.Reference{Id: toID("test_group")}},
		},
	}, result.ServiceUsers)
}

// TestDownloader_NoRequestedServiceUserDetails tests that downloading a service user without details fails as expected.
func TestDownloader_NoRequestedServiceUserDetails(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)

	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{{Email: "service.user@some.org", Name: "service_user", Description: accountmanagement.PtrString("A service user")}}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "service.user@some.org", accountUUID).Return(nil, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestDownloader_GetEnvironmentsAndMZonesErrors tests that downloading fails if GetEnvironmentsAndMZones errors.
func TestDownloader_GetEnvironmentsAndMZonesErrors(t *testing.T) {

	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return(nil, nil, originalErr)

	result, err := downloader.DownloadResources(t.Context())
	assert.Nil(t, result)
	assert.ErrorContains(t, err, originalErr.Error(), "Returned error must contain original error")
	assert.ErrorContains(t, err, "failed to get a list of environments and management zones for account", "Return error must contain additional information")
}

// TestDownloader_GetPoliciesErrors tests that downloading fails if GetPolicies errors.
func TestDownloader_GetPoliciesErrors(t *testing.T) {

	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return(nil, originalErr)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Nil(t, result)
	assert.ErrorContains(t, err, originalErr.Error(), "Returned error must contain original error")
	assert.ErrorContains(t, err, "failed to get a list of policies for account", "Return error must contain additional information")
}

// TestDownloader_GetUsersErrors tests that downloading fails if GetUsers errors.
func TestDownloader_GetUsersErrors(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return(nil, originalErr)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Nil(t, result)
	assert.ErrorContains(t, err, originalErr.Error(), "Returned error must contain original error")
	assert.ErrorContains(t, err, "failed to get a list of users for account", "Return error must contain additional information")
}

func TestDownloader_GetUsersWithMissingReference(t *testing.T) {
	// It may be the case during download that new groups are added and assigned to a user
	// Fetched groups -> fetching users **group-and-user-update-here** -> assigning the reference user.groups to fetched groups (configID)

	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{
		{
			Uid:   "uuid",
			Email: "mail",
		},
	}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "mail", accountUUID).Return(&accountmanagement.GroupUserDto{Email: "mail", Groups: []accountmanagement.AccountGroupDto{
		{
			GroupName: "new-group",
			Uuid:      "g-uuid",
		},
	}}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return([]accountmanagement.ExternalServiceUserDto{}, nil)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	require.NoError(t, err)
	assert.NotContains(t, result.Users["mail"].Groups, nil)
}

// TestDownloader_GetGroupsErrors tests that downloading fails if GetGroups errors.
func TestDownloader_GetGroupsErrors(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return(nil, originalErr)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Nil(t, result)
	assert.ErrorContains(t, err, originalErr.Error(), "Returned error must contain original error")
	assert.ErrorContains(t, err, "failed to get a list of groups for account", "Return error must contain additional information")
}

// TestDownloader_GetGroupsForUserErrors tests that downloading fails if GetGroupsForUsers errors.
func TestDownloader_GetGroupsForUserErrors(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{{Email: "usert@some.org"}}, nil)
	client.EXPECT().GetGroupsForUser(gomock.Any(), "usert@some.org", accountUUID).Return(nil, originalErr)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Nil(t, result)
	assert.ErrorContains(t, err, originalErr.Error(), "Returned error must contain original error")
	assert.ErrorContains(t, err, "failed to get a list of bind groups for user", "Return error must contain additional information")
}

// TestDownloader_GetServiceUsersErrors tests that downloading fails if GetServiceUsers errors.
func TestDownloader_GetServiceUsersErrors(t *testing.T) {
	client := http.NewMockhttpClient(gomock.NewController(t))
	downloader := downloader.NewForTesting(&account.AccountInfo{
		Name:        "test",
		AccountUUID: accountUUID,
	}, client)

	client.EXPECT().GetEnvironmentsAndMZones(gomock.Any(), accountUUID).Return([]accountmanagement.TenantResourceDto{}, []accountmanagement.ManagementZoneResourceDto{}, nil)
	client.EXPECT().GetPolicies(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyOverview{}, nil)
	client.EXPECT().GetGroups(gomock.Any(), accountUUID).Return([]accountmanagement.GetGroupDto{}, nil)
	client.EXPECT().GetUsers(gomock.Any(), accountUUID).Return([]accountmanagement.UsersDto{}, nil)
	client.EXPECT().GetServiceUsers(gomock.Any(), accountUUID).Return(nil, originalErr)
	if featureflags.Boundaries.Enabled() {
		client.EXPECT().GetBoundaries(gomock.Any(), accountUUID).Return([]accountmanagement.PolicyBoundaryOverview{}, nil)
	}

	result, err := downloader.DownloadResources(t.Context())
	assert.Nil(t, result)
	assert.ErrorContains(t, err, originalErr.Error(), "Returned error must contain original error")
	assert.ErrorContains(t, err, "failed to get a list of service users for account", "Return error must contain additional information")
}
