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
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
)

type DeletePointer struct {
	Type     string
	ConfigId string
}

func DeleteConfigs(client client.Client, apis map[string]api.Api, entriesToDelete map[string][]DeletePointer) []error {
	errs := make([]error, 0)

	for targetApi, entries := range entriesToDelete {
		theApi, found := apis[targetApi]

		// handle settings 2.0 objects
		if !found {
			deleteErrs := deleteSettingsObject(client, entries)
			errs = append(errs, deleteErrs...)

		} else {
			deleteErrs := deleteClassicConfig(client, theApi, entries, targetApi)
			errs = append(errs, deleteErrs...)

		}
	}

	return errs
}

func deleteClassicConfig(client client.Client, theApi api.Api, entries []DeletePointer, targetApi string) []error {
	errors := make([]error, 0)

	values, err := client.List(theApi)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to fetch existing configs of api `%v`. Skipping deletion all configs of this api. Reason: %w", theApi.GetId(), err))
	}

	values, errs := filterValuesToDelete(entries, values, theApi.GetId())
	errors = append(errors, errs...)

	log.Info("Deleting configs of type %s...", theApi.GetId())

	if len(values) == 0 {
		log.Debug("No values found to delete (%s)", targetApi)
	}

	for _, v := range values {
		log.Debug("Deleting %v (%v)", v, targetApi)
		if err := client.DeleteById(theApi, v.Id); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func deleteSettingsObject(c client.Client, entries []DeletePointer) []error {
	errors := make([]error, 0)

	for _, e := range entries {
		externalID := idutils.GenerateExternalID(e.Type, e.ConfigId)
		// get settings objects with matching external ID
		objects, err := c.ListSettings(e.Type, client.ListSettingsOptions{DiscardValue: true, Filter: func(o client.DownloadSettingsObject) bool { return o.ExternalId == externalID }})
		if err.WrappedError != nil {
			errors = append(errors, fmt.Errorf("could not fetch settings 2.0 objects with schema ID %s: %w", e.Type, err))
			continue
		}

		if len(objects) == 0 {
			log.Debug("No settings object found to delete: %s/%s", e.Type, e.ConfigId)
			continue
		}

		for _, obj := range objects {
			log.Debug("Deleting settings object %s/%s with objectId %s", e.Type, e.ConfigId, obj.ObjectId)
			err := c.DeleteSettings(obj.ObjectId)
			if err != nil {
				errors = append(errors, fmt.Errorf("could not delete settings 2.0 object with object ID %s", obj.ObjectId))
			}
		}
	}

	return errors
}

// filterValuesToDelete filters the given values for only values we want to delete.
// We first search the names of the config-to-be-deleted, and if we find it, return them.
// If we don't find it, we look if the name is actually an id, and if we find it, return them.
// If a given name is found multiple times, we return an error for each name.
func filterValuesToDelete(entries []DeletePointer, existingValues []api.Value, apiName string) ([]api.Value, []error) {

	toDeleteByName := make(map[string][]api.Value, len(entries))
	valuesById := make(map[string]api.Value, len(existingValues))

	for _, v := range existingValues {
		valuesById[v.Id] = v

		for _, entry := range entries {
			if toDeleteByName[entry.ConfigId] == nil {
				toDeleteByName[entry.ConfigId] = []api.Value{}
			}

			if v.Name == entry.ConfigId {
				toDeleteByName[entry.ConfigId] = append(toDeleteByName[entry.ConfigId], v)
			}
		}
	}

	result := make([]api.Value, 0, len(entries))

	var errs []error

	for name, valuesToDelete := range toDeleteByName {

		switch len(valuesToDelete) {
		case 1:
			result = append(result, valuesToDelete[0])
			break

		case 0:
			v, found := valuesById[name]

			if found {
				result = append(result, v)
			} else {
				log.Debug("No config found with the name or ID '%v' (%v)", name, apiName)
			}

		default:
			// multiple configs with this name found -> error
			errs = append(errs, fmt.Errorf("multiple configs found with the name '%v' (%v). Configs: %v", name, apiName, valuesToDelete))
		}
	}

	return result, errs
}

func DeleteAllConfigs(client client.ConfigClient, apis map[string]api.Api) (errors []error) {

	for _, api := range apis {
		log.Info("Collecting configs of type %s...", api.GetId())
		values, err := client.List(api)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		log.Info("Deleting %d configs of type %s...", len(values), api.GetId())

		for _, v := range values {
			// TODO(improvement): this could be improved by filtering for default configs the same way as Download does
			err := client.DeleteById(api, v.Id)

			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
