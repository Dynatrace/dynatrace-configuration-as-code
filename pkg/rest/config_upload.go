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
	"strings"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

func upsertDynatraceObject(client *http.Client, fullUrl string, objectName string, theApi api.Api, configJson string, apiToken string) (api.DynatraceEntity, error) {

	isDashBoard, existingObjectId, err := getObjectIdIfAlreadyExists(client, theApi, fullUrl, objectName, apiToken)
	var dtEntity api.DynatraceEntity
	if err != nil {
		return dtEntity, err
	}
	var resp Response
	path := fullUrl
	body := configJson
	configType := theApi.GetId()

	// The calculated-metrics-log API doesn't have a POST endpoint, to create a new log metric we need to use PUT which
	// requires a metric key for which we can just take the objectName
	if configType == "calculated-metrics-log" && existingObjectId == "" {
		existingObjectId = objectName
	}

	if existingObjectId != "" {
		path = joinUrl(fullUrl, existingObjectId)
		// Updating a dashboard requires the ID to be contained in the JSON, so we just add it...
		if isDashBoard {
			body = strings.Replace(configJson, "{", "{\n\"id\":\""+existingObjectId+"\",\n", 1)
		}
		resp = put(client, path, body, apiToken)
	} else {
		if configType == "app-detection-rule" {
			path += "?position=PREPEND"
		}
		resp = post(client, path, body, apiToken)

		// It can happen that the post fails because config needs time to be propagated on all cluster nodes. If the error
		// constraintViolations":[{"path":"name","message":"X must have a unique name...
		// is returned, try once again
		if !success(resp) && strings.Contains(string(resp.Body), "must have a unique name") {
			// Try again after 5 seconds:
			util.Log.Warn("\t\tConfig '%s - %s' needs to have a unique name. Waiting for 5 seconds before retry...", configType, objectName)
			time.Sleep(5 * time.Second)
			resp = post(client, path, body, apiToken)
		}
		// It can take longer until request attributes are ready to be used
		if !success(resp) && strings.Contains(string(resp.Body), "must specify a known request attribute") {
			util.Log.Warn("\t\tSpecified request attribute not known for %s. Waiting for 10 seconds before retry...", objectName)
			time.Sleep(10 * time.Second)
			resp = post(client, path, body, apiToken)
		}
	}
	if !success(resp) {
		return dtEntity, fmt.Errorf("Failed to upsert DT object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body))
	}
	if updateSuccess(resp) {
		util.Log.Debug("\t\t\tUpdated existing object for %s (%s)", objectName, existingObjectId)
		return api.DynatraceEntity{
			Id:          existingObjectId,
			Name:        objectName,
			Description: "Updated existing object",
		}, nil
	}

	if configType == "synthetic-monitor" || configType == "synthetic-location" {
		var entity api.SyntheticEntity
		err := json.Unmarshal(resp.Body, &entity)
		if util.CheckError(err, "Cannot unmarshal Synthetic API response") {
			return dtEntity, err
		}
		dtEntity = translateSyntheticEntityResponse(entity, objectName)

	} else if available, locationSlice := isLocationHeaderAvailable(resp); available {

		// The SLO API does not return the ID of the config in its response. Instead, it contains a Location header,
		// which contains the URL to the created resource. This URL needs to be cleaned, to get the ID of the
		// config.

		if len(locationSlice) == 0 {
			return dtEntity,
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
			return dtEntity, err
		}
	}
	util.Log.Debug("\t\t\tCreated new object for %s (%s)", dtEntity.Name, dtEntity.Id)

	return dtEntity, nil
}

func joinUrl(urlBase string, path string) string {
	if strings.HasSuffix(urlBase, "/") {
		return urlBase + path
	}
	return urlBase + "/" + path
}

func isLocationHeaderAvailable(resp Response) (headerAvailable bool, headerArray []string) {
	if resp.Headers["Location"] != nil {
		return true, resp.Headers["Location"]
	}
	return false, make([]string, 0)
}

func deleteDynatraceObject(client *http.Client, api api.Api, name string, url string, token string) error {

	_, existingId, err := getObjectIdIfAlreadyExists(client, api, url, name, token)
	if err != nil {
		return err
	}

	if len(existingId) > 0 {
		deleteConfig(client, url, token, existingId)
	}
	return nil
}

func getObjectIdIfAlreadyExists(client *http.Client, api api.Api, url string, objectName string, apiToken string) (isDashboard bool, existingId string, err error) {

	isDashboard, values, err := getExistingValuesFromEndpoint(client, api, url, apiToken)
	if err != nil {
		return isDashboard, "", err
	}

	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.Name == objectName {
			return isDashboard, value.Id, nil
		}
	}
	return isDashboard, "", nil
}

func getExistingValuesFromEndpoint(client *http.Client, theApi api.Api, url string, apiToken string) (isDashboard bool, values []api.Value, err error) {

	values = make([]api.Value, 0)
	resp := get(client, url, apiToken)
	isDashboard = theApi.GetId() == "dashboard"

	for {
		var objmap map[string]interface{}

		err, values = unmarshalJson(theApi, err, resp, values, objmap, isDashboard)
		if err != nil {
			return isDashboard, values, err
		}

		// Does the API support paging?
		if isPaginated, nextPage := isPaginatedResponse(objmap); isPaginated {
			resp = get(client, url+"?nextPageKey="+nextPage, apiToken)
		} else {
			break
		}
	}

	return isDashboard, values, nil
}

func unmarshalJson(theApi api.Api, err error, resp Response, values []api.Value, objmap map[string]interface{}, isDashboard bool) (error, []api.Value) {

	if theApi.GetId() == "synthetic-location" {

		var jsonResp api.SyntheticLocationResponse
		err = json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
			return err, values
		}
		values = translateSyntheticValues(jsonResp.Locations)

	} else if theApi.GetId() == "synthetic-monitor" {

		var jsonResp api.SyntheticMonitorsResponse
		err = json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
			return err, values
		}
		values = translateSyntheticValues(jsonResp.Monitors)

	} else if theApi.GetId() == "aws-credentials" {

		var jsonResp []api.Value
		err := json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing aws-credentials") {
			return err, values
		}
		values = jsonResp

	} else if !theApi.IsStandardApi() {

		if err := json.Unmarshal(resp.Body, &objmap); err != nil {
			return err, values
		}

		if available, array := isResultArrayAvailable(objmap, theApi); available {
			genericValues, err := translateGenericValues(array, theApi.GetId())
			if err != nil {
				return err, values
			}
			values = append(values, genericValues...)
		}

	} else {

		var jsonResponse api.ValuesResponse
		err = json.Unmarshal(resp.Body, &jsonResponse)
		if util.CheckError(err, "Cannot unmarshal API response for existing objects") {
			return err, values
		}
		values = jsonResponse.Values
	}

	return nil, values
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

func updateSuccess(resp Response) bool {
	return resp.StatusCode == http.StatusNoContent
}
