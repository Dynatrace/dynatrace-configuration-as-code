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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/exp/maps"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/cache"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
)

type ConfigClient struct {
	client *corerest.Client

	// retrySettings are the settings to be used for retrying failed http requests
	retrySettings RetrySettings

	// configCache caches config API values
	configCache cache.Cache[[]Value]
}

// WithRetrySettings sets the retry settings to be used by the ConfigClient
func WithRetrySettingsForClassic(retrySettings RetrySettings) func(*ConfigClient) {
	return func(d *ConfigClient) {
		d.retrySettings = retrySettings
	}
}

// WithCachingDisabledForConfigClient allows disabling the client's builtin caching mechanism for classic configs.
// Disabling the caching is recommended in situations where configs are fetched immediately after their creation (e.g. in test scenarios).
func WithCachingDisabledForConfigClient(disabled bool) func(client *ConfigClient) {
	return func(d *ConfigClient) {
		if disabled {
			d.configCache = &cache.NoopCache[[]Value]{}
		}
	}
}

func NewClassicConfigClient(client *corerest.Client, opts ...func(dynatraceClient *ConfigClient)) (*ConfigClient, error) {
	d := &ConfigClient{
		client:        client,
		retrySettings: DefaultRetrySettings,
		configCache:   &cache.DefaultCache[[]Value]{},
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

func (d *ConfigClient) Cache(ctx context.Context, api api.API) error {
	_, err := d.List(ctx, api)
	return err
}

func (d *ConfigClient) Get(ctx context.Context, api api.API, id string) (json []byte, err error) {
	var dtUrl = api.URLPath
	if !api.SingleConfiguration {
		dtUrl = dtUrl + "/" + url.PathEscape(id)
	}

	response, err := coreapi.AsResponseOrError(d.client.GET(ctx, dtUrl, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (d *ConfigClient) Delete(ctx context.Context, api api.API, id string) error {
	parsedURL, err := url.Parse(api.URLPath)
	if err != nil {
		return err
	}
	parsedURL = parsedURL.JoinPath(id)

	_, err = coreapi.AsResponseOrError(d.client.DELETE(ctx, parsedURL.String(), corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		apiError := coreapi.APIError{}
		if errors.As(err, &apiError) && apiError.StatusCode == http.StatusNotFound {
			log.Debug("No config with id '%s' found to delete (HTTP 404 response)", id)
			return nil
		}
		return err
	}

	return nil
}

func (d *ConfigClient) ExistsWithName(ctx context.Context, api api.API, name string) (exists bool, id string, err error) {
	if api.SingleConfiguration {
		// check that a single configuration is there by actually reading it.
		_, err := d.Get(ctx, api, "")
		return err == nil, "", nil
	}

	existingObjectId, err := d.getExistingObjectId(ctx, name, api, nil)
	return existingObjectId != "", existingObjectId, err
}

func (d *ConfigClient) UpsertByName(ctx context.Context, a api.API, name string, payload []byte) (entity DynatraceEntity, err error) {
	if a.ID == api.Extension {
		return d.uploadExtension(ctx, a, name, payload)
	}
	return d.upsertDynatraceObject(ctx, a, name, payload)
}

func (d *ConfigClient) upsertDynatraceObject(ctx context.Context, theApi api.API, objectName string, payload []byte) (DynatraceEntity, error) {
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
			return d.updateDynatraceObject(ctx, objectName, existingObjectID, theApi, payload)
		} else {
			return d.createDynatraceObject(ctx, objectName, theApi, payload)
		}
	}

	if obj, err := doUpsert(); err == nil {
		return obj, nil
	} else {
		d.configCache.Delete(theApi.ID)
		return doUpsert()
	}
}

func (d *ConfigClient) UpsertByNonUniqueNameAndId(ctx context.Context, theApi api.API, entityId string, objectName string, payload []byte, duplicate bool) (entity DynatraceEntity, err error) {
	body := payload

	existingEntities, err := d.List(ctx, theApi)
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
		entity, err := d.updateDynatraceObject(ctx, objectName, entityId, theApi, body)
		return entity, err
	}

	// check if we are dealing with a duplicate non-unique name configuration, if not, go ahead and update the known entity
	if featureflags.UpdateNonUniqueByNameIfSingleOneExists.Enabled() && len(entitiesWithSameName) == 1 && !duplicate {
		existingUuid := entitiesWithSameName[0].Id
		entity, err := d.updateDynatraceObject(ctx, objectName, existingUuid, theApi, body)
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

	return d.updateDynatraceObject(ctx, objectName, entityId, theApi, body)
}

func (d *ConfigClient) createDynatraceObject(ctx context.Context, objectName string, theApi api.API, payload []byte) (DynatraceEntity, error) {
	endpoint := theApi.URLPath
	if theApi.ID == api.KeyUserActionsMobile {
		endpoint = joinUrl(endpoint, objectName)
	}

	body := payload

	queryParams := url.Values{}
	if theApi.ID == api.AppDetectionRule {
		queryParams.Add("position", "PREPEND")
	}

	resp, err := d.callWithRetryOnKnowTimingIssue(ctx, d.client.POST, endpoint, body, theApi, corerest.RequestOptions{QueryParams: queryParams, CustomShouldRetryFunc: corerest.RetryIfTooManyRequests})
	if err != nil {
		return DynatraceEntity{}, err
	}

	return unmarshalCreateResponse(ctx, *resp, theApi.ID, objectName)
}

func unmarshalCreateResponse(ctx context.Context, resp coreapi.Response, configType string, objectName string) (DynatraceEntity, error) {
	var dtEntity DynatraceEntity

	if configType == api.SyntheticMonitor || configType == api.SyntheticLocation {
		var entity SyntheticEntity
		err := json.Unmarshal(resp.Data, &entity)
		if errutils.CheckError(err, "Failed to unmarshal Synthetic API response") {
			return DynatraceEntity{}, fmt.Errorf("failed to unmarshal Synthetic API response: %w", err)
		}
		dtEntity = translateSyntheticEntityResponse(entity, objectName)

	} else if locationSlice, exist := getLocationFromHeader(resp); exist {

		// The POST of the SLO API does not return the ID of the config in its response. Instead, it contains a
		// Location header, which contains the URL to the created resource. This URL needs to be cleaned, to get the
		// ID of the config.

		if len(locationSlice) == 0 {
			return DynatraceEntity{}, fmt.Errorf("location response header was empty for API %s (name: %s): %s", configType, objectName, resp.Request.URL)
		}

		location := locationSlice[0]

		// Some APIs prepend the environment URL. If available, trim it from the location
		objectId := strings.TrimPrefix(location, resp.Request.URL)
		objectId = strings.TrimPrefix(objectId, "/")

		dtEntity = DynatraceEntity{
			Id:          objectId,
			Name:        objectName,
			Description: "Created object",
		}

	} else {
		err := json.Unmarshal(resp.Data, &dtEntity)
		if errutils.CheckError(err, "Failed to unmarshal API response") {
			return DynatraceEntity{}, fmt.Errorf("failed to unmarshal API response: %w", err)
		}
		if dtEntity.Id == "" && dtEntity.Name == "" {
			return DynatraceEntity{}, fmt.Errorf("cannot parse API response '%s' into Dynatrace Entity with Id and Name", string(resp.Data))
		}
	}
	log.WithCtxFields(ctx).Debug("\tCreated new object for '%s' (%s)", dtEntity.Name, dtEntity.Id)

	return dtEntity, nil
}

func (d *ConfigClient) updateDynatraceObject(ctx context.Context, objectName string, existingObjectId string, theApi api.API, payload []byte) (DynatraceEntity, error) {
	endpoint := theApi.URLPath
	if !theApi.SingleConfiguration {
		endpoint = joinUrl(endpoint, existingObjectId)
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

	_, err := d.callWithRetryOnKnowTimingIssue(ctx, d.client.PUT, endpoint, payload, theApi, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests})
	if err != nil {
		return DynatraceEntity{}, err
	}

	if theApi.NonUniqueName {
		log.WithCtxFields(ctx).Debug("Created/Updated object by ID for '%s'", getNameIDDescription(objectName, existingObjectId))
	} else {
		log.WithCtxFields(ctx).Debug("Updated existing object for '%s'", getNameIDDescription(objectName, existingObjectId))
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
// retrying in case of known errors on upload.
func (d *ConfigClient) callWithRetryOnKnowTimingIssue(ctx context.Context, restCall SendRequestWithBody, endpoint string, requestBody []byte, theApi api.API, options corerest.RequestOptions) (*coreapi.Response, error) {
	resp, err := coreapi.AsResponseOrError(restCall(ctx, endpoint, bytes.NewReader(requestBody), options))
	if err == nil {
		return resp, nil
	}

	apiError := coreapi.APIError{}
	if !errors.As(err, &apiError) {
		return nil, err
	}

	var rs RetrySetting
	// It can take longer until calculated service metrics are ready to be used in SLOs
	if isCalculatedMetricNotReadyYet(apiError) ||
		// It can take longer until management zones are ready to be used in SLOs
		isManagementZoneNotReadyYet(apiError) ||
		// It can take longer until Credentials are ready to be used in Synthetic Monitors
		isCredentialNotReadyYet(apiError) ||
		// It can take some time for configurations to propagate to all cluster nodes - indicated by an incorrect constraint violation error
		isGeneralDependencyNotReadyYet(apiError) ||
		// Synthetic and related APIs sometimes run into issues of finding objects quickly after creation
		isGeneralSyntheticAPIError(apiError, theApi) ||
		// Network zone deployments can fail due to fact that the feature not effectively enabled yet
		isNetworkZoneFeatureNotEnabledYet(apiError, theApi) {

		rs = d.retrySettings.Normal
	}

	// It can take even longer until request attributes are ready to be used
	if isRequestAttributeNotYetReady(apiError) {
		rs = d.retrySettings.Long
	}

	// It can take even longer until applications are ready to be used in synthetic tests and calculated metrics
	if isApplicationNotReadyYet(apiError, theApi) {
		rs = d.retrySettings.VeryLong
	}

	if rs.MaxRetries > 0 {
		return SendWithRetry(ctx, restCall, endpoint, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}, requestBody, rs)
	}

	return resp, err
}

func isGeneralDependencyNotReadyYet(apiError coreapi.APIError) bool {
	return strings.Contains(string(apiError.Body), "must have a unique name")
}

func isCalculatedMetricNotReadyYet(apiError coreapi.APIError) bool {
	body := string(apiError.Body)
	return strings.Contains(body, "Metric selector") &&
		strings.Contains(body, "invalid")
}

func isRequestAttributeNotYetReady(apiError coreapi.APIError) bool {
	body := string(apiError.Body)
	return strings.Contains(body, "must specify a known request attribute")
}

func isManagementZoneNotReadyYet(apiError coreapi.APIError) bool {
	body := string(apiError.Body)
	return strings.Contains(body, "Entity selector is invalid") ||
		(strings.Contains(body, "SLO validation failed") &&
			strings.Contains(body, "Management-Zone not found")) ||
		strings.Contains(body, "Unknown management zone")
}

func isApplicationNotReadyYet(apiError coreapi.APIError, theApi api.API) bool {
	body := string(apiError.Body)
	return isCalculatedMetricsError(apiError, theApi) ||
		isSyntheticMonitorServerError(apiError, theApi) ||
		isApplicationAPIError(apiError, theApi) ||
		isApplicationDetectionRuleException(apiError, theApi) ||
		isKeyUserActionMobile(theApi) ||
		isKeyUserActionWeb(theApi) ||
		isUserSessionPropertiesMobile(theApi) ||
		strings.Contains(body, "Unknown application(s)")
}
func isNetworkZoneFeatureNotEnabledYet(apiError coreapi.APIError, theApi api.API) bool {
	body := string(apiError.Body)
	return strings.HasPrefix(theApi.ID, "network-zone") && (apiError.Is4xxError()) && strings.Contains(body, "network zones are disabled")
}

func isCalculatedMetricsError(apiError coreapi.APIError, theApi api.API) bool {
	return strings.HasPrefix(theApi.ID, "calculated-metrics") && (apiError.Is4xxError() || apiError.Is5xxError())
}
func isSyntheticMonitorServerError(apiError coreapi.APIError, theApi api.API) bool {
	return theApi.ID == api.SyntheticMonitor && apiError.Is5xxError()
}

func isGeneralSyntheticAPIError(apiError coreapi.APIError, theApi api.API) bool {
	return (strings.HasPrefix(theApi.ID, "synthetic-") || theApi.ID == api.CredentialVault) && (apiError.StatusCode == http.StatusNotFound || apiError.Is5xxError())
}

func isApplicationAPIError(apiError coreapi.APIError, theApi api.API) bool {
	return isAnyApplicationApi(theApi) &&
		(apiError.Is5xxError() || apiError.StatusCode == http.StatusConflict || apiError.StatusCode == http.StatusNotFound)
}

// Special case (workaround):
// Sometimes, the API returns 500 Internal Server Error e.g. when an application referenced by
// an application detection rule is not fully "ready" yet.
func isApplicationDetectionRuleException(apiError coreapi.APIError, theApi api.API) bool {
	return theApi.ID == api.AppDetectionRule && (apiError.Is4xxError() || apiError.Is5xxError())
}

func isCredentialNotReadyYet(apiError coreapi.APIError) bool {
	body := string(apiError.Body)
	return strings.Contains(body, "credential-vault") &&
		strings.Contains(body, "was not available")
}

func joinUrl(urlBase string, path string) string {
	trimmedUrl := strings.TrimSuffix(urlBase, "/")

	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return trimmedUrl
	}

	return trimmedUrl + "/" + url.PathEscape(trimmedPath)
}

func getLocationFromHeader(resp coreapi.Response) ([]string, bool) {
	const locationHeader = "Location"
	if resp.Header[locationHeader] != nil {
		return resp.Header[locationHeader], true
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

func (d *ConfigClient) getExistingObjectId(ctx context.Context, objectName string, theApi api.API, payload []byte) (string, error) {
	var objID string
	// if there is a custom equal function registered, use that instead of just the Object name
	// in order to search for existing values
	if theApi.CheckEqualFunc != nil {
		values, err := d.List(ctx, theApi)
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
		values, err := d.List(ctx, theApi)
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

func (d *ConfigClient) List(ctx context.Context, theApi api.API) ([]Value, error) {
	// caching cannot be used for subPathAPI as well because there is potentially more than one config per api type/id to consider.
	// the cache cannot deal with that
	if (!theApi.NonUniqueName && !theApi.HasParent()) && // there is potentially more than one config per api type/id to consider
		(theApi.ID != api.ApplicationWeb && theApi.ID != api.ApplicationMobile) { // there is no refresh mechanism for delete; outdated values can cause decreasing performance during delete (unnecessary retrying)
		if values, cached := d.configCache.Get(theApi.ID); cached {
			return values, nil
		}
	}

	queryParams := getQueryParamsForNonStandardApis(theApi)

	// For any subpath API like e.g. Key user Actions, it can be that we need to do longer retries
	// because the parent config (e.g. an application) is not ready yet
	var retrySetting RetrySetting
	if theApi.HasParent() {
		retrySetting = d.retrySettings.Long
	} else {
		retrySetting = d.retrySettings.Normal
	}

	resp, err := GetWithRetry(ctx, *d.client, theApi.URLPath, corerest.RequestOptions{QueryParams: queryParams, CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}, retrySetting)
	if err != nil {
		return nil, err
	}

	var existingValues []Value
	for {
		values, err := unmarshalJson(ctx, theApi, resp.Data)
		if err != nil {
			return nil, err
		}

		existingValues = append(existingValues, values...)

		nextPageKey, _ := getPaginationValues(resp.Data)
		if nextPageKey == "" {
			break
		}

		resp, err = GetWithRetry(ctx, *d.client, theApi.URLPath, corerest.RequestOptions{QueryParams: makeQueryParamsWithNextPageKey(theApi.URLPath, queryParams, nextPageKey), CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}, retrySetting)
		if err != nil {

			apiError := coreapi.APIError{}
			if errors.As(err, &apiError) && apiError.StatusCode == http.StatusBadRequest {
				log.WithCtxFields(ctx).Warn("Failed to get additional data from paginated API %s - pages may have been removed during request.\n    Response was: %s", theApi.ID, string(apiError.Body))
				break
			}

			return nil, err
		}
	}
	d.configCache.Set(theApi.ID, existingValues)
	return existingValues, nil
}

func (d *ConfigClient) findUniqueByName(ctx context.Context, values []Value, objectName string) string {
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

func (d *ConfigClient) findUnique(ctx context.Context, values []Value, payload []byte, checkEqualFunc func(map[string]any, map[string]any) bool) (string, error) {
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
		log.WithCtxFields(ctx).Warn("Found %d configs with same name. Please delete duplicates.", matchingObjectsFound)
	}

	return objectId, nil
}

func getQueryParamsForNonStandardApis(theApi api.API) url.Values {
	queryParams := url.Values{}
	if theApi.ID == api.AnomalyDetectionMetrics {
		queryParams.Add("includeEntityFilterMetricEvents", "true")
	}
	if theApi.ID == api.Slo {
		queryParams.Add("enabledSlos", "all")
	}
	return queryParams
}

func unmarshalJson(ctx context.Context, theApi api.API, body []byte) ([]Value, error) {

	var values []Value
	var objmap map[string]interface{}

	// This API returns an untyped list as a response -> it needs a special handling
	if theApi.ID == api.AwsCredentials {
		var jsonResp []Value
		err := json.Unmarshal(body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing aws-credentials") {
			return values, err
		}
		values = jsonResp
	} else if theApi.ID == api.SyntheticLocation {
		var jsonResp SyntheticLocationResponse
		err := json.Unmarshal(body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
			return nil, err
		}
		values = translateSyntheticValues(jsonResp.Locations)
	} else if theApi.ID == api.SyntheticMonitor {
		var jsonResp SyntheticMonitorsResponse
		err := json.Unmarshal(body, &jsonResp)
		if errutils.CheckError(err, "Cannot unmarshal API response for existing synthetic location") {
			return nil, err
		}
		values = translateSyntheticValues(jsonResp.Monitors)
	} else if theApi.ID == api.KeyUserActionsMobile {
		var jsonResp KeyUserActionsMobileResponse
		err := json.Unmarshal(body, &jsonResp)
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
		err := json.Unmarshal(body, &jsonResp)
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
		err := json.Unmarshal(body, &jsonResp)
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
		if err := json.Unmarshal(body, &objmap); err != nil {
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
		err := json.Unmarshal(body, &jsonResponse)
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
