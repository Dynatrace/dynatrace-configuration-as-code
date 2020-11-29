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

func upsertDynatraceObject(client *http.Client, apiPath string, objectName string, configType string, configJson string, apiToken string) (api.DynatraceEntity, error) {

	isDashBoard, existingObjectId, err := getObjectIdIfAlreadyExists(client, configType, apiPath, objectName, apiToken)
	var dtEntity api.DynatraceEntity
	if err != nil {
		return dtEntity, err
	}
	var resp Response
	path := apiPath
	body := configJson

	// The calculated-metrics-log API doesn't have a POST endpoint, to create a new log metric we need to use PUT which
	// requires a metric key for which we can just take the objectName
	if configType == "calculated-metrics-log" && existingObjectId == "" {
		existingObjectId = objectName
	}

	if existingObjectId != "" {
		path = apiPath + "/" + existingObjectId
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
	} else {
		err := json.Unmarshal(resp.Body, &dtEntity)
		if util.CheckError(err, "Cannot unmarshal API response") {
			return dtEntity, err
		}
	}
	util.Log.Debug("\t\t\tCreated new object for %s (%s)", dtEntity.Name, dtEntity.Id)

	return dtEntity, nil
}

func deleteDynatraceObject(client *http.Client, configType string, name string, url string, token string) error {

	_, existingId, err := getObjectIdIfAlreadyExists(client, configType, url, name, token)
	if err != nil {
		return err
	}

	if len(existingId) > 0 {
		deleteConfig(client, url, token, existingId)
	}
	return nil
}

func getObjectIdIfAlreadyExists(client *http.Client, configType string, url string, objectName string, apiToken string) (isDashboard bool, existingId string, err error) {
	isDashboard, values, err := getExistingValuesFromEndpoint(client, configType, url, apiToken)
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

func getExistingValuesFromEndpoint(client *http.Client, configType string, url string, apiToken string) (isDashboard bool, values []api.Value, err error) {

	resp := get(client, url, apiToken)

	switch configType {
	case "dashboard":
		var jsonResp api.DashboardResponse
		err = json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing dashboards") {
			return isDashboard, values, err
		}
		isDashboard = true
		values = jsonResp.Dashboards
	case "synthetic-location":
		var jsonResp api.SyntheticLocationResponse
		err = json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
			return isDashboard, values, err
		}
		values = translateSyntheticValues(jsonResp.Locations)
	case "synthetic-monitor":
		var jsonResp api.SyntheticMonitorsResponse
		err = json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
			return isDashboard, values, err
		}
		values = translateSyntheticValues(jsonResp.Monitors)
	case "extension":
		var jsonResp api.ExtensionsResponse
		err = json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing extension") {
			return isDashboard, values, err
		}
		values = jsonResp.Values
	case "aws-credentials":
		var jsonResp []api.Value
		err := json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing aws-credentials") {
			return isDashboard, values, err
		}
		values = jsonResp
	default:
		var jsonResponse api.ValuesResponse
		err = json.Unmarshal(resp.Body, &jsonResponse)
		if util.CheckError(err, "Cannot unmarshal API response for existing objects") {
			return isDashboard, values, err
		}
		values = jsonResponse.Values
	}

	return isDashboard, values, nil
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
