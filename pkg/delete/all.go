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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/segment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/slo"
)

// All collects and deletes ALL configuration objects using the provided ClientSet.
// To delete specific configurations use Configs instead!
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - clients (ClientSet): A set of API clients used to collect and delete configurations from an environment.
func All(ctx context.Context, clients client.ClientSet, apis api.APIs) error {
	errCount := 0

	if clients.ConfigClient == nil {
		log.WarnContext(ctx, "Skipped deletion of classic configurations as API client was unavailable.")
	} else if err := classic.DeleteAll(ctx, clients.ConfigClient, apis); err != nil {
		log.ErrorContext(ctx, "Failed to delete all classic API configurations: %v", err)
		errCount++
	}

	if clients.SettingsClient == nil {
		log.WarnContext(ctx, "Skipped deletion of settings configurations as API client was unavailable.")
	} else if err := settings.DeleteAll(ctx, clients.SettingsClient); err != nil {
		log.ErrorContext(ctx, "Failed to delete all Settings 2.0 objects: %v", err)
		errCount++
	}

	if clients.AutClient == nil {
		log.WarnContext(ctx, "Skipped deletion of Automation configurations as API client was unavailable.")
	} else if err := automation.DeleteAll(ctx, clients.AutClient); err != nil {
		log.ErrorContext(ctx, "Failed to delete all Automation configurations: %v", err)
		errCount++
	}

	if clients.BucketClient == nil {
		log.WarnContext(ctx, "Skipped deletion of Grail Bucket configurations as API client was unavailable.")
	} else if err := bucket.NewDeleter(clients.BucketClient).DeleteAll(ctx); err != nil {
		log.ErrorContext(ctx, "Failed to delete all Grail Bucket configurations: %v", err)
		errCount++
	}

	if clients.DocumentClient == nil {
		log.WarnContext(ctx, "Skipped deletion of Documents configurations as appropriate client was unavailable.")
	} else if err := document.NewDeleter(clients.DocumentClient).DeleteAll(ctx); err != nil {
		log.ErrorContext(ctx, "Failed to delete all Document configurations: %v", err)
		errCount++
	}

	if clients.SegmentClient == nil {
		log.WarnContext(ctx, "Skipped deletion of %s configurations as appropriate client was unavailable.", config.SegmentID)
	} else if err := segment.NewDeleter(clients.SegmentClient).DeleteAll(ctx); err != nil {
		log.ErrorContext(ctx, "Failed to delete all %s configurations: %v", config.SegmentID, err)
		errCount++
	}

	if clients.ServiceLevelObjectiveClient == nil {
		log.WarnContext(ctx, "Skipped deletion of %s configurations as appropriate client was unavailable.", config.SegmentID)
	} else if err := slo.NewDeleter(clients.ServiceLevelObjectiveClient).DeleteAll(ctx); err != nil {
		log.ErrorContext(ctx, "Failed to delete all %s configurations: %v", config.ServiceLevelObjective{}, err)
		errCount++
	}

	if errCount > 0 {
		return fmt.Errorf("failed to delete all configurations for %d types", errCount)
	}
	return nil
}
