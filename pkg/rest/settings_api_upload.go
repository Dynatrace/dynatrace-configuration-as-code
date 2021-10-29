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
	"github.com/PaesslerAG/jsonpath"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"strconv"
)

type settingsApiRestClient struct {
}

func (s *settingsApiRestClient) upsertDynatraceObject(d *dynatraceClientImpl, fullUrl string, objectName string, theApi api.Api, payload []byte, validateOnly bool) (api.DynatraceEntity, error) {

	existingObjectId, err := s.getObjectIdIfAlreadyExists(d, theApi, fullUrl, objectName)
	if err != nil {
		return api.DynatraceEntity{}, err
	}

	isUpdate := existingObjectId != ""
	theApi = api.Settings20ObjectsApi

	if isUpdate {
		return s.put(d, theApi, existingObjectId, payload, objectName, validateOnly)
	} else {
		return s.post(d, theApi, payload, objectName, validateOnly)
	}
}

func (s *settingsApiRestClient) put(d *dynatraceClientImpl, theApi api.Api, existingObjectId string, body []byte, objectName string, validateOnly bool) (api.DynatraceEntity, error) {

	path := joinUrl(theApi.GetUrlFromEnvironmentUrl(d.environmentUrl), existingObjectId) + "?validateOnly=" + strconv.FormatBool(validateOnly)
	resp, err := put(d.client, path, body, d.token)
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
	}

	return api.DynatraceEntity{}, fmt.Errorf("Failed to update DT object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body))
}

func (s *settingsApiRestClient) post(d *dynatraceClientImpl, theApi api.Api, body []byte, objectName string, validateOnly bool) (api.DynatraceEntity, error) {

	// The API requires the body to be wrapped in an untyped list:
	body = []byte("[" + string(body) + "]")

	path := theApi.GetUrlFromEnvironmentUrl(d.environmentUrl) + "?validateOnly=" + strconv.FormatBool(validateOnly)
	resp, err := post(d.client, path, body, d.token)

	if err != nil {
		return api.DynatraceEntity{}, err
	}
	if !success(resp) {
		return api.DynatraceEntity{}, fmt.Errorf("Failed to create DT object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body))
	}

	return s.getDynatraceEntityFromObectApiResponse(resp, objectName)
}

func (s *settingsApiRestClient) getDynatraceEntityFromObectApiResponse(resp Response, objectName string) (api.DynatraceEntity, error) {

	var objmap []map[string]interface{}
	err := json.Unmarshal(resp.Body, &objmap)
	if err != nil {
		return api.DynatraceEntity{}, err
	}

	if len(objmap) == 0 {
		return api.DynatraceEntity{}, fmt.Errorf("response list was empty for object object %s (HTTP %d)! \nResponse was: %s", objectName, resp.StatusCode, string(resp.Body))
	}

	return api.DynatraceEntity{
		Id:          objmap[0]["objectId"].(string),
		Name:        objectName,
		Description: "Created object",
	}, nil
}

func (s *settingsApiRestClient) getObjectIdIfAlreadyExists(d *dynatraceClientImpl, api api.Api, url string, objectName string) (existingId string, err error) {

	values, err := s.getExistingValuesFromEndpoint(d, api, url)
	if err != nil {
		return "", err
	}
	return filterValuesByName(values, objectName)
}

func (s *settingsApiRestClient) getExistingValuesFromEndpoint(d *dynatraceClientImpl, theApi api.Api, url string) (values []api.Value, err error) {

	var existingValues []api.Value

	if schemaItem, ok := d.settings20Schemas[theApi.GetId()]; ok {

		// Get settings 2.0 schemas:
		resp, err := get(d.client, api.Settings20SchemaApi.GetUrlFromEnvironmentUrl(d.environmentUrl)+"/"+schemaItem.SchemaId, d.token)
		if err != nil {
			return nil, err
		}

		// Unmarshal the schema details from the API:
		var schemaDetails api.Settings20SchemaItemDetailsResponse
		err = json.Unmarshal(resp.Body, &schemaDetails)
		if util.CheckError(err, "Cannot unmarshal API response for schema "+schemaItem.SchemaId) {
			return nil, err
		}

		scopes := s.extractAllScopes(schemaDetails)

		settings20ObjectUrl := api.Settings20ObjectsApi.GetUrlFromEnvironmentUrl(d.environmentUrl) + "?schemaIds=" + schemaItem.SchemaId + "&scopes=" + scopes
		resp, err = get(d.client, settings20ObjectUrl, d.token)
		if err != nil {
			return nil, err
		}

		for {

			err, values, objmap := s.unmarshalJson(theApi, err, resp)
			if err != nil {
				return values, err
			}
			existingValues = append(existingValues, values...)

			// Does the API support paging?
			if isPaginated, nextPage := isPaginatedResponse(objmap); isPaginated {

				settings20ObjectUrl = api.Settings20ObjectsApi.GetUrlFromEnvironmentUrl(d.environmentUrl) + "?nextPageKey=" + nextPage
				resp, err = get(d.client, settings20ObjectUrl, d.token)
				if err != nil {
					return nil, err
				}
			} else {
				break
			}
		}

	} else {
		return nil, fmt.Errorf("settings schema of type %s was not available on environment %s", theApi.GetId(), d.environmentUrl)
	}
	return existingValues, nil
}

func (s *settingsApiRestClient) extractAllScopes(schemaDetails api.Settings20SchemaItemDetailsResponse) string {
	scopes := ""
	for _, scope := range schemaDetails.AllowedScopes {
		scopes += scope
		scopes += ","
	}
	scopes = scopes[:len(scopes)-1]
	return scopes
}

func (s *settingsApiRestClient) unmarshalJson(theApi api.Api, err error, resp Response) (error, []api.Value, map[string]interface{}) {

	var values []api.Value
	var objmap map[string]interface{}

	var schemaDetails api.Settings20ObjectResponse
	err = json.Unmarshal(resp.Body, &schemaDetails)
	if err != nil {
		return fmt.Errorf("unmarshalling setting object response failed: %s", err), values, objmap
	}

	for _, item := range schemaDetails.Items {

		value := api.Value{}

		// is the API unique?
		if theApi.GetPropertyNameOfGetAllResponse() == "" {
			value = api.Value{
				Id:   item.ObjectId,
				Name: "Not needed",
			}
		} else {

			name, err := jsonpath.Get(theApi.GetPropertyNameOfGetAllResponse(), item.Value)
			if err != nil {
				return fmt.Errorf("could not extract value with jsonpath %s: %s", theApi.GetPropertyNameOfGetAllResponse(), err), values, objmap
			}

			value = api.Value{
				Id:   item.ObjectId,
				Name: name.(string),
			}
		}
		values = append(values, value)
	}

	if err := json.Unmarshal(resp.Body, &objmap); err != nil {
		return err, nil, nil
	}

	return nil, values, objmap
}

func (s *settingsApiRestClient) deleteDynatraceObject(d *dynatraceClientImpl, api api.Api, name string, url string) error {

	existingId, err := s.getObjectIdIfAlreadyExists(d, api, url, name)
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
