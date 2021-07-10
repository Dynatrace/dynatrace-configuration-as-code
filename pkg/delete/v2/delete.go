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

		log.Debug("\tTrying to delete configs of type %s %s", api.GetId(), names)
		err := client.BulkDeleteByName(api, names)

		if err != nil {
			errors = append(errors, err...)
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
