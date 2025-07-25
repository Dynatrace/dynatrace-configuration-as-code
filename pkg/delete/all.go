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
	purgers := make([]Purger, 0)
	if clients.ConfigClient == nil {
		slog.WarnContext(ctx, "Skipped deletion of classic configurations as API client was unavailable.")
	} else {
		purgers = append(purgers, classic.NewPurger(clients.ConfigClient, apis))
	}

	if clients.SettingsClient == nil {
		slog.WarnContext(ctx, "Skipped deletion of settings configurations as API client was unavailable.")
	} else {
		purgers = append(purgers, settings.NewDeleter(clients.SettingsClient))
	}
	if clients.AutClient == nil {
		slog.WarnContext(ctx, "Skipped deletion of automation configurations as API client was unavailable.")
	} else {
		purgers = append(purgers, automation.NewDeleter(clients.AutClient))
	}
	if clients.BucketClient == nil {
		slog.WarnContext(ctx, "Skipped deletion of Grail bucket configurations as API client was unavailable.")
	} else {
		purgers = append(purgers, bucket.NewDeleter(clients.BucketClient))
	}
	if clients.DocumentClient == nil {
		slog.WarnContext(ctx, "Skipped deletion of document configurations as API client was unavailable.")
	} else {
		purgers = append(purgers, document.NewDeleter(clients.DocumentClient))
	}
	if clients.SegmentClient == nil {
		slog.WarnContext(ctx, "Skipped deletion of segment configurations as API client was unavailable.")
	} else {
		purgers = append(purgers, segment.NewDeleter(clients.SegmentClient))
	}
	if clients.ServiceLevelObjectiveClient == nil {
		slog.WarnContext(ctx, "Skipped deletion of SLO-v2 configurations as API client was unavailable.")
	} else {
		purgers = append(purgers, slo.NewDeleter(clients.ServiceLevelObjectiveClient))
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
