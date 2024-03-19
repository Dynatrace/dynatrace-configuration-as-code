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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"golang.org/x/exp/maps"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func (d *DynatraceClient) upsertDynatraceObject(ctx context.Context, theApi api.API, objectName string, payload []byte) (DynatraceEntity, error) {
	doUpsert := func() (DynatraceEntity, error) {
		existingObjectID, err := d.getExistingObjectId(ctx, objectName, theApi, payload)
		if err != nil {
			return DynatraceEntity{}, err
		}

		// Single configurations with a parent use the parent's ID
		if theApi.SingleConfiguration && theApi.HasParent() {
			existingObjectID = theApi.AppliedParentObjectID
		}

		// The network-zone API doesn't have a POST endpoint, hence, we need to treat it as an update operation
		// per default
		if theApi.ID == api.UserActionAndSessionPropertiesMobile {
			existingObjectID = objectName
		}

		// The network-zone API doesn't have a POST endpoint, hence, we need to treat it as an update operation
		// per default
		if theApi.ID == api.NetworkZone {
			existingObjectID = objectName
		}

		// The calculated-metrics-log API doesn't have a POST endpoint, to create a new log metric we need to use PUT which
		// requires a metric key for which we can just take the objectName
		if theApi.ID == api.CalculatedMetricsLog && existingObjectID == "" {
			existingObjectID = objectName
		}

		// Single configuration APIs don't have a POST, but a PUT endpoint
		// and therefore always require an update
		if existingObjectID != "" || theApi.SingleConfiguration {
			return d.updateDynatraceObject(ctx, theApi.CreateURL(d.environmentURLClassic), objectName, existingObjectID, theApi, payload)
		} else {
			return d.createDynatraceObject(ctx, theApi.CreateURL(d.environmentURLClassic), objectName, theApi, payload)
		}
	}

	if obj, err := doUpsert(); err == nil {
		return obj, nil
	} else {
		d.classicConfigsCache.Delete(theApi.ID)
		return doUpsert()
	}
}

func (d *DynatraceClient) upsertDynatraceEntityByNonUniqueNameAndId(
	ctx context.Context,
	entityId string,
	objectName string,
	theApi api.API,
	payload []byte,
	duplicate bool,
) (DynatraceEntity, error) {
	fullUrl := theApi.CreateURL(d.environmentURLClassic)
	body := payload

	existingEntities, err := d.fetchExistingValues(ctx, theApi, fullUrl)
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
		entity, err := d.updateDynatraceObject(ctx, fullUrl, objectName, entityId, theApi, body)
		return entity, err
	}

	// check if we are dealing with a duplicate non-unique name configuration, if not, go ahead and update the known entity
	if featureflags.UpdateNonUniqueByNameIfSingleOneExists().Enabled() && len(entitiesWithSameName) == 1 && !duplicate {
		existingUuid := entitiesWithSameName[0].Id
		entity, err := d.updateDynatraceObject(ctx, fullUrl, objectName, existingUuid, theApi, body)
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

	return d.updateDynatraceObject(ctx, fullUrl, objectName, entityId, theApi, body)
}

func (d *DynatraceClient) createDynatraceObject(ctx context.Context, urlString string, objectName string, theApi api.API, payload []byte) (DynatraceEntity, error) {

	if theApi.ID == api.KeyUserActionsMobile {
		urlString = joinUrl(urlString, objectName)
	}

	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("invalid URL for creating Dynatrace config: %w", err)
	}
	body := payload

	if theApi.ID == api.AppDetectionRule {
		queryParams := parsedUrl.Query()
		queryParams.Add("position", "PREPEND")
		parsedUrl.RawQuery = queryParams.Encode()
	}

	resp, err := d.callWithRetryOnKnowTimingIssue(ctx, d.classicClient.Post, parsedUrl.String(), body, theApi)
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

	return unmarshalCreateResponse(ctx, resp, urlString, theApi.ID, objectName)
}

func unmarshalCreateResponse(ctx context.Context, resp rest.Response, fullUrl string, configType string, objectName string) (DynatraceEntity, error) {
	var dtEntity DynatraceEntity

	if configType == api.SyntheticMonitor || configType == api.SyntheticLocation {
		var entity SyntheticEntity
		err := json.Unmarshal(resp.Body, &entity)
		if errutils.CheckError(err, "Failed to unmarshal Synthetic API response") {
			return DynatraceEntity{}, rest.NewRespErr("Failed to unmarshal Synthetic API response", resp).WithRequestInfo(http.MethodPost, fullUrl).WithErr(err)
		}
		dtEntity = translateSyntheticEntityResponse(entity, objectName)

	} else if locationSlice, exist := getLocationFromHeader(resp); exist {

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

func (d *DynatraceClient) updateDynatraceObject(ctx context.Context, fullUrl string, objectName string, existingObjectId string, theApi api.API, payload []byte) (DynatraceEntity, error) {
	path := fullUrl
	if !theApi.SingleConfiguration {
		path = joinUrl(fullUrl, existingObjectId)
	}

	// Updating a dashboard, reports or any service detection API requires the ID to be contained in the JSON, so we just add it...
	if isApiDashboard(theApi) || isApiDashboardShareSettings(theApi) || isReportsApi(theApi) || isAnyServiceDetectionApi(theApi) {
		payload = addObjectIDToPayload(payload, existingObjectId)
	}

	// Updating a Mobile Application does not allow changing the applicationType as such this property required on Create, must be stripped on Update
	if isMobileApp(theApi) {
		payload = stripCreateOnlyPropertiesFromAppMobile(payload)
	}

	// Key user mobile actions can't be updated, thus we return immediately
	if isKeyUserActionMobile(theApi) || isKeyUserActionWeb(theApi) {
		return DynatraceEntity{
			Id:   existingObjectId,
			Name: objectName,
		}, nil
	}

	resp, err := d.callWithRetryOnKnowTimingIssue(ctx, d.classicClient.Put, path, payload, theApi)

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
		log.WithCtxFields(ctx).Debug("Created/Updated object by ID for %s", getNameIDDescription(objectName, existingObjectId))
	} else {
		log.WithCtxFields(ctx).Debug("Updated existing object for %s", getNameIDDescription(objectName, existingObjectId))
	}

	return DynatraceEntity{
		Id:          existingObjectId,
		Name:        objectName,
		Description: "Updated existing object",
	}, nil
}

func addObjectIDToPayload(payload []byte, objectID string) []byte {
	return []byte(strings.Replace(string(payload), "{", "{\n\"id\":\""+objectID+"\",\n", 1))
}

func getNameIDDescription(objectName, objectID string) string {
	if objectName == "" {
		return objectID
	}

	return fmt.Sprintf("%s (%s)", objectName, objectID)
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
func (d *DynatraceClient) callWithRetryOnKnowTimingIssue(ctx context.Context, restCall rest.SendRequestWithBody, path string, body []byte, theApi api.API) (rest.Response, error) {
	resp, err := restCall(ctx, path, body)
	if err == nil && resp.IsSuccess() {
		return resp, nil
	}
	if err != nil {
		log.WithCtxFields(ctx).WithFields(field.Error(err)).Warn("Failed to send HTTP request: %v", err)
	} else {

		log.WithCtxFields(ctx).WithFields(field.F("statusCode", resp.StatusCode)).Warn("Failed to send HTTP request: (HTTP %d)!\n    Response was: %s", resp.StatusCode, string(resp.Body))
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
		isGeneralSyntheticAPIError(resp, theApi) ||
		// Network zone deployments can fail due to fact that the feature not effectively enabled yet
		isNetworkZoneFeatureNotEnabledYet(resp, theApi) {

		setting = d.retrySettings.Normal
	}

	// It can take even longer until request attributes are ready to be used
	if isRequestAttributeNotYetReady(resp) {
		setting = d.retrySettings.Long
	}

	// It can take even longer until applications are ready to be used in synthetic tests and calculated metrics
	if isApplicationNotReadyYet(resp, theApi) {
		setting = d.retrySettings.VeryLong
	}

	if setting.MaxRetries > 0 {
		return rest.SendWithRetry(ctx, restCall, path, body, setting)
	}
	return resp, err
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
		isApplicationDetectionRuleException(resp, theApi) ||
		isKeyUserActionMobile(theApi) ||
		isKeyUserActionWeb(theApi) ||
		isUserSessionPropertiesMobile(theApi) ||
		strings.Contains(string(resp.Body), "Unknown application(s)")
}
func isNetworkZoneFeatureNotEnabledYet(resp rest.Response, theApi api.API) bool {
	return strings.HasPrefix(theApi.ID, "network-zone") && (resp.Is4xxError()) && strings.Contains(string(resp.Body), "network zones are disabled")
}

func isCalculatedMetricsError(resp rest.Response, theApi api.API) bool {
	return strings.HasPrefix(theApi.ID, "calculated-metrics") && (resp.Is4xxError() || resp.Is5xxError())
}
func isSyntheticMonitorServerError(resp rest.Response, theApi api.API) bool {
	return theApi.ID == api.SyntheticMonitor && resp.Is5xxError()
}

func isGeneralSyntheticAPIError(resp rest.Response, theApi api.API) bool {
	return (strings.HasPrefix(theApi.ID, "synthetic-") || theApi.ID == api.CredentialVault) && (resp.StatusCode == http.StatusNotFound || resp.Is5xxError())
}

func isApplicationAPIError(resp rest.Response, theApi api.API) bool {
	return isAnyApplicationApi(theApi) &&
		(resp.Is5xxError() || resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusNotFound)
}

// Special case (workaround):
// Sometimes, the API returns 500 Internal Server Error e.g. when an application referenced by
// an application detection rule is not fully "ready" yet.
func isApplicationDetectionRuleException(resp rest.Response, theApi api.API) bool {
	return theApi.ID == api.AppDetectionRule && !resp.IsSuccess()
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

func getLocationFromHeader(resp rest.Response) ([]string, bool) {
	if resp.Headers["Location"] != nil {
		return resp.Headers["Location"], true
	}
	return make([]string, 0), false
}

func isApiDashboard(a api.API) bool {
	return a.ID == api.Dashboard || a.ID == api.DashboardV2
}

func isApiDashboardShareSettings(a api.API) bool {
	return a.ID == api.DashboardShareSettings
}

func isReportsApi(a api.API) bool {
	return a.ID == api.Reports
}

func isAnyServiceDetectionApi(api api.API) bool {
	return strings.HasPrefix(api.ID, "service-detection-")
}

func isAnyApplicationApi(api api.API) bool {
	return strings.HasPrefix(api.ID, "application-")
}

func isMobileApp(a api.API) bool {
	return a.ID == api.ApplicationMobile
}

func isKeyUserActionMobile(a api.API) bool {
	return a.ID == api.KeyUserActionsMobile
}

func isKeyUserActionWeb(a api.API) bool {
	return a.ID == api.KeyUserActionsWeb
}

func isUserSessionPropertiesMobile(a api.API) bool {
	return a.ID == api.UserActionAndSessionPropertiesMobile
}

func (d *DynatraceClient) getExistingObjectId(ctx context.Context, objectName string, theApi api.API, payload []byte) (string, error) {
	var objID string
	// if there is a custom equal function registered, use that instead of just the Object name
	// in order to search for existing values
	if theApi.CheckEqualFunc != nil {
		values, err := d.fetchExistingValues(ctx, theApi, theApi.CreateURL(d.environmentURLClassic))
		if err != nil {
			return "", err
		}
		objID, err = d.findUnique(ctx, values, payload, theApi.CheckEqualFunc)
		if err != nil {
			return "", err
		}
		return objID, nil
	}

	// Single configuration APIs don't have an id which allows skipping this step
	if !theApi.SingleConfiguration {
		values, err := d.fetchExistingValues(ctx, theApi, theApi.CreateURL(d.environmentURLClassic))
		if err != nil {
			return "", err
		}
		objID = d.findUniqueByName(ctx, values, objectName)
	}

	if objID != "" {
		log.WithCtxFields(ctx).Debug("Found existing config of type %s with id %s", theApi.ID, objID)
	}
	return objID, nil
}

func (d *DynatraceClient) fetchExistingValues(ctx context.Context, theApi api.API, urlString string) (values []Value, err error) {
	// caching cannot be used for subPathAPI as well because there is potentially more than one config per api type/id to consider.
	// the cache cannot deal with that
	if !theApi.NonUniqueName && !theApi.HasParent() {
		if values, cached := d.classicConfigsCache.Get(theApi.ID); cached {
			return values, nil
		}
	}
	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("invalid URL for getting existing Dynatrace configs: %w", err)
	}

	parsedUrl = addQueryParamsForNonStandardApis(theApi, parsedUrl)

	var resp rest.Response
	// For any subpath API like e.g. Key user Actions, it can be that we need to do longer retries
	// because the parent config (e.g. an application) is not ready yet
	if theApi.HasParent() {
		resp, err = rest.SendWithRetryWithInitialTry(ctx, func(ctx context.Context, url string, _ []byte) (rest.Response, error) {
			return d.classicClient.Get(ctx, url)
		}, parsedUrl.String(), nil, rest.DefaultRetrySettings.Long)
	} else {
		resp, err = d.classicClient.Get(ctx, parsedUrl.String())
	}

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

			resp, err = d.classicClient.GetWithRetry(ctx, parsedUrl.String(), d.retrySettings.Normal)

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
	d.classicConfigsCache.Set(theApi.ID, existingValues)
	return existingValues, nil
}

func (d *DynatraceClient) findUniqueByName(ctx context.Context, values []Value, objectName string) string {
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
	return objectId
}

func escapeApiValueName(ctx context.Context, value Value) string {
	valueName, err := template.EscapeSpecialCharactersInValue(value.Name, template.FullStringEscapeFunction)
	if err != nil {
		log.WithCtxFields(ctx).Warn("failed to string escape API value '%s' while checking if object exists, check directly", value.Name)
		return value.Name
	}
	return valueName.(string)
}

func (d *DynatraceClient) findUnique(ctx context.Context, values []Value, payload []byte, checkEqualFunc func(map[string]any, map[string]any) bool) (string, error) {
	if checkEqualFunc == nil {
		return "", nil
	}
	var objectId = ""
	var matchingObjectsFound = 0
	for i := 0; i < len(values); i++ {
		value := values[i]
		var pl map[string]any
		if err := json.Unmarshal(payload, &pl); err != nil {
			return "", err
		}
		if checkEqualFunc(value.CustomFields, pl) {
			if matchingObjectsFound == 0 {
				objectId = value.Id
			}
			matchingObjectsFound++
		}
	}

	if matchingObjectsFound > 1 {
		log.WithCtxFields(ctx).Warn("Found %d configs with same name: %s. Please delete duplicates.", matchingObjectsFound)
	}

	return objectId, nil
}

func addQueryParamsForNonStandardApis(theApi api.API, url *url.URL) *url.URL {

	queryParams := url.Query()
	if theApi.ID == api.AnomalyDetectionMetrics {
		queryParams.Add("includeEntityFilterMetricEvents", "true")
	}
	if theApi.ID == api.Slo {
		queryParams.Add("enabledSlos", "all")
	}
	url.RawQuery = queryParams.Encode()
	return url
}

func unmarshalJson(ctx context.Context, theApi api.API, resp rest.Response) ([]Value, error) {

	var values []Value
	var objmap map[string]interface{}

	// This API returns an untyped list as a response -> it needs a special handling
	if theApi.ID == api.AwsCredentials {
		var jsonResp []Value
		err := json.Unmarshal(resp.Body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing aws-credentials") {
			return values, err
		}
		values = jsonResp
	} else if theApi.ID == api.SyntheticLocation {
		var jsonResp SyntheticLocationResponse
		err := json.Unmarshal(resp.Body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
			return nil, err
		}
		values = translateSyntheticValues(jsonResp.Locations)
	} else if theApi.ID == api.SyntheticMonitor {
		var jsonResp SyntheticMonitorsResponse
		err := json.Unmarshal(resp.Body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
			return nil, err
		}
		values = translateSyntheticValues(jsonResp.Monitors)
	} else if theApi.ID == api.KeyUserActionsMobile {
		var jsonResp KeyUserActionsMobileResponse
		err := json.Unmarshal(resp.Body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing key user action") {
			return nil, err
		}
		for _, kua := range jsonResp.KeyUserActions {
			values = append(values, Value{
				Id:   kua.Name,
				Name: kua.Name,
			})
		}
	} else if theApi.ID == api.KeyUserActionsWeb {
		var jsonResp struct {
			List []struct {
				Name         string `json:"name"`
				MeIdentifier string `json:"meIdentifier"`
				Domain       string `json:"domain"`
				ActionType   string `json:"actionType"`
			} `json:"keyUserActionList"`
		}
		err := json.Unmarshal(resp.Body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing key user action") {
			return nil, err
		}
		for _, kua := range jsonResp.List {
			values = append(values, Value{
				Id:           kua.MeIdentifier,
				Name:         kua.Name,
				CustomFields: map[string]any{"name": kua.Name, "domain": kua.Domain, "actionType": kua.ActionType},
			})
		}
	} else if theApi.ID == api.UserActionAndSessionPropertiesMobile {
		var jsonResp UserActionAndSessionPropertyResponse
		err := json.Unmarshal(resp.Body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing key user action") {
			return nil, err
		}

		// The entries are potentially duplicated, that's why we need to map by the unique key
		entries := map[string]Value{}
		for _, entry := range jsonResp.UserActionProperties {
			entries[entry.Key] = Value{Id: entry.Key, Name: entry.Key}
		}
		for _, entry := range jsonResp.SessionProperties {
			entries[entry.Key] = Value{Id: entry.Key, Name: entry.Key}
		}
		values = maps.Values(entries)

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
			isReportsApi := configType == api.Reports
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
