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

package http

import (
	"context"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
)

func (c *Client) GetUsers(ctx context.Context, accountUUID string) ([]accountmanagement.UsersDto, error) {
	r, resp, err := c.UserManagementAPI.GetUsers(ctx, accountUUID).ServiceUsers(true).Execute() //TODO: who are service users? do we need them?
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err); err != nil {
		return nil, err
	}

	return r.Items, nil
}

func (c *Client) GetGroupsForUser(ctx context.Context, userEmail string, accountUUID string) ([]accountmanagement.AccountGroupDto, error) {
	r, resp, err := c.UserManagementAPI.GetUserGroups(ctx, accountUUID, userEmail).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err); err != nil {
		return nil, err
	}

	return r.Groups, nil
}
