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
	"strings"
)

type (
	Groups []group

	levelID = string
	group   struct {
		group         *account.Group
		dto           *accountmanagement.GetGroupDto
		permissionDTO *accountmanagement.PermissionsGroupDto
		bindings      map[levelID]*accountmanagement.LevelPolicyBindingDto
	}
)

func (a *Downloader) groups(ctx context.Context, policies Policies, tenants Environments) (Groups, error) {
	log.Info("Downloading groups...")
	groupDTOs, err := a.httpClient.GetGroups(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, err
	}

	var retVal Groups
	for i := range groupDTOs {
		if groupDTOs[i].Uuid == nil {
			log.Error("Group %q does not have a uuid set, skipping", groupDTOs[i].Name)
			continue
		}

		log.Debug("Downloading definition for group %q (uuid: %q)", groupDTOs[i].Name, *groupDTOs[i].Uuid)
		g := group{
			dto:      &groupDTOs[i],
			bindings: make(map[levelID]*accountmanagement.LevelPolicyBindingDto, len(tenants)),
		}

		log.Debug("Downloading group's policies...")
		binding, err := a.httpClient.GetBindingsFor(ctx, "account", a.accountInfo.AccountUUID)
		if err != nil {
			return nil, err
		}
		g.bindings["account"] = binding

		log.Debug("Downloading group's permissions...")
		perDTO, err := a.httpClient.GetPermissionFor(ctx, a.accountInfo.AccountUUID, *groupDTOs[i].Uuid)
		if err != nil {
			return nil, err
		}
		g.permissionDTO = perDTO

		acc := account.Account{
			Permissions: getPermissionFor("account", perDTO),
			Policies:    policies.RefOn(getPoliciesFor(binding, *g.dto.Uuid)...),
		}

		var envs []account.Environment
		var mzs []account.ManagementZone
		for _, t := range tenants {
			log.Debug("Downloading policies for %q...", t)
			binding, err := a.httpClient.GetBindingsFor(ctx, "environment", t.id)
			if err != nil {
				return nil, err
			}
			g.bindings[t.id] = binding

			envs = append(envs, account.Environment{
				Name:        t.id,
				Permissions: getPermissionFor(t.id, perDTO),
				Policies:    policies.RefOn(getPoliciesFor(binding, *g.dto.Uuid)...),
			})

			for k, v := range getManagementZonesFor(t.id, perDTO) {
				mzs = append(mzs, account.ManagementZone{
					Environment:    t.id,
					ManagementZone: tenants.getMzoneName(k),
					Permissions:    v,
				})
			}

		}

		g.group = &account.Group{
			ID:             uuid.New().String(),
			Name:           g.dto.Name,
			Description:    g.dto.GetDescription(),
			Account:        effectiveAccount(acc),
			Environment:    effectiveEnvironments(envs),
			ManagementZone: mzs,
			OriginObjectID: *g.dto.Uuid,
		}

		retVal = append(retVal, g)
	}
	return retVal, nil
}

func (g Groups) asAccountGroups() map[account.GroupId]account.Group {
	retVal := make(map[account.GroupId]account.Group)
	for i := range g {
		retVal[g[i].group.ID] = *g[i].group
	}
	return retVal
}

func (g Groups) refOn(groupUUID string) account.Ref {
	for i := range g {
		if *g[i].dto.Uuid == groupUUID {
			return account.Reference{
				Type: "reference",
				Id:   g[i].group.ID,
			}
		}
	}
	return nil
}

func (g Groups) refFromDTOs(dtos []accountmanagement.AccountGroupDto) []account.Ref {
	var retVal []account.Ref
	for _, dto := range dtos {
		retVal = append(retVal, g.refOn(dto.Uuid))
	}
	return retVal
}

func getPermissionFor(scope string, perDTOs *accountmanagement.PermissionsGroupDto) []string {
	var retVal []string
	for _, p := range perDTOs.Permissions {
		if p.ScopeType == scope || p.Scope == scope {
			retVal = append(retVal, p.PermissionName)
		}
	}
	return retVal
}

func getManagementZonesFor(scope string, perDTOs *accountmanagement.PermissionsGroupDto) map[string][]string {
	retVal := make(map[string][]string)
	for _, p := range perDTOs.Permissions {
		if p.ScopeType == "management-zone" {
			if after, found := strings.CutPrefix(p.Scope, scope+":"); found {
				retVal[after] = append(retVal[after], p.PermissionName)
			}
		}
	}
	return retVal
}

func getPoliciesFor(binding *accountmanagement.LevelPolicyBindingDto, groupUUID string) []string {
	var retVal []string
	for _, b := range binding.PolicyBindings {
		for _, g := range b.Groups {
			if g == groupUUID {
				retVal = append(retVal, b.PolicyUuid)
				break
			}
		}
	}
	return retVal
}

func effectiveAccount(a account.Account) *account.Account {
	if len(a.Policies) == 0 && len(a.Permissions) == 0 {
		return nil
	}
	return &a
}

func effectiveEnvironments(es []account.Environment) []account.Environment {
	var retVal []account.Environment
	for _, e := range es {
		if len(e.Policies) > 0 || len(e.Permissions) > 0 {
			retVal = append(retVal, e)
		}
	}
	return retVal
}
