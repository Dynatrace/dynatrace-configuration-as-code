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
	"log/slog"
	"math"
	"net/http"

	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/accounts"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

type (
	Permissions    = accountmanagement.PermissionsDto
	Policy         = accountmanagement.CreateOrUpdateLevelPolicyRequestDto
	Boundary       = accountmanagement.PolicyBoundaryDto
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

func (c *accountManagementClient) getAccountInfo() account.AccountInfo {
	return c.accountInfo
}

func (c *accountManagementClient) getBoundaryIds(ctx context.Context) (map[string]remoteId, error) {
	boundaries, err := c.getBoundaries(ctx)
	if err != nil {
		return nil, err
	}

	result := map[string]remoteId{}
	for _, bnd := range boundaries {
		result[bnd.Name] = bnd.GetUuid()
	}
	return result, nil
}

func (c *accountManagementClient) getGlobalPolicies(ctx context.Context) (map[string]remoteId, error) {
	globalPolicies, resp, err := c.client.PolicyManagementAPI.GetLevelPolicies(ctx, "global", "global").Execute()
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

func (c *accountManagementClient) getAllGroups(ctx context.Context) (map[string]remoteId, error) {
	groups, resp, err := c.client.GroupManagementAPI.GetGroups(ctx, c.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable get all groups for account "+c.accountInfo.AccountUUID); err != nil {
		return nil, err
	}
	result := make(map[string]remoteId)
	for _, glP := range groups.GetItems() {
		result[glP.Name] = glP.GetUuid()
	}
	return result, nil

}

func (c *accountManagementClient) getManagementZones(ctx context.Context) ([]ManagementZone, error) {
	envResources, resp, err := c.client.EnvironmentManagementAPI.GetEnvironmentResources(ctx, c.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to get environment resources for account "+c.accountInfo.AccountUUID); err != nil {
		return nil, err
	}
	if envResources == nil {
		return []ManagementZone{}, nil
	}
	return envResources.ManagementZoneResources, nil
}

func (c *accountManagementClient) upsertBoundary(ctx context.Context, boundaryId string, boundary Boundary) (remoteId, error) {
	if boundaryId == "" {
		slog.DebugContext(ctx, "Trying to get boundary", slog.String("name", boundary.Name))
		bnd, err := c.getBoundaryByName(ctx, boundary.Name)
		if err != nil {
			var rnfErr *ResourceNotFoundError
			if !errors.As(err, &rnfErr) {
				return "", err
			}

			slog.DebugContext(ctx, "No boundary found. Creating a new one", slog.String("name", boundary.Name))
			return c.createBoundary(ctx, boundary)
		}
		boundaryId = bnd.Uuid
	}

	return c.updateBoundary(ctx, boundaryId, boundary)
}

func (c *accountManagementClient) updateBoundary(ctx context.Context, boundaryId string, boundary Boundary) (string, error) {
	slog.DebugContext(ctx, "Trying to update boundary", slog.String("name", boundary.Name), slog.String("uuid", boundaryId))
	_, resp, err := c.client.PolicyManagementAPI.PutPolicyBoundary(ctx, boundaryId, c.accountInfo.AccountUUID).PolicyBoundaryDto(boundary).Execute()
	defer closeResponseBody(resp)

	// handle a 404 here if need be as handleClientResponseError discards it!
	if is404(resp) {
		return "", ResourceNotFoundError{Identifier: boundaryId}
	}

	if err = handleClientResponseError(resp, err, "unable to update boundary with name: "+boundary.Name); err != nil {
		return "", err
	}
	return boundaryId, nil
}

func (c *accountManagementClient) createBoundary(ctx context.Context, boundary Boundary) (string, error) {
	createdBoundary, resp, err := c.client.PolicyManagementAPI.PostPolicyBoundary(ctx, c.accountInfo.AccountUUID).PolicyBoundaryDto(boundary).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to create boundary with name: "+boundary.Name); err != nil {
		return "", err
	}

	if createdBoundary == nil {
		return "", errors.New("the received data is empty")
	}

	return createdBoundary.Uuid, nil
}

func (c *accountManagementClient) getBoundaryByName(ctx context.Context, name string) (*accountmanagement.PolicyBoundaryOverview, error) {
	boundaries, err := c.getBoundaries(ctx)
	if err != nil {
		return nil, err
	}

	var foundBoundary *accountmanagement.PolicyBoundaryOverview
	for _, b := range boundaries {
		if b.Name == name {
			if foundBoundary != nil {
				return nil, fmt.Errorf("found multiple boundaries with name '%s'", name)
			}
			foundBoundary = &b
		}
	}
	if foundBoundary == nil {
		return nil, &ResourceNotFoundError{Identifier: name}
	}

	return foundBoundary, nil
}

func (c *accountManagementClient) getBoundaries(ctx context.Context) ([]accountmanagement.PolicyBoundaryOverview, error) {
	boundaries := []accountmanagement.PolicyBoundaryOverview{}
	const pageSize = 100
	for page := (int32)(1); page < math.MaxInt32; page++ {
		r, err := c.getBoundariesPage(ctx, c.accountInfo.AccountUUID, page, pageSize)
		if err != nil {
			return nil, err
		}

		boundaries = append(boundaries, r.Content...)
		// If the amount of boundaries returned on the page is less than the requested page size, we can assume it was the last page.
		if len(r.Content) < pageSize {
			break
		}
	}

	return boundaries, nil
}

func (c *accountManagementClient) getBoundariesPage(ctx context.Context, accountUUID string, page int32, pageSize int32) (*accountmanagement.PolicyBoundaryDtoList, error) {
	r, resp, err := c.client.PolicyManagementAPI.GetPolicyBoundaries(ctx, accountUUID).Page(page).Size(pageSize).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "failed to get boundaries"); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("the received data is empty")
	}
	return r, nil
}

func (c *accountManagementClient) upsertPolicy(ctx context.Context, policyLevel string, policyLevelId string, policyId string, policy Policy) (remoteId, error) {
	if policyId != "" {

		slog.DebugContext(ctx, "Trying to update policy", slog.String("uuid", policyId))
		_, resp, err := c.client.PolicyManagementAPI.UpdateLevelPolicy(ctx, policyId, policyLevelId, policyLevel).CreateOrUpdateLevelPolicyRequestDto(policy).Execute()
		defer closeResponseBody(resp)
		if err = handleClientResponseError(resp, err, "unable to update policy with UUID: "+policyId); err != nil {
			return "", err
		}
		return policyId, nil
	}

	slog.DebugContext(ctx, "Trying to get policy", slog.String("name", policy.Name))
	result, resp, err := c.client.PolicyManagementAPI.GetLevelPolicies(ctx, policyLevelId, policyLevel).Name(policy.Name).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to get policy with name: "+policy.Name); err != nil {
		return "", err
	}

	existingPolicies := result.GetPolicies()

	if len(existingPolicies) == 0 {
		slog.DebugContext(ctx, "No policy found. Creating a new one", slog.String("name", policy.Name))
		var createdPolicy *accountmanagement.LevelPolicyDto
		createdPolicy, resp, err = c.client.PolicyManagementAPI.CreateLevelPolicy(ctx, policyLevelId, policyLevel).CreateOrUpdateLevelPolicyRequestDto(policy).Execute()
		defer closeResponseBody(resp)
		if err = handleClientResponseError(resp, err, "unable to create policy with name: "+policy.Name); err != nil {
			return "", err
		}
		return createdPolicy.GetUuid(), nil
	}

	if len(existingPolicies) > 1 { // shouldn't happen
		slog.DebugContext(ctx, "Found multiple policies", slog.String("name", policy.Name), slog.String("uuid", existingPolicies[0].GetUuid()))
	}

	slog.DebugContext(ctx, "Trying to update existing policy", slog.String("name", policy.Name), slog.String("uuid", existingPolicies[0].GetUuid()))
	_, resp, err = c.client.PolicyManagementAPI.UpdateLevelPolicy(ctx, existingPolicies[0].GetUuid(), policyLevelId, policyLevel).CreateOrUpdateLevelPolicyRequestDto(policy).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to update policy with name: "+policy.Name); err != nil {
		return "", err
	}
	return existingPolicies[0].GetUuid(), nil
}

func (c *accountManagementClient) upsertGroup(ctx context.Context, groupId string, group Group) (remoteId, error) {
	if groupId != "" {
		slog.DebugContext(ctx, "Trying to update group", slog.String("id", groupId))
		existingGroup, err := c.getGroupByID(ctx, groupId)
		if err != nil {
			return "", err
		}

		return c.updateExistingGroup(ctx, *existingGroup, group)
	}

	existingGroupsWithName, err := c.getGroupsByName(ctx, group.Name)
	if err != nil {
		return "", err
	}

	if len(existingGroupsWithName) == 0 {
		return c.createGroup(ctx, group)
	}

	if len(existingGroupsWithName) > 1 { // shouldn't happen
		slog.DebugContext(ctx, "Updating multiple groups", slog.String("name", group.Name), slog.String("uuid", existingGroupsWithName[0].GetUuid()))
	}

	return c.updateExistingGroup(ctx, existingGroupsWithName[0], group)
}

func (c *accountManagementClient) getGroupByID(ctx context.Context, groupID string) (*accountmanagement.GetGroupDto, error) {
	result, resp, err := c.client.GroupManagementAPI.GetGroups(ctx, c.accountInfo.AccountUUID).Execute()
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

func (c *accountManagementClient) getGroupsByName(ctx context.Context, name string) ([]accountmanagement.GetGroupDto, error) {
	groupList, resp, err := c.client.GroupManagementAPI.GetGroups(ctx, c.accountInfo.AccountUUID).Execute()
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

func (c *accountManagementClient) createGroup(ctx context.Context, group Group) (remoteId, error) {
	var createdGroups []accountmanagement.GetGroupDto
	createdGroups, resp, err := c.client.GroupManagementAPI.
		CreateGroups(ctx, c.accountInfo.AccountUUID).
		InsertGroupDto([]accountmanagement.InsertGroupDto{
			accountmanagement.InsertGroupDto(group),
		}).
		Execute()

	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to create group with name: "+group.Name); err != nil {
		return "", err
	}
	if len(createdGroups) < 1 {
		return "", fmt.Errorf("unable to get UUID of created group with name: %s", group.Name)
	}
	return createdGroups[0].GetUuid(), nil
}

func (c *accountManagementClient) updateExistingGroup(ctx context.Context, existingGroup accountmanagement.GetGroupDto, group Group) (remoteId, error) {
	// Groups with owner "SCIM" or "ALL_USERS" cannot be modified and so updates should be skipped
	if featureflags.SkipReadOnlyAccountGroupUpdates.Enabled() && ((existingGroup.Owner == "SCIM") || (existingGroup.Owner == "ALL_USERS")) {
		return existingGroup.GetUuid(), nil
	}

	resp, err := c.client.GroupManagementAPI.EditGroup(ctx, c.accountInfo.AccountUUID, existingGroup.GetUuid()).PutGroupDto(group).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to update group with UUID: "+existingGroup.GetUuid()); err != nil {
		return "", err
	}
	return existingGroup.GetUuid(), nil
}

func (c *accountManagementClient) upsertUser(ctx context.Context, userId string) (remoteId, error) {
	_, resp, err := c.client.UserManagementAPI.GetUserGroups(ctx, c.accountInfo.AccountUUID, userId).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to create user with email: "+userId); err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp, err = c.client.UserManagementAPI.CreateUserForAccount(ctx, c.accountInfo.AccountUUID).UserEmailDto(accountmanagement.UserEmailDto{Email: userId}).Execute()
		defer closeResponseBody(resp)
		if err = handleClientResponseError(resp, err, "unable to create user with email: "+userId); err != nil {
			return "", err
		}

		return userId, nil
	}

	return userId, nil
}

func (c *accountManagementClient) upsertServiceUser(ctx context.Context, serviceUserId string, data ServiceUser) (remoteId, error) {
	if serviceUserId == "" {
		suId, err := c.getServiceUserIDByName(ctx, data.Name)
		if err != nil {
			var rnfErr *ResourceNotFoundError
			if !errors.As(err, &rnfErr) {
				return "", err
			}

			return c.createServiceUser(ctx, data)
		}
		serviceUserId = suId
	}

	return c.updateServiceUser(ctx, serviceUserId, data)
}

func (c *accountManagementClient) createServiceUser(ctx context.Context, dto accountmanagement.ServiceUserDto) (string, error) {
	externalServiceUserWithGroupUuidDto, resp, err := c.client.ServiceUserManagementAPI.CreateServiceUserForAccount(ctx, c.accountInfo.AccountUUID).ServiceUserDto(dto).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "failed to create service user"); err != nil {
		return "", err
	}

	if externalServiceUserWithGroupUuidDto == nil {
		return "", errors.New("the received data are empty")
	}

	return externalServiceUserWithGroupUuidDto.Uid, nil
}

func (c *accountManagementClient) updateServiceUser(ctx context.Context, serviceUserId string, dto accountmanagement.ServiceUserDto) (string, error) {
	resp, err := c.client.ServiceUserManagementAPI.UpdateServiceUserForAccount(ctx, c.accountInfo.AccountUUID, serviceUserId).ServiceUserDto(dto).Execute()
	defer closeResponseBody(resp)

	// handle a 404 here if need be as handleClientResponseError discards it!
	if is404(resp) {
		return "", ResourceNotFoundError{Identifier: serviceUserId}
	}

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

func (c *accountManagementClient) getServiceUserByUid(ctx context.Context, uid string) (*accountmanagement.ExternalServiceUserWithGroupUuidDto, error) {
	serviceUser, resp, err := c.client.ServiceUserManagementAPI.GetServiceUser(ctx, c.accountInfo.AccountUUID, uid).Execute()
	defer closeResponseBody(resp)

	if is404(resp) {
		return nil, ResourceNotFoundError{Identifier: uid}
	}
	if err = handleClientResponseError(resp, err, "failed to get service users"); err != nil {
		return nil, err
	}
	if serviceUser == nil {
		return nil, errors.New("the received data are empty")
	}
	return serviceUser, nil
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

func (c *accountManagementClient) updatePermissions(ctx context.Context, groupId string, permissions []accountmanagement.PermissionsDto) error {
	if groupId == "" {
		return fmt.Errorf("group id must not be empty")
	}

	if permissions == nil {
		permissions = []accountmanagement.PermissionsDto{}
	}

	resp, err := c.client.PermissionManagementAPI.OverwriteGroupPermissions(ctx, c.accountInfo.AccountUUID, groupId).PermissionsDto(permissions).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to update permissions of group with UUID "+groupId); err != nil {
		return err
	}

	return nil
}

func (c *accountManagementClient) updateAccountPolicyBindings(ctx context.Context, groupId string, boundariesForPolicyIds map[string][]string) error {
	if groupId == "" {
		return fmt.Errorf("group id must not be empty")
	}

	policyIds := []string{}
	if boundariesForPolicyIds != nil {
		policyIds = maps.Keys(boundariesForPolicyIds)
	}
	data := accountmanagement.PolicyUuidsDto{PolicyUuids: policyIds}

	resp, err := c.client.PolicyManagementAPI.UpdatePolicyBindingsToGroup(ctx, groupId, c.accountInfo.AccountUUID, "account").PolicyUuidsDto(data).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to update policy binding between group with UUID "+groupId+" and policies with UUIDs "+fmt.Sprintf("%v", policyIds)); err != nil {
		return err
	}

	for policyId, boundaryIds := range boundariesForPolicyIds {
		if err := c.updateBoundariesForPolicyBinding(ctx, "account", c.accountInfo.AccountUUID, groupId, policyId, boundaryIds); err != nil {
			return err
		}
	}
	return nil
}

func (c *accountManagementClient) updateEnvironmentPolicyBindings(ctx context.Context, envName string, groupId string, boundariesForPolicyIds map[string][]string) error {
	if envName == "" {
		return fmt.Errorf("environment name must not be empty")
	}
	if groupId == "" {
		return fmt.Errorf("group id must not be empty")
	}
	var policyIds []string
	if boundariesForPolicyIds == nil {
		policyIds = []string{}
	} else {
		policyIds = maps.Keys(boundariesForPolicyIds)
	}
	data := accountmanagement.PolicyUuidsDto{PolicyUuids: policyIds}
	resp, err := c.client.PolicyManagementAPI.UpdatePolicyBindingsToGroup(ctx, groupId, envName, "environment").PolicyUuidsDto(data).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to update policy binding between group with UUID "+groupId+" and policies with UUIDs "+fmt.Sprintf("%v", policyIds)); err != nil {
		return err
	}

	for policyId, boundaryIds := range boundariesForPolicyIds {
		if err := c.updateBoundariesForPolicyBinding(ctx, "environment", envName, groupId, policyId, boundaryIds); err != nil {
			return err
		}
	}
	return nil
}

func (c *accountManagementClient) deleteAllEnvironmentPolicyBindings(ctx context.Context, groupId string) error {
	environments, resp, err := c.client.EnvironmentManagementAPI.GetEnvironments(ctx, c.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to get all environments for account with id"+c.accountInfo.AccountUUID); err != nil {
		return err
	}

	for _, e := range environments.Data {
		if !e.Active {
			// inactive environments can't be updated and result in a 403
			continue
		}
		policies, resp, err := c.client.PolicyManagementAPI.GetPolicyUuidsBindings(ctx, groupId, e.Id, "environment").Execute()
		closeResponseBody(resp)
		if err = handleClientResponseError(resp, err, "unable to list all environments policy bindings for account with UUID "+c.accountInfo.AccountUUID+" and group with UUID "+groupId); err != nil {
			return err
		}
		for _, pol := range policies.PolicyUuids {
			resp, err = c.client.PolicyManagementAPI.DeleteLevelPolicyBindingsForPolicyAndGroup(ctx, groupId, pol, e.Id, "environment").ForceMultiple(true).Execute()
			closeResponseBody(resp)
			if err = handleClientResponseError(resp, err, "unable to delete all environments policy bindings for account with UUID "+c.accountInfo.AccountUUID+" and group with UUID "+groupId); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *accountManagementClient) updateBoundariesForPolicyBinding(ctx context.Context, levelType string, levelId string, groupId string, policyId string, boundaryIds []string) error {
	if groupId == "" {
		return fmt.Errorf("group id must not be empty")
	}
	if policyId == "" {
		return fmt.Errorf("policy id must not be empty")
	}
	if boundaryIds == nil {
		boundaryIds = []string{}
	}

	data := accountmanagement.AppendLevelPolicyBindingForGroupDto{Boundaries: boundaryIds}
	resp, err := c.client.PolicyManagementAPI.UpdateLevelPolicyBindingForPolicyAndGroup(ctx, groupId, policyId, levelId, levelType).AppendLevelPolicyBindingForGroupDto(data).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, fmt.Sprintf("unable to update boundaries for policy binding between group with UUID %s, policy with UUID %s and boundaries with UUIDs %v", groupId, policyId, boundaryIds)); err != nil {
		return err
	}
	return nil
}

func (c *accountManagementClient) updateGroupBindings(ctx context.Context, userId string, groupIds []string) error {
	if userId == "" {
		return fmt.Errorf("user id must not be empty")
	}
	if groupIds == nil {
		groupIds = []string{}
	}
	resp, err := c.client.UserManagementAPI.ReplaceUserGroups(ctx, c.accountInfo.AccountUUID, userId).RequestBody(groupIds).Execute()
	defer closeResponseBody(resp)
	if err = handleClientResponseError(resp, err, "unable to add user "+userId+" to groups "+fmt.Sprintf("%v", groupIds)); err != nil {
		return err
	}
	return nil
}

func is404(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
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
