// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

type DeletePointer struct {
	Type     string
	ConfigId string
}

func DeleteConfigs(client rest.ConfigClient, apis map[string]api.Api, entriesToDelete map[string][]DeletePointer) (errors []error) {

	for targetApi, entries := range entriesToDelete {
		theApi, found := apis[targetApi]

		if !found {
			errors = append(errors, fmt.Errorf("unknown api `%s`", targetApi))
			continue
		}

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

func DeleteAllConfigs(client rest.ConfigClient, apis map[string]api.Api) (errors []error) {

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
