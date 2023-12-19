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
)

type (
	Environments []environment

	environment struct {
		id              string
		name            string
		managementZones []managementZone
	}

	managementZone struct {
		name     string
		originID string
	}
)

func (e environment) String() string {
	return e.id
}

func (e Environments) getMzoneName(originID string) string {
	for _, env := range e {
		for _, mz := range env.managementZones {
			if mz.originID == originID {
				return mz.name
			}
		}
	}
	return ""
}

func (a *Downloader) environments(ctx context.Context) (Environments, error) {
	log.WithCtxFields(ctx).Info("Fetching environments")

	dto, err := a.httpClient.GetEnvironments(ctx, a.accountInfo.AccountUUID)
	if err != nil {
		return nil, err
	}

	retVal := make(Environments, 0, len(dto.TenantResources))
	for i := range dto.TenantResources {
		e := fromTenantResourceDto(dto.TenantResources[i])
		e.managementZones = fromManagementZoneResourceDto(dto.ManagementZoneResources, dto.TenantResources[i].Id)
		retVal = append(retVal, e)
	}

	log.WithCtxFields(ctx).Info("Fetched environments: %q", retVal)
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
