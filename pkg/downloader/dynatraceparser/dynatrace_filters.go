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

//filterConfiguration evaluates multiple filters to check if downloaded configuration is valid
func NotSupportedConfiguration(apiID string, dat map[string]interface{}) bool {
	res := isDefaultEntity(apiID, dat)
	if res == true {
		return res
	}
	//TODO: Add filter for extensions since Monaco only supports JMX
	//TODO: Add filter for extensions JMX that don't provide a name
	return false
}

//isDefaultEntity returns if the object from the dynatrace API is readonly, in which case it shouldn't be downloaded
func isDefaultEntity(apiID string, dat map[string]interface{}) bool {

	switch apiID {
	case "dashboard":
		if dat["dashboardMetadata"] != nil {
			metadata := dat["dashboardMetadata"].(map[string]interface{})
			if metadata["preset"] != nil && metadata["preset"] == true {
				return true
			}
		}
		return false
	case "synthetic-location":
		if dat["type"] == "PRIVATE" {
			return false
		}
		return true
	case "synthetic-monitor":
		return false
	case "extension":
		return false
	case "aws-credentials":
		return false
	default:
		return false
	}
}
