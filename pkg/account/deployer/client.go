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

package deployer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

type (
	Permissions    = accountmanagement.PermissionsDto
	Policy         = accountmanagement.CreateOrUpdateLevelPolicyRequestDto
	Group          = accountmanagement.PutGroupDto
	ServiceUser    = accountmanagement.ServiceUserDto
	ManagementZone = accountmanagement.ManagementZoneResourceDto

	accountManagementClient struct {
		accountInfo account.AccountInfo
		client      *accounts.Client
	}
)

func NewClient(info account.AccountInfo, client *accounts.Client) *accountManagementClient {
	return &accountManagementClient{
		accountInfo: info,
		client:      client,
	}
}

func (d *accountManagementClient) getAccountInfo() account.AccountInfo {
	return d.accountInfo
}

func (d *accountManagementClient) getGlobalPolicies(ctx context.Context) (map[string]remoteId, error) {
	globalPolicies, resp, err := d.client.PolicyManagementAPI.GetLevelPolicies(ctx, "global", "global").Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable get global policies"); err != nil {
		return nil, err
	}

	result := make(map[string]remoteId)
	for _, glP := range globalPolicies.GetPolicies() {
		result[glP.Name] = glP.GetUuid()
	}
	return result, nil
}

func (d *accountManagementClient) getAllGroups(ctx context.Context) (map[string]remoteId, error) {
	groups, resp, err := d.client.GroupManagementAPI.GetGroups(ctx, d.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable get all groups for account "+d.accountInfo.AccountUUID); err != nil {
		return nil, err
	}
	result := make(map[string]remoteId)
	for _, glP := range groups.GetItems() {
		result[glP.Name] = glP.GetUuid()
	}
	return result, nil

}

func (d *accountManagementClient) getManagementZones(ctx context.Context) ([]ManagementZone, error) {
	envResources, resp, err := d.client.EnvironmentManagementAPI.GetEnvironmentResources(ctx, d.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to get environment resources for account "+d.accountInfo.AccountUUID); err != nil {
		return nil, err
	}
	if envResources == nil {
		return []ManagementZone{}, nil
	}
	return envResources.ManagementZoneResources, nil
}

func (d *accountManagementClient) upsertPolicy(ctx context.Context, policyLevel string, policyLevelId string, policyId string, policy Policy) (remoteId, error) {
	if policyId != "" {

		logr.FromContextOrDiscard(ctx).V(1).Info("Trying to update policy with origin object ID (UUID) " + policyId)
		_, resp, err := d.client.PolicyManagementAPI.UpdateLevelPolicy(ctx, policyId, policyLevelId, policyLevel).CreateOrUpdateLevelPolicyRequestDto(policy).Execute()
		defer closeResponseBody(resp)
		if err = handleClientResponseError(resp, err, "unable to update policy with UUID: "+policyId); err != nil {
			return "", err
		}
		return policyId, nil
	}

	logr.FromContextOrDiscard(ctx).V(1).Info("Trying to get policy with name " + policy.Name)
	result, resp, err := d.client.PolicyManagementAPI.GetLevelPolicies(ctx, policyLevelId, policyLevel).Name(policy.Name).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to get policy with name: "+policy.Name); err != nil {
		return "", err
	}

	existingPolicies := result.GetPolicies()

	if len(existingPolicies) == 0 {
		logr.FromContextOrDiscard(ctx).V(1).Info("No policy with name " + policy.Name + " found. Creating a new one")
		var createdPolicy *accountmanagement.LevelPolicyDto
		createdPolicy, resp, err = d.client.PolicyManagementAPI.CreateLevelPolicy(ctx, policyLevelId, policyLevel).CreateOrUpdateLevelPolicyRequestDto(policy).Execute()
		defer closeResponseBody(resp)
		if err = handleClientResponseError(resp, err, "unable to create policy with name: "+policy.Name); err != nil {
			return "", err
		}
		return createdPolicy.GetUuid(), nil
	}

	if len(existingPolicies) > 1 { // shouldn't happen
		logr.FromContextOrDiscard(ctx).V(-1).Info("Found multiple policies with name " + policy.Name + ". Updating policy with UUID " + existingPolicies[0].GetUuid())
	}

	logr.FromContextOrDiscard(ctx).V(1).Info("Trying to update existing policy with name " + policy.Name + " and UUID " + existingPolicies[0].GetUuid())
	_, resp, err = d.client.PolicyManagementAPI.UpdateLevelPolicy(ctx, existingPolicies[0].GetUuid(), policyLevelId, policyLevel).CreateOrUpdateLevelPolicyRequestDto(policy).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to update policy with name: "+policy.Name); err != nil {
		return "", err
	}
	return existingPolicies[0].GetUuid(), nil
}

func (d *accountManagementClient) upsertGroup(ctx context.Context, groupId string, group Group) (remoteId, error) {
	if groupId != "" {
		logr.FromContextOrDiscard(ctx).V(1).Info("Trying to update group with origin object ID (UUID) " + groupId)
		existingGroup, err := d.getGroupByID(ctx, groupId)
		if err != nil {
			return "", err
		}

		return d.updateExistingGroup(ctx, *existingGroup, group)
	}

	existingGroupsWithName, err := d.getGroupsByName(ctx, group.Name)
	if err != nil {
		return "", err
	}

	if len(existingGroupsWithName) == 0 {
		return d.createGroup(ctx, group)
	}

	if len(existingGroupsWithName) > 1 { // shouldn't happen
		logr.FromContextOrDiscard(ctx).V(-1).Info("Updating multiple policies with name " + group.Name + ". Updating group with UUID " + existingGroupsWithName[0].GetUuid())
	}

	return d.updateExistingGroup(ctx, existingGroupsWithName[0], group)
}

func (d *accountManagementClient) getGroupByID(ctx context.Context, groupID string) (*accountmanagement.GetGroupDto, error) {
	result, resp, err := d.client.GroupManagementAPI.GetGroups(ctx, d.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to get group with ID: "+groupID); err != nil {
		return nil, err
	}

	for _, g := range result.GetItems() {
		if g.GetUuid() == groupID {
			return &g, nil
		}
	}

	return nil, fmt.Errorf("unable to get group with ID: %s", groupID)
}

func (d *accountManagementClient) getGroupsByName(ctx context.Context, name string) ([]accountmanagement.GetGroupDto, error) {
	groupList, resp, err := d.client.GroupManagementAPI.GetGroups(ctx, d.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to get group with name: "+name); err != nil {
		return nil, err
	}

	var groupsMatchingName []accountmanagement.GetGroupDto
	for _, g := range groupList.GetItems() {
		if g.GetName() == name {
			groupsMatchingName = append(groupsMatchingName, g)
		}
	}

	return groupsMatchingName, nil
}

func (d *accountManagementClient) createGroup(ctx context.Context, group Group) (remoteId, error) {
	var createdGroups []accountmanagement.GetGroupDto
	createdGroups, resp, err := d.client.GroupManagementAPI.CreateGroups(ctx, d.accountInfo.AccountUUID).PutGroupDto([]accountmanagement.PutGroupDto{group}).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to create group with name: "+group.Name); err != nil {
		return "", err
	}
	if len(createdGroups) < 1 {
		return "", fmt.Errorf("unable to get UUID of created group with name: %s", group.Name)
	}
	return createdGroups[0].GetUuid(), nil
}

func (d *accountManagementClient) updateExistingGroup(ctx context.Context, existingGroup accountmanagement.GetGroupDto, group Group) (remoteId, error) {
	// Groups with owner "SCIM" or "ALL_USERS" cannot be modified and so updates should be skipped
	if featureflags.SkipReadOnlyAccountGroupUpdates.Enabled() && ((existingGroup.Owner == "SCIM") || (existingGroup.Owner == "ALL_USERS")) {
		return existingGroup.GetUuid(), nil
	}

	resp, err := d.client.GroupManagementAPI.EditGroup(ctx, d.accountInfo.AccountUUID, existingGroup.GetUuid()).PutGroupDto(group).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to update group with UUID: "+existingGroup.GetUuid()); err != nil {
		return "", err
	}
	return existingGroup.GetUuid(), nil
}

func (d *accountManagementClient) upsertUser(ctx context.Context, userId string) (remoteId, error) {
	_, resp, err := d.client.UserManagementAPI.GetUserGroups(ctx, d.accountInfo.AccountUUID, userId).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to create user with email: "+userId); err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp, err = d.client.UserManagementAPI.CreateUserForAccount(ctx, d.accountInfo.AccountUUID).UserEmailDto(accountmanagement.UserEmailDto{Email: userId}).Execute()
		defer closeResponseBody(resp)
		if err = handleClientResponseError(resp, err, "unable to create user with email: "+userId); err != nil {
			return "", err
		}

		return userId, nil
	}

	return userId, nil
}

func (d *accountManagementClient) upsertServiceUser(ctx context.Context, serviceUserId string, data ServiceUser) (remoteId, error) {
	if serviceUserId == "" {
		suId, err := d.getServiceUserIDByName(ctx, data.Name)
		if err != nil {
			var rnfErr *ResourceNotFoundError
			if !errors.As(err, &rnfErr) {
				return "", err
			}

			return d.createServiceUser(ctx, data)
		}
		serviceUserId = suId
	}

	return d.updateServiceUser(ctx, serviceUserId, data)
}

func (d *accountManagementClient) createServiceUser(ctx context.Context, dto accountmanagement.ServiceUserDto) (string, error) {
	uuidDto, resp, err := d.client.ServiceUserManagementAPI.CreateServiceUserForAccount(ctx, d.accountInfo.AccountUUID).ServiceUserDto(dto).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "failed to create service user"); err != nil {
		return "", err
	}

	if uuidDto == nil {
		return "", errors.New("the received data are empty")
	}

	return uuidDto.Uuid, nil
}

func (d *accountManagementClient) updateServiceUser(ctx context.Context, serviceUserId string, dto accountmanagement.ServiceUserDto) (string, error) {
	resp, err := d.client.ServiceUserManagementAPI.UpdateServiceUserForAccount(ctx, d.accountInfo.AccountUUID, serviceUserId).ServiceUserDto(dto).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "failed to update service user"); err != nil {
		return "", err
	}

	return serviceUserId, nil
}

// ResourceNotFoundError is an error signifying that the desired resource was not found.
type ResourceNotFoundError struct {
	Identifier string
}

func (e ResourceNotFoundError) Error() string {
	return fmt.Sprintf("resource '%s' not found", e.Identifier)
}

func (c *accountManagementClient) getServiceUserIDByName(ctx context.Context, name string) (string, error) {
	serviceUser, err := c.getServiceUserByName(ctx, name)
	if err != nil {
		return "", err
	}

	return serviceUser.Uid, nil
}

func (c *accountManagementClient) getServiceUserEmailByName(ctx context.Context, name string) (string, error) {
	serviceUser, err := c.getServiceUserByName(ctx, name)
	if err != nil {
		return "", err
	}

	return serviceUser.Email, nil
}

func (c *accountManagementClient) getServiceUserByName(ctx context.Context, name string) (*accountmanagement.ExternalServiceUserDto, error) {
	serviceUsers, err := c.getServiceUsers(ctx)
	if err != nil {
		return nil, err
	}

	var foundServiceUser *accountmanagement.ExternalServiceUserDto
	for _, s := range serviceUsers {
		if s.Name == name {
			if foundServiceUser != nil {
				return nil, fmt.Errorf("found multiple service users with name '%s'", name)
			}
			foundServiceUser = &s
		}
	}
	if foundServiceUser == nil {
		return nil, &ResourceNotFoundError{Identifier: name}
	}

	return foundServiceUser, nil
}

func (c *accountManagementClient) getServiceUserEmailByUid(ctx context.Context, uid string) (string, error) {
	serviceUser, err := c.getServiceUserByUid(ctx, uid)
	if err != nil {
		return "", err
	}

	return serviceUser.Email, nil
}

func (c *accountManagementClient) getServiceUserByUid(ctx context.Context, uid string) (*accountmanagement.ExternalServiceUserDto, error) {
	serviceUsers, err := c.getServiceUsers(ctx)
	if err != nil {
		return nil, err
	}

	var foundServiceUser *accountmanagement.ExternalServiceUserDto
	for _, s := range serviceUsers {
		if s.Uid == uid {
			if foundServiceUser != nil {
				return nil, fmt.Errorf("found multiple service users with id '%s'", uid)
			}
			foundServiceUser = &s
		}
	}
	if foundServiceUser == nil {
		return nil, &ResourceNotFoundError{Identifier: uid}
	}

	return foundServiceUser, nil
}

func (c *accountManagementClient) getServiceUsers(ctx context.Context) ([]accountmanagement.ExternalServiceUserDto, error) {
	serviceUsers := []accountmanagement.ExternalServiceUserDto{}
	const pageSize = 1000
	page := (int32)(1)
	for {
		r, err := c.getServiceUsersPage(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}

		serviceUsers = append(serviceUsers, r.Results...)

		if r.NextPageKey == nil {
			break
		}
		page++
	}

	return serviceUsers, nil
}

func (c *accountManagementClient) getServiceUsersPage(ctx context.Context, page int32, pageSize int32) (*accountmanagement.ExternalServiceUsersPageDto, error) {
	r, resp, err := c.client.ServiceUserManagementAPI.GetServiceUsersFromAccount(ctx, c.accountInfo.AccountUUID).Page(page).PageSize(pageSize).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "failed to get service users"); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("the received data are empty")
	}
	return r, nil
}

func (d *accountManagementClient) updatePermissions(ctx context.Context, groupId string, permissions []accountmanagement.PermissionsDto) error {
	if groupId == "" {
		return fmt.Errorf("group id must not be empty")
	}

	if permissions == nil {
		permissions = []accountmanagement.PermissionsDto{}
	}

	resp, err := d.client.PermissionManagementAPI.OverwriteGroupPermissions(ctx, d.accountInfo.AccountUUID, groupId).PermissionsDto(permissions).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to update permissions of group with UUID "+groupId); err != nil {
		return err
	}

	return nil
}

func (d *accountManagementClient) updateAccountPolicyBindings(ctx context.Context, groupId string, policyIds []string) error {
	if groupId == "" {
		return fmt.Errorf("group id must not be empty")
	}
	if policyIds == nil {
		policyIds = []string{}
	}
	data := accountmanagement.PolicyUuidsDto{PolicyUuids: policyIds}

	resp, err := d.client.PolicyManagementAPI.UpdatePolicyBindingsToGroup(ctx, groupId, d.accountInfo.AccountUUID, "account").PolicyUuidsDto(data).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to update policy binding between group with UUID "+groupId+" and policies with UUIDs "+fmt.Sprintf("%v", policyIds)); err != nil {
		return err
	}

	return nil
}

func (d *accountManagementClient) updateEnvironmentPolicyBindings(ctx context.Context, envName string, groupId string, policyIds []string) error {
	if envName == "" {
		return fmt.Errorf("environment name must not be empty")
	}
	if groupId == "" {
		return fmt.Errorf("group id must not be empty")
	}
	if policyIds == nil {
		policyIds = []string{}
	}
	data := accountmanagement.PolicyUuidsDto{PolicyUuids: policyIds}
	resp, err := d.client.PolicyManagementAPI.UpdatePolicyBindingsToGroup(ctx, groupId, envName, "environment").PolicyUuidsDto(data).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to update policy binding between group with UUID "+groupId+" and policies with UUIDs "+fmt.Sprintf("%v", policyIds)); err != nil {
		return err
	}
	return nil
}

func (d *accountManagementClient) deleteAllEnvironmentPolicyBindings(ctx context.Context, groupId string) error {
	environments, resp, err := d.client.EnvironmentManagementAPI.GetEnvironments(ctx, d.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to get all environments for account with id"+d.accountInfo.AccountUUID); err != nil {
		return err
	}

	for _, e := range environments.Data {
		policies, resp, err := d.client.PolicyManagementAPI.GetPolicyUuidsBindings(ctx, groupId, e.Id, "environment").Execute()
		closeResponseBody(resp)
		if err = handleClientResponseError(resp, err, "unable to list all environments policy bindings for account with UUID "+d.accountInfo.AccountUUID+" and group with UUID "+groupId); err != nil {
			return err
		}
		for _, pol := range policies.PolicyUuids {
			resp, err = d.client.PolicyManagementAPI.DeleteLevelPolicyBindingsForPolicyAndGroup(ctx, groupId, pol, e.Id, "environment").ForceMultiple(true).Execute()
			closeResponseBody(resp)
			if err = handleClientResponseError(resp, err, "unable to delete all environments policy bindings for account with UUID "+d.accountInfo.AccountUUID+" and group with UUID "+groupId); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *accountManagementClient) updateGroupBindings(ctx context.Context, userId string, groupIds []string) error {
	if userId == "" {
		return fmt.Errorf("user id must not be empty")
	}
	if groupIds == nil {
		groupIds = []string{}
	}
	resp, err := d.client.UserManagementAPI.ReplaceUserGroups(ctx, d.accountInfo.AccountUUID, userId).RequestBody(groupIds).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to add user "+userId+" to groups "+fmt.Sprintf("%v", groupIds)); err != nil {
		return err
	}
	return nil
}

func handleClientResponseError(resp *http.Response, clientErr error, errMessage string) error {
	if clientErr != nil && (resp == nil || rest.IsSuccess(resp)) {
		return fmt.Errorf(errMessage+": %w", clientErr)
	}

	if !rest.IsSuccess(resp) && resp.StatusCode != http.StatusNotFound {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		return fmt.Errorf(errMessage+" (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func closeResponseBody(resp *http.Response) {
	if resp != nil {
		_ = resp.Body.Close()
	}
}
