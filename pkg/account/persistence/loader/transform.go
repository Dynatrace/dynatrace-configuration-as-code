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
	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/internal/types"
)

func transformToAccountResources(resources *persistence.Resources) *account.Resources {
	return &account.Resources{
		Policies:     transformPolicies(resources.Policies),
		Groups:       transformGroups(resources.Groups),
		Users:        transformUsers(resources.Users),
		ServiceUsers: transformServiceUsers(resources.ServiceUsers),
	}
}

func transformPolicies(pPolicies map[string]persistence.Policy) map[account.PolicyId]account.Policy {
	policies := make(map[account.PolicyId]account.Policy, len(pPolicies))
	for id, pPolicy := range pPolicies {
		policies[id] = transformPolicy(pPolicy)
	}
	return policies
}

func transformPolicy(pPolicy persistence.Policy) account.Policy {
	return account.Policy{
		ID:             pPolicy.ID,
		Name:           pPolicy.Name,
		Level:          transformLevel(pPolicy.Level),
		Description:    pPolicy.Description,
		Policy:         pPolicy.Policy,
		OriginObjectID: pPolicy.OriginObjectID,
	}
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
	for id, pGroup := range pGroups {
		groups[id] = transformGroup(pGroup)
	}
	return groups
}

func transformGroup(pGroup persistence.Group) account.Group {
	return account.Group{
		ID:                       pGroup.ID,
		Name:                     pGroup.Name,
		Description:              pGroup.Description,
		FederatedAttributeValues: pGroup.FederatedAttributeValues,
		Account:                  transformAccount(pGroup.Account),
		Environment:              transformEnvironments(pGroup.Environment),
		ManagementZone:           transformManagementZones(pGroup.ManagementZone),
		OriginObjectID:           pGroup.OriginObjectID,
	}
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
	for id, pUser := range pUsers {
		users[id] = transformUser(pUser)
	}
	return users
}

func transformUser(pUser persistence.User) account.User {
	return account.User{
		Email:  pUser.Email,
		Groups: transformReferences(pUser.Groups),
	}
}

func transformServiceUsers(pServiceUsers map[string]persistence.ServiceUser) map[account.ServiceUserId]account.ServiceUser {
	serviceUsers := make(map[account.ServiceUserId]account.ServiceUser, len(pServiceUsers))
	for id, pServiceUser := range pServiceUsers {
		serviceUsers[id] = transformServiceUser(pServiceUser)
	}
	return serviceUsers
}

func transformServiceUser(pServiceUser persistence.ServiceUser) account.ServiceUser {
	return account.ServiceUser{
		Name:        pServiceUser.Name,
		Description: pServiceUser.Description,
		Groups:      transformReferences(pServiceUser.Groups),
	}
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
