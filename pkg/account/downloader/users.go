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
	"errors"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

func (a *Account) Users(knownGroups []account.Group) ([]account.User, error) {
	dtos, err := a.getUsers(context.TODO())
	if err != nil {
		return nil, err
	}

	var users []account.User
	for _, dto := range dtos {
		gg, err := a.getGroupsForUser(context.TODO(), dto.Email)
		if err != nil {
			return nil, err
		}

		groups := make([]account.Ref, 0, len(gg))
		for _, g := range gg {
			groups = append(groups, createReferenceOnGroup(g, knownGroups))
		}

		users = append(users, account.User{
			Email:  dto.Email,
			Groups: groups,
		})
	}

	return users, nil
}

func (a *Account) getUsers(ctx context.Context) ([]accountmanagement.UsersDto, error) {
	log.Debug("Downloading users for account %q", a.accountInfo)
	r, resp, err := a.httpClient.UserManagementAPI.GetUsers(ctx, a.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to get users"); err != nil {
		return nil, err
	}
	if r != nil && int(r.Count) != len(r.Items) {
		return nil, errors.New("the received data are incomplete")
	}
	log.Debug("%d user downloaded", len(r.Items))

	return r.Items, nil
}

func (a *Account) getGroupsForUser(ctx context.Context, userEmail string) ([]accountmanagement.AccountGroupDto, error) {
	log.Debug("Downloading list of groups for user %q", userEmail)
	r, resp, err := a.httpClient.UserManagementAPI.GetUserGroups(ctx, a.accountInfo.AccountUUID, userEmail).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to get groups for the users"); err != nil {
		return nil, err
	}
	return r.Groups, nil
}

func createReferenceOnGroup(dto accountmanagement.AccountGroupDto, groups []account.Group) account.Ref {
	for _, kg := range groups {
		if kg.OriginObjectID == dto.Uuid {
			return account.Reference{
				Type: "reference",
				Id:   kg.ID,
			}
		}
	}

	return account.StrReference(dto.GroupName)
}
