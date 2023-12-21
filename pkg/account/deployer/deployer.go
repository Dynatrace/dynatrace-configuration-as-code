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
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/go-logr/logr"
	"strings"
	"sync"
)

type (
	localId    = string // local (monaco related) identifier
	envName    = string // dt environment name
	remoteId   = string // dt entity identifier
	idLookupFn func(id localId) remoteId

	permissions []accountmanagement.PermissionsDto
)

func (p permissions) String() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for _, e := range p {
		fmt.Fprintf(&sb, "{%s %s}", e.PermissionName, e.ScopeType)
	}
	sb.WriteString("]")
	return sb.String()
}

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
	getAccountInfo() account.AccountInfo
}

type AccountDeployer struct {
	accountManagementClient client

	deployedPolicies  map[localId]remoteId
	deployedGroups    map[localId]remoteId
	deployedMgmtZones []accountmanagement.ManagementZoneResourceDto

	logger loggers.Logger
}

func NewAccountDeployer(client client) *AccountDeployer {

	return &AccountDeployer{
		accountManagementClient: client,

		deployedPolicies: make(map[localId]remoteId),
		deployedGroups:   make(map[localId]remoteId),

		logger: log.WithFields(field.F("account", client.getAccountInfo().Name)),
	}

}

func (d *AccountDeployer) Deploy(res *account.Resources) error {
	var ers []error
	var waitForExistingResources sync.WaitGroup
	waitForExistingResources.Add(3)
	errCh := make(chan error, 3)

	go fetchResources(d.fetchGlobalPolicies, &waitForExistingResources, errCh)
	go fetchResources(d.fetchManagementZones, &waitForExistingResources, errCh)
	go fetchResources(d.fetchGroups, &waitForExistingResources, errCh)

	d.waitForResources(&waitForExistingResources, errCh, &ers)
	if len(ers) > 0 {
		return errors.Join(ers...)
	}

	var waitForResources sync.WaitGroup
	errCh = make(chan error, 3)
	waitForResources.Add(3)

	go deployResources(res.Policies, d.deployPolicies, &waitForResources, errCh)
	go deployResources(res.Groups, d.deployGroups, &waitForResources, errCh)
	go deployResources(res.Users, d.deployUsers, &waitForResources, errCh)

	d.waitForResources(&waitForResources, errCh, &ers)
	if len(ers) > 0 {
		return errors.Join(ers...)
	}

	var waitForBindings sync.WaitGroup
	errCh = make(chan error, 2)
	waitForBindings.Add(2)

	go deployResources(res.Groups, d.deployGroupBindings, &waitForBindings, errCh)
	go deployResources(res.Users, d.deployUserBindings, &waitForBindings, errCh)

	d.waitForResources(&waitForBindings, errCh, &ers)
	if len(ers) > 0 {
		return errors.Join(ers...)
	}

	return nil
}

func (d *AccountDeployer) waitForResources(waitGroup *sync.WaitGroup, errCh chan error, ers *[]error) {
	waitGroup.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			*ers = append(*ers, err)
		}
	}
}

func (d *AccountDeployer) fetchGlobalPolicies() error {
	var err error
	d.logger.Debug("Getting existing global policies")
	d.deployedPolicies, err = d.accountManagementClient.getGlobalPolicies(d.logCtx())
	return err
}

func (d *AccountDeployer) fetchManagementZones() error {
	var err error
	d.logger.Debug("Getting existing management zones")
	d.deployedMgmtZones, err = d.accountManagementClient.getManagementZones(d.logCtx())
	return err
}

func (d *AccountDeployer) fetchGroups() error {
	var err error
	d.logger.Debug("Getting existing groups")
	d.deployedGroups, err = d.accountManagementClient.getAllGroups(d.logCtx())
	return err

}

func fetchResources(fetchFunc func() error, wg *sync.WaitGroup, errCh chan error) {
	defer wg.Done()
	errCh <- fetchFunc()
}

func deployResources[T any](resources map[localId]T, deployFunc func(map[string]T) error, wg *sync.WaitGroup, errCh chan error) {
	defer wg.Done()
	errCh <- deployFunc(resources)
}

func (d *AccountDeployer) deployPolicies(policies map[string]account.Policy) error {
	var errs []error
	for _, policy := range policies {
		d.logger.Info("Deploying policy %s", policy.Name)
		pUuid, err := d.upsertPolicy(d.logCtx(), policy)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to deploy policy for account %s: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err))
		}
		d.deployedPolicies[policy.ID] = pUuid
	}
	return errors.Join(errs...)
}

func (d *AccountDeployer) deployGroups(groups map[string]account.Group) error {
	var errs []error
	for _, group := range groups {
		d.logger.Info("Deploying group %s", group.Name)
		gUuid, err := d.upsertGroup(d.logCtx(), group)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to deploy group for account %s: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err))
		}
		d.deployedGroups[group.ID] = gUuid

	}
	return errors.Join(errs...)
}
func (d *AccountDeployer) deployGroupBindings(groups map[account.GroupId]account.Group) error {
	var errs []error
	for _, group := range groups {
		d.logger.Info("Updating policy bindings and permissions for group %s", group.Name)
		if err := d.updateGroupPolicyBindings(d.logCtx(), group); err != nil {
			errs = append(errs, fmt.Errorf("unable to deploy policy binding for account %s: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err))
		}

		if err := d.updateGroupPermissions(d.logCtx(), group); err != nil {
			errs = append(errs, fmt.Errorf("unable to deploy permissions for account %s: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err))
		}
	}
	return errors.Join(errs...)
}

func (d *AccountDeployer) deployUsers(users map[string]account.User) error {
	var errs []error
	for _, user := range users {
		d.logger.Info("Deploying user %s", user.Email)
		if _, err := d.upsertUser(d.logCtx(), user); err != nil {
			errs = append(errs, fmt.Errorf("unable to deploy user for account %s: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err))
		}
	}
	return errors.Join(errs...)
}

func (d *AccountDeployer) deployUserBindings(users map[account.UserId]account.User) error {
	var errs []error
	for _, user := range users {
		d.logger.Info("Updating group bindings for user %s", user.Email)
		if err := d.updateUserGroupBindings(d.logCtx(), user); err != nil {
			errs = append(errs, fmt.Errorf("unable to deploy user binding for account %s: %w", d.accountManagementClient.getAccountInfo().AccountUUID, err))
		}
	}
	return errors.Join(errs...)
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
		return fmt.Errorf("unable to determine UUID for group %s", group.Name)
	}
	accPolicyUuids, err := d.getAccountPolicyRefs(group)
	if err != nil {
		return fmt.Errorf("failed to fetch policies for group %s: %w", group.Name, err)
	}

	d.logger.Debug("Updating account level policy bindings for group with ID %s --> %v", remoteGroupId, accPolicyUuids)
	if err = d.accountManagementClient.updateAccountPolicyBindings(ctx, remoteGroupId, accPolicyUuids); err != nil {
		return fmt.Errorf("failed to update group-account-policy bindings for group %s: %w", group.Name, err)
	}

	envPolicyUuids, err := d.getEnvPolicyRefs(group)
	if err != nil {
		return err
	}

	if len(envPolicyUuids) == 0 {
		d.logger.Debug("Deleting all policy bindings of group with ID %s", remoteGroupId)
		return d.accountManagementClient.deleteAllEnvironmentPolicyBindings(ctx, remoteGroupId)
	}

	for env, uuids := range envPolicyUuids {
		d.logger.WithFields().Debug("Updating environment level policy bindings for group with ID %s and environment with name %s --> %v", remoteGroupId, env, uuids)
		if err = d.accountManagementClient.updateEnvironmentPolicyBindings(ctx, env, remoteGroupId, uuids); err != nil {
			return fmt.Errorf("failed to update group-environment-policy bindings for group %s and environment %s: %w", group.Name, env, err)
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
	allPermissions := make(permissions, 0)

	if group.Account != nil {
		perms := d.getAccountPermissions(group.Account)
		allPermissions = append(allPermissions, perms...)
	}

	if group.Environment != nil {
		perms := d.getEnvironmentPermissions(group.Environment)
		allPermissions = append(allPermissions, perms...)
	}

	if len(group.ManagementZone) > 0 {
		perms, err := d.getManagementZonePermissions(group.ManagementZone)
		if err != nil {
			return err
		}
		allPermissions = append(allPermissions, perms...)
	}

	remoteGroupId := d.groupIdLookup(group.ID)
	if remoteGroupId == "" {
		return fmt.Errorf("no group UUID found to update group <--> permissions bindings %s", group.ID)
	}

	d.logger.Debug("Updating permissions for group with ID %s --> %v", remoteGroupId, allPermissions)
	if err := d.accountManagementClient.updatePermissions(ctx, remoteGroupId, allPermissions); err != nil {
		return fmt.Errorf("unable to update group %s: %w", group.ID, err)
	}
	return nil
}

func (d *AccountDeployer) getAccountPermissions(acc *account.Account) []accountmanagement.PermissionsDto {
	var perms []accountmanagement.PermissionsDto
	for _, p := range acc.Permissions {
		perm := accountmanagement.PermissionsDto{
			PermissionName: p,
			ScopeType:      "account",
			Scope:          d.accountManagementClient.getAccountInfo().AccountUUID,
		}
		perms = append(perms, perm)
	}
	return perms
}

func (d *AccountDeployer) getEnvironmentPermissions(environments []account.Environment) []accountmanagement.PermissionsDto {
	var perms []accountmanagement.PermissionsDto
	for _, env := range environments {
		for _, p := range env.Permissions {
			perm := accountmanagement.PermissionsDto{
				PermissionName: p,
				ScopeType:      "tenant",
				Scope:          env.Name,
			}
			perms = append(perms, perm)
		}
	}
	return perms
}

func (d *AccountDeployer) getManagementZonePermissions(mzones []account.ManagementZone) ([]accountmanagement.PermissionsDto, error) {
	var perms []accountmanagement.PermissionsDto
	for _, mz := range mzones {
		mzId := d.managementZoneIdLookup(mz.Environment, mz.ManagementZone)
		if mzId == "" {
			return nil, fmt.Errorf("unable to lookup id for management zone %s of environment %s", mz.ManagementZone, mz.Environment)
		}

		for _, p := range mz.Permissions {
			perm := accountmanagement.PermissionsDto{
				PermissionName: p,
				ScopeType:      "management-zone",
				Scope:          fmt.Sprintf("%s:%s", mz.Environment, mzId),
			}
			perms = append(perms, perm)
		}
	}

	return perms, nil
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

func (d *AccountDeployer) logCtx() context.Context {
	return logr.NewContext(context.TODO(), d.logger.GetLogr())
}
