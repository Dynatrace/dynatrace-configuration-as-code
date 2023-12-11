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
)

type (
	Users []user

	user struct {
		user      *account.User
		dto       *accountmanagement.UsersDto
		dtoGroups []accountmanagement.AccountGroupDto
	}
)

func (a *Account) users(ctx context.Context, groups Groups) (Users, error) {
	log.Info("Downloading users...")
	dtos, err := a.httpClient.GetUsers(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, err
	}

	retVal := make(Users, 0, len(dtos))
	for i := range dtos {
		log.Info("Downloading details for user %q", dtos[i].Email)
		dtoGroups, err := a.httpClient.GetGroupsForUser(ctx, dtos[i].Email, a.accountInfo.AccountUUID)
		if err != nil {
			return nil, err
		}

		g := &account.User{
			Email:  dtos[i].Email,
			Groups: groups.refFromDTOs(dtoGroups),
		}

		retVal = append(retVal, user{
			user:      g,
			dto:       &dtos[i],
			dtoGroups: dtoGroups,
		})
	}

	log.Info("%d users downloads.", len(retVal))
	return retVal, nil
}

func (u Users) asAccountUsers() map[account.UserId]account.User {
	retVal := make(map[account.UserId]account.User, len(u))
	for i := range u {
		retVal[u[i].user.Email] = *u[i].user
	}
	return retVal
}
