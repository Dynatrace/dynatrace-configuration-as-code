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

package dynatraceparser

import (
	"fmt"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
)

//GetNameForConfig gets the name of the configuration or fills the name if empty
func GetNameForConfig(name string, dat map[string]interface{}, api api.Api) (string, error) {
	//for the apis that return a name for the config
	if name != "" {
		return name, nil
	}
	if api.GetId() == "reports" {
		return dat["dashboardId"].(string), nil
	}

	return "", fmt.Errorf("error getting name for config in api %q", api.GetId())
}

//replaceKeyProperties replaces name or displayname for each config
func ReplaceKeyProperties(dat map[string]interface{}) (string, map[string]interface{}) {
	//removes id field
	configId := dat["id"].(string)
	delete(dat, "id")
	if dat["name"] != nil {
		dat["name"] = "{{.name}}"
	}
	if dat["displayName"] != nil {
		dat["displayName"] = "{{.name}}"
	}
	//for reports
	if dat["dashboardId"] != nil {
		dat["dashboardId"] = "{{.name}}"
	}
	return configId, dat
}
