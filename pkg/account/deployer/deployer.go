package deployer

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-logr/logr"

	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
)

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
	accClient            client
	idMap                idMap
	logger               *log.WrappedLogger
	maxConcurrentDeploys int
}

func WithMaxConcurrentDeploys(maxConcurrentDeploys int) func(*AccountDeployer) {
	return func(d *AccountDeployer) {
		d.maxConcurrentDeploys = maxConcurrentDeploys
	}
}

func NewAccountDeployer(client client, opts ...func(*AccountDeployer)) *AccountDeployer {
	ac := &AccountDeployer{
		accClient: client,
		idMap:     newIdMap(),
		logger:    log.WithFields(field.F("account", client.getAccountInfo().Name)),
	}
	for _, o := range opts {
		o(ac)
	}
	return ac
}

func (d *AccountDeployer) Deploy(res *account.Resources) error {
	err := d.fetchExistingResources()
	if err != nil {
		return err
	}

	err = d.deployResources(res)
	if err != nil {
		return err
	}

	err = d.updateBindings(res)
	if err != nil {
		return err
	}

	return nil
}

func (d *AccountDeployer) fetchExistingResources() error {
	dispatcher := NewDispatcher(d.maxConcurrentDeploys)
	dispatcher.Run()
	defer dispatcher.Stop()

	fetchResourcesJob := func(wg *sync.WaitGroup, errCh chan error) {
		fetchResources(d.fetchGlobalPolicies, wg, errCh)

	}
	fetchMZonesJob := func(wg *sync.WaitGroup, errCh chan error) {
		fetchResources(d.fetchManagementZones, wg, errCh)
	}

	fetchGroupsJob := func(wg *sync.WaitGroup, errCh chan error) {
		fetchResources(d.fetchGroups, wg, errCh)
	}

	dispatcher.AddJob(fetchResourcesJob)
	dispatcher.AddJob(fetchMZonesJob)
	dispatcher.AddJob(fetchGroupsJob)

	return dispatcher.Wait()

}

func (d *AccountDeployer) deployResources(res *account.Resources) error {
	dispatcher := NewDispatcher(d.maxConcurrentDeploys)
	dispatcher.Run()
	defer dispatcher.Stop()

	d.deployPolicies(res.Policies, dispatcher)
	d.deployGroups(res.Groups, dispatcher)
	d.deployUsers(res.Users, dispatcher)

	return dispatcher.Wait()

}

func (d *AccountDeployer) updateBindings(res *account.Resources) error {
	dispatcher := NewDispatcher(d.maxConcurrentDeploys)
	dispatcher.Run()
	defer dispatcher.Stop()

	d.deployGroupBindings(res.Groups, dispatcher)
	d.deployUserBindings(res.Users, dispatcher)
	return dispatcher.Wait()

}

func (d *AccountDeployer) fetchGlobalPolicies() error {
	d.logger.Debug("Getting existing global policies")
	deployedPolicies, err := d.accClient.getGlobalPolicies(d.logCtx())
	if err != nil {
		return err
	}
	d.idMap.addPolicies(deployedPolicies)
	return nil
}

func (d *AccountDeployer) fetchManagementZones() error {
	d.logger.Debug("Getting existing management zones")
	deployedMgmtZones, err := d.accClient.getManagementZones(d.logCtx())
	if err != nil {
		return err
	}
	d.idMap.addMZones(deployedMgmtZones)
	return nil
}

func (d *AccountDeployer) fetchGroups() error {
	d.logger.Debug("Getting existing groups")
	deployedGroups, err := d.accClient.getAllGroups(d.logCtx())
	if err != nil {
		return err
	}
	d.idMap.addGroups(deployedGroups)
	return err

}

func fetchResources(fetchFunc func() error, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	errCh <- fetchFunc()
}

func (d *AccountDeployer) deployPolicies(policies map[string]account.Policy, dispatcher *Dispatcher) { // nolint:dupl
	for _, policy := range policies {
		policy := policy
		deployPolicyJob := func(wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			d.logger.Info("Deploying policy %s", policy.Name)
			pUuid, err := d.upsertPolicy(d.logCtx(), policy)
			if err != nil {
				errCh <- fmt.Errorf("unable to deploy policy for account %s: %w", d.accClient.getAccountInfo().AccountUUID, err)
			}
			d.idMap.addPolicy(policy.ID, pUuid)
		}
		dispatcher.AddJob(deployPolicyJob)
	}
}

func (d *AccountDeployer) deployGroups(groups map[string]account.Group, dispatcher *Dispatcher) { // nolint:dupl
	for _, group := range groups {
		group := group
		deployGroupJob := func(wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			d.logger.Info("Deploying group %s", group.Name)
			gUuid, err := d.upsertGroup(d.logCtx(), group)
			if err != nil {
				errCh <- fmt.Errorf("unable to deploy group for account %s: %w", d.accClient.getAccountInfo().AccountUUID, err)
			}
			d.idMap.addGroup(group.ID, gUuid)

		}
		dispatcher.AddJob(deployGroupJob)

	}
}

func (d *AccountDeployer) deployUsers(users map[string]account.User, dispatcher *Dispatcher) {
	for _, user := range users {
		user := user
		deployUserJob := func(wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			d.logger.Info("Deploying user %s", user.Email)
			if _, err := d.upsertUser(d.logCtx(), user); err != nil {
				errCh <- fmt.Errorf("unable to deploy user for account %s: %w", d.accClient.getAccountInfo().AccountUUID, err)
			}
		}
		dispatcher.AddJob(deployUserJob)

	}
}

func (d *AccountDeployer) deployGroupBindings(groups map[account.GroupId]account.Group, dispatcher *Dispatcher) {
	for _, group := range groups {
		group := group
		d.logger.Info("Updating policy bindings and permissions for group %s", group.Name)

		updateBindingsJob := func(wg *sync.WaitGroup, errCh chan error) {
			defer wg.Done()
			if err := d.updateGroupPolicyBindings(d.logCtx(), group); err != nil {
				errCh <- fmt.Errorf("unable to deploy policy binding for account %s: %w", d.accClient.getAccountInfo().AccountUUID, err)
			}

			if err := d.updateGroupPermissions(d.logCtx(), group); err != nil {
				errCh <- fmt.Errorf("unable to deploy permissions for account %s: %w", d.accClient.getAccountInfo().AccountUUID, err)
			}
		}

		dispatcher.AddJob(updateBindingsJob)

	}
}

func (d *AccountDeployer) deployUserBindings(users map[account.UserId]account.User, dispatcher *Dispatcher) {
	for _, user := range users {
		user := user
		deployUserBindingsJob :=
			func(wg *sync.WaitGroup, errCh chan error) {
				defer wg.Done()
				d.logger.Info("Updating group bindings for user %s", user.Email)
				if err := d.updateUserGroupBindings(d.logCtx(), user); err != nil {
					errCh <- fmt.Errorf("unable to deploy user binding for account %s: %w", d.accClient.getAccountInfo().AccountUUID, err)
				}
			}

		dispatcher.AddJob(deployUserBindingsJob)

	}
}
func (d *AccountDeployer) upsertPolicy(ctx context.Context, policy account.Policy) (remoteId, error) {
	var policyLevel string
	var policyLevelID string

	if _, ok := policy.Level.(account.PolicyLevelAccount); ok {
		policyLevel = "account"
		policyLevelID = d.accClient.getAccountInfo().AccountUUID
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

	return d.accClient.upsertPolicy(ctx, policyLevel, policyLevelID, policy.OriginObjectID, data)
}

func (d *AccountDeployer) upsertGroup(ctx context.Context, group account.Group) (remoteId, error) {
	data := accountmanagement.PutGroupDto{
		Name:                     group.Name,
		Description:              &group.Description,
		FederatedAttributeValues: group.FederatedAttributeValues,
	}
	return d.accClient.upsertGroup(ctx, group.OriginObjectID, data)
}

func (d *AccountDeployer) upsertUser(ctx context.Context, user account.User) (remoteId, error) {
	return d.accClient.upsertUser(ctx, user.Email.Value())
}

func (d *AccountDeployer) updateGroupPolicyBindings(ctx context.Context, group account.Group) error {

	remoteGroupId := d.idMap.getGroupUUID(group.ID)
	if remoteGroupId == "" {
		return fmt.Errorf("unable to determine UUID for group %s", group.Name)
	}
	accPolicyUuids, err := d.getAccountPolicyRefs(group)
	if err != nil {
		return fmt.Errorf("failed to fetch policies for group %s: %w", group.Name, err)
	}

	d.logger.Debug("Updating account level policy bindings for group with ID %s --> %v", remoteGroupId, accPolicyUuids)
	if err = d.accClient.updateAccountPolicyBindings(ctx, remoteGroupId, accPolicyUuids); err != nil {
		return fmt.Errorf("failed to update group-account-policy bindings for group %s: %w", group.Name, err)
	}

	envPolicyUuids, err := d.getEnvPolicyRefs(group)
	if err != nil {
		return err
	}

	if len(envPolicyUuids) == 0 {
		d.logger.Debug("Deleting all policy bindings of group with ID %s", remoteGroupId)
		return d.accClient.deleteAllEnvironmentPolicyBindings(ctx, remoteGroupId)
	}

	for env, uuids := range envPolicyUuids {
		d.logger.WithFields().Debug("Updating environment level policy bindings for group with ID %s and environment with name %s --> %v", remoteGroupId, env, uuids)
		if err = d.accClient.updateEnvironmentPolicyBindings(ctx, env, remoteGroupId, uuids); err != nil {
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

	if err := d.accClient.updateGroupBindings(ctx, user.Email.Value(), remoteGroupIds); err != nil {
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

	remoteGroupId := d.idMap.getGroupUUID(group.ID)
	if remoteGroupId == "" {
		return fmt.Errorf("no group UUID found to update group <--> permissions bindings %s", group.ID)
	}

	d.logger.Debug("Updating permissions for group with ID %s --> %v", remoteGroupId, allPermissions)
	if err := d.accClient.updatePermissions(ctx, remoteGroupId, allPermissions); err != nil {
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
			Scope:          d.accClient.getAccountInfo().AccountUUID,
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
				ScopeType:      api.ManagementZone,
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
	return d.idMap.getPolicyUUID(id)
}

func (d *AccountDeployer) groupIdLookup(id localId) remoteId {
	return d.idMap.getGroupUUID(id)
}

func (d *AccountDeployer) managementZoneIdLookup(envName, mzName string) remoteId {
	return d.idMap.getMZoneUUID(envName, mzName)
}

func (d *AccountDeployer) logCtx() context.Context {
	return logr.NewContextWithSlogLogger(context.TODO(), d.logger.SLogger())
}
