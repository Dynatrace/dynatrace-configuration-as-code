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
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

type Account struct {
	httpClient  *accounts.Client
	accountInfo *account.AccountInfo
}

func New(accountInfo *account.AccountInfo, client *accounts.Client) *Account {
	return &Account{
		httpClient:  client,
		accountInfo: accountInfo,
	}
}

func (a *Account) DownloadConfiguration() (*account.Resources, error) {
	gg, err := a.Groups()
	if err != nil {
		return nil, err
	}
	groups := make(map[account.GroupId]account.Group)
	for i := range gg {
		groups[gg[i].ID] = gg[i]
	}

	pp, err := a.Policies()
	if err != nil {
		return nil, err
	}
	policies := make(map[account.PolicyId]account.Policy, len(pp))
	for i := range pp {
		policies[pp[i].ID] = pp[i]
	}

	uu, err := a.Users(gg)
	if err != nil {
		return nil, err
	}
	users := make(map[account.UserId]account.User)
	for i := range uu {
		users[uu[i].Email] = uu[i]
	}

	r := account.Resources{
		Users:    users,
		Groups:   groups,
		Policies: policies,
	}

	return &r, nil
}
