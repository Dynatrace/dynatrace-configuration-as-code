/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package loader

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/internal/types"
)

func transformToAccountResources(resources *persistence.Resources) *account.Resources {
	return &account.Resources{
		Policies: transformPolicies(resources.Policies),
		Groups:   transformGroups(resources.Groups),
		Users:    transformUsers(resources.Users),
	}
}

func transformPolicies(pPolicies map[string]persistence.Policy) map[account.PolicyId]account.Policy {
	policies := make(map[account.PolicyId]account.Policy, len(pPolicies))
	for id, v := range pPolicies {
		policies[id] = account.Policy{
			ID:             v.ID,
			Name:           v.Name,
			Level:          transformLevel(v.Level),
			Description:    v.Description,
			Policy:         v.Policy,
			OriginObjectID: v.OriginObjectID,
		}
	}
	return policies
}

func transformLevel(pLevel persistence.PolicyLevel) any {
	switch pLevel.Type {
	case persistence.PolicyLevelAccount:
		return account.PolicyLevelAccount{Type: pLevel.Type}
	case persistence.PolicyLevelEnvironment:
		return account.PolicyLevelEnvironment{Type: pLevel.Type, Environment: pLevel.Environment}
	default:
		panic("unable to convert persistence model")
	}
}

func transformGroups(pGroups map[string]persistence.Group) map[account.GroupId]account.Group {
	groups := make(map[account.GroupId]account.Group, len(pGroups))
	for id, v := range pGroups {
		groups[id] = account.Group{
			ID:                       v.ID,
			Name:                     v.Name,
			Description:              v.Description,
			FederatedAttributeValues: v.FederatedAttributeValues,
			Account:                  transformAccount(v.Account),
			Environment:              transformEnvironments(v.Environment),
			ManagementZone:           transformManagementZones(v.ManagementZone),
			OriginObjectID:           v.OriginObjectID,
		}
	}
	return groups
}

func transformAccount(pAccount *persistence.Account) *account.Account {
	if pAccount == nil {
		return nil
	}

	return &account.Account{
		Permissions: pAccount.Permissions,
		Policies:    transformReferences(pAccount.Policies),
	}
}

func transformEnvironments(pEnvironments []persistence.Environment) []account.Environment {
	env := make([]account.Environment, len(pEnvironments))
	for i, e := range pEnvironments {
		env[i] = account.Environment{
			Name:        e.Name,
			Permissions: e.Permissions,
			Policies:    transformReferences(e.Policies),
		}
	}
	return env
}

func transformManagementZones(pManagementZones []persistence.ManagementZone) []account.ManagementZone {
	managementZones := make([]account.ManagementZone, len(pManagementZones))
	for i, m := range pManagementZones {
		managementZones[i] = account.ManagementZone{
			Environment:    m.Environment,
			ManagementZone: m.ManagementZone,
			Permissions:    m.Permissions,
		}
	}
	return managementZones
}

func transformUsers(pUsers map[string]persistence.User) map[account.UserId]account.User {
	users := make(map[account.UserId]account.User, len(pUsers))
	for id, v := range pUsers {
		users[id] = account.User{
			Email:  v.Email,
			Groups: transformReferences(v.Groups),
		}
	}
	return users
}

func transformReferences(pReferences []persistence.Reference) []account.Ref {
	res := make([]account.Ref, len(pReferences))
	for i, el := range pReferences {
		switch el.Type {
		case persistence.ReferenceType:
			res[i] = account.Reference{Id: el.Id}
		case "":
			res[i] = account.StrReference(el.Value)
		default:
			panic("unable to convert persistence model")
		}
	}
	return res
}
