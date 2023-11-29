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

package downloader

import (
	"context"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/google/uuid"
)

func (a *Account) Policies() ([]account.Policy, error) {
	dtos, err := a.getPolicies(context.TODO())
	if err != nil {
		return nil, err
	}

	var retVal []account.Policy
	for _, dto := range dtos {
		l := getPolicyLevel(dto)
		if l != nil {
			retVal = append(retVal, account.Policy{
				ID:             uuid.New().String(),
				Name:           dto.Name,
				Level:          getPolicyLevel(dto),
				Description:    dto.Description,
				OriginObjectID: dto.Uuid,
			})
		}
	}
	return retVal, nil
}

func getPolicyLevel(dto accountmanagement.PolicyOverview) account.PolicyLevel {
	var retVal account.PolicyLevel
	switch dto.LevelType {
	case "account":
		retVal = account.PolicyLevelAccount{Type: "account"}
	case "environment":
		retVal = account.PolicyLevelEnvironment{
			Type:        "environment",
			Environment: dto.LevelId,
		}
	}
	return retVal
}

func (a *Account) getPolicies(ctx context.Context) ([]accountmanagement.PolicyOverview, error) {
	log.Debug("Downloading policies for account %q", a.accountInfo)
	r, resp, err := a.httpClient.PolicyManagementAPI.GetPolicyOverviewList(ctx, "account", a.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to get groups"); err != nil {
		return nil, err
	}

	log.Debug("%d policy downloaded", len(r.PolicyOverviewList))

	return r.PolicyOverviewList, nil
}
