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

		err := client.BulkDeleteByName(api, names)

		if err != nil {
			errors = append(errors, err)
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
