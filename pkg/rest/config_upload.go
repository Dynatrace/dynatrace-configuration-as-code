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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"net/http"
	"net/url"
	"strings"
)

// internalRestClient is the internal rest client
// As of now there are two implementations: one for classic APIs (mainly configuration APIs) and one for +
// settings 2.0 APIs
type internalRestClient interface {
	getExistingValuesFromEndpoint(d *dynatraceClientImpl, theApi api.Api, url string) (values []api.Value, err error)
	upsertDynatraceObject(d *dynatraceClientImpl, fullUrl string, objectName string, theApi api.Api, payload []byte, validateOnly bool) (api.DynatraceEntity, error)
	getObjectIdIfAlreadyExists(d *dynatraceClientImpl, api api.Api, url string, objectName string) (existingId string, err error)
	unmarshalJson(theApi api.Api, err error, resp Response) (error, []api.Value, map[string]interface{})
	deleteDynatraceObject(d *dynatraceClientImpl, api api.Api, name string, url string) error
}

func newConfigUploader(theApi api.Api) internalRestClient {

	if theApi.IsSettings20Api() {
		return &settingsApiRestClient{}
	}
	return &classicApiRestClient{}
}

func joinUrl(urlBase string, path string) string {
	if strings.HasSuffix(urlBase, "/") {
		return urlBase + url.PathEscape(path)
	}
	return urlBase + "/" + url.PathEscape(path)
}

func filterValuesByName(values []api.Value, objectName string) (existingId string, err error) {

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

func success(resp Response) bool {
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent
}
