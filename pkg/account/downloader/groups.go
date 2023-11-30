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
	"github.com/google/uuid"
)

func (a *Account) Groups() ([]account.Group, error) {
	ctx := context.TODO()
	dtos, err := a.getGroups(ctx)
	if err != nil {
		return nil, err
	}

	var retVal []account.Group
	for _, dto := range dtos {
		pp, err := a.getPermissionFor(ctx, *dto.Uuid)
		if err != nil {
			return nil, err
		}

		a := &account.Account{
			Permissions: extractAccountPermissions(pp.Permissions),
		}

		var e []account.Environment
		for scope, permissions := range sortPermissionsByTenant(pp.Permissions) {
			e = append(e, account.Environment{
				Name:        scope,
				Permissions: permissions,
			})
		}
		retVal = append(retVal, account.Group{
			ID:             uuid.New().String(),
			Name:           dto.Name,
			Description:    dto.GetDescription(),
			Account:        a,
			Environment:    e,
			ManagementZone: nil,
			OriginObjectID: *dto.Uuid,
		})
	}
	return retVal, nil
}

func (a *Account) getGroups(ctx context.Context) ([]accountmanagement.GetGroupDto, error) {
	log.Debug("Downloading groups for account %q", a.accountInfo)
	r, resp, err := a.httpClient.GroupManagementAPI.GetGroups(ctx, a.accountInfo.AccountUUID).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to get groups"); err != nil {
		return nil, err
	}
	if r != nil && int(r.Count) != len(r.Items) {
		return nil, errors.New("the received data are inconsistent")
	}

	log.Debug("%d group downloaded", len(r.Items))

	return r.Items, nil
}

func (a *Account) getPermissionFor(ctx context.Context, groupUuid string) (*accountmanagement.PermissionsGroupDto, error) {
	log.Debug("Downloading permissions for group %q", groupUuid) //TODO: or should be account.Group.ID ?
	r, resp, err := a.httpClient.PermissionManagementAPI.GetGroupPermissions(ctx, a.accountInfo.AccountUUID, groupUuid).Execute()
	defer closeResponseBody(resp)

	if err = handleClientResponseError(resp, err, "unable to get groups"); err != nil {
		return nil, err
	}

	return r, nil
}

func extractAccountPermissions(p []accountmanagement.PermissionsDto) []string {
	var retVal []string
	for i := range p {
		if p[i].ScopeType == "account" {
			retVal = append(retVal, p[i].PermissionName)
		}
	}
	return retVal
}

func sortPermissionsByTenant(permissionDTOs []accountmanagement.PermissionsDto) map[string][]string {
	retVal := map[string][]string{}
	for _, p := range permissionDTOs {
		if p.ScopeType == "tenant" {
			retVal[p.Scope] = append(retVal[p.Scope], p.GetPermissionName())
		}
	}
	return retVal
}
