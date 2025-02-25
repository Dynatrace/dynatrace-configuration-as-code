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

package delete

import (
	"context"
	"errors"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
)

type User struct {
	Email secret.Email
}

type ServiceUser struct {
	Name string
}

type Group struct {
	Name string
}
type AccountPolicy struct {
	Name string
}
type EnvironmentPolicy struct {
	Name        string
	Environment string
}

// Resources defines which account resources to delete. Each field defines the information required to delete that type.
type Resources struct {
	Users               []User
	ServiceUsers        []ServiceUser
	Groups              []Group
	AccountPolicies     []AccountPolicy
	EnvironmentPolicies []EnvironmentPolicy
}

// Account defines everything required to access the account management API
type Account struct {
	// Name of this account - as defined in the manifest.Manifest
	Name string
	// UUID of this account
	UUID string
	// APIClient is a Client for authenticated access to delete resources for this Account
	APIClient Client
}

func (a Account) String() string {
	return fmt.Sprintf("%q (%s)", a.Name, a.UUID)
}

// DeleteAccountResources removes all given Resources from the given Account
// Returns an error if any resource fails to be deleted, but attempts to delete as many resources as possible and only returns an error at the end.
func DeleteAccountResources(ctx context.Context, account Account, resourcesToDelete Resources) error {
	totalErrorCount := deleteUsers(ctx, account, resourcesToDelete.Users) +
		deleteServiceUsers(ctx, account, resourcesToDelete.ServiceUsers) +
		deleteGroups(ctx, account, resourcesToDelete.Groups) +
		deleteAccountPolicies(ctx, account, resourcesToDelete.AccountPolicies) +
		deleteEnvironmentPolicies(ctx, account, resourcesToDelete.EnvironmentPolicies)

	if totalErrorCount > 0 {
		return fmt.Errorf("encountered %d errors - please check logs for details", totalErrorCount)
	}
	return nil
}

func deleteUsers(ctx context.Context, account Account, users []User) int {
	errCount := 0
	for _, user := range users {
		err := account.APIClient.DeleteUser(ctx, user.Email.Value())
		if err == nil {
			log.Info("Deleted user %q from account %s", user.Email, account)
			continue
		}

		if errors.Is(err, NotFoundErr) {
			log.Info("User %q does not exist for account %s", user.Email, account)
			continue
		}

		log.Error("Failed to delete user %q from account %s: %v", user.Email, account, err)
		errCount++
	}
	return errCount
}

func deleteServiceUsers(ctx context.Context, account Account, serviceUsers []ServiceUser) int {
	errCount := 0
	for _, user := range serviceUsers {
		err := account.APIClient.DeleteServiceUser(ctx, user.Name)
		if err == nil {
			log.Info("Deleted service user %q from account %s", user.Name, account)
			continue
		}

		if errors.Is(err, NotFoundErr) {
			log.Info("Service user %q does not exist for account %s", user.Name, account)
			continue
		}

		log.Error("Failed to delete service user %q from account %s: %v", user.Name, account, err)
		errCount++
	}
	return errCount
}

func deleteGroups(ctx context.Context, account Account, groups []Group) int {
	errCount := 0
	for _, group := range groups {
		err := account.APIClient.DeleteGroup(ctx, group.Name)
		if err == nil {
			log.Info("Deleted group %q from account %s", group.Name, account)
			continue
		}

		if errors.Is(err, NotFoundErr) {
			log.Info("Group %q does not exist for account %s", group.Name, account)
			continue
		}

		log.Error("Failed to delete group %q from account %s: %v", group.Name, account, err)
		errCount++
	}
	return errCount
}

func deleteAccountPolicies(ctx context.Context, account Account, accountPolicies []AccountPolicy) int {
	errCount := 0
	for _, policy := range accountPolicies {
		err := account.APIClient.DeleteAccountPolicy(ctx, policy.Name)
		if err == nil {
			log.Info("Deleted policy %q from account %s", policy.Name, account)
			continue
		}

		if errors.Is(err, NotFoundErr) {
			log.Info("Policy %q does not exist for account %s", policy.Name, account)
		}

		log.Error("Failed to delete policy %q from account %s: %v", policy.Name, account, err)
		errCount++
	}
	return errCount
}

func deleteEnvironmentPolicies(ctx context.Context, account Account, environmentPolicies []EnvironmentPolicy) int {
	errCount := 0
	for _, policy := range environmentPolicies {
		err := account.APIClient.DeleteEnvironmentPolicy(ctx, policy.Environment, policy.Name)
		if err == nil {
			log.Info("Deleted policy %q for environment %q", policy.Name, policy.Environment)
			continue
		}

		if errors.Is(err, NotFoundErr) {
			log.Info("Policy %q does not exist for environment %s", policy.Name, policy.Environment)
			continue
		}

		log.Error("Failed to delete policy %q for environment %s: %v", policy.Name, policy.Environment, err)
		errCount++
	}
	return errCount
}
