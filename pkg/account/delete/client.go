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
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"golang.org/x/net/context"
	"io"
	"net/http"
)

// Client for deleting resources from the Account Management API
type Client interface {
	DeleteUser(ctx context.Context, email string) error
	DeleteGroup(ctx context.Context, name string) error
	DeleteAccountPolicy(ctx context.Context, name string) error
	DeleteEnvironmentPolicy(ctx context.Context, environment, name string) error
	GetAccountUUID() string
}

var _ Client = (*AccountAPIClient)(nil)

// AccountAPIClient is the default implementation of a delete Client, accessing the Account Management API using an accounts.Client
type AccountAPIClient struct {
	accountUUID string
	client      *accounts.Client
}

func NewAccountAPIClient(accountUUID string, restClient *accounts.Client) Client {
	return &AccountAPIClient{
		accountUUID: accountUUID,
		client:      restClient,
	}
}

// DeleteUser removes the user with the given email from the account
// Returns error if any API call fails unless the user is already not present (HTTP 404)
func (c *AccountAPIClient) DeleteUser(ctx context.Context, email string) error {
	resp, err := c.client.UserManagementAPI.RemoveUserFromAccount(ctx, c.accountUUID, email).Execute()
	defer closeResponseBody(resp)
	if resp != nil && resp.StatusCode == 404 {
		log.Info("User %q does not exist for account %q", email, c.accountUUID)
		return nil
	}
	if err := handleClientResponseError(resp, err, fmt.Sprintf("failed to delete user %q", email)); err != nil {
		return err
	}
	log.Info("Deleted user %q from account %q", email, c.accountUUID)
	return nil
}

// DeleteGroup removes the group with the given name from the account
// Returns error if any API call fails unless the group is already not present (HTTP 404)
func (c *AccountAPIClient) DeleteGroup(ctx context.Context, name string) error {

	uuid, err := c.getGroupID(ctx, c.accountUUID, name)
	if err != nil {
		if errors.Is(err, notFoundErr) {
			log.Info("Group %q does not exist for account %q", name, c.accountUUID)
			return nil
		}
		return err
	}

	resp, err := c.client.GroupManagementAPI.DeleteGroup(ctx, c.accountUUID, uuid).Execute()
	defer closeResponseBody(resp)
	if resp != nil && resp.StatusCode == 404 {
		log.Info("Group %q does not exist for account %q", name, c.accountUUID)
		return nil
	}
	if err := handleClientResponseError(resp, err, fmt.Sprintf("failed to delete group %q", name)); err != nil {
		return err
	}
	log.Info("Deleted group %q (%s) from account %q", name, uuid, c.accountUUID)
	return nil
}

var notFoundErr = errors.New("nothing with given name found")

func (c *AccountAPIClient) getGroupID(ctx context.Context, accountUUID, name string) (string, error) {
	groups, resp, err := c.client.GroupManagementAPI.GetGroups(ctx, accountUUID).Execute()
	defer closeResponseBody(resp)
	if err := handleClientResponseError(resp, err, fmt.Sprintf("failed to fetch UUID for group %q", name)); err != nil {
		return "", err
	}
	for _, g := range groups.GetItems() {
		if g.Name == name {
			return g.GetUuid(), nil
		}
	}
	return "", notFoundErr
}

// DeleteAccountPolicy removes the account-level policy with the given name from the account
// If the policy is still bound to any groups, it will be force removed from them.
// Returns error if any API call fails unless the policy is already not present (HTTP 404)
func (c *AccountAPIClient) DeleteAccountPolicy(ctx context.Context, name string) error {
	return c.deletePolicy(ctx, "account", c.accountUUID, name)
}

// DeleteEnvironmentPolicy removes the environment-level policy with the given name from the given environment.
// If the policy is still bound to any groups, it will be force removed from them.
// Returns error if any API call fails unless the policy is already not present (HTTP 404)
func (c *AccountAPIClient) DeleteEnvironmentPolicy(ctx context.Context, environmentID, name string) error {
	return c.deletePolicy(ctx, "environment", environmentID, name)
}

func (c *AccountAPIClient) deletePolicy(ctx context.Context, levelType string, levelID, name string) error {
	uuid, err := c.getPolicyID(ctx, levelType, levelID, name)
	if err != nil {
		if errors.Is(err, notFoundErr) {
			log.Info("Policy %q does not exist for %s %q", name, levelType, levelID)
			return nil
		}
		return err
	}

	resp, err := c.client.PolicyManagementAPI.DeleteLevelPolicy(ctx, levelType, levelID, uuid).Force(true).Execute()
	defer closeResponseBody(resp)
	if resp != nil && resp.StatusCode == 404 {
		log.Info("Policy %q does not exist for %s %q", name, levelType, levelID)
		return nil
	}
	if err := handleClientResponseError(resp, err, fmt.Sprintf("failed to delete policy %q", name)); err != nil {
		return err
	}
	log.Info("Deleted policy %q (%s) for %s %q", name, uuid, levelType, levelID)
	return nil
}

func (c *AccountAPIClient) getPolicyID(ctx context.Context, levelType, levelID, name string) (string, error) {
	policies, resp, err := c.client.PolicyManagementAPI.GetLevelPolicies(ctx, levelType, levelID).Execute()
	defer closeResponseBody(resp)
	if err := handleClientResponseError(resp, err, fmt.Sprintf("failed to fetch UUID for policy %q", name)); err != nil {
		return "", err
	}
	for _, p := range policies.GetPolicies() {
		if p.Name == name {
			return p.GetUuid(), nil
		}
	}
	return "", notFoundErr
}

// GetAccountUUID returns the UUID of the account this Client has access to
func (c *AccountAPIClient) GetAccountUUID() string {
	return c.accountUUID
}

func handleClientResponseError(resp *http.Response, clientErr error, errMessage string) error {
	if clientErr != nil && resp == nil {
		return fmt.Errorf("%s: %w", errMessage, clientErr)
	}

	if !rest.IsSuccess(resp) && resp.StatusCode != http.StatusNotFound {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%s: unable to read response body %w", errMessage, err)
		}
		return fmt.Errorf("%s (HTTP %d): %s", errMessage, resp.StatusCode, string(body))
	}
	return nil
}

func closeResponseBody(resp *http.Response) {
	_ = resp.Body.Close()
}
