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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"io"
	"net/http"
)

func (d *Account) Users() ([]account.User, error) {
	uu, err := d.getUsers(context.TODO())
	if err != nil {
		return nil, err
	}

	var users []account.User
	for _, u := range uu {
		gg, err := d.getGroupsFor(context.TODO(), u.Email)
		if err != nil {
			return nil, err
		}

		groups := make([]account.Ref, 0, len(gg))
		for _, g := range gg { //TODO: change to real reference
			groups = append(groups, account.StrReference(g.GroupName))
		}

		users = append(users, account.User{
			Email:  u.Email,
			Groups: groups,
		})
	}

	return users, nil
}

func (d *Account) getUsers(ctx context.Context) ([]accountmanagement.UsersDto, error) {
	log.Debug("Downloading users for account %q", d.accountInfo)
	r, resp, err := d.httpClient2.UserManagementAPI.GetUsers(ctx, d.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to get users"); err != nil {
		return nil, err
	}
	if r != nil && int(r.Count) != len(r.Items) {
		return nil, errors.New("the received data are incomplete")
	}
	log.Debug("%d users record reviewed", len(r.Items))

	return r.Items, nil
}

func (d *Account) getGroupsFor(ctx context.Context, userEmail string) ([]accountmanagement.AccountGroupDto, error) {
	log.Debug("Downloading list of groups for user %q", userEmail)
	r, resp, err := d.httpClient2.UserManagementAPI.GetUserGroups(ctx, d.accountInfo.AccountUUID, userEmail).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to get groups for the users"); err != nil {
		return nil, err
	}
	return r.Groups, nil
}

func closeResponseBody(resp *http.Response) {
	_ = resp.Body.Close()
}

func handleClientResponseError(resp *http.Response, clientErr error, errMessage string) error {
	if clientErr != nil && resp == nil {
		return fmt.Errorf(errMessage+": %w", clientErr)
	}

	if !rest.IsSuccess(resp) && resp.StatusCode != http.StatusNotFound {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body %w", err)
		}
		return fmt.Errorf(errMessage+" (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
