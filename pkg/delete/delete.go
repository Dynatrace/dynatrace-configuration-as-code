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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/internal/setting"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"reflect"
)

type ClientSet struct {
	Classic    dtclient.Client
	Settings   dtclient.Client
	Automation automation.Client
	Buckets    bucket.Client
}

type configurationType = string

// DeleteEntries is a map of configuration type to slice of delete pointers
type DeleteEntries = map[configurationType][]pointer.DeletePointer

// Configs removes all given entriesToDelete from the Dynatrace environment the given client connects to
func Configs(ctx context.Context, clients ClientSet, apis api.APIs, automationResources map[string]config.AutomationResource, entriesToDelete DeleteEntries) error {
	deleteErrors := 0
	for entryType, entries := range entriesToDelete {
		var err error
		if targetApi, isClassicAPI := apis[entryType]; isClassicAPI {
			err = classic.Delete(ctx, clients.Classic, targetApi, entries, entryType)
		} else if targetAutomation, isAutomationAPI := automationResources[entryType]; isAutomationAPI {
			if reflect.ValueOf(clients.Automation).IsNil() {
				log.WithCtxFields(ctx).WithFields(field.Type(entryType)).Warn("Skipped deletion of %d Automation configuration(s) of type %q as API client was unavailable.", len(entries), entryType)
				continue
			}
			err = automation.Delete(ctx, clients.Automation, targetAutomation, entries)
		} else if entryType == "bucket" {
			if reflect.ValueOf(clients.Buckets).IsNil() {
				log.WithCtxFields(ctx).WithFields(field.Type(entryType)).Warn("Skipped deletion of %d Grail Bucket configuration(s) as API client was unavailable.", len(entries))
				continue
			}
			err = bucket.Delete(ctx, clients.Buckets, entries)
		} else { // assume it's a Settings Schema
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

// AllConfigs collects and deletes classic API configuration objects using the provided ConfigClient.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - client (dtclient.ConfigClient): An implementation of the ConfigClient interface for managing configuration objects.
//   - apis (api.APIs): A list of APIs for which configuration values need to be collected and deleted.
//
// Returns:
//   - []error: A slice of errors encountered during the collection and deletion of configuration values.
func AllConfigs(ctx context.Context, client dtclient.ConfigClient, apis api.APIs) []error {
	if err := classic.DeleteAll(ctx, client, apis); err != nil {
		return []error{err}
	}

	return nil
}

// AllSettingsObjects collects and deletes settings objects using the provided SettingsClient.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - c (dtclient.SettingsClient): An implementation of the SettingsClient interface for managing settings objects.
//
// Returns:
//   - []error: A slice of errors encountered during the collection and deletion of settings objects.
func AllSettingsObjects(ctx context.Context, c dtclient.SettingsClient) []error {
	if err := setting.DeleteAll(ctx, c); err != nil {
		return []error{err}
	}
	return nil
}

// AllAutomations collects and deletes automations resources using the given automation client.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - c (automationClient): An implementation of the automationClient interface for performing automation-related operations.
//
// Returns:
//   - []error: A slice of errors encountered during the collection and deletion of automations.
func AllAutomations(ctx context.Context, c automation.Client) []error {
	if err := automation.DeleteAll(ctx, c); err != nil {
		return []error{err}
	}
	return nil
}

// AllBuckets collects and deletes objects of type "bucket" using the provided bucketClient.
//
// Parameters:
//   - ctx (context.Context): The context for the operation.
//   - c (bucketClient): The bucketClient used for listing and deleting objects.
//
// Returns:
//   - []error: A slice of errors encountered during the operation. It may contain listing errors,
//     deletion errors, or API errors.
func AllBuckets(ctx context.Context, c bucket.Client) []error {
	if err := bucket.DeleteAll(ctx, c); err != nil {
		return []error{err}
	}
	return nil
}
