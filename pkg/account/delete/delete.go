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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
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

// AccountResources removes all given Resources from the given Account
// Returns an error if any resource fails to be deleted, but attempts to delete as many resources as possible and only returns an error at the end.
func AccountResources(ctx context.Context, account Account, resourcesToDelete Resources) error {

	deleteErrors := 0

	for _, user := range resourcesToDelete.Users {
		if err := account.APIClient.DeleteUser(ctx, user.Email.Value()); err != nil && errors.Is(err, NotFoundErr) {
			log.Info("User %q does not exist for account %s", user.Email, account)
		} else if err != nil {
			log.Error("Failed to delete user %q from account %s: %v", user.Email, account, err)
			deleteErrors++
		} else {
			log.Info("Deleted user %q from account %s", user.Email, account)
		}
	}

	if featureflags.ServiceUsers.Enabled() {
		for _, serviceUser := range resourcesToDelete.ServiceUsers {
			if err := account.APIClient.DeleteServiceUser(ctx, serviceUser.Name); err != nil && errors.Is(err, NotFoundErr) {
				log.Info("Service user %q does not exist for account %s", serviceUser.Name, account)
			} else if err != nil {
				log.Error("Failed to delete service user %q from account %s: %v", serviceUser.Name, account, err)
				deleteErrors++
			} else {
				log.Info("Deleted service user %q from account %s", serviceUser.Name, account)
			}
		}
	}

	for _, group := range resourcesToDelete.Groups {
		if err := account.APIClient.DeleteGroup(ctx, group.Name); err != nil && errors.Is(err, NotFoundErr) {
			log.Info("Group %q does not exist for account %s", group.Name, account)
		} else if err != nil {
			log.Error("Failed to delete group %q from account %s: %v", group.Name, account, err)
			deleteErrors++
		} else {
			log.Info("Deleted group %q from account %s", group.Name, account)
		}
	}
	for _, policy := range resourcesToDelete.AccountPolicies {
		if err := account.APIClient.DeleteAccountPolicy(ctx, policy.Name); err != nil && errors.Is(err, NotFoundErr) {
			log.Info("Policy %q does not exist for account %s", policy.Name, account)
		} else if err != nil {
			log.Error("Failed to delete policy %q from account %s: %v", policy.Name, account, err)
			deleteErrors++
		} else {
			log.Info("Deleted policy %q from account %s", policy.Name, account)
		}
	}
	for _, policy := range resourcesToDelete.EnvironmentPolicies {
		if err := account.APIClient.DeleteEnvironmentPolicy(ctx, policy.Environment, policy.Name); err != nil && errors.Is(err, NotFoundErr) {
			log.Info("Policy %q does not exist for environment %s", policy.Name, policy.Environment)
		} else if err != nil {
			log.Error("Failed to delete policy %q for environment %s: %v", policy.Name, policy.Environment, err)
			deleteErrors++
		} else {
			log.Info("Deleted policy %q for environment %q", policy.Name, policy.Environment)
		}
	}

	if deleteErrors > 0 {
		return fmt.Errorf("encountered %d errors - please check logs for details", deleteErrors)
	}
	return nil
}
