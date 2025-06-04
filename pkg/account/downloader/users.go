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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

type (
	Users []user

	user struct {
		user      *account.User
		dto       *accountmanagement.UsersDto
		dtoGroups *accountmanagement.GroupUserDto
	}
)

func (a *Downloader) users(ctx context.Context, groups Groups) (Users, error) {
	log.InfoContext(ctx, "Downloading users")
	dtos, err := a.httpClient.GetUsers(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of users for account %q from DT: %w", a.accountInfo, err)
	}

	retVal := make(Users, 0, len(dtos))
	for i := range dtos {
		log.DebugContext(ctx, "Downloading details for user %q", secret.Email(dtos[i].Email))
		dtoGroups, err := a.httpClient.GetGroupsForUser(ctx, dtos[i].Email, a.accountInfo.AccountUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get a list of bind groups for user %q: %w", secret.Email(dtos[i].Email), err)
		}
		if dtoGroups == nil {
			return nil, fmt.Errorf("failed to get a list of bind groups for the user %q", secret.Email(dtos[i].Email))
		}

		g := &account.User{
			Email:  secret.Email(dtos[i].Email),
			Groups: groups.refFromDTOs(dtoGroups.Groups),
		}

		retVal = append(retVal, user{
			user:      g,
			dto:       &dtos[i],
			dtoGroups: dtoGroups,
		})
	}

	log.InfoContext(ctx, "Fetched %d users", len(retVal))
	return retVal, nil
}

func (u Users) asAccountUsers() map[account.UserId]account.User {
	retVal := make(map[account.UserId]account.User, len(u))
	for i := range u {
		retVal[u[i].user.Email.Value()] = *u[i].user
	}
	return retVal
}
