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
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
)

type Client accounts.Client

func (c *Client) GetPoliciesFroAccount(ctx context.Context, account string) ([]accountmanagement.PolicyOverview, error) {
	r, resp, err := c.PolicyManagementAPI.GetPolicyOverviewList(ctx, "account", account).Execute()
	defer closeResponseBody(resp)

	if err != nil {
		return nil, err
	}

	return r.PolicyOverviewList, nil
}

func (c *Client) GetPolicyDefinition(ctx context.Context, dto accountmanagement.PolicyOverview) (*accountmanagement.LevelPolicyDto, error) {
	r, resp, err := c.PolicyManagementAPI.GetLevelPolicy(ctx, dto.LevelType, dto.LevelId, dto.Uuid).Execute()
	defer closeResponseBody(resp)

	if err != nil {
		return nil, err
	}

	return r, nil
}
