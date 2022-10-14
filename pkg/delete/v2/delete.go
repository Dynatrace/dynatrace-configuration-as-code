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
	ApiId string
	Name  string
}

func DeleteConfigs(client rest.DynatraceClient, apis map[string]api.Api,
	entriesToDelete map[string][]DeletePointer) (errors []error) {

	for targetApi, entries := range entriesToDelete {
		api, found := apis[targetApi]

		if !found {
			errors = append(errors, fmt.Errorf("invalid api `%s`", targetApi))
			continue
		}

		names := toNames(entries)

		if api.IsNonUniqueNameApi() {
			log.Warn("Detected non-unique naming API (%s) - can not safely delete configurations [%s]. "+
				"Please delete the correct configuration from your environment manually and remove from delete.yaml",
				targetApi, names)
			continue
		}

		log.Info("Deleting configs of type %s...", api.GetId())
		log.Debug("\tconfigs: %s", names)
		err := client.BulkDeleteByName(api, names)

		if err != nil {
			errors = append(errors, err...)
		}
	}

	return errors
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

func toNames(pointers []DeletePointer) []string {
	result := make([]string, len(pointers))

	for i, p := range pointers {
		result[i] = p.Name
	}

	return result
}
