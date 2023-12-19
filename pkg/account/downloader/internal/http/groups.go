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

func (c *Client) GetGroups(ctx context.Context, accUUID string) ([]accountmanagement.GetGroupDto, error) {
	r, resp, err := c.GroupManagementAPI.GetGroups(ctx, accUUID).Execute()
	defer closeResponseBody(resp)

	if err != nil {
		return nil, err
	}

	return r.Items, nil
}

func (c *Client) GetPermissionFor(ctx context.Context, accUUID string, groupUUID string) (*accountmanagement.PermissionsGroupDto, error) {
	r, resp, err := c.PermissionManagementAPI.GetGroupPermissions(ctx, accUUID, groupUUID).Execute()
	defer closeResponseBody(resp)

	if err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) GetBindingsFor(ctx context.Context, levelType string, levelId string) (*accountmanagement.LevelPolicyBindingDto, error) {
	r, resp, err := c.PolicyManagementAPI.GetAllLevelPoliciesBindings(ctx, levelType, levelId).Execute()
	defer closeResponseBody(resp)

	if err != nil {
		return nil, err
	}

	return r, nil
}
