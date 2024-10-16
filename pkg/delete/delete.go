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
	coreAutomation "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"

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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/setting"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
)

type ClientSet struct {
	Classic    client.ConfigClient
	Settings   client.SettingsClient
	Automation client.AutomationClient
	Buckets    client.BucketClient
	Documents  client.DocumentClient
}

type configurationType = string

// DeleteEntries is a map of configuration type to slice of delete pointers
type DeleteEntries = map[configurationType][]pointer.DeletePointer

// Configs removes all given entriesToDelete from the Dynatrace environment the given client connects to
func Configs(ctx context.Context, clients ClientSet, _ api.APIs, automationResources map[string]config.AutomationResource, entriesToDelete DeleteEntries) error {
	copiedDeleteEntries := make(DeleteEntries)
	for k, v := range entriesToDelete {
		copiedDeleteEntries[k] = v
	}

	var deleteErrors int

	// Delete automation resources (in the specified order)
	automationTypeOrder := []config.AutomationResource{config.Workflow, config.SchedulingRule, config.BusinessCalendar}
	for _, key := range automationTypeOrder {
		entries := copiedDeleteEntries[string(key)]
		if clients.Automation == nil {
			log.WithCtxFields(ctx).WithFields(field.Type(key)).Warn("Skipped deletion of %d Automation configuration(s) of type %q as API client was unavailable.", len(entries), key)
			delete(copiedDeleteEntries, string(key))
			continue
		}
		err := automation.Delete(ctx, clients.Automation, automationResources[string(key)], entries)
		if err != nil {
			log.WithFields(field.Error(err)).Error("Error during deletion: %v", err)
			deleteErrors += 1
		}
		delete(copiedDeleteEntries, string(key))
	}

	//  Dashboard share settings cannot be deleted
	if _, ok := copiedDeleteEntries[api.DashboardShareSettings]; ok {
		log.Warn("Classic config of type %s cannot be deleted. Note, that they can be removed by deleting the associated dashboard.", api.DashboardShareSettings)
		delete(copiedDeleteEntries, api.DashboardShareSettings)

	}

	// Delete rest of config types
	for t, entries := range copiedDeleteEntries {
		var err error
		if _, ok := api.NewAPIs()[t]; ok {
			if clients.Classic == (*dtclient.DynatraceClient)(nil) {
				log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Classic configuration(s) as API client was unavailable.", len(entries))
				continue
			}
			err = classic.Delete(ctx, clients.Classic, entries)
		} else if t == "bucket" {
			if clients.Buckets == (*buckets.Client)(nil) {
				log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Grail Bucket configuration(s) as API client was unavailable.", len(entries))
				continue
			}
			err = bucket.Delete(ctx, clients.Buckets, entries)
		} else if t == "document" {
			if featureflags.Temporary[featureflags.Documents].Enabled() && featureflags.Temporary[featureflags.DeleteDocuments].Enabled() {
				if clients.Documents == (*documents.Client)(nil) {
					log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Document configuration(s) as API client was unavailable.", len(entries))
					continue
				}
				err = document.Delete(ctx, clients.Documents, entries)
			}
		} else {
			if clients.Settings == (*dtclient.DynatraceClient)(nil) {
				log.WithCtxFields(ctx).WithFields(field.Type(t)).Warn("Skipped deletion of %d Settings configuration(s) as API client was unavailable.", len(entries))
				continue
			}
			err = setting.Delete(ctx, clients.Settings, entries)
		}

		if err != nil {
			log.WithFields(field.Error(err)).Error("Error during deletion: %v", err)
			deleteErrors += 1
		}
	}

	if deleteErrors > 0 {
		return fmt.Errorf("encountered %d errors", deleteErrors)
	}
	return nil
}

// All collects and deletes ALL configuration objects using the provided ClientSet.
// To delete specific configurations use Configs instead!
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - clients (ClientSet): A set of API clients used to collect and delete configurations from an environment.
func All(ctx context.Context, clients ClientSet, apis api.APIs) error {
	errs := 0

	if clients.Classic == (*dtclient.DynatraceClient)(nil) {
		log.Warn("Skipped deletion of classic configurations as API client was unavailable.")
	} else if err := classic.DeleteAll(ctx, clients.Classic, apis); err != nil {
		log.Error("Failed to delete all classic API configurations: %v", err)
		errs++
	}

	if clients.Settings == (*dtclient.DynatraceClient)(nil) {
		log.Warn("Skipped deletion of settings configurations as API client was unavailable.")
	} else if err := setting.DeleteAll(ctx, clients.Settings); err != nil {
		log.Error("Failed to delete all Settings 2.0 objects: %v", err)
		errs++
	}

	if clients.Automation == (*coreAutomation.Client)(nil) {
		log.Warn("Skipped deletion of Automation configurations as API client was unavailable.")
	} else if err := automation.DeleteAll(ctx, clients.Automation); err != nil {
		log.Error("Failed to delete all Automation configurations: %v", err)
		errs++
	}

	if clients.Buckets == (*buckets.Client)(nil) {
		log.Warn("Skipped deletion of Grail Bucket configurations as API client was unavailable.")
	} else if err := bucket.DeleteAll(ctx, clients.Buckets); err != nil {
		log.Error("Failed to delete all Grail Bucket configurations: %v", err)
		errs++
	}

	if featureflags.Temporary[featureflags.Documents].Enabled() && featureflags.Temporary[featureflags.DeleteDocuments].Enabled() {
		if clients.Documents == (*documents.Client)(nil) {
			log.Warn("Skipped deletion of Documents configurations as appropriate client was unavailable.")
		} else if err := document.DeleteAll(ctx, clients.Documents); err != nil {
			log.Error("Failed to delete all Document configurations: %v", err)
			errs++
		}
	}

	if errs > 0 {
		return fmt.Errorf("failed to delete all configurations for %d types", errs)
	}
	return nil
}
