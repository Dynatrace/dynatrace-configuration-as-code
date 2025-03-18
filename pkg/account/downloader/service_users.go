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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
)

type (
	ServiceUsers []serviceUser

	serviceUser struct {
		serviceUser *account.ServiceUser
		dto         *accountmanagement.ExternalServiceUserDto
		dtoGroups   *accountmanagement.GroupUserDto
	}
)

func (a *Downloader) serviceUsers(ctx context.Context, groups Groups) (ServiceUsers, error) {
	log.WithCtxFields(ctx).Info("Downloading service users")
	dtos, err := a.httpClient.GetServiceUsers(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of service users for account %q from DT: %w", a.accountInfo, err)
	}

	retVal := make(ServiceUsers, 0, len(dtos))
	for _, dto := range dtos {
		log.WithCtxFields(ctx).Debug("Downloading details for service user %q", dto.Name)
		dtoGroups, err := a.httpClient.GetGroupsForUser(ctx, dto.Email, a.accountInfo.AccountUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get a list of bind groups for service user %q: %w", dto.Name, err)
		}
		if dtoGroups == nil {
			return nil, fmt.Errorf("failed to get a list of bind groups for the service user %q", dto.Name)
		}

		su := &account.ServiceUser{
			Name:           dto.Name,
			OriginObjectID: dtoGroups.Uid,
			Description:    dto.GetDescription(),
			Groups:         groups.refFromDTOs(dtoGroups.Groups),
		}

		retVal = append(retVal, serviceUser{
			serviceUser: su,
			dto:         &dto,
			dtoGroups:   dtoGroups,
		})
	}

	log.WithCtxFields(ctx).Info("Fetched %d service users", len(retVal))
	return retVal, nil
}

func (sus ServiceUsers) asAccountServiceUsers() []account.ServiceUser {
	retVal := make([]account.ServiceUser, 0, len(sus))
	for _, su := range sus {
		retVal = append(retVal, *su.serviceUser)
	}
	return retVal
}
