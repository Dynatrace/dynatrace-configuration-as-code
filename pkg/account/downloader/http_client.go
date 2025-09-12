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

//go:generate mockgen -source http_client.go -package=http -destination=internal/http/client_mock.go
type httpClient interface {
	GetUsers(ctx context.Context, accountUUID string) ([]accountmanagement.UsersDto, error)
	GetServiceUsers(ctx context.Context, accountUUID string) ([]accountmanagement.ExternalServiceUserDto, error)
	GetGroupsForUser(ctx context.Context, userEmail string, accountUUID string) (*accountmanagement.GroupUserDto, error)
	GetBoundaries(ctx context.Context, account string) ([]accountmanagement.PolicyBoundaryOverview, error)
	GetPolicies(ctx context.Context, account string) ([]accountmanagement.PolicyOverview, error)
	GetPolicyDefinition(ctx context.Context, dto accountmanagement.PolicyOverview) (*accountmanagement.LevelPolicyDto, error)
	GetPolicyGroupBindings(ctx context.Context, levelType string, levelId string) (*accountmanagement.LevelPolicyBindingDto, error)
	GetGroups(ctx context.Context, accUUID string) ([]accountmanagement.GetGroupDto, error)
	GetPermissionFor(ctx context.Context, accUUID string, groupUUID string) (*accountmanagement.PermissionsGroupDto, error)
	GetEnvironmentsAndMZones(ctx context.Context, account string) ([]accountmanagement.TenantResourceDto, []accountmanagement.ManagementZoneResourceDto, error)
}
