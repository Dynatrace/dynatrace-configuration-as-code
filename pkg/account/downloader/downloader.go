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
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/downloader/internal/http"
)

type Downloader struct {
	httpClient  httpClient
	accountInfo *account.AccountInfo
}

func New(accountInfo *account.AccountInfo, client *accounts.Client) *Downloader {
	return &Downloader{
		httpClient:  (*http.Client)(client),
		accountInfo: accountInfo,
	}
}

func (a *Downloader) DownloadResources(ctx context.Context) (*account.Resources, error) {
	slog.InfoContext(ctx, "Starting download")
	tenants, err := a.environments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch environments: %w", err)
	}

	boundaries, err := a.boundaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch boundaries: %w", err)
	}

	policies, err := a.policies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch policies: %w", err)
	}

	groups, err := a.groups(ctx, policies, boundaries, tenants)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}

	users, err := a.users(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	serviceUsers, err := a.serviceUsers(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch service users: %w", err)
	}

	r := account.Resources{
		Users:        users.asAccountUsers(),
		ServiceUsers: serviceUsers.asAccountServiceUsers(),
		Groups:       groups.asAccountGroups(),
		Policies:     policies.asAccountPolicies(),
		Boundaries:   boundaries.asAccountBoundaries(),
	}

	return &r, nil
}
