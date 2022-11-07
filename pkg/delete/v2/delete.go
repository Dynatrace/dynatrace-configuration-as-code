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
	"strings"
)

type DeletePointer struct {
	ApiId string
	Name  string
}

func DeleteConfigs(client rest.DynatraceClient, apis map[string]api.Api, entriesToDelete map[string][]DeletePointer) (errors []error) {

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

		names, ids, errs := findConfigsToDelete(entries, values, theApi.GetId())
		errors = append(errors, errs...)

		log.Info("Deleting configs of type %s...", theApi.GetId())

		log.Debug("\tconfig-names: %s", names)
		if errs := client.BulkDeleteByName(theApi, names); errs != nil {
			errors = append(errors, errs...)
		}

		// delete by id if we need to
		if len(ids) != 0 {
			log.Debug("\tconfig-ids: %s", names)
			for _, v := range ids {
				if err := client.DeleteById(theApi, v); err != nil {
					errors = append(errors, err)
				}

			}
		}
	}

	return errors
}

// splitConfigsToDelete splits the configs to be deleted into name-deletes, and id-deletes.
// We first search the names of the config-to-be-deleted, and if we find it, return them.
// If we don't find it, we look if the name is actually an id, and if we find it, return them.
// If a given name is found multiple times, we return an error for each name.
func findConfigsToDelete(entries []DeletePointer, values []api.Value, apiName string) ([]string, []string, []error) {

	configIdsByConfigName := make(map[string][]string, len(entries))
	for _, entry := range entries {
		configIdsByConfigName[entry.Name] = []string{}
		for _, value := range values {
			if value.Name == entry.Name {
				configIdsByConfigName[entry.Name] = append(configIdsByConfigName[entry.Name], value.Id)
			}
		}
	}

	names := make([]string, 0, len(entries))
	idsToDelete := make([]string, 0, len(entries))
	var errs []error

	for name, ids := range configIdsByConfigName {
		occurrences := len(ids)

		if occurrences == 0 {
			// no configs found -> name might be an id. If we find the id, we use it to delete by id

			found := false
			for _, value := range values {
				if value.Id == name {
					idsToDelete = append(idsToDelete, name)
					found = true
					break
				}
			}

			if !found {
				log.Debug("No config found with the name or ID '%v' (%v)", name, apiName)
			}
		} else if occurrences == 1 {
			names = append(names, name)
		} else {
			// multiple configs with this name found -> error
			errs = append(errs, fmt.Errorf("multiple configs found with the name '%v' (%v). Ids: %v", name, apiName, strings.Join(ids, ", ")))
		}
	}

	return names, idsToDelete, errs
}

func DeleteAllConfigs(client rest.DynatraceClient, apis map[string]api.Api) (errors []error) {

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
