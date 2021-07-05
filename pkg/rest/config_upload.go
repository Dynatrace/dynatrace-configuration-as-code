/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

func upsertDynatraceObject(client *http.Client, fullUrl string, objectName string, theApi api.Api, payload []byte, apiToken string) (api.DynatraceEntity, error) {

	existingObjectId, err := getObjectIdIfAlreadyExists(client, theApi, fullUrl, objectName, apiToken)
	if err != nil {
		return api.DynatraceEntity{}, err
	}

	var resp Response
	path := fullUrl
	body := payload
	configType := theApi.GetId()

	// The calculated-metrics-log API doesn't have a POST endpoint, to create a new log metric we need to use PUT which
	// requires a metric key for which we can just take the objectName
	if configType == "calculated-metrics-log" && existingObjectId == "" {
		existingObjectId = objectName
	}

	isUpdate := existingObjectId != ""

	if isUpdate {
		path = joinUrl(fullUrl, existingObjectId)
		// Updating a dashboard requires the ID to be contained in the JSON, so we just add it...
		if isApiDashboard(theApi) {
			tmp := strings.Replace(string(payload), "{", "{\n\"id\":\""+existingObjectId+"\",\n", 1)
			body = []byte(tmp)
		}
		resp, err = put(client, path, body, apiToken)

		if err != nil {
			return api.DynatraceEntity{}, err
		}

		if success(resp) {
			util.Log.Debug("\t\t\tUpdated existing object for %s (%s)", objectName, existingObjectId)
			return api.DynatraceEntity{
				Id:          existingObjectId,
				Name:        objectName,
				Description: "Updated existing object",
			}, nil
		} else {
			return api.DynatraceEntity{}, fmt.Errorf("Failed to update DT object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body))
		}

	} else {
		if configType == "app-detection-rule" {
			path += "?position=PREPEND"
		}
		resp, err = post(client, path, body, apiToken)

		if err != nil {
			return api.DynatraceEntity{}, err
		}

		// It can happen that the post fails because config needs time to be propagated on all cluster nodes. If the error
		// constraintViolations":[{"path":"name","message":"X must have a unique name...
		// is returned, try once again
		if !success(resp) && strings.Contains(string(resp.Body), "must have a unique name") {
			// Try again after 5 seconds:
			util.Log.Warn("\t\tConfig '%s - %s' needs to have a unique name. Waiting for 5 seconds before retry...", configType, objectName)
			time.Sleep(5 * time.Second)
			resp, err = post(client, path, body, apiToken)

			if err != nil {
				return api.DynatraceEntity{}, err
			}
		}
		// It can take longer until request attributes are ready to be used
		if !success(resp) && strings.Contains(string(resp.Body), "must specify a known request attribute") {
			util.Log.Warn("\t\tSpecified request attribute not known for %s. Waiting for 10 seconds before retry...", objectName)
			time.Sleep(10 * time.Second)
			resp, err = post(client, path, body, apiToken)

			if err != nil {
				return api.DynatraceEntity{}, err
			}
		}
		if !success(resp) {
			return api.DynatraceEntity{}, fmt.Errorf("Failed to create DT object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body))
		}
	}

	var dtEntity api.DynatraceEntity

	if configType == "synthetic-monitor" || configType == "synthetic-location" {
		var entity api.SyntheticEntity
		err := json.Unmarshal(resp.Body, &entity)
		if util.CheckError(err, "Cannot unmarshal Synthetic API response") {
			return api.DynatraceEntity{}, err
		}
		dtEntity = translateSyntheticEntityResponse(entity, objectName)

	} else if available, locationSlice := isLocationHeaderAvailable(resp); available {

		// The POST of the SLO API does not return the ID of the config in its response. Instead, it contains a
		// Location header, which contains the URL to the created resource. This URL needs to be cleaned, to get the
		// ID of the config.

		if len(locationSlice) == 0 {
			return api.DynatraceEntity{},
				fmt.Errorf("location response header was empty for API %s (name: %s)", configType, objectName)
		}

		location := locationSlice[0]

		// Some APIs prepend the environment URL. If available, trim it from the location
		existingObjectId = strings.TrimPrefix(location, fullUrl)
		existingObjectId = strings.TrimPrefix(existingObjectId, "/")

		dtEntity = api.DynatraceEntity{
			Id:          existingObjectId,
			Name:        objectName,
			Description: "Created object",
		}

	} else {
		err := json.Unmarshal(resp.Body, &dtEntity)
		if util.CheckError(err, "Cannot unmarshal API response") {
			return api.DynatraceEntity{}, err
		}
	}
	util.Log.Debug("\t\t\tCreated new object for %s (%s)", dtEntity.Name, dtEntity.Id)

	return dtEntity, nil
}

func joinUrl(urlBase string, path string) string {
	if strings.HasSuffix(urlBase, "/") {
		return urlBase + url.PathEscape(path)
	}
	return urlBase + "/" + url.PathEscape(path)
}

func isLocationHeaderAvailable(resp Response) (headerAvailable bool, headerArray []string) {
	if resp.Headers["Location"] != nil {
		return true, resp.Headers["Location"]
	}
	return false, make([]string, 0)
}

func deleteDynatraceObject(client *http.Client, api api.Api, name string, url string, token string) error {

	existingId, err := getObjectIdIfAlreadyExists(client, api, url, name, token)
	if err != nil {
		return err
	}

	if len(existingId) > 0 {
		deleteConfig(client, url, token, existingId)
	}
	return nil
}

func getObjectIdIfAlreadyExists(client *http.Client, api api.Api, url string, objectName string, apiToken string) (existingId string, err error) {

	values, err := getExistingValuesFromEndpoint(client, api, url, apiToken)
	if err != nil {
		return "", err
	}

	var configName = ""
	var configsFound = 0
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.Name == objectName {
			if configsFound == 0 {
				configName = value.Id
			}
			configsFound++

		}
	}

	if configsFound > 1 {
		util.Log.Error("\t\t\tFound %d configs with same name: %s. Please delete duplicates.", configsFound, objectName)
	}
	return configName, nil
}

func isApiDashboard(api api.Api) bool {
	return api.GetId() == "dashboard"
}

func getExistingValuesFromEndpoint(client *http.Client, theApi api.Api, url string, apiToken string) (values []api.Value, err error) {

	var existingValues []api.Value
	resp, err := get(client, url, apiToken)

	if err != nil {
		return nil, err
	}

	for {

		err, values, objmap := unmarshalJson(theApi, err, resp)
		if err != nil {
			return values, err
		}
		existingValues = append(existingValues, values...)

		// Does the API support paging?
		if isPaginated, nextPage := isPaginatedResponse(objmap); isPaginated {
			resp, err = get(client, url+"?nextPageKey="+nextPage, apiToken)

			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	return existingValues, nil
}

func unmarshalJson(theApi api.Api, err error, resp Response) (error, []api.Value, map[string]interface{}) {

	var values []api.Value
	var objmap map[string]interface{}

	// This API returns an untyped list as a response -> it needs a special handling
	if theApi.GetId() == "aws-credentials" {

		var jsonResp []api.Value
		err := json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing aws-credentials") {
			return err, values, objmap
		}
		values = jsonResp

	} else {

		if err := json.Unmarshal(resp.Body, &objmap); err != nil {
			return err, nil, nil
		}

		if theApi.GetId() == "synthetic-location" {

			var jsonResp api.SyntheticLocationResponse
			err = json.Unmarshal(resp.Body, &jsonResp)
			if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
				return err, nil, nil
			}
			values = translateSyntheticValues(jsonResp.Locations)

		} else if theApi.GetId() == "synthetic-monitor" {

			var jsonResp api.SyntheticMonitorsResponse
			err = json.Unmarshal(resp.Body, &jsonResp)
			if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
				return err, nil, nil
			}
			values = translateSyntheticValues(jsonResp.Monitors)

		} else if !theApi.IsStandardApi() {

			if available, array := isResultArrayAvailable(objmap, theApi); available {
				jsonResp, err := translateGenericValues(array, theApi.GetId())
				if err != nil {
					return err, nil, nil
				}
				values = jsonResp
			}

		} else {

			var jsonResponse api.ValuesResponse
			err = json.Unmarshal(resp.Body, &jsonResponse)
			if util.CheckError(err, "Cannot unmarshal API response for existing objects") {
				return err, nil, nil
			}
			values = jsonResponse.Values
		}
	}

	return nil, values, objmap
}

func isResultArrayAvailable(jsonResponse map[string]interface{}, theApi api.Api) (resultArrayAvailable bool, results []interface{}) {
	if jsonResponse[theApi.GetPropertyNameOfGetAllResponse()] != nil {
		return true, jsonResponse[theApi.GetPropertyNameOfGetAllResponse()].([]interface{})
	}
	return false, make([]interface{}, 0)
}

func isPaginatedResponse(jsonResponse map[string]interface{}) (paginated bool, pageKey string) {
	if jsonResponse["nextPageKey"] != nil {
		return true, jsonResponse["nextPageKey"].(string)
	}
	return false, ""
}

func translateGenericValues(inputValues []interface{}, configType string) ([]api.Value, error) {

	numValues := len(inputValues)
	values := make([]api.Value, numValues, numValues)

	for i := 0; i < numValues; i++ {
		input := inputValues[i].(map[string]interface{})

		if input["id"] == nil {
			return values, fmt.Errorf("config of type %s was invalid: No id", configType)
		}

		// repair invalid configs - but log them
		if input["name"] == nil {
			jsonStr, err := json.Marshal(input)
			if err != nil {
				util.Log.Warn("Config of type %s was invalid. Ignoring it!", configType)
				continue
			}

			util.Log.Warn("Config of type %s was invalid. Auto-corrected to use ID as name!\nInvalid config: %s", configType, string(jsonStr))

			values[i] = api.Value{
				Id:   input["id"].(string),
				Name: input["id"].(string), // use the id as name
			}
			continue
		}

		values[i] = api.Value{
			Id:   input["id"].(string),
			Name: input["name"].(string),
		}
	}
	return values, nil
}

func translateSyntheticValues(syntheticValues []api.SyntheticValue) []api.Value {
	numValues := len(syntheticValues)
	values := make([]api.Value, numValues, numValues)
	for i := 0; i < numValues; i++ {
		loc := syntheticValues[i]
		values[i] = api.Value{
			Id:   loc.EntityId,
			Name: loc.Name,
		}
	}
	return values
}

func translateSyntheticEntityResponse(resp api.SyntheticEntity, objectName string) api.DynatraceEntity {
	return api.DynatraceEntity{
		Name: objectName,
		Id:   resp.EntityId,
	}
}

func success(resp Response) bool {
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent
}
