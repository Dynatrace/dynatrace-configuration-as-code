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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
)

type User struct {
	Email string
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
	Groups              []Group
	AccountPolicies     []AccountPolicy
	EnvironmentPolicies []EnvironmentPolicy
}

// AccountResources removes all given Account configurations defined as DeleteEntries from an account using the supplied Client
// Returns an error if any resource fails to be deleted, but attempts to delete as many resources as possible and only returns an error at the end.
func AccountResources(ctx context.Context, client Client, resourcesToDelete Resources) error {

	deleteErrors := 0

	for _, user := range resourcesToDelete.Users {
		if err := client.DeleteUser(ctx, user.Email); err != nil {
			log.Error("Failed to delete user %q from account %q: %v", user.Email, client.GetAccountUUID(), err)
			deleteErrors++
		}
	}
	for _, group := range resourcesToDelete.Groups {
		if err := client.DeleteGroup(ctx, group.Name); err != nil {
			log.Error("Failed to delete group %q from account %q: %v", group.Name, client.GetAccountUUID(), err)
			deleteErrors++
		}
	}
	for _, policy := range resourcesToDelete.AccountPolicies {
		if err := client.DeleteAccountPolicy(ctx, policy.Name); err != nil {
			log.Error("Failed to delete account policy %q from account %q: %v", policy.Name, client.GetAccountUUID(), err)
			deleteErrors++
		}
	}
	for _, policy := range resourcesToDelete.EnvironmentPolicies {
		if err := client.DeleteEnvironmentPolicy(ctx, policy.Environment, policy.Name); err != nil {
			log.Error("Failed to delete policy %q for environment %q: %v", policy.Name, policy.Environment, err)
			deleteErrors++
		}
	}

	if deleteErrors > 0 {
		return fmt.Errorf("encountered %d errors - please check logs for details", deleteErrors)
	}
	return nil
}
