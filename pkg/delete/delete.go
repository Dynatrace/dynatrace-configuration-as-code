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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/slo"
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
			log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d %s configuration(s) as API client was unavailable.", len(entries), config.SegmentID)
		}
	} else if t == string(config.ServiceLevelObjectiveID) {
		if featureflags.ServiceLevelObjective.Enabled() {
			if clients.ServiceLevelObjectiveClient != nil {
				return slo.Delete(ctx, clients.ServiceLevelObjectiveClient, entries)
			}
			log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d %s configuration(s) as API client was unavailable.", len(entries), config.ServiceLevelObjectiveID)
		}
	} else {
		if clients.SettingsClient != nil {
			return setting.Delete(ctx, clients.SettingsClient, entries)
		}
		log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Settings configuration(s) as API client was unavailable.", len(entries))
	}
	return nil
}
