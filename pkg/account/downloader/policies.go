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
	"fmt"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/google/uuid"
)

type (
	Policies []policy

	policy struct {
		policy        *account.Policy
		dto           *accountmanagement.PolicyOverview
		dtoDefinition *accountmanagement.LevelPolicyDto
	}
)

func (a *Account) Policies() (Policies, error) {
	return a.policies(context.TODO())
}

func (a *Account) policies(ctx context.Context) (Policies, error) {
	log.Info("Downloading policies for account %q", a.accountInfo)
	dto, err := a.httpClient2.GetPoliciesFroAccount(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies for account %q: %w", a.accountInfo, err)
	}
	log.Debug("%d policy downloaded (global + custom)", len(dto))

	retVal := make(Policies, 0, len(dto))

	for i := range dto {
		var dtoDef *accountmanagement.LevelPolicyDto
		var p *account.Policy
		if isCustom(dto[i]) {
			log.Debug("Downloading definition for policy %q (uuid: %q)", dto[i].Name, dto[i].Uuid) //TODO: or should be account.Policy.ID ?
			dtoDef, err = a.httpClient2.GetPolicyDefinition(ctx, dto[i])
			if err != nil {
				return nil, err
			}

			p = toAccountPolicy(&dto[i], dtoDef)
		}

		retVal = append(retVal, policy{
			policy:        p,
			dto:           &dto[i],
			dtoDefinition: dtoDef,
		})
	}

	log.Debug("Number of policies: %d", len(retVal.asAccountPolicies()))
	return retVal, nil
}

func toAccountPolicy(dto *accountmanagement.PolicyOverview, dtoDef *accountmanagement.LevelPolicyDto) *account.Policy {
	return &account.Policy{
		ID:             uuid.New().String(),
		Name:           dto.Name,
		Level:          getPolicyLevel(dto),
		Description:    dto.Description,
		Policy:         dtoDef.StatementQuery,
		OriginObjectID: dto.Uuid,
	}
}

func (p Policies) asAccountPolicies() map[account.PolicyId]account.Policy {
	retVal := make(map[account.PolicyId]account.Policy)
	for i := range p {
		if p[i].isCustom() {
			retVal[p[i].policy.ID] = *p[i].policy
		}
	}
	return retVal
}

func (p *policy) isCustom() bool {
	return isCustom(*p.dto)
}

func isCustom(dto accountmanagement.PolicyOverview) bool {
	return dto.LevelType == "account" || dto.LevelType == "environment"
}

func (p *policy) Ref() *account.Ref {
	return nil
}

func getPolicyLevel(dto *accountmanagement.PolicyOverview) account.PolicyLevel {
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
