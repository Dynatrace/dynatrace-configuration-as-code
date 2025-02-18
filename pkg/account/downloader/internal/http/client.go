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
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
)

type Client accounts.Client

func (c *Client) GetUsers(ctx context.Context, accountUUID string) ([]accountmanagement.UsersDto, error) {
	r, resp, err := c.UserManagementAPI.GetUsers(ctx, accountUUID).ServiceUsers(false).Execute() //service users are not yet implemented in DT
	defer closeResponseBody(resp)
	if err = getErrorMessageFromResponse(resp, err); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("the received data are empty")
	}
	return r.Items, nil
}

func (c *Client) GetServiceUsers(ctx context.Context, accountUUID string) ([]accountmanagement.ExternalServiceUserDto, error) {
	serviceUsers := []accountmanagement.ExternalServiceUserDto{}
	const pageSize = 10
	for page := (int32)(1); page < math.MaxInt32; page++ {
		r, err := c.getServiceUsersPage(ctx, accountUUID, page, pageSize)
		if err != nil {
			return nil, err
		}

		serviceUsers = append(serviceUsers, r.Results...)

		if r.NextPageKey == nil {
			break
		}
	}

	return serviceUsers, nil
}

func (c *Client) getServiceUsersPage(ctx context.Context, accountUUID string, page int32, pageSize int32) (*accountmanagement.ExternalServiceUsersPageDto, error) {
	r, resp, err := c.ServiceUserManagementAPI.GetServiceUsersFromAccount(ctx, accountUUID).Page(page).PageSize(pageSize).Execute()
	defer closeResponseBody(resp)
	if err = getErrorMessageFromResponse(resp, err); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("the received data are empty")
	}
	return r, nil
}

func (c *Client) GetGroupsForUser(ctx context.Context, userEmail string, accountUUID string) (*accountmanagement.GroupUserDto, error) {
	r, resp, err := c.UserManagementAPI.GetUserGroups(ctx, accountUUID, userEmail).Execute()
	defer closeResponseBody(resp)
	if is404(resp) {
		return nil, nil
	}
	if err = getErrorMessageFromResponse(resp, err); err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) GetPolicies(ctx context.Context, account string) ([]accountmanagement.PolicyOverview, error) {
	r, resp, err := c.PolicyManagementAPI.GetPolicyOverviewList(ctx, account, "account").Execute()
	defer closeResponseBody(resp)
	if err = getErrorMessageFromResponse(resp, err); err != nil {
		return nil, err
	}
	return r.PolicyOverviewList, nil
}

func (c *Client) GetPolicyDefinition(ctx context.Context, dto accountmanagement.PolicyOverview) (*accountmanagement.LevelPolicyDto, error) {
	r, resp, err := c.PolicyManagementAPI.GetLevelPolicy(ctx, dto.Uuid, dto.LevelId, dto.LevelType).Execute()
	defer closeResponseBody(resp)
	if is404(resp) {
		return nil, nil
	}
	if err = getErrorMessageFromResponse(resp, err); err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) GetGroups(ctx context.Context, accUUID string) ([]accountmanagement.GetGroupDto, error) {
	r, resp, err := c.GroupManagementAPI.GetGroups(ctx, accUUID).Execute()
	defer closeResponseBody(resp)
	if err = getErrorMessageFromResponse(resp, err); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("no group response data received")
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

func (c *Client) GetPolicyGroupBindings(ctx context.Context, levelType string, levelId string) (*accountmanagement.LevelPolicyBindingDto, error) {
	r, resp, err := c.PolicyManagementAPI.GetAllLevelPoliciesBindings(ctx, levelId, levelType).Execute()
	defer closeResponseBody(resp)
	if err = getErrorMessageFromResponse(resp, err); err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) GetEnvironmentsAndMZones(ctx context.Context, account string) ([]accountmanagement.TenantResourceDto, []accountmanagement.ManagementZoneResourceDto, error) {
	r, resp, err := c.EnvironmentManagementAPI.GetEnvironmentResources(ctx, account).Execute()
	defer closeResponseBody(resp)
	if err = getErrorMessageFromResponse(resp, err); err != nil {
		return nil, nil, err
	}
	return r.GetTenantResources(), r.GetManagementZoneResources(), nil
}

func is404(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}

// similar as handleClientResponseError
func getErrorMessageFromResponse(resp *http.Response, clientErr error) error {
	if clientErr != nil && resp != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w: %w", err, clientErr)
		}
		return fmt.Errorf("(HTTP %d): %s", resp.StatusCode, string(body))
	}
	return clientErr
}

func closeResponseBody(resp *http.Response) {
	if resp != nil {
		_ = resp.Body.Close()
	}
}
