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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"reflect"
)

// DeletePointer contains all data needed to identify an object to be deleted from a Dynatrace environment.
// DeletePointer is similar but not fully equivalent to config.Coordinate as it may contain an Identifier that is either
// a Name or a ConfigID - only in case of a ConfigID is it actually equivalent to a Coordinate
type DeletePointer struct {
	Project string
	Type    string

	//Identifier will either be the Name of a classic Config API object, or a configID for newer types like Settings
	Identifier string
}

func (d DeletePointer) asCoordinate() coordinate.Coordinate {
	return coordinate.Coordinate{
		Project:  d.Project,
		Type:     d.Type,
		ConfigId: d.Identifier,
	}
}

type ClientSet struct {
	Classic    dtclient.Client
	Settings   dtclient.Client
	Automation automationClient
}

type automationClient interface {
	Delete(resourceType automation.ResourceType, id string) (err error)
	List(ctx context.Context, resourceType automation.ResourceType) (result []automation.Response, err error)
}

// Configs removes all given entriesToDelete from the Dynatrace environment the given client connects to
func Configs(ctx context.Context, clients ClientSet, apis api.APIs, automationResources map[string]config.AutomationResource, entriesToDelete map[string][]DeletePointer) []error {
	errs := make([]error, 0)

	for entryType, entries := range entriesToDelete {
		if targetApi, isApi := apis[entryType]; isApi {
			deleteErrs := deleteClassicConfig(ctx, clients.Classic, targetApi, entries, entryType)
			errs = append(errs, deleteErrs...)
		} else if targetAutomation, isAutomation := automationResources[entryType]; isAutomation {
			if reflect.ValueOf(clients.Automation).IsNil() {
				log.WithCtxFields(ctx).WithFields(field.Type(entryType)).Warn("Skipped deletion of %d Automation configurations of type %q as API client was unavailable.", len(entries), entryType)
				continue
			}

			deleteErrs := deleteAutomations(clients.Automation, targetAutomation, entries)
			errs = append(errs, deleteErrs...)
		} else { // assume it's a Settings Schema
			deleteErrs := deleteSettingsObject(ctx, clients.Settings, entries)
			errs = append(errs, deleteErrs...)
		}
	}

	return errs
}

func deleteClassicConfig(ctx context.Context, client dtclient.Client, theApi api.API, entries []DeletePointer, targetApi string) []error {
	errors := make([]error, 0)

	values, err := client.ListConfigs(ctx, theApi)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to fetch existing configs of api `%v`. Skipping deletion all configs of this api. Reason: %w", theApi.ID, err))
	}

	values, errs := filterValuesToDelete(ctx, entries, values, theApi.ID)
	errors = append(errors, errs...)

	log.WithCtxFields(ctx).WithFields(field.Type(theApi.ID)).Info("Deleting configs of type %s...", theApi.ID)

	if len(values) == 0 {
		log.WithCtxFields(ctx).WithFields(field.Type(theApi.ID)).Debug("No values found to delete (%s).", targetApi)
	}

	for _, v := range values {
		log.WithCtxFields(ctx).WithFields(field.Type(theApi.ID), field.F("value", v)).Debug("Deleting %v (%v).", v, targetApi)
		if err := client.DeleteConfigById(theApi, v.Id); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func deleteSettingsObject(ctx context.Context, c dtclient.Client, entries []DeletePointer) []error {
	errors := make([]error, 0)

	for _, e := range entries {

		ctx = context.WithValue(ctx, log.CtxKeyCoord{}, e.asCoordinate())

		if e.Project == "" {
			log.WithCtxFields(ctx).Warn("Generating legacy externalID for deletion of %q - this will fail to identify a newer Settings object. Consider defining a 'project' for this delete entry.", e)
		}
		externalID, err := idutils.GenerateExternalID(e.asCoordinate())

		if err != nil {
			errors = append(errors, fmt.Errorf("unable to generate external id: %w", err))
			continue
		}
		// get settings objects with matching external ID
		objects, err := c.ListSettings(ctx, e.Type, dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId == externalID }})
		if err != nil {
			errors = append(errors, fmt.Errorf("could not fetch settings 2.0 objects with schema ID %s: %w", e.Type, err))
			continue
		}

		if len(objects) == 0 {
			log.WithCtxFields(ctx).Debug("No settings object found to delete: %s/%s", e.Type, e.Identifier)
			continue
		}

		for _, obj := range objects {
			if obj.ModificationInfo != nil && !obj.ModificationInfo.Deletable {
				log.WithCtxFields(ctx).WithFields(field.F("object", obj)).Warn("Requested settings object %s/%s (%s) is not deletable.", e.Type, e.Identifier, obj.ObjectId)
				continue
			}

			log.WithCtxFields(ctx).Debug("Deleting settings object %s/%s with objectId %s.", e.Type, e.Identifier, obj.ObjectId)
			err := c.DeleteSettings(obj.ObjectId)
			if err != nil {
				errors = append(errors, fmt.Errorf("could not delete settings 2.0 object with object ID %s", obj.ObjectId))
			}
		}
	}

	return errors
}

func deleteAutomations(c automationClient, automationResource config.AutomationResource, entries []DeletePointer) []error {
	errors := make([]error, 0)

	for _, e := range entries {

		id := idutils.GenerateUUIDFromCoordinate(e.asCoordinate())

		resourceType, err := automationutils.ClientResourceTypeFromConfigType(automationResource)
		if err != nil {
			errors = append(errors, fmt.Errorf("could not delete Automation object with ID %q: %w", id, err))
		}

		err = c.Delete(resourceType, id)
		if err != nil {
			errors = append(errors, fmt.Errorf("could not delete Automation object with ID %q: %w", id, err))
		}
	}

	return errors
}

// filterValuesToDelete filters the given values for only values we want to delete.
// We first search the names of the config-to-be-deleted, and if we find it, return them.
// If we don't find it, we look if the name is actually an id, and if we find it, return them.
// If a given name is found multiple times, we return an error for each name.
func filterValuesToDelete(ctx context.Context, entries []DeletePointer, existingValues []dtclient.Value, apiName string) ([]dtclient.Value, []error) {

	toDeleteByName := make(map[string][]dtclient.Value, len(entries))
	valuesById := make(map[string]dtclient.Value, len(existingValues))

	for _, v := range existingValues {
		valuesById[v.Id] = v

		for _, entry := range entries {
			if toDeleteByName[entry.Identifier] == nil {
				toDeleteByName[entry.Identifier] = []dtclient.Value{}
			}

			if v.Name == entry.Identifier {
				toDeleteByName[entry.Identifier] = append(toDeleteByName[entry.Identifier], v)
			}
		}
	}

	result := make([]dtclient.Value, 0, len(entries))

	var errs []error

	for name, valuesToDelete := range toDeleteByName {

		switch len(valuesToDelete) {
		case 1:
			result = append(result, valuesToDelete[0])
		case 0:
			v, found := valuesById[name]

			if found {
				result = append(result, v)
			} else {
				log.WithCtxFields(ctx).WithFields(field.Type(apiName), field.F("expectedID", name)).Debug("No config found with the name or ID '%v' (%v)", name, apiName)
			}

		default:
			// multiple configs with this name found -> error
			errs = append(errs, fmt.Errorf("multiple configs found with the name '%v' (%v). Configs: %v", name, apiName, valuesToDelete))
		}
	}

	return result, errs
}

// AllConfigs deletes ALL classic Config API objects it can find from the Dynatrace environment the given client connects to
func AllConfigs(ctx context.Context, client dtclient.ConfigClient, apis api.APIs) (errors []error) {

	for _, a := range apis {
		log.WithCtxFields(ctx).WithFields(field.Type(a.ID)).Info("Collecting configs of type %s...", a.ID)
		values, err := client.ListConfigs(ctx, a)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		log.WithCtxFields(ctx).WithFields(field.Type(a.ID)).Info("Deleting %d configs of type %s...", len(values), a.ID)

		for _, v := range values {
			log.WithCtxFields(ctx).WithFields(field.Type(a.ID), field.F("value", v)).Debug("Deleting config %s/%s...", a.ID, v.Id)
			// TODO(improvement): this could be improved by filtering for default configs the same way as Download does
			err := client.DeleteConfigById(a, v.Id)

			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}

// AllSettingsObjects deletes all settings objects it can find from the Dynatrace environment the given client connects to
func AllSettingsObjects(ctx context.Context, c dtclient.SettingsClient) []error {
	var errs []error

	schemas, err := c.ListSchemas()
	if err != nil {
		return []error{fmt.Errorf("failed to fetch settings schemas. No settings will be deleted. Reason: %w", err)}
	}

	schemaIds := make([]string, len(schemas))
	for i := range schemas {
		schemaIds[i] = schemas[i].SchemaId
	}

	log.WithCtxFields(ctx).Debug("Deleting settings of schemas %v...", schemaIds)

	for _, s := range schemaIds {
		log.WithCtxFields(ctx).WithFields(field.Type(s)).Info("Collecting Settings of type %s...", s)
		settings, err := c.ListSettings(ctx, s, dtclient.ListSettingsOptions{DiscardValue: true})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		log.WithCtxFields(ctx).WithFields(field.Type(s)).Info("Deleting %d configs of type %s...", len(settings), s)
		for _, setting := range settings {
			if setting.ModificationInfo != nil && !setting.ModificationInfo.Deletable {
				continue
			}

			log.WithCtxFields(ctx).WithFields(field.Type(s), field.F("object", setting)).Debug("Deleting settings object with objectId %q...", setting.ObjectId)
			err := c.DeleteSettings(setting.ObjectId)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

// AllAutomations deletes all Automation objects it can find from the Dynatrace environment the given client connects to
func AllAutomations(ctx context.Context, c automationClient) []error {
	var errs []error

	resources := []config.AutomationResource{config.Workflow, config.BusinessCalendar, config.SchedulingRule}
	for _, resource := range resources {
		t, err := automationutils.ClientResourceTypeFromConfigType(resource)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		log.WithCtxFields(ctx).WithFields(field.Type(string(resource))).Info("Collecting Automations of type %s...", resource)
		objects, err := c.List(ctx, t)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		log.WithCtxFields(ctx).WithFields(field.Type(string(resource))).Info("Deleting %d Automations of type %s...", len(objects), resource)
		for _, o := range objects {
			log.WithCtxFields(ctx).WithFields(field.Type(string(resource)), field.F("object", o)).Debug("Deleting Automation object with id %q...", o.ID)
			err = c.Delete(t, o.ID)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}
