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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/segment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/setting"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/slo"
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
		log.Warn("Skipped deletion of classic configurations as API client was unavailable.")
	} else if err := classic.DeleteAll(ctx, clients.ConfigClient, apis); err != nil {
		log.Error("Failed to delete all classic API configurations: %v", err)
		errCount++
	}

	if clients.SettingsClient == nil {
		log.Warn("Skipped deletion of settings configurations as API client was unavailable.")
	} else if err := setting.DeleteAll(ctx, clients.SettingsClient); err != nil {
		log.Error("Failed to delete all Settings 2.0 objects: %v", err)
		errCount++
	}

	if clients.AutClient == nil {
		log.Warn("Skipped deletion of Automation configurations as API client was unavailable.")
	} else if err := automation.DeleteAll(ctx, clients.AutClient); err != nil {
		log.Error("Failed to delete all Automation configurations: %v", err)
		errCount++
	}

	if clients.BucketClient == nil {
		log.Warn("Skipped deletion of Grail Bucket configurations as API client was unavailable.")
	} else if err := bucket.DeleteAll(ctx, clients.BucketClient); err != nil {
		log.Error("Failed to delete all Grail Bucket configurations: %v", err)
		errCount++
	}

	if clients.DocumentClient == nil {
		log.Warn("Skipped deletion of Documents configurations as appropriate client was unavailable.")
	} else if err := document.DeleteAll(ctx, clients.DocumentClient); err != nil {
		log.Error("Failed to delete all Document configurations: %v", err)
		errCount++
	}

	if featureflags.Segments.Enabled() {
		if clients.SegmentClient == nil {
			log.Warn("Skipped deletion of %s configurations as appropriate client was unavailable.", config.SegmentID)
		} else if err := segment.DeleteAll(ctx, clients.SegmentClient); err != nil {
			log.Error("Failed to delete all %s configurations: %v", config.SegmentID, err)
			errCount++
		}
	}

	if featureflags.ServiceLevelObjective.Enabled() {
		if clients.ServiceLevelObjectiveClient == nil {
			log.Warn("Skipped deletion of %s configurations as appropriate client was unavailable.", config.SegmentID)
		} else if err := slo.DeleteAll(ctx, clients.ServiceLevelObjectiveClient); err != nil {
			log.Error("Failed to delete all %s configurations: %v", config.ServiceLevelObjective{}, err)
			errCount++
		}
	}

	if errCount > 0 {
		return fmt.Errorf("failed to delete all configurations for %d types", errCount)
	}
	return nil
}
