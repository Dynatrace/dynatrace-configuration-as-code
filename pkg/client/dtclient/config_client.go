/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package dtclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func upsertDynatraceObject(
	ctx context.Context,
	client *http.Client,
	environmentUrl string,
	objectName string,
	theApi api.API,
	payload []byte,
	retrySettings rest.RetrySettings,
) (DynatraceEntity, error) {
	isSingleConfigurationApi := theApi.SingleConfiguration
	existingObjectId := ""

	fullUrl := theApi.CreateURL(environmentUrl)

	// Single configuration APIs don't have an id which allows skipping this step
	if !isSingleConfigurationApi {
		var err error
		existingObjectId, err = getObjectIdIfAlreadyExists(ctx, client, theApi, fullUrl, objectName, retrySettings)
		if err != nil {
			return DynatraceEntity{}, err
		}
	}

	body := payload
	configType := theApi.ID

	// The calculated-metrics-log API doesn't have a POST endpoint, to create a new log metric we need to use PUT which
	// requires a metric key for which we can just take the objectName
	if configType == "calculated-metrics-log" && existingObjectId == "" {
		existingObjectId = objectName
	}

	isUpdate := existingObjectId != ""

	// Single configuration APIs don't have a POST, but a PUT endpoint
	// and therefore always require an update
	if isUpdate || isSingleConfigurationApi {
		return updateDynatraceObject(ctx, client, fullUrl, objectName, existingObjectId, theApi, body, retrySettings)
	} else {
		return createDynatraceObject(ctx, client, fullUrl, objectName, theApi, body, retrySettings)
	}
}

func upsertDynatraceEntityByNonUniqueNameAndId(
	ctx context.Context,
	client *http.Client,
	environmentUrl string,
	entityId string,
	objectName string,
	theApi api.API,
	payload []byte,
	retrySettings rest.RetrySettings,
) (DynatraceEntity, error) {
	fullUrl := theApi.CreateURL(environmentUrl)
	body := payload

	existingEntities, err := getExistingValuesFromEndpoint(ctx, client, theApi, fullUrl, retrySettings)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to query existing entities: %w", err)
	}

	var entitiesWithSameName []Value
	var entityExists bool

	for _, e := range existingEntities {
		if e.Name == objectName {
			entitiesWithSameName = append(entitiesWithSameName, e)
			if e.Id == entityId {
				entityExists = true
			}
		}
	}

	if entityExists || len(entitiesWithSameName) == 0 { // create with fixed ID or update (if this moves to client logging can clearly state things)
		entity, err := updateDynatraceObject(ctx, client, fullUrl, objectName, entityId, theApi, body, retrySettings)
		return entity, err
	}

	if len(entitiesWithSameName) == 1 { // name is currently unique, update know entity
		existingUuid := entitiesWithSameName[0].Id
		entity, err := updateDynatraceObject(ctx, client, fullUrl, objectName, existingUuid, theApi, body, retrySettings)
		return entity, err
	}

	msg := strings.Builder{} // builder errors are ignored as write string always return nil error
	msg.WriteString("%d %q entities with name %q exist - monaco will create a new entity with known UUID %q to allow automated updates")
	msg.WriteString("\n\tYou may have to manually remove pre-existing configuration if this duplicates manually created configuration, and if possible consider giving your configurations unique names.")
	msg.WriteString("\n\tPre-existing %q configurations with same name:")
	for _, e := range entitiesWithSameName {
		msg.WriteString(fmt.Sprintf("\n\t- %s", e.Id))
	}
	log.WithCtxFields(ctx).Warn(msg.String(), len(entitiesWithSameName), theApi.ID, objectName, entityId, theApi.ID)

	return updateDynatraceObject(ctx, client, fullUrl, objectName, entityId, theApi, body, retrySettings)
}

func createDynatraceObject(ctx context.Context, client *http.Client, urlString string, objectName string, theApi api.API, payload []byte, retrySettings rest.RetrySettings) (DynatraceEntity, error) {
	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("invalid URL for creating Dynatrace config: %w", err)
	}
	body := payload

	configType := theApi.ID

	if configType == "app-detection-rule" {
		queryParams := parsedUrl.Query()
		queryParams.Add("position", "PREPEND")
		parsedUrl.RawQuery = queryParams.Encode()
	}

	resp, err := callWithRetryOnKnowTimingIssue(ctx, client, rest.Post, objectName, parsedUrl.String(), body, theApi, retrySettings)
	if err != nil {
		var respErr rest.RespError
		if errors.As(err, &respErr) {
			return DynatraceEntity{}, respErr.WithRequestInfo(http.MethodPost, parsedUrl.String())
		}
		return DynatraceEntity{}, err
	}

	if !resp.IsSuccess() {
		return DynatraceEntity{}, rest.NewRespErr(fmt.Sprintf("Failed to create DT object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodPost, parsedUrl.String())
	}

	return unmarshalCreateResponse(ctx, resp, urlString, configType, objectName)
}

func unmarshalCreateResponse(ctx context.Context, resp rest.Response, fullUrl string, configType string, objectName string) (DynatraceEntity, error) {
	var dtEntity DynatraceEntity

	if configType == "synthetic-monitor" || configType == "synthetic-location" {
		var entity SyntheticEntity
		err := json.Unmarshal(resp.Body, &entity)
		if errutils.CheckError(err, "Failed to unmarshal Synthetic API response") {
			return DynatraceEntity{}, rest.NewRespErr("Failed to unmarshal Synthetic API response", resp).WithRequestInfo(http.MethodPost, fullUrl).WithErr(err)
		}
		dtEntity = translateSyntheticEntityResponse(entity, objectName)

	} else if available, locationSlice := isLocationHeaderAvailable(resp); available {

		// The POST of the SLO API does not return the ID of the config in its response. Instead, it contains a
		// Location header, which contains the URL to the created resource. This URL needs to be cleaned, to get the
		// ID of the config.

		if len(locationSlice) == 0 {
			return DynatraceEntity{}, rest.NewRespErr(fmt.Sprintf("location response header was empty for API %s (name: %s)", configType, objectName), resp).WithRequestInfo(http.MethodPost, fullUrl)
		}

		location := locationSlice[0]

		// Some APIs prepend the environment URL. If available, trim it from the location
		objectId := strings.TrimPrefix(location, fullUrl)
		objectId = strings.TrimPrefix(objectId, "/")

		dtEntity = DynatraceEntity{
			Id:          objectId,
			Name:        objectName,
			Description: "Created object",
		}

	} else {
		err := json.Unmarshal(resp.Body, &dtEntity)
		if errutils.CheckError(err, "Failed to unmarshal API response") {
			return DynatraceEntity{}, rest.NewRespErr("Failed to unmarshal API response", resp).WithRequestInfo(http.MethodPost, fullUrl).WithErr(err)
		}
		if dtEntity.Id == "" && dtEntity.Name == "" {
			return DynatraceEntity{}, rest.NewRespErr(fmt.Sprintf("cannot parse API response '%s' into Dynatrace Entity with Id and Name", resp.Body), resp).WithRequestInfo(http.MethodPost, fullUrl)
		}
	}
	log.WithCtxFields(ctx).Debug("\tCreated new object for %s (%s)", dtEntity.Name, dtEntity.Id)

	return dtEntity, nil
}

func updateDynatraceObject(ctx context.Context, client *http.Client, fullUrl string, objectName string, existingObjectId string, theApi api.API, payload []byte, retrySettings rest.RetrySettings) (DynatraceEntity, error) {
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

	resp, err := callWithRetryOnKnowTimingIssue(ctx, client, rest.Put, objectName, path, body, theApi, retrySettings)

	if err != nil {
		var respErr rest.RespError
		if errors.As(err, &respErr) {
			return DynatraceEntity{}, respErr.WithRequestInfo(http.MethodPut, path)
		}
		return DynatraceEntity{}, err
	}

	if !resp.IsSuccess() {
		return DynatraceEntity{}, rest.NewRespErr(fmt.Sprintf("Failed to update Config object %s (HTTP %d)!\n    Response was: %s", objectName, resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodPut, path)
	}

	if theApi.NonUniqueName {
		log.WithCtxFields(ctx).Debug("\tCreated/Updated object by ID for %s (%s)", objectName, existingObjectId)
	} else {
		log.WithCtxFields(ctx).Debug("\tUpdated existing object for %s (%s)", objectName, existingObjectId)
	}

	return DynatraceEntity{
		Id:          existingObjectId,
		Name:        objectName,
		Description: "Updated existing object",
	}, nil
}

func stripCreateOnlyPropertiesFromAppMobile(payload []byte) []byte {
	// applicationType is required on creation, but not allowed to be updated
	r := regexp.MustCompile(`"applicationType":.*?,`)
	tmp := r.ReplaceAllString(string(payload), "")
	newPayload := []byte(tmp)

	return newPayload
}

// callWithRetryOnKnowTimingIssue handles several know cases in which Dynatrace has a slight delay before newly created objects
// can be used in further configuration. This is a cheap way to allow monaco to work around this, by waiting, then
// retrying in case of know errors on upload.
func callWithRetryOnKnowTimingIssue(ctx context.Context, client *http.Client, restCall rest.SendRequestWithBody, objectName string, path string, body []byte, theApi api.API, retrySettings rest.RetrySettings) (rest.Response, error) {

	resp, err := restCall(ctx, client, path, body)

	if err == nil && resp.IsSuccess() {
		return resp, nil
	}

	var setting rest.RetrySetting

	// It can take longer until calculated service metrics are ready to be used in SLOs
	if isCalculatedMetricNotReadyYet(resp) ||
		// It can take longer until management zones are ready to be used in SLOs
		isManagementZoneNotReadyYet(resp) ||
		// It can take longer until Credentials are ready to be used in Synthetic Monitors
		isCredentialNotReadyYet(resp) ||
		// It can take some time for configurations to propagate to all cluster nodes - indicated by an incorrect constraint violation error
		isGeneralDependencyNotReadyYet(resp) ||
		// Synthetic and related APIs sometimes run into issues of finding objects quickly after creation
		isGeneralSyntheticAPIError(resp, theApi) {

		setting = retrySettings.Normal
	}

	// It can take even longer until request attributes are ready to be used
	if isRequestAttributeNotYetReady(resp) {
		setting = retrySettings.Long
	}

	// It can take even longer until applications are ready to be used in synthetic tests and calculated metrics
	if isApplicationNotReadyYet(resp, theApi) {
		setting = retrySettings.VeryLong
	}

	if setting.MaxRetries > 0 {
		return rest.SendWithRetry(ctx, client, restCall, objectName, path, body, setting)
	}
	return resp, nil
}

func isGeneralDependencyNotReadyYet(resp rest.Response) bool {
	return strings.Contains(string(resp.Body), "must have a unique name")
}

func isCalculatedMetricNotReadyYet(resp rest.Response) bool {
	return strings.Contains(string(resp.Body), "Metric selector") &&
		strings.Contains(string(resp.Body), "invalid")
}

func isRequestAttributeNotYetReady(resp rest.Response) bool {
	return strings.Contains(string(resp.Body), "must specify a known request attribute")
}

func isManagementZoneNotReadyYet(resp rest.Response) bool {
	return strings.Contains(string(resp.Body), "Entity selector is invalid") ||
		(strings.Contains(string(resp.Body), "SLO validation failed") &&
			strings.Contains(string(resp.Body), "Management-Zone not found")) ||
		strings.Contains(string(resp.Body), "Unknown management zone")
}

func isApplicationNotReadyYet(resp rest.Response, theApi api.API) bool {
	return isCalculatedMetricsError(resp, theApi) ||
		isSyntheticMonitorServerError(resp, theApi) ||
		isApplicationAPIError(resp, theApi) ||
		strings.Contains(string(resp.Body), "Unknown application(s)")
}

func isCalculatedMetricsError(resp rest.Response, theApi api.API) bool {
	return strings.HasPrefix(theApi.ID, "calculated-metrics") && (resp.Is4xxError() || resp.Is5xxError())
}
func isSyntheticMonitorServerError(resp rest.Response, theApi api.API) bool {
	return theApi.ID == "synthetic-monitor" && resp.Is5xxError()
}

func isGeneralSyntheticAPIError(resp rest.Response, theApi api.API) bool {
	return (strings.HasPrefix(theApi.ID, "synthetic-") || theApi.ID == "credential-vault") && (resp.StatusCode == http.StatusNotFound || resp.Is5xxError())
}

func isApplicationAPIError(resp rest.Response, theApi api.API) bool {
	return isAnyApplicationApi(theApi) &&
		(resp.Is5xxError() || resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusNotFound)
}

func isCredentialNotReadyYet(resp rest.Response) bool {
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

func isLocationHeaderAvailable(resp rest.Response) (headerAvailable bool, headerArray []string) {
	if resp.Headers["Location"] != nil {
		return true, resp.Headers["Location"]
	}
	return false, make([]string, 0)
}

func getObjectIdIfAlreadyExists(ctx context.Context, client *http.Client, api api.API, url string, objectName string, retrySettings rest.RetrySettings) (string, error) {
	values, err := getExistingValuesFromEndpoint(ctx, client, api, url, retrySettings)

	if err != nil {
		return "", err
	}

	var objectId = ""
	var matchingObjectsFound = 0
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.Name == objectName || escapeApiValueName(ctx, value) == objectName {
			if matchingObjectsFound == 0 {
				objectId = value.Id
			}
			matchingObjectsFound++
		}
	}

	if matchingObjectsFound > 1 {
		log.WithCtxFields(ctx).Warn("Found %d configs with same name: %s. Please delete duplicates.", matchingObjectsFound, objectName)
	}

	if objectId != "" {
		log.WithCtxFields(ctx).Debug("Found existing config %s (%s) with id %s", objectName, api.ID, objectId)
	}

	return objectId, nil
}

func escapeApiValueName(ctx context.Context, value Value) string {
	valueName, err := template.EscapeSpecialCharactersInValue(value.Name, template.FullStringEscapeFunction)
	if err != nil {
		log.WithCtxFields(ctx).Warn("failed to string escape API value '%s' while checking if object exists, check directly", value.Name)
		return value.Name
	}
	return valueName.(string)
}

func isApiDashboard(api api.API) bool {
	return api.ID == "dashboard" || api.ID == "dashboard-v2"
}

func isReportsApi(api api.API) bool {
	return api.ID == "reports"
}

func isAnyServiceDetectionApi(api api.API) bool {
	return strings.HasPrefix(api.ID, "service-detection-")
}

func isAnyApplicationApi(api api.API) bool {
	return strings.HasPrefix(api.ID, "application-")
}

func isMobileApp(api api.API) bool {
	return api.ID == "application-mobile"
}

func getExistingValuesFromEndpoint(ctx context.Context, client *http.Client, theApi api.API, urlString string, retrySettings rest.RetrySettings) (values []Value, err error) {

	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("invalid URL for getting existing Dynatrace configs: %w", err)
	}

	parsedUrl = addQueryParamsForNonStandardApis(theApi, parsedUrl)

	resp, err := rest.Get(ctx, client, parsedUrl.String())

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, rest.NewRespErr(fmt.Sprintf("Failed to get existing configs for API %s (HTTP %d)!\n    Response was: %s", theApi.ID, resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodGet, parsedUrl.String())
	}

	var existingValues []Value
	for {
		values, err := unmarshalJson(ctx, theApi, resp)
		if err != nil {
			return nil, err
		}
		existingValues = append(existingValues, values...)

		if resp.NextPageKey != "" {
			parsedUrl = rest.AddNextPageQueryParams(parsedUrl, resp.NextPageKey)

			resp, err = rest.GetWithRetry(ctx, client, parsedUrl.String(), retrySettings.Normal)

			if err != nil {
				return nil, err
			}

			if !resp.IsSuccess() && resp.StatusCode != http.StatusBadRequest {
				return nil, rest.NewRespErr(fmt.Sprintf("Failed to get further configs from paginated API %s (HTTP %d)!\n    Response was: %s", theApi.ID, resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodGet, parsedUrl.String())
			} else if resp.StatusCode == http.StatusBadRequest {
				log.WithCtxFields(ctx).Warn("Failed to get additional data from paginated API %s - pages may have been removed during request.\n    Response was: %s", theApi.ID, string(resp.Body))
				break
			}

		} else {
			break
		}
	}

	return existingValues, nil
}

func addQueryParamsForNonStandardApis(theApi api.API, url *url.URL) *url.URL {

	queryParams := url.Query()
	if theApi.ID == "anomaly-detection-metrics" {
		queryParams.Add("includeEntityFilterMetricEvents", "true")
	}
	if theApi.ID == "slo" {
		queryParams.Add("enabledSlos", "all")
	}
	url.RawQuery = queryParams.Encode()
	return url
}

func unmarshalJson(ctx context.Context, theApi api.API, resp rest.Response) ([]Value, error) {

	var values []Value
	var objmap map[string]interface{}

	// This API returns an untyped list as a response -> it needs a special handling
	if theApi.ID == "aws-credentials" {

		var jsonResp []Value
		err := json.Unmarshal(resp.Body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing aws-credentials") {
			return values, err
		}
		values = jsonResp

	} else {

		if theApi.ID == "synthetic-location" {

			var jsonResp SyntheticLocationResponse
			err := json.Unmarshal(resp.Body, &jsonResp)
			if errutils.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
				return nil, err
			}
			values = translateSyntheticValues(jsonResp.Locations)

		} else if theApi.ID == "synthetic-monitor" {

			var jsonResp SyntheticMonitorsResponse
			err := json.Unmarshal(resp.Body, &jsonResp)
			if errutils.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
				return nil, err
			}
			values = translateSyntheticValues(jsonResp.Monitors)

		} else if !theApi.IsStandardAPI() || isReportsApi(theApi) {

			if err := json.Unmarshal(resp.Body, &objmap); err != nil {
				return nil, err
			}

			if available, array := isResultArrayAvailable(objmap, theApi); available {
				jsonResp, err := translateGenericValues(ctx, array, theApi.ID)
				if err != nil {
					return nil, err
				}
				values = jsonResp
			}

		} else {

			var jsonResponse ValuesResponse
			err := json.Unmarshal(resp.Body, &jsonResponse)
			if errutils.CheckError(err, "Cannot unmarshal API response for existing objects") {
				return nil, err
			}
			values = jsonResponse.Values
		}
	}

	return values, nil
}

func isResultArrayAvailable(jsonResponse map[string]interface{}, theApi api.API) (resultArrayAvailable bool, results []interface{}) {
	if jsonResponse[theApi.PropertyNameOfGetAllResponse] != nil {
		return true, jsonResponse[theApi.PropertyNameOfGetAllResponse].([]interface{})
	}
	return false, make([]interface{}, 0)
}

func translateGenericValues(ctx context.Context, inputValues []interface{}, configType string) ([]Value, error) {

	values := make([]Value, 0, len(inputValues))

	for _, input := range inputValues {
		input := input.(map[string]interface{})

		if input["id"] == nil {
			return values, fmt.Errorf("config of type %s was invalid: No id", configType)
		}

		// Substitute missing name attribute
		if input["name"] == nil {
			jsonStr, err := json.Marshal(input)
			if err != nil {
				log.WithCtxFields(ctx).Warn("Config of type %s was invalid. Ignoring it!", configType)
				continue
			}

			substitutedName := ""

			// Differentiate handling for reports API from others
			isReportsApi := configType == "reports"
			if isReportsApi {
				// Substitute name with dashboard id since it is unique identifier for entity
				substitutedName = input["dashboardId"].(string)
				log.WithCtxFields(ctx).Debug("Rewriting response of config-type '%v', name missing. Using dashboardId as name. Invalid json: %v", configType, string(jsonStr))

			} else {
				// Substitute name with id since it is unique identifier for entity
				substitutedName = input["id"].(string)
				log.WithCtxFields(ctx).Debug("Rewriting response of config-type '%v', name missing. Using id as name. Invalid json: %v", configType, string(jsonStr))
			}

			values = append(values, Value{
				Id:   input["id"].(string),
				Name: substitutedName,
			})
			continue
		}

		value := Value{
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

func translateSyntheticValues(syntheticValues []SyntheticValue) []Value {
	values := make([]Value, 0, len(syntheticValues))
	for _, loc := range syntheticValues {
		values = append(values, Value{
			Id:   loc.EntityId,
			Name: loc.Name,
		})
	}
	return values
}

func translateSyntheticEntityResponse(resp SyntheticEntity, objectName string) DynatraceEntity {
	return DynatraceEntity{
		Name: objectName,
		Id:   resp.EntityId,
	}
}
