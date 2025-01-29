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

package delete

import (
	"context"
	"fmt"
	"maps"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/segment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/setting"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type configurationType = string

// DeleteEntries is a map of configuration type to slice of delete pointers
type DeleteEntries = map[configurationType][]pointer.DeletePointer

// Configs removes all given entriesToDelete from the Dynatrace environment the given client connects to
func Configs(ctx context.Context, clients client.ClientSet, entriesToDelete DeleteEntries) error {
	remainingEntriesToDelete, errCount := deleteAutomationConfigs(ctx, clients.AutClient, entriesToDelete)

	//  Dashboard share settings cannot be deleted
	if _, ok := remainingEntriesToDelete[api.DashboardShareSettings]; ok {
		log.Warn("Classic config of type %s cannot be deleted. Note, that they can be removed by deleting the associated dashboard.", api.DashboardShareSettings)
		delete(remainingEntriesToDelete, api.DashboardShareSettings)
	}

	// Delete rest of config types
	for t, entries := range remainingEntriesToDelete {
		if err := deleteConfig(ctx, clients, t, entries); err != nil {
			log.WithFields(field.Error(err)).Error("Error during deletion: %v", err)
			errCount += 1
		}
	}

	if errCount > 0 {
		return fmt.Errorf("encountered %d errors", errCount)
	}
	return nil
}

func deleteAutomationConfigs(ctx context.Context, autClient client.AutomationClient, allEntries DeleteEntries) (DeleteEntries, int) {
	remainingDeleteEntries := maps.Clone(allEntries)
	errCount := 0
	automationTypeOrder := []config.AutomationResource{config.Workflow, config.SchedulingRule, config.BusinessCalendar}
	for _, key := range automationTypeOrder {
		entries := allEntries[string(key)]
		delete(remainingDeleteEntries, string(key))
		if autClient == nil {
			log.WithCtxFields(ctx).WithFields(field.Type(key)).Warn("Skipped deletion of %d Automation configuration(s) of type %q as API client was unavailable.", len(entries), key)
			continue
		}
		err := automation.Delete(ctx, autClient, key, entries)
		if err != nil {
			log.WithFields(field.Error(err)).Error("Error during deletion: %v", err)
			errCount += 1
		}
	}
	return remainingDeleteEntries, errCount
}

func deleteConfig(ctx context.Context, clients client.ClientSet, t string, entries []pointer.DeletePointer) error {
	if _, ok := api.NewAPIs()[t]; ok {
		if clients.ConfigClient != nil {
			return classic.Delete(ctx, clients.ConfigClient, entries)
		}
		log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Classic configuration(s) as API client was unavailable.", len(entries))
	} else if t == "bucket" {
		if clients.BucketClient != nil {
			return bucket.Delete(ctx, clients.BucketClient, entries)
		}
		log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Grail Bucket configuration(s) as API client was unavailable.", len(entries))
	} else if t == "document" {
		if clients.DocumentClient != nil {
			return document.Delete(ctx, clients.DocumentClient, entries)
		}
		log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Document configuration(s) as API client was unavailable.", len(entries))
	} else if t == string(config.SegmentID) {
		if featureflags.Segments.Enabled() {
			if clients.SegmentClient != nil {
				return segment.Delete(ctx, clients.SegmentClient, entries)
			}
			log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Segment configuration(s) as API client was unavailable.", len(entries))
		}
	} else {
		if clients.SettingsClient != nil {
			return setting.Delete(ctx, clients.SettingsClient, entries)
		}
		log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Settings configuration(s) as API client was unavailable.", len(entries))
	}
	return nil
}

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

	if errCount > 0 {
		return fmt.Errorf("failed to delete all configurations for %d types", errCount)
	}
	return nil
}
