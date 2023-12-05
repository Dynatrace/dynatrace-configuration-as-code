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
)

func (a *Account) Environments() (Environments, error) {
	return a.environments(context.TODO())
}

func (a *Account) environments(ctx context.Context) (Environments, error) {
	dto, err := a.httpClient2.GetTenants(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, err
	}

	retVal := make(Environments, 0, len(dto.TenantResources))
	for i := range dto.TenantResources {
		e := fromTenantResourceDto(dto.TenantResources[i])
		e.managementZones = fromManagementZoneResourceDto(dto.ManagementZoneResources, dto.TenantResources[i].Id)
		retVal = append(retVal, e)
	}

	return retVal, nil
}

func fromTenantResourceDto(dto accountmanagement.TenantResourceDto) environment {
	return environment{
		id:   dto.Id,
		name: dto.Name,
	}
}

func fromManagementZoneResourceDto(dtos []accountmanagement.ManagementZoneResourceDto, tenantID string) []managementZone {
	var retVal []managementZone
	for _, dto := range dtos {
		if dto.Parent == tenantID {
			retVal = append(retVal, managementZone{
				name:     dto.Name,
				originID: dto.Id,
			})
		}
	}
	return retVal
}
