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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/segment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/slo"
)

type Deleter interface {
	Delete(context.Context, []pointer.DeletePointer) error
}

type configurationType = string

// DeleteEntries is a map of configuration type to slice of delete pointers
type DeleteEntries = map[configurationType][]pointer.DeletePointer

// Configs removes all given entriesToDelete from the Dynatrace environment the given client connects to
func Configs(ctx context.Context, clients client.ClientSet, entriesToDelete DeleteEntries) error {
	deleters := clientSetToDeleters(clients)

	remainingEntriesToDelete, errCount := deleteAutomationConfigs(ctx, deleters, entriesToDelete)

	removeNonDeletableClassicAPIs(ctx, remainingEntriesToDelete)

	// Delete rest of config types
	for t, entries := range remainingEntriesToDelete {
		if err := deleters.Delete(ctx, t, entries); err != nil {
			log.With(log.ErrorAttr(err)).ErrorContext(ctx, "Error during deletion: %v", err)
			errCount += 1
		}
	}

	if errCount > 0 {
		return fmt.Errorf("encountered %d errors", errCount)
	}
	return nil
}

func removeNonDeletableClassicAPIs(ctx context.Context, remainingEntriesToDelete DeleteEntries) {
	for _, classicApi := range api.NewAPIs() {
		if !classicApi.NonDeletable {
			continue
		}

		if _, found := remainingEntriesToDelete[classicApi.ID]; !found {
			continue
		}

		if classicApi.Parent == nil {
			log.WarnContext(ctx, "Classic config of type %s cannot be deleted.", classicApi.ID)
		} else {
			log.WarnContext(ctx, "Classic config of type %s cannot be deleted. Note, that they can be removed by deleting the associated '%s' type.", classicApi.ID, classicApi.Parent.ID)
		}
		delete(remainingEntriesToDelete, classicApi.ID)
	}
}

func deleteAutomationConfigs(ctx context.Context, deleters Deleters, allEntries DeleteEntries) (DeleteEntries, int) {
	remainingDeleteEntries := maps.Clone(allEntries)
	errCount := 0
	automationTypeOrder := []config.AutomationResource{config.Workflow, config.SchedulingRule, config.BusinessCalendar}
	for _, key := range automationTypeOrder {
		entries := allEntries[string(key)]
		delete(remainingDeleteEntries, string(key))
		if len(entries) == 0 {
			continue
		}

		err := deleters.Delete(ctx, string(key), entries)
		if err != nil {
			log.With(log.ErrorAttr(err)).ErrorContext(ctx, "Error during deletion: %v", err)
			errCount += 1
		}
	}
	return remainingDeleteEntries, errCount
}

type Deleters struct {
	deleterForType     map[string]Deleter
	unknownTypeDeleter Deleter
}

func (d Deleters) Delete(ctx context.Context, configType string, entries []pointer.DeletePointer) error {
	chosenDeleter, ok := d.deleterForType[configType]
	if !ok {
		chosenDeleter = d.unknownTypeDeleter
	}

	if chosenDeleter == nil {
		log.With(log.TypeAttr(configType)).WarnContext(ctx, "Skipped deletion of %d %s configuration(s) as API client was unavailable.", len(entries), configType)
		return nil
	}

	return chosenDeleter.Delete(ctx, entries)
}

func clientSetToDeleters(clients client.ClientSet) Deleters {
	deleterForConfigType := make(map[string]Deleter)
	var classicDeleter Deleter = nil
	if clients.ConfigClient != nil {
		classicDeleter = classic.NewDeleter(clients.ConfigClient)
	}
	for configType := range api.NewAPIs() {
		deleterForConfigType[configType] = classicDeleter
	}

	var automationDeleter Deleter = nil
	if clients.AutClient != nil {
		automationDeleter = automation.NewDeleter(clients.AutClient)
	}
	for _, configType := range []config.AutomationResource{config.Workflow, config.SchedulingRule, config.BusinessCalendar} {
		deleterForConfigType[string(configType)] = automationDeleter
	}

	deleterForConfigType[string(config.BucketTypeID)] = nil
	if clients.BucketClient != nil {
		deleterForConfigType[string(config.BucketTypeID)] = bucket.NewDeleter(clients.BucketClient)
	}

	deleterForConfigType[string(config.DocumentTypeID)] = nil
	if clients.DocumentClient != nil {
		deleterForConfigType[string(config.DocumentTypeID)] = document.NewDeleter(clients.DocumentClient)
	}

	deleterForConfigType[string(config.SegmentID)] = nil
	if clients.SegmentClient != nil {
		deleterForConfigType[string(config.SegmentID)] = segment.NewDeleter(clients.SegmentClient)
	}

	deleterForConfigType[string(config.ServiceLevelObjectiveID)] = nil
	if clients.ServiceLevelObjectiveClient != nil {
		deleterForConfigType[string(config.ServiceLevelObjectiveID)] = slo.NewDeleter(clients.ServiceLevelObjectiveClient)
	}

	var unknownTypeDeleter Deleter = nil
	if clients.SettingsClient != nil {
		unknownTypeDeleter = settings.NewDeleter(clients.SettingsClient)
	}

	return Deleters{
		deleterForType:     deleterForConfigType,
		unknownTypeDeleter: unknownTypeDeleter,
	}
}
