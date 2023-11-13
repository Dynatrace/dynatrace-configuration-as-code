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

type Client interface {
	DeleteUser(ctx context.Context, accountUUID, email string) error
	DeleteGroup(ctx context.Context, accountUUID, name string) error
	DeletePolicy(ctx context.Context, levelType, levelID, name string) error
}

type AccountAPIClient struct {
	Client *accounts.Client
}

func (c *AccountAPIClient) DeleteUser(ctx context.Context, accountUUID, email string) error {
	resp, err := c.Client.UserManagementAPI.RemoveUserFromAccount(ctx, accountUUID, email).Execute()
	if resp != nil && resp.StatusCode == 404 {
		log.Info("User %q does not exist for account %q", email, accountUUID)
		return nil
	}
	if err := handleClientResponseError(resp, err, fmt.Sprintf("failed to delete user %q", email)); err != nil {
		return err
	}
	log.Info("Deleted user %q from account %q", email, accountUUID)
	return nil
}

func (c *AccountAPIClient) DeleteGroup(ctx context.Context, accountUUID, name string) error {

	uuid, err := c.getGroupID(ctx, accountUUID, name)
	if err != nil {
		if errors.Is(err, notFoundErr) {
			log.Info("Group %q does not exist for account %q", name, accountUUID)
			return nil
		}
		return err
	}

	resp, err := c.Client.GroupManagementAPI.DeleteGroup(ctx, accountUUID, uuid).Execute()
	if resp != nil && resp.StatusCode == 404 {
		log.Info("Group %q does not exist for account %q", name, accountUUID)
		return nil
	}
	if err := handleClientResponseError(resp, err, fmt.Sprintf("failed to delete group %q", name)); err != nil {
		return err
	}
	log.Info("Deleted group %q (%s) from account %q", name, uuid, accountUUID)
	return nil
}

var notFoundErr = errors.New("nothing with given name found")

func (c *AccountAPIClient) getGroupID(ctx context.Context, accountUUID, name string) (string, error) {
	groups, resp, err := c.Client.GroupManagementAPI.GetGroups(ctx, accountUUID).Execute()
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

func (c *AccountAPIClient) DeletePolicy(ctx context.Context, levelType string, levelID, name string) error {
	uuid, err := c.getPolicyID(ctx, levelType, levelID, name)
	if err != nil {
		if errors.Is(err, notFoundErr) {
			log.Info("Policy %q does not exist for %s %q", name, levelType, levelID)
			return nil
		}
		return err
	}

	resp, err := c.Client.PolicyManagementAPI.DeleteLevelPolicy(ctx, levelType, levelID, uuid).Force(true).Execute()
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
	policies, resp, err := c.Client.PolicyManagementAPI.GetLevelPolicies(ctx, levelType, levelID).Execute()
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
