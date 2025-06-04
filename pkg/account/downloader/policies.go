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
	stringutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

type (
	Policies []policy

	policy struct {
		policy        *account.Policy
		dto           *accountmanagement.PolicyOverview
		dtoDefinition *accountmanagement.LevelPolicyDto
	}
)

func (a *Downloader) policies(ctx context.Context) (Policies, error) {
	log.WithCtxFields(ctx).InfoContext(ctx, "Downloading policies")
	dtos, err := a.httpClient.GetPolicies(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of policies for account %q from DT: %w", a.accountInfo, err)
	}
	log.WithCtxFields(ctx).Debug("Downloaded %d policies (global + custom)", len(dtos))

	retVal := make(Policies, 0, len(dtos))
	for i := range dtos {
		var dtoDef *accountmanagement.LevelPolicyDto
		var p *account.Policy
		if isCustom(dtos[i]) {
			log.WithCtxFields(ctx).Debug("Downloading definition for policy %q (uuid: %q)", dtos[i].Name, dtos[i].Uuid)
			dtoDef, err = a.httpClient.GetPolicyDefinition(ctx, dtos[i])
			if err != nil {
				return nil, fmt.Errorf("failed to get the definition for the policy %q (uuid: %q) from DT: %w", dtos[i].Name, dtos[i].Uuid, err)
			}
			if dtoDef == nil {
				return nil, fmt.Errorf("failed to get the definition for the policy %q (uuid: %q) from DT", dtos[i].Name, dtos[i].Uuid)
			}

			p = toAccountPolicy(&dtos[i], dtoDef)
		}

		retVal = append(retVal, policy{
			policy:        p,
			dto:           &dtos[i],
			dtoDefinition: dtoDef,
		})
	}

	log.WithCtxFields(ctx).InfoContext(ctx, "Downloaded %d policies", len(retVal.asAccountPolicies()))
	return retVal, nil
}

func toAccountPolicy(dto *accountmanagement.PolicyOverview, dtoDef *accountmanagement.LevelPolicyDto) *account.Policy {
	return &account.Policy{
		ID:             stringutils.Sanitize(dto.Name),
		Name:           dto.Name,
		Level:          toAccountPolicyLevel(dto),
		Description:    dto.Description,
		Policy:         dtoDef.StatementQuery,
		OriginObjectID: dto.Uuid,
	}
}

func toAccountPolicyLevel(dto *accountmanagement.PolicyOverview) account.PolicyLevel {
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

func (p Policies) asAccountPolicies() map[account.PolicyId]account.Policy {
	retVal := make(map[account.PolicyId]account.Policy)
	for i := range p {
		if p[i].isCustom() {
			retVal[p[i].policy.ID] = *p[i].policy
		}
	}
	return retVal
}

func (p Policies) RefOn(policyUUID ...string) []account.Ref {
	var retVal []account.Ref
	for _, pol := range p {
		for _, uuid := range policyUUID {
			if pol.dto.Uuid == uuid {
				retVal = append(retVal, pol.RefOn())
				break
			}
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

func (p *policy) RefOn() account.Ref {
	if p.isCustom() {
		return account.Reference{Id: p.policy.ID}
	}
	return account.StrReference(p.dto.Name)
}
