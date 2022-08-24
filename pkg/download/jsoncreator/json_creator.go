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
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
)

//go:generate mockgen -source=json_creator.go -destination=json_creator_mock.go -package=jsoncreator JSONCreator

// JSONCreator interface allows to mock the methods for unit testing
type JSONCreator interface {
	CreateJSONConfig(
		fs afero.Fs,
		client rest.DynatraceClient,
		api api.Api,
		entityId string,
		path string,
	) (filter bool, err error)
}

// JSONCreatorImp object
type JsonCreatorImp struct{}

// NewJSONCreator creates a new instance of the jsonCreator
func NewJSONCreator() *JsonCreatorImp {
	result := JsonCreatorImp{}
	return &result
}

// CreateJSONConfig creates a json file using the specified path and API data
func (d *JsonCreatorImp) CreateJSONConfig(fs afero.Fs, client rest.DynatraceClient, api api.Api, entityId string,
	jsonFilePath string) (filter bool, err error) {
	data, filter, err := getDetailFromAPI(client, api, entityId)
	if err != nil {
		log.Error("error getting detail %s from API", api.GetId())
		return false, err
	}

	if filter {
		return true, nil
	}

	jsonfile, err := processJSONFile(data, entityId)
	if err != nil {
		log.Error("error processing jsonfile %s", api.GetId())
		return false, err
	}

	err = afero.WriteFile(fs, jsonFilePath, jsonfile, 0664)
	if err != nil {
		log.Error("error writing detail %s", api.GetId())
		return false, err
	}

	return false, nil
}

func getDetailFromAPI(client rest.DynatraceClient, api api.Api, entityId string) (dat map[string]interface{}, filter bool, err error) {

	resp, err := client.ReadById(api, entityId)
	if err != nil {
		log.Error("error getting detail for API %s for entity %v", api.GetId(), escapedEntityId)
		return nil, false, err
	}

	err = json.Unmarshal(resp, &dat)
	if err != nil {
		log.Error("error transforming %s from json to object", escapedEntityId)
		return nil, false, err
	}

	filter = isDefaultEntity(api.GetId(), dat)
	if filter {
		log.Debug("Non-user-created default Object has been filtered out", escapedEntityId)
		return nil, true, err
	}

	return dat, false, nil
}

// processJSONFile removes and replaces properties for each json config to make them compatible with monaco standard
func processJSONFile(data map[string]interface{}, id string) ([]byte, error) {
	data = replaceKeyProperties(data)

	jsonfile, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Error("error creating json file  %s", id)
		return nil, err
	}

	return jsonfile, nil
}

// replaceKeyProperties replaces name or displayname for each config
func replaceKeyProperties(dat map[string]interface{}) map[string]interface{} {

	dat = removeKey(dat, []string{"metadata"})
	dat = removeKey(dat, []string{"id"})
	dat = removeKey(dat, []string{"identifier"})
	dat = removeKey(dat, []string{"rules", "id"})
	dat = removeKey(dat, []string{"rules", "methodRules", "id"})
	dat = removeKey(dat, []string{"entityId"})

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

// removes key with specified path
func removeKey(dat map[string]interface{}, key []string) map[string]interface{} {
	if len(key) == 0 || dat == nil {
		//noting todo
		return dat
	}
	if len(key) == 1 {
		delete(dat, key[0])
		return dat
	}
	if dat[key[0]] == nil {
		// no field: nothing to do
		return dat
	}
	if field, ok := dat[key[0]].(map[string]interface{}); ok {
		dat[key[0]] = removeKey(field, key[1:])
		return dat
	}

	if arrayOfFields, ok := dat[key[0]].([]interface{}); ok {
		for i := range arrayOfFields {
			if field, ok := arrayOfFields[i].(map[string]interface{}); ok {
				arrayOfFields[i] = removeKey(field, key[1:])
			}
		}
		dat[key[0]] = arrayOfFields
	}
	return dat
}

// isDefaultEntity returns if the object from the dynatrace API is readonly, in which case it shouldn't be downloaded
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
		if id, ok := dat["id"].(string); ok && strings.HasPrefix(id, "custom.") {
			return false
		}
		return true
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
