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

package rest

import (
	"encoding/json"
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"strings"
	"time"
)

type classicApiRestClient struct {
}

func (c *classicApiRestClient) upsertDynatraceObject(d *dynatraceClientImpl, fullUrl string, objectName string, theApi api.Api, payload []byte, validateOnly bool) (api.DynatraceEntity, error) {

	if validateOnly {
		return api.DynatraceEntity{}, fmt.Errorf("validation is not supported for api %s", theApi.GetId())
	}

	existingObjectId, err := c.getObjectIdIfAlreadyExists(d, theApi, fullUrl, objectName)
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
		if c.isApiDashboard(theApi) {
			tmp := strings.Replace(string(payload), "{", "{\n\"id\":\""+existingObjectId+"\",\n", 1)
			body = []byte(tmp)
		}
		resp, err = put(d.client, path, body, d.token)

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
		resp, err = post(d.client, path, body, d.token)

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
			resp, err = post(d.client, path, body, d.token)

			if err != nil {
				return api.DynatraceEntity{}, err
			}
		}
		// It can take longer until request attributes are ready to be used
		if !success(resp) && strings.Contains(string(resp.Body), "must specify a known request attribute") {
			util.Log.Warn("\t\tSpecified request attribute not known for %s. Waiting for 10 seconds before retry...", objectName)
			time.Sleep(10 * time.Second)
			resp, err = post(d.client, path, body, d.token)

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
		dtEntity = c.translateSyntheticEntityResponse(entity, objectName)

	} else if available, locationSlice := c.isLocationHeaderAvailable(resp); available {

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

func (c *classicApiRestClient) getObjectIdIfAlreadyExists(d *dynatraceClientImpl, api api.Api, url string, objectName string) (existingId string, err error) {

	values, err := c.getExistingValuesFromEndpoint(d, api, url)
	if err != nil {
		return "", err
	}
	return filterValuesByName(values, objectName)
}

func (c *classicApiRestClient) getExistingValuesFromEndpoint(d *dynatraceClientImpl, theApi api.Api, url string) (values []api.Value, err error) {

	var existingValues []api.Value

	resp, err := get(d.client, url, d.token)

	if err != nil {
		return nil, err
	}

	for {

		err, values, objmap := c.unmarshalJson(theApi, err, resp)
		if err != nil {
			return values, err
		}
		existingValues = append(existingValues, values...)

		// Does the API support paging?
		if isPaginated, nextPage := isPaginatedResponse(objmap); isPaginated {
			resp, err = get(d.client, url+"?nextPageKey="+nextPage, d.token)

			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	return existingValues, nil
}

func (c *classicApiRestClient) unmarshalJson(theApi api.Api, err error, resp Response) (error, []api.Value, map[string]interface{}) {

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
			values = c.translateSyntheticValues(jsonResp.Locations)

		} else if theApi.GetId() == "synthetic-monitor" {

			var jsonResp api.SyntheticMonitorsResponse
			err = json.Unmarshal(resp.Body, &jsonResp)
			if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
				return err, nil, nil
			}
			values = c.translateSyntheticValues(jsonResp.Monitors)

		} else if !theApi.IsStandardApi() {

			if available, array := isResultArrayAvailable(objmap, theApi); available {
				jsonResp, err := c.translateGenericValues(array, theApi.GetId())
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

func (c *classicApiRestClient) deleteDynatraceObject(d *dynatraceClientImpl, api api.Api, name string, url string) error {

	existingId, err := c.getObjectIdIfAlreadyExists(d, api, url, name)
	if err != nil {
		return err
	}

	if len(existingId) > 0 {
		err = deleteConfig(d.client, url, d.token, existingId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *classicApiRestClient) isLocationHeaderAvailable(resp Response) (headerAvailable bool, headerArray []string) {
	if resp.Headers["Location"] != nil {
		return true, resp.Headers["Location"]
	}
	return false, make([]string, 0)
}

func (c *classicApiRestClient) isApiDashboard(api api.Api) bool {
	return api.GetId() == "dashboard"
}

func (c *classicApiRestClient) translateGenericValues(inputValues []interface{}, configType string) ([]api.Value, error) {

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

func (c *classicApiRestClient) translateSyntheticValues(syntheticValues []api.SyntheticValue) []api.Value {
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

func (c *classicApiRestClient) translateSyntheticEntityResponse(resp api.SyntheticEntity, objectName string) api.DynatraceEntity {
	return api.DynatraceEntity{
		Name: objectName,
		Id:   resp.EntityId,
	}
}
