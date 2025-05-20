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

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
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
	log.WithCtxFields(ctx).Info("Starting download")
	ctx = logr.NewContextWithSlogLogger(ctx, log.WithCtxFields(ctx).SLogger())
	tenants, err := a.environments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch environments: %w", err)
	}

	policies, err := a.policies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch policies: %w", err)
	}

	groups, err := a.groups(ctx, policies, tenants)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}

	users, err := a.users(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	serviceUsers := ServiceUsers{}
	if featureflags.ServiceUsers.Enabled() {
		serviceUsers, err = a.serviceUsers(ctx, groups)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch service users: %w", err)
		}
	}

	r := account.Resources{
		Users:        users.asAccountUsers(),
		ServiceUsers: serviceUsers.asAccountServiceUsers(),
		Groups:       groups.asAccountGroups(),
		Policies:     policies.asAccountPolicies(),
	}

	return &r, nil
}
