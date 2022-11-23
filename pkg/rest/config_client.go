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
	"regexp"
	"strings"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

func upsertDynatraceObject(
	client *http.Client,
	environmentUrl string,
	objectName string,
	theApi api.Api,
	payload []byte,
	apiToken string,
) (api.DynatraceEntity, error) {
	isSingleConfigurationApi := theApi.IsSingleConfigurationApi()
	existingObjectId := ""

	fullUrl := theApi.GetUrl(environmentUrl)

	// Single configuration APIs don't have an id which allows skipping this step
	if !isSingleConfigurationApi {
		var err error
		existingObjectId, err = getObjectIdIfAlreadyExists(client, theApi, fullUrl, objectName, apiToken)
		if err != nil {
			return api.DynatraceEntity{}, err
		}
	}

	body := payload
	configType := theApi.GetId()

	// The calculated-metrics-log API doesn't have a POST endpoint, to create a new log metric we need to use PUT which
	// requires a metric key for which we can just take the objectName
	if configType == "calculated-metrics-log" && existingObjectId == "" {
		existingObjectId = objectName
	}

	isUpdate := existingObjectId != ""

	// Single configuration APIs don't have a POST, but a PUT endpoint
	// and therefore always require an update
	if isUpdate || isSingleConfigurationApi {
		return updateDynatraceObject(client, fullUrl, objectName, existingObjectId, theApi, body, apiToken)
	} else {
		return createDynatraceObject(client, fullUrl, objectName, theApi, body, apiToken)
	}
}

func upsertDynatraceEntityById(
	client *http.Client,
	environmentUrl string,
	entityId string,
	objectName string,
	theApi api.Api,
	payload []byte,
	apiToken string,
) (api.DynatraceEntity, error) {
	fullUrl := theApi.GetUrl(environmentUrl)
	body := payload

	return updateDynatraceObject(client, fullUrl, objectName, entityId, theApi, body, apiToken)
}

func createDynatraceObject(client *http.Client, fullUrl string, objectName string, theApi api.Api, payload []byte, apiToken string) (api.DynatraceEntity, error) {
	path := fullUrl
	body := payload

	configType := theApi.GetId()

	if configType == "app-detection-rule" {
		path += "?position=PREPEND"
	}

	resp, err := callWithRetryOnKnowTimingIssue(client, post, objectName, path, body, theApi, apiToken)
	if err != nil {
		return api.DynatraceEntity{}, err
	}

	if !success(resp) {
		return api.DynatraceEntity{}, fmt.Errorf("Failed to create DT object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body))
	}

	return unmarshalResponse(resp, fullUrl, configType, objectName)
}

func unmarshalResponse(resp Response, fullUrl string, configType string, objectName string) (api.DynatraceEntity, error) {
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
		objectId := strings.TrimPrefix(location, fullUrl)
		objectId = strings.TrimPrefix(objectId, "/")

		dtEntity = api.DynatraceEntity{
			Id:          objectId,
			Name:        objectName,
			Description: "Created object",
		}

	} else {
		err := json.Unmarshal(resp.Body, &dtEntity)
		if util.CheckError(err, "Cannot unmarshal API response") {
			return api.DynatraceEntity{}, err
		}
	}
	log.Debug("\tCreated new object for %s (%s)", dtEntity.Name, dtEntity.Id)

	return dtEntity, nil
}

func updateDynatraceObject(client *http.Client, fullUrl string, objectName string, existingObjectId string, theApi api.Api, payload []byte, apiToken string) (api.DynatraceEntity, error) {
	path := joinUrl(fullUrl, existingObjectId)
	body := payload

	// Updating a dashboard, reports or any service detection API requires the ID to be contained in the JSON, so we just add it...
	if isApiDashboard(theApi) || isReportsApi(theApi) || isAnyServiceDetectionApi(theApi) {
		tmp := strings.Replace(string(payload), "{", "{\n\"id\":\""+existingObjectId+"\",\n", 1)
		body = []byte(tmp)
	}

	// Updating a Mobile Application does not allow changing the applicationType as such this property required on Create, must be stripped on Update
	if isMobileApp(theApi) {
		body = stripCreateOnlyPropertiesFromAppMobile(body)
	}

	resp, err := callWithRetryOnKnowTimingIssue(client, put, objectName, path, body, theApi, apiToken)

	if err != nil {
		return api.DynatraceEntity{}, err
	}

	if !success(resp) {
		return api.DynatraceEntity{}, fmt.Errorf("Failed to update DT object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body))
	}

	if theApi.IsNonUniqueNameApi() {
		log.Debug("\tCreated/Updated object by ID for %s (%s)", objectName, existingObjectId)
	} else {
		log.Debug("\tUpdated existing object for %s (%s)", objectName, existingObjectId)
	}

	return api.DynatraceEntity{
		Id:          existingObjectId,
		Name:        objectName,
		Description: "Updated existing object",
	}, nil
}

func stripCreateOnlyPropertiesFromAppMobile(payload []byte) []byte {
	//applicationType is required on creation, but not allowed to be updated
	r := regexp.MustCompile(`"applicationType":.*?,`)
	tmp := r.ReplaceAllString(string(payload), "")
	newPayload := []byte(tmp)

	return newPayload
}

// callWithRetryOnKnowTimingIssue handles several know cases in which Dynatrace has a slight delay before newly created objects
// can be used in further configuration. This is a cheap way to allow monaco to work around this, by waiting, then
// retrying in case of know errors on upload.
func callWithRetryOnKnowTimingIssue(client *http.Client, restCall sendingRequest, objectName string, path string, body []byte, theApi api.Api, apiToken string) (Response, error) {

	resp, err := restCall(client, path, body, apiToken)

	if err == nil && success(resp) {
		return resp, nil
	}

	// It can take longer until calculated service metrics are ready to be used in SLOs
	if isCalculatedMetricNotReadyYet(resp) ||
		// It can take longer until management zones are ready to be used in SLOs
		isManagementZoneNotReadyYet(resp) ||
		// It can take longer until Credentials are ready to be used in Synthetic Monitors
		isCredentialNotReadyYet(resp) ||
		// It can take some time for configurations to propagate to all cluster nodes - indicated by an incorrect constraint violation error
		isGeneralDependencyNotReadyYet(resp) {

		return retry(client, restCall, objectName, path, body, apiToken, 3, 5*time.Second)
	}

	// It can take even longer until request attributes are ready to be used
	if isRequestAttributeNotYetReady(resp) {
		return retry(client, restCall, objectName, path, body, apiToken, 3, 10*time.Second)
	}

	// It can take even longer until applications are ready to be used in synthetic tests
	if isApplicationNotReadyYet(resp, theApi) {
		return retry(client, restCall, objectName, path, body, apiToken, 5, 15*time.Second)
	}

	return resp, nil
}

func retry(client *http.Client, restCall sendingRequest, objectName string, path string, body []byte, apiToken string, maxRetries int, timeout time.Duration) (resp Response, err error) {
	for i := 0; i < maxRetries; i++ {
		log.Warn("\t\t\tDependency of config %s was not available. Waiting for %s before retry...", objectName, timeout)
		time.Sleep(timeout)
		resp, err = restCall(client, path, body, apiToken)
		if err == nil && success(resp) {
			return resp, err
		}
	}

	var retryErr error
	if err != nil {
		retryErr = fmt.Errorf("dependency of config %s was not available after %d retries: %w", objectName, maxRetries, err)
	} else {
		retryErr = fmt.Errorf("dependency of config %s was not available after %d retries: (HTTP %d)!\n    Response was: %s", objectName, maxRetries, resp.StatusCode, resp.Body)
	}
	return Response{}, retryErr
}

func isGeneralDependencyNotReadyYet(resp Response) bool {
	return strings.Contains(string(resp.Body), "must have a unique name")
}

func isCalculatedMetricNotReadyYet(resp Response) bool {
	return strings.Contains(string(resp.Body), "Metric selector") &&
		strings.Contains(string(resp.Body), "invalid")
}

func isRequestAttributeNotYetReady(resp Response) bool {
	return strings.Contains(string(resp.Body), "must specify a known request attribute")
}

func isManagementZoneNotReadyYet(resp Response) bool {
	return strings.Contains(string(resp.Body), "Entity selector is invalid") ||
		(strings.Contains(string(resp.Body), "SLO validation failed") &&
			strings.Contains(string(resp.Body), "Management-Zone not found")) ||
		strings.Contains(string(resp.Body), "Unknown management zone")
}

func isApplicationNotReadyYet(resp Response, theApi api.Api) bool {
	return isServerError(resp) && (theApi.GetId() == "synthetic-monitor" || isAnyApplicationApi(theApi)) ||
		strings.Contains(string(resp.Body), "Unknown application(s)")
}

func isCredentialNotReadyYet(resp Response) bool {
	s := string(resp.Body)
	return strings.Contains(s, "credential-vault") &&
		strings.Contains(s, "was not available")
}

func joinUrl(urlBase string, path string) string {
	trimmedUrl := strings.TrimSuffix(urlBase, "/")

	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return trimmedUrl
	}

	return trimmedUrl + "/" + url.PathEscape(trimmedPath)
}

func isLocationHeaderAvailable(resp Response) (headerAvailable bool, headerArray []string) {
	if resp.Headers["Location"] != nil {
		return true, resp.Headers["Location"]
	}
	return false, make([]string, 0)
}

func getObjectIdIfAlreadyExists(client *http.Client, api api.Api, url string, objectName string, apiToken string) (string, error) {
	values, err := getExistingValuesFromEndpoint(client, api, url, apiToken)

	if err != nil {
		return "", err
	}

	var objectId = ""
	var matchingObjectsFound = 0
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.Name == objectName || escapeApiValueName(value) == objectName {
			if matchingObjectsFound == 0 {
				objectId = value.Id
			}
			matchingObjectsFound++
		}
	}

	if matchingObjectsFound > 1 {
		log.Warn("Found %d configs with same name: %s. Please delete duplicates.", matchingObjectsFound, objectName)
	}

	if objectId != "" {
		log.Debug("Found existing config %s (%s) with id %s", objectName, api.GetId(), objectId)
	}

	return objectId, nil
}

func escapeApiValueName(value api.Value) string {
	valueName, err := util.EscapeSpecialCharactersInValue(value.Name, util.FullStringEscapeFunction)
	if err != nil {
		log.Warn("failed to string escape API value '%s' while checking if object exists, check directly", value.Name)
		return value.Name
	}
	return valueName.(string)
}

func isApiDashboard(api api.Api) bool {
	return api.GetId() == "dashboard" || api.GetId() == "dashboard-v2"
}

func isReportsApi(api api.Api) bool {
	return api.GetId() == "reports"
}

func isAnyServiceDetectionApi(api api.Api) bool {
	return strings.HasPrefix(api.GetId(), "service-detection-")
}

func isAnyApplicationApi(api api.Api) bool {
	return strings.HasPrefix(api.GetId(), "application-")
}

func isMobileApp(api api.Api) bool {
	return api.GetId() == "application-mobile"
}

func getExistingValuesFromEndpoint(client *http.Client, theApi api.Api, urlString string, apiToken string) (values []api.Value, err error) {

	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	parsedUrl = addQueryParamsForNonStandardApis(theApi, parsedUrl)

	resp, err := get(client, parsedUrl.String(), apiToken)

	if err != nil {
		return nil, err
	}

	if !success(resp) {
		return nil, fmt.Errorf("Failed to get existing configs for API %s (HTTP %d)!\n    Response was: %s", theApi.GetId(), resp.StatusCode, string(resp.Body))
	}

	var existingValues []api.Value
	for {

		values, objmap, err := unmarshalJson(theApi, resp)
		if err != nil {
			return values, err
		}
		existingValues = append(existingValues, values...)

		// Does the API support paging?
		if isPaginated, nextPage := isPaginatedResponse(objmap); isPaginated {
			parsedUrl = addNextPageQueryParams(theApi, parsedUrl, nextPage)

			resp, err = getWithRetry(client, parsedUrl.String(), apiToken, 3, 5*time.Second)

			if err != nil {
				return nil, err
			}

			if !success(resp) {
				return nil, fmt.Errorf("Failed to get further configs from paginated API %s (HTTP %d)!\n    Response was: %s", theApi.GetId(), resp.StatusCode, string(resp.Body))
			}

		} else {
			break
		}
	}

	return existingValues, nil
}

// getWithRetry works similarly to retry does for PUT and POST
// this method can be used for API calls we know to have occasional timing issues on GET - e.g. paginated queries that are impacted by replication lag, returning unequal amounts of objects/pages per node
func getWithRetry(client *http.Client, url string, apiToken string, maxRetries int, timeout time.Duration) (Response, error) {
	resp, err := get(client, url, apiToken)

	if err == nil && success(resp) {
		return resp, nil
	}

	for i := 0; i < maxRetries; i++ {
		util.Log.Warn("Retrying failed GET request %s after error (HTTP %d): %w", url, resp.StatusCode, err)
		time.Sleep(timeout)
		resp, err = get(client, url, apiToken)
		if err == nil && success(resp) {
			return resp, err
		}
	}

	var retryErr error
	if err != nil {
		retryErr = fmt.Errorf("GET request %s failed after %d retries: %w", url, maxRetries, err)
	} else {
		retryErr = fmt.Errorf("GET request %s failed after %d retries: (HTTP %d)!\n    Response was: %s", url, maxRetries, resp.StatusCode, resp.Body)
	}
	return Response{}, retryErr
}

func addQueryParamsForNonStandardApis(theApi api.Api, url *url.URL) *url.URL {

	queryParams := url.Query()

	if theApi.GetId() == "anomaly-detection-metrics" {
		queryParams.Add("includeEntityFilterMetricEvents", "true")
	}
	if theApi.GetId() == "slo" {
		queryParams.Add("enabledSlos", "all")
	}
	url.RawQuery = queryParams.Encode()
	return url
}

func addNextPageQueryParams(theApi api.Api, url *url.URL, nextPage string) *url.URL {
	queryParams := url.Query()

	if theApi.GetId() == "slo" {
		//SLO API requires enabledSlos to not be set together with nextPageKey, that information is already encoded in the key
		queryParams.Del("enabledSlos")
	}

	queryParams.Set("nextPageKey", nextPage)
	url.RawQuery = queryParams.Encode()
	return url
}

func unmarshalJson(theApi api.Api, resp Response) ([]api.Value, map[string]interface{}, error) {

	var values []api.Value
	var objmap map[string]interface{}

	// This API returns an untyped list as a response -> it needs a special handling
	if theApi.GetId() == "aws-credentials" {

		var jsonResp []api.Value
		err := json.Unmarshal(resp.Body, &jsonResp)
		if util.CheckError(err, "Cannot unmarshal API response for existing aws-credentials") {
			return values, objmap, err
		}
		values = jsonResp

	} else {

		if err := json.Unmarshal(resp.Body, &objmap); err != nil {
			return nil, nil, err
		}

		if theApi.GetId() == "synthetic-location" {

			var jsonResp api.SyntheticLocationResponse
			err := json.Unmarshal(resp.Body, &jsonResp)
			if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
				return nil, nil, err
			}
			values = translateSyntheticValues(jsonResp.Locations)

		} else if theApi.GetId() == "synthetic-monitor" {

			var jsonResp api.SyntheticMonitorsResponse
			err := json.Unmarshal(resp.Body, &jsonResp)
			if util.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
				return nil, nil, err
			}
			values = translateSyntheticValues(jsonResp.Monitors)

		} else if !theApi.IsStandardApi() || isReportsApi(theApi) {

			if available, array := isResultArrayAvailable(objmap, theApi); available {
				jsonResp, err := translateGenericValues(array, theApi.GetId())
				if err != nil {
					return nil, nil, err
				}
				values = jsonResp
			}

		} else {

			var jsonResponse api.ValuesResponse
			err := json.Unmarshal(resp.Body, &jsonResponse)
			if util.CheckError(err, "Cannot unmarshal API response for existing objects") {
				return nil, nil, err
			}
			values = jsonResponse.Values
		}
	}

	return values, objmap, nil
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

	values := make([]api.Value, 0, len(inputValues))

	for _, input := range inputValues {
		input := input.(map[string]interface{})

		if input["id"] == nil {
			return values, fmt.Errorf("config of type %s was invalid: No id", configType)
		}

		// Substitute missing name attribute
		if input["name"] == nil {
			jsonStr, err := json.Marshal(input)
			if err != nil {
				log.Warn("Config of type %s was invalid. Ignoring it!", configType)
				continue
			}

			substitutedName := ""

			// Differentiate handling for reports API from others
			isReportsApi := configType == "reports"
			if isReportsApi {
				// Substitute name with dashboard id since it is unique identifier for entity
				substitutedName = input["dashboardId"].(string)
				log.Debug("Rewriting response of config-type '%v', name missing. Using dashboardId as name. Invalid json: %v", configType, string(jsonStr))

			} else {
				// Substitute name with id since it is unique identifier for entity
				substitutedName = input["id"].(string)
				log.Debug("Rewriting response of config-type '%v', name missing. Using id as name. Invalid json: %v", configType, string(jsonStr))
			}

			values = append(values, api.Value{
				Id:   input["id"].(string),
				Name: substitutedName,
			})
			continue
		}

		value := api.Value{
			Id:   input["id"].(string),
			Name: input["name"].(string),
		}

		if v, ok := input["owner"].(string); ok {
			value.Owner = &v
		}

		values = append(values, value)
	}
	return values, nil
}

func translateSyntheticValues(syntheticValues []api.SyntheticValue) []api.Value {
	values := make([]api.Value, 0, len(syntheticValues))
	for _, loc := range syntheticValues {
		values = append(values, api.Value{
			Id:   loc.EntityId,
			Name: loc.Name,
		})
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
	return resp.StatusCode >= 200 && resp.StatusCode <= 299
}

func isServerError(resp Response) bool {
	return resp.StatusCode >= 500 && resp.StatusCode <= 599
}
