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
	"fmt"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

type (
	AccountInfo struct {
		Name        string
		AccountUUID string
	}
	localId    = string // local (monaco related) identifier
	envName    = string // dt environment name
	remoteId   = string // dt entity identifier
	idLookupFn func(id localId) remoteId
)

type Options struct {
	DryRun bool
}

//go:generate mockgen -source=deployer.go -destination=client_mock.go -package=deployer client
type client interface {
	getAllGroups(ctx context.Context) (map[string]remoteId, error)
	getGlobalPolicies(ctx context.Context) (map[string]remoteId, error)
	getManagementZones(ctx context.Context) ([]ManagementZone, error)
	upsertPolicy(ctx context.Context, policyLevel string, policyLevelId string, policyId string, policy Policy) (remoteId, error)
	upsertGroup(ctx context.Context, groupId string, group Group) (remoteId, error)
	upsertUser(ctx context.Context, userId string) (remoteId, error)
	updateAccountPolicyBindings(ctx context.Context, groupId string, policyIds []string) error
	updateEnvironmentPolicyBindings(ctx context.Context, envName string, groupId string, policyIds []string) error
	deleteAllEnvironmentPolicyBindings(ctx context.Context, groupId string) error
	updateGroupBindings(ctx context.Context, userId string, groupIds []string) error
	updatePermissions(ctx context.Context, groupId string, permissions []accountmanagement.PermissionsDto) error
	getAccountInfo() AccountInfo
}

type AccountDeployer struct {
	accountManagementClient client

	deployedPolicies  map[localId]remoteId
	deployedGroups    map[localId]remoteId
	deployedMgmtZones []accountmanagement.ManagementZoneResourceDto
}

func NewAccountDeployer(client client) *AccountDeployer {
	return &AccountDeployer{
		accountManagementClient: client,

		deployedPolicies: make(map[localId]remoteId),
		deployedGroups:   make(map[localId]remoteId),
	}
}

func (d *AccountDeployer) Deploy(res *account.Resources) error {
	var err error
	d.deployedPolicies, err = d.accountManagementClient.getGlobalPolicies(context.TODO())
	if err != nil {
		return err
	}

	d.deployedMgmtZones, err = d.accountManagementClient.getManagementZones(context.TODO())
	if err != nil {
		return err
	}

	d.deployedGroups, err = d.accountManagementClient.getAllGroups(context.TODO())
	if err != nil {
		return err
	}

	if err = d.deployPolicies(res.Policies); err != nil {
		return err
	}

	if err = d.deployGroups(res.Groups); err != nil {
		return err
	}

	if err = d.deployUsers(res.Users); err != nil {
		return err
	}

	return nil
}

func (d *AccountDeployer) deployPolicies(policies map[string]account.Policy) error {
	for _, policy := range policies {
		log.Info("Deploying policy %s to account %s...", policy.Name, d.accountManagementClient.getAccountInfo().Name)
		pUuid, err := d.upsertPolicy(context.TODO(), policy)
		if err != nil {
			return fmt.Errorf("unable to deploy policy for account %q: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err)
		}
		d.deployedPolicies[policy.ID] = pUuid
	}
	return nil
}

func (d *AccountDeployer) deployGroups(groups map[string]account.Group) error {
	for _, group := range groups {
		log.Info("Deploying group %s to account %s...", group.Name, d.accountManagementClient.getAccountInfo().Name)
		gUuid, err := d.upsertGroup(context.TODO(), group)
		if err != nil {
			return fmt.Errorf("unable to deploy group for account %q: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err)
		}
		d.deployedGroups[group.ID] = gUuid

		log.Info("Updating policy bindings and permissions...")
		if err = d.updateGroupPolicyBindings(context.TODO(), group); err != nil {
			return fmt.Errorf("unable to deploy policy binding for account %q: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err)
		}

		if err = d.updateGroupPermissions(context.TODO(), group); err != nil {
			return fmt.Errorf("unable to deploy permissions for account %q: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err)
		}
	}
	return nil
}

func (d *AccountDeployer) deployUsers(users map[string]account.User) error {
	for _, user := range users {
		log.Info("Deploying user %s to account %s...", user.Email, d.accountManagementClient.getAccountInfo().Name)
		if _, err := d.upsertUser(context.TODO(), user); err != nil {
			return fmt.Errorf("unable to deploy user for account %q: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err)
		}
		log.Info("Updating group bindings...")
		if err := d.updateUserGroupBindings(context.TODO(), user); err != nil {
			return fmt.Errorf("unable to deploy user binding for account %q: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err)
		}
	}
	return nil
}

func (d *AccountDeployer) upsertPolicy(ctx context.Context, policy account.Policy) (remoteId, error) {
	var policyLevel string
	var policyLevelID string

	if _, ok := policy.Level.(account.PolicyLevelAccount); ok {
		policyLevel = "account"
		policyLevelID = d.accountManagementClient.getAccountInfo().AccountUUID
	}
	if p, ok := policy.Level.(account.PolicyLevelEnvironment); ok {
		policyLevel = "environment"
		policyLevelID = p.Environment
	}
	data := accountmanagement.CreateOrUpdateLevelPolicyRequestDto{
		Name:           policy.Name,
		Description:    policy.Description,
		StatementQuery: policy.Policy,
	}

	return d.accountManagementClient.upsertPolicy(ctx, policyLevel, policyLevelID, policy.OriginObjectID, data)
}

func (d *AccountDeployer) upsertGroup(ctx context.Context, group account.Group) (remoteId, error) {
	data := accountmanagement.PutGroupDto{
		Name:        group.Name,
		Description: &group.Description,
	}
	return d.accountManagementClient.upsertGroup(ctx, group.OriginObjectID, data)
}

func (d *AccountDeployer) upsertUser(ctx context.Context, user account.User) (remoteId, error) {
	return d.accountManagementClient.upsertUser(ctx, user.Email)
}

func (d *AccountDeployer) updateGroupPolicyBindings(ctx context.Context, group account.Group) error {
	remoteGroupId := d.groupIdLookup(group.ID)
	if remoteGroupId == "" {
		return fmt.Errorf("unable to determine UUID for group %q", group.Name)
	}
	remoteIds, err := d.getAccountPolicyRefs(group)
	if err != nil {
		return err
	}

	if err = d.accountManagementClient.updateAccountPolicyBindings(ctx, remoteGroupId, remoteIds); err != nil {
		return err
	}

	envPolicyUuids, err := d.getEnvPolicyRefs(group)
	if err != nil {
		return err
	}

	if len(envPolicyUuids) == 0 {
		return d.accountManagementClient.deleteAllEnvironmentPolicyBindings(ctx, remoteGroupId)
	}

	for env, uuids := range envPolicyUuids {
		if err = d.accountManagementClient.updateEnvironmentPolicyBindings(ctx, env, remoteGroupId, uuids); err != nil {
			return err
		}
	}
	return nil
}

func (d *AccountDeployer) updateUserGroupBindings(ctx context.Context, user account.User) error {
	remoteGroupIds, err := d.getUserGroupRefs(user)
	if err != nil {
		return err
	}

	if err := d.accountManagementClient.updateGroupBindings(ctx, user.Email, remoteGroupIds); err != nil {
		return err
	}
	return nil
}

func (d *AccountDeployer) updateGroupPermissions(ctx context.Context, group account.Group) error {
	var permissions []accountmanagement.PermissionsDto

	if group.Account != nil {
		perms := d.getAccountPermissions(group.Account)
		permissions = append(permissions, perms...)
	}

	if group.Environment != nil {
		perms := d.getEnvironmentPermissions(group.Environment)
		permissions = append(permissions, perms...)
	}

	if len(group.ManagementZone) > 0 {
		perms, err := d.getManagementZonePermissions(group.ManagementZone)
		if err != nil {
			return err
		}
		permissions = append(permissions, perms...)
	}

	if len(permissions) > 0 {
		remoteGroupId := d.groupIdLookup(group.ID)
		if remoteGroupId == "" {
			return fmt.Errorf("no group UUID found to update group <--> permissions bindings %q", group.ID)
		}
		if err := d.accountManagementClient.updatePermissions(ctx, remoteGroupId, permissions); err != nil {
			return err
		}
	}
	return nil
}

func (d *AccountDeployer) getAccountPermissions(acc *account.Account) []accountmanagement.PermissionsDto {
	var permissions []accountmanagement.PermissionsDto
	for _, p := range acc.Permissions {
		perm := accountmanagement.PermissionsDto{
			PermissionName: p,
			ScopeType:      "account",
			Scope:          d.accountManagementClient.getAccountInfo().AccountUUID,
		}
		permissions = append(permissions, perm)
	}
	return permissions
}

func (d *AccountDeployer) getEnvironmentPermissions(environments []account.Environment) []accountmanagement.PermissionsDto {
	var permissions []accountmanagement.PermissionsDto
	for _, env := range environments {
		for _, p := range env.Permissions {
			perm := accountmanagement.PermissionsDto{
				PermissionName: p,
				ScopeType:      "tenant",
				Scope:          env.Name,
			}
			permissions = append(permissions, perm)
		}
	}
	return permissions
}

func (d *AccountDeployer) getManagementZonePermissions(mzones []account.ManagementZone) ([]accountmanagement.PermissionsDto, error) {
	var permissions []accountmanagement.PermissionsDto
	for _, mz := range mzones {
		mzId := d.managementZoneIdLookup(mz.Environment, mz.ManagementZone)
		if mzId == "" {
			return nil, fmt.Errorf("unable to lookup id for management zone %q of environment %q", mz.ManagementZone, mz.Environment)
		}

		for _, p := range mz.Permissions {
			perm := accountmanagement.PermissionsDto{
				PermissionName: p,
				ScopeType:      "management-zone",
				Scope:          fmt.Sprintf("%s:%s", mz.Environment, mzId),
			}
			permissions = append(permissions, perm)
		}
	}

	return permissions, nil
}
func (d *AccountDeployer) getAccountPolicyRefs(group account.Group) ([]remoteId, error) {
	var policyIds []remoteId
	var err error
	if group.Account != nil {
		policyIds, err = d.processItems(group.Account.Policies, d.policyIdLookup)
		if err != nil {
			return nil, err
		}
	}
	return policyIds, nil
}

func (d *AccountDeployer) getEnvPolicyRefs(group account.Group) (map[envName][]remoteId, error) {
	result := make(map[envName][]remoteId)
	var err error
	if group.Environment != nil {
		for _, e := range group.Environment {
			result[e.Name], err = d.processItems(e.Policies, d.policyIdLookup)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (d *AccountDeployer) getUserGroupRefs(user account.User) ([]remoteId, error) {
	return d.processItems(user.Groups, d.groupIdLookup)
}

func (d *AccountDeployer) processItems(items []account.Ref, remoteIdLookup idLookupFn) ([]remoteId, error) {
	ids := []remoteId{}
	var notFoundLocalIds []localId

	for _, item := range items {
		rid := remoteIdLookup(item.ID())
		if rid == "" {
			notFoundLocalIds = append(notFoundLocalIds, item.ID())
			continue
		}

		ids = append(ids, rid)
	}

	if len(notFoundLocalIds) > 0 {
		return nil, fmt.Errorf("could not find remote Ids for the following items: %v", notFoundLocalIds)
	}

	return ids, nil
}

func (d *AccountDeployer) policyIdLookup(id localId) remoteId {
	return d.deployedPolicies[id]
}

func (d *AccountDeployer) groupIdLookup(id localId) remoteId {
	return d.deployedGroups[id]
}

func (d *AccountDeployer) managementZoneIdLookup(envName, mzName string) remoteId {
	for _, z := range d.deployedMgmtZones {
		if z.Parent == envName && z.Name == mzName {
			return z.Id
		}
	}
	return ""
}
