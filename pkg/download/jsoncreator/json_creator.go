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

package jsoncreator

import (
	"encoding/json"
	"net/url"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
)

//go:generate mockgen -source=json_creator.go -destination=json_creator_mock.go -package=jsoncreator JSONCreator

//JSONCreator interface allows to mock the methods for unit testing
type JSONCreator interface {
	CreateJSONConfig(fs afero.Fs, client rest.DynatraceClient, api api.Api, value api.Value,
		path string) (filter bool, err error)
}

//JSONCreatorImp object
type JsonCreatorImp struct{}

//NewJSONCreator creates a new instance of the jsonCreator
func NewJSONCreator() *JsonCreatorImp {
	result := JsonCreatorImp{}
	return &result
}

//CreateJSONConfig creates a json file using the specified path and API data
func (d *JsonCreatorImp) CreateJSONConfig(fs afero.Fs, client rest.DynatraceClient, api api.Api, value api.Value,
	jsonFilePath string) (filter bool, err error) {
	data, filter, err := getDetailFromAPI(client, api, value.Id)
	if err != nil {
		util.Log.Error("error getting detail %s from API", api.GetId())
		return false, err
	}

	if filter {
		return true, nil
	}

	jsonfile, err := processJSONFile(data, value.Id, value.Name, api)
	if err != nil {
		util.Log.Error("error processing jsonfile %s", api.GetId())
		return false, err
	}

	err = afero.WriteFile(fs, jsonFilePath, jsonfile, 0664)
	if err != nil {
		util.Log.Error("error writing detail %s", api.GetId())
		return false, err
	}

	return false, nil
}

func getDetailFromAPI(client rest.DynatraceClient, api api.Api, name string) (dat map[string]interface{}, filter bool, err error) {

	name = url.QueryEscape(name)
	resp, err := client.ReadById(api, name)
	if err != nil {
		util.Log.Error("error getting detail for API %s", api.GetId(), name)
		return nil, false, err
	}
	err = json.Unmarshal(resp, &dat)
	if err != nil {
		util.Log.Error("error transforming %s from json to object", name)
		return nil, false, err
	}
	filter = isDefaultEntity(api.GetId(), dat)
	if filter {
		util.Log.Debug("Non-user-created default Object has been filtered out", name)
		return nil, true, err
	}
	return dat, false, nil
}

//processJSONFile removes and replaces properties for each json config to make them compatible with monaco standard
func processJSONFile(data map[string]interface{}, id string, name string, api api.Api) ([]byte, error) {
	data = replaceKeyProperties(data)

	jsonfile, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		util.Log.Error("error creating json file  %s", id)
		return nil, err
	}

	return jsonfile, nil
}

//replaceKeyProperties replaces name or displayname for each config
func replaceKeyProperties(dat map[string]interface{}) map[string]interface{} {
	//removes id field
	delete(dat, "id")

	// Removes metadata field
	delete(dat, "metadata")

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
	return dat
}

//isDefaultEntity returns if the object from the dynatrace API is readonly, in which case it shouldn't be downloaded
func isDefaultEntity(apiID string, dat map[string]interface{}) bool {

	switch apiID {
	case "dashboard", "dashboard-v2":
		if dat["dashboardMetadata"] != nil {
			metadata := dat["dashboardMetadata"].(map[string]interface{})

			// dashboards can be flagged as "preset" which makes them public in a specific environment.
			// Only dashboards that are flaged "preset" and are owned by "Dynatrace" are default and can be skipped.
			isPreset := metadata["preset"] != nil && metadata["preset"] == true
			isOwnerDynatrace := metadata["owner"] != nil && metadata["owner"] == "Dynatrace"

			if isPreset && isOwnerDynatrace {
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
	case "hosts-auto-update":
		_, ok := dat["updateWindows"]
		if !ok {
			return false
		}

		definedWindows, ok := dat["updateWindows"].(map[string]interface{})["windows"].([]interface{})
		if !ok {
			return false
		}

		numberDefinedWindows := len(definedWindows)
		if numberDefinedWindows < 1 {
			return true
		}

		return false
	default:
		return false
	}
}
