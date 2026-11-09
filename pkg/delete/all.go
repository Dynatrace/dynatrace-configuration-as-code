/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package delete

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/credentialvault"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/segment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/slo"
)

type Purger interface {
	DeleteAll(context.Context) error
}

// All collects and deletes ALL configuration objects using the provided ClientSet.
// To delete specific configurations use Configs instead!
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - clients (ClientSet): A set of API clients used to collect and delete configurations from an environment.
func All(ctx context.Context, clients client.ClientSet, apis api.APIs) error {
	candidates := []struct {
		name      string
		available bool
		newPurger func() Purger
	}{
		{"classic", clients.ConfigClient != nil, func() Purger { return classic.NewPurger(clients.ConfigClient, apis) }},
		{"settings", clients.SettingsClient != nil, func() Purger { return settings.NewDeleter(clients.SettingsClient) }},
		{"automation", clients.AutClient != nil, func() Purger { return automation.NewDeleter(clients.AutClient) }},
		{"Grail bucket", clients.BucketClient != nil, func() Purger { return bucket.NewDeleter(clients.BucketClient) }},
		{"document", clients.DocumentClient != nil, func() Purger { return document.NewDeleter(clients.DocumentClient) }},
		{"segment", clients.SegmentClient != nil, func() Purger { return segment.NewDeleter(clients.SegmentClient) }},
		{"SLO-v2", clients.ServiceLevelObjectiveClient != nil, func() Purger { return slo.NewDeleter(clients.ServiceLevelObjectiveClient) }},
		{"credential-vault-v2", clients.CredentialVaultClient != nil, func() Purger { return credentialvault.NewDeleter(clients.CredentialVaultClient) }},
	}

	purgers := make([]Purger, 0, len(candidates))
	for _, c := range candidates {
		if !c.available {
			slog.WarnContext(ctx, fmt.Sprintf("Skipped deletion of %s configurations as API client was unavailable.", c.name))
			continue
		}
		purgers = append(purgers, c.newPurger())
	}

	return all(ctx, purgers)
}

func all(ctx context.Context, purgers []Purger) error {
	errCount := 0

	for _, pp := range purgers {
		if err := pp.DeleteAll(ctx); err != nil {
			errCount++
		}
	}

	if errCount > 0 {
		return fmt.Errorf("failed to delete all configurations for %d types", errCount)
	}
	return nil
}
