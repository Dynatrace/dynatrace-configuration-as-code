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

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"

	. "github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
)

// ConfigClient is responsible for the classic Dynatrace configs. For settings objects, the [SettingsClient] is responsible.
// Each config endpoint is described by an [Api] object to describe endpoints, structure, and behavior.
type ConfigClient interface {
	// List lists the available configs for an API.
	// It calls the underlying GET endpoint of the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	// The result is expressed using a list of Value (id and name tuples).
	List(a Api) (values []Value, err error)

	// ReadById reads a Dynatrace config identified by id from the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	ReadById(a Api, id string) (json []byte, err error)

	// UpsertByName creates a given Dynatrace config if it doesn't exist and updates it otherwise using its name.
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	UpsertByName(a Api, name string, payload []byte) (entity DynatraceEntity, err error)

	// UpsertByNonUniqueNameAndId creates a given Dynatrace config if it doesn't exist and updates it based on specific rules if it does not
	// - if only one config with the name exist, behave like any other type and just update this entity
	// - if an exact match is found (same name and same generated UUID) update that entity
	// - if several configs exist, but non match the generated UUID create a new entity with generated UUID
	// It calls the underlying GET and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//	 GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//	 PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... with the given (or found by unique name) entity ID
	UpsertByNonUniqueNameAndId(a Api, entityId string, name string, payload []byte) (entity DynatraceEntity, err error)

	// DeleteById removes a given config for a given API using its id.
	// It calls the DELETE endpoint for the API. E.g. for alerting profiles this would be:
	//    DELETE <environment-url>/api/config/v1/alertingProfiles/<id> ... to delete the config
	DeleteById(a Api, id string) error

	// ExistsByName checks if a config with the given name exists for the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	ExistsByName(a Api, name string) (exists bool, id string, err error)
}

// DownloadSettingsObject is the response type for the ListSettings operation
type DownloadSettingsObject struct {
	ExternalId    string          `json:"externalId"`
	SchemaVersion string          `json:"schemaVersion"`
	SchemaId      string          `json:"schemaId"`
	ObjectId      string          `json:"objectId"`
	Scope         string          `json:"scope"`
	Value         json.RawMessage `json:"value"`
}

// SettingsClient is the abstraction layer for CRUD operations on the Dynatrace Settings API.
// Its design is intentionally not dependent on Monaco objects.
//
// This interface exclusively accesses the [settings api] of Dynatrace.
//
// The base mechanism for all methods is the same:
// We identify objects to be updated/deleted by their external-id. If an object can not be found using its external-id, we assume
// that it does not exist.
// More documentation is written in each method's documentation.
//
// [settings api]: https://www.dynatrace.com/support/help/dynatrace-api/environment-api/settings
type SettingsClient interface {
	// UpsertSettings either creates the supplied object, or updates an existing one.
	// First, we try to find the external-id of the object. If we can't find it, we create the object, if we find it, we
	// update the object.
	UpsertSettings(SettingsObject) (DynatraceEntity, error)

	// ListSchemas returns all schemas that the Dynatrace environment reports
	ListSchemas() (SchemaList, error)

	// ListSettings returns all settings objects for a given schema.
	ListSettings(string, ListSettingsOptions) ([]DownloadSettingsObject, error)

	// DeleteSettings deletes a settings object giving its object ID
	DeleteSettings(string) error
}

// defaultListSettingsFields  are the fields we are interested in when getting setting objects
const defaultListSettingsFields = "objectId,value,externalId,schemaVersion,schemaId,scope"

// reducedListSettingsFields are the fields we are interested in when getting settings objects but don't care about the
// actual value payload
const reducedListSettingsFields = "objectId,externalId,schemaVersion,schemaId,scope"
const defaultPageSize = "500"
const defaultPageSizeEntities = "4000"

const defaultEntityDurationTimeframeFrom = -5 * 7 * 24 * time.Hour

// Not extracting the last 10 minutes to make sure what we extract is stable
// And avoid extracting more entities than the TotalCount from the first page of extraction
const defaultEntityDurationTimeframeTo = -10 * time.Minute

// ListSettingsOptions are additional options for the ListSettings method
// of the Settings client
type ListSettingsOptions struct {
	// DiscardValue specifies whether the value field of the returned
	// settings object shall be included in the payload
	DiscardValue bool
	// ListSettingsFilter can be set to pre-filter the result given a special logic
	Filter ListSettingsFilter
}

// ListSettingsFilter can be used to filter fetched settings objects with custom criteria, e.g. o.ExternalId == ""
type ListSettingsFilter func(DownloadSettingsObject) bool

// EntitiesClient is the abstraction layer for read-only operations on the Dynatrace Entities v2 API.
// Its design is intentionally not dependent on Monaco objects.
//
// This interface exclusively accesses the [entities api] of Dynatrace.
//
// More documentation is written in each method's documentation.
//
// [entities api]: https://www.dynatrace.com/support/help/dynatrace-api/environment-api/entity-v2
type EntitiesClient interface {

	// ListSchemas returns all schemas that the Dynatrace environment reports
	ListEntitiesTypes() ([]EntitiesType, error)

	// ListEntities returns all settings objects for a given schema.
	ListEntities(EntitiesType) ([]string, error)
}

//go:generate mockgen -source=client.go -destination=client_mock.go -package=client -imports .=github.com/dynatrace/dynatrace-configuration-as-code/pkg/api DynatraceClient

// Client provides the functionality for performing basic CRUD operations on any Dynatrace API
// supported by monaco.
// It encapsulates the configuration-specific inconsistencies of certain APIs in one place to provide
// a common interface to work with. After all: A user of Client shouldn't care about the
// implementation details of each individual Dynatrace API.
// Its design is intentionally not dependent on the Config and Environment interfaces included in monaco.
// This makes sure, that Client can be used as a base for future tooling, which relies on
// a standardized way to access Dynatrace APIs.
type Client interface {
	ConfigClient
	SettingsClient
	EntitiesClient
}

// DynatraceClient is the default implementation of the HTTP
// client targeting the relevant Dynatrace APIs for Monaco
type DynatraceClient struct {
	environmentUrl string
	token          string
	client         *http.Client
	retrySettings  rest.RetrySettings
}

var (
	_ EntitiesClient = (*DynatraceClient)(nil)
	_ SettingsClient = (*DynatraceClient)(nil)
	_ ConfigClient   = (*DynatraceClient)(nil)
	_ Client         = (*DynatraceClient)(nil)
)

// WithRetrySettings sets the retry settings to be used by the DynatraceClient
func WithRetrySettings(retrySettings rest.RetrySettings) func(*DynatraceClient) {
	return func(d *DynatraceClient) {
		d.retrySettings = retrySettings
	}
}

// WithHTTPClient sets the http client to be used by the DynatraceClient
func WithHTTPClient(client *http.Client) func(dynatraceClient *DynatraceClient) {
	return func(d *DynatraceClient) {
		d.client = client
	}
}

// NewDynatraceClient creates a new DynatraceClient
func NewDynatraceClient(environmentURL string, token string, opts ...func(dynatraceClient *DynatraceClient)) (*DynatraceClient, error) {
	environmentURL = strings.TrimSuffix(environmentURL, "/")

	if environmentURL == "" {
		return nil, errors.New("no environment url")
	}

	if token == "" {
		return nil, errors.New("no token")
	}

	parsedUrl, err := url.ParseRequestURI(environmentURL)
	if err != nil {
		return nil, errors.New("environment url " + environmentURL + " was not valid")
	}

	if parsedUrl.Scheme != "https" {
		return nil, errors.New("environment url " + environmentURL + " was not valid")
	}

	if !isNewDynatraceTokenFormat(token) {
		log.Warn("You used an old token format. Please consider switching to the new 1.205+ token format.")
		log.Warn("More information: https://www.dynatrace.com/support/help/dynatrace-api/basics/dynatrace-api-authentication/#-dynatrace-version-1205--token-format")
	}

	dtClient := &DynatraceClient{
		environmentUrl: environmentURL,
		token:          token,
		client:         &http.Client{},
		retrySettings:  rest.DefaultRetrySettings,
	}

	for _, o := range opts {
		o(dtClient)
	}

	return dtClient, nil
}

func isNewDynatraceTokenFormat(token string) bool {
	return strings.HasPrefix(token, "dt0c01.") && strings.Count(token, ".") == 2
}

func (d *DynatraceClient) UpsertSettings(obj SettingsObject) (DynatraceEntity, error) {
	externalId := util.GenerateExternalID(obj.SchemaId, obj.Id)

	// list all settings with matching external ID
	settings, err := d.ListSettings(obj.SchemaId, ListSettingsOptions{
		// we don't care about the actual value setting
		DiscardValue: true,
		// only consider objects with an external id that matches the expected one
		Filter: func(o DownloadSettingsObject) bool {
			return o.ExternalId == externalId
		},
	})
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to retrieve known settings for upsert operation: %w", err)
	}

	// should never happen
	if len(settings) > 1 {
		return DynatraceEntity{}, fmt.Errorf("failed to perform upsert operation: found more than one settings object with the same external id: %s. Settings objects: %v", externalId, settings)
	}

	if len(settings) == 0 {
		// special handling of this Settings object.
		// It is delete-protected BUT has a key property which is internally
		// used to find the object to be updated
		if obj.SchemaId == "builtin:oneagent.features" {
			externalId = ""
			obj.OriginObjectId = ""
		}
		payload, err := buildPostRequestPayload(obj, externalId)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to build settings object for upsert: %w", err)
		}

		requestUrl := d.environmentUrl + pathSettingsObjects

		resp, err := rest.SendWithRetryWithInitialTry(d.client, rest.Post, obj.Id, requestUrl, payload, d.token, d.retrySettings.Normal)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to upsert dynatrace obj: %w", err)
		}

		if !success(resp) {
			return DynatraceEntity{}, fmt.Errorf("failed to update settings object with externalId %s (HTTP %d)!\n\tResponse was: %s", externalId, resp.StatusCode, string(resp.Body))
		}

		entity, err := parsePostResponse(resp)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to parse response: %w", err)
		}

		log.Debug("\tCreated object %s (%s) with externalId %s", obj.Id, obj.SchemaId, externalId)
		return entity, nil
	} else {
		payload, err := buildPutRequestPayload(obj, externalId)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to build settings object for upsert: %w", err)
		}

		requestUrl := d.environmentUrl + pathSettingsObjects + "/" + settings[0].ObjectId

		resp, err := rest.SendWithRetryWithInitialTry(d.client, rest.Put, obj.Id, requestUrl, payload, d.token, d.retrySettings.Long)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to upsert dynatrace obj: %w", err)
		}

		if !success(resp) {
			return DynatraceEntity{}, fmt.Errorf("failed to update settings object with externalId %s (HTTP %d)!\n\tResponse was: %s", externalId, resp.StatusCode, string(resp.Body))
		}

		entity, err := parsePutResponse(resp)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to parse response: %w", err)
		}

		log.Debug("\tUpdated object %s (%s) with externalId %s", obj.Id, obj.SchemaId, externalId)
		return entity, nil
	}
}

func (d *DynatraceClient) List(api Api) (values []Value, err error) {

	fullUrl := api.GetUrl(d.environmentUrl)
	values, err = getExistingValuesFromEndpoint(d.client, api, fullUrl, d.token, d.retrySettings)
	return values, err
}

func (d *DynatraceClient) ReadById(api Api, id string) (json []byte, err error) {
	var dtUrl string
	isSingleConfigurationApi := api.IsSingleConfigurationApi()

	if isSingleConfigurationApi {
		dtUrl = api.GetUrl(d.environmentUrl)
	} else {
		dtUrl = api.GetUrl(d.environmentUrl) + "/" + url.PathEscape(id)
	}

	response, err := rest.Get(d.client, dtUrl, d.token)

	if err != nil {
		return nil, err
	}

	if !success(response) {
		return nil, fmt.Errorf("Failed to get existing config for api %v (HTTP %v)!\n    Response was: %v", api.GetId(), response.StatusCode, string(response.Body))
	}

	return response.Body, nil
}

func (d *DynatraceClient) DeleteById(api Api, id string) error {

	return rest.DeleteConfig(d.client, api.GetUrl(d.environmentUrl), d.token, id)
}

func (d *DynatraceClient) ExistsByName(api Api, name string) (exists bool, id string, err error) {
	apiURL := api.GetUrl(d.environmentUrl)
	existingObjectId, err := getObjectIdIfAlreadyExists(d.client, api, apiURL, name, d.token, d.retrySettings)
	return existingObjectId != "", existingObjectId, err
}

func (d *DynatraceClient) UpsertByName(api Api, name string, payload []byte) (entity DynatraceEntity, err error) {

	if api.GetId() == "extension" {
		fullUrl := api.GetUrl(d.environmentUrl)
		return uploadExtension(d.client, fullUrl, name, payload, d.token)
	}
	return upsertDynatraceObject(d.client, d.environmentUrl, name, api, payload, d.token, d.retrySettings)
}

func (d *DynatraceClient) UpsertByNonUniqueNameAndId(api Api, entityId string, name string, payload []byte) (entity DynatraceEntity, err error) {
	return upsertDynatraceEntityByNonUniqueNameAndId(d.client, d.environmentUrl, entityId, name, api, payload, d.token, d.retrySettings)
}

// SchemaListResponse is the response type returned by the ListSchemas operation
type SchemaListResponse struct {
	Items      SchemaList `json:"items"`
	TotalCount int        `json:"totalCount"`
}
type SchemaList []struct {
	SchemaId string `json:"schemaId"`
}

func (d *DynatraceClient) ListSchemas() (SchemaList, error) {
	u, err := url.Parse(d.environmentUrl + pathSchemas)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	// getting all schemas does not have pagination
	resp, err := rest.Get(d.client, u.String(), d.token)
	if err != nil {
		return nil, fmt.Errorf("failed to GET schemas: %w", err)
	}

	if !success(resp) {
		return nil, fmt.Errorf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body))
	}

	var result SchemaListResponse
	err = json.Unmarshal(resp.Body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.TotalCount != len(result.Items) {
		log.Warn("Total count of settings 2.0 schemas (=%d) does not match with count of actually downloaded settings 2.0 schemas (=%d)", result.TotalCount, len(result.Items))
	}

	return result.Items, nil
}

func (d *DynatraceClient) ListSettings(schemaId string, opts ListSettingsOptions) ([]DownloadSettingsObject, error) {
	log.Debug("Downloading all settings for schema %s", schemaId)

	listSettingsFields := defaultListSettingsFields
	if opts.DiscardValue {
		listSettingsFields = reducedListSettingsFields
	}
	params := url.Values{
		"schemaIds": []string{schemaId},
		"pageSize":  []string{defaultPageSize},
		"fields":    []string{listSettingsFields},
	}

	result := make([]DownloadSettingsObject, 0)

	addToResult := func(body []byte) (int, int, error) {
		var parsed struct {
			Items []DownloadSettingsObject `json:"items"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return 0, len(result), fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// eventually apply filter
		if opts.Filter == nil {
			result = append(result, parsed.Items...)
		} else {
			for _, i := range parsed.Items {
				if opts.Filter(i) {
					result = append(result, i)
				}
			}
		}

		return len(parsed.Items), len(result), nil
	}

	_, err := d.ListPaginated(pathSettingsObjects, params, schemaId, addToResult)

	if err != nil {
		return nil, err
	}

	return result, err
}

type EntitiesTypeListResponse struct {
	Types []EntitiesType `json:"types"`
}
type EntitiesType struct {
	EntitiesTypeId  string                   `json:"type"`
	ToRelationships []map[string]interface{} `json:"toRelationships"`
	Properties      []map[string]interface{} `json:"properties"`
}

func (d *DynatraceClient) ListEntitiesTypes() ([]EntitiesType, error) {

	params := url.Values{
		"pageSize": []string{defaultPageSize},
	}

	result := make([]EntitiesType, 0)

	addToResult := func(body []byte) (int, int, error) {
		var parsed EntitiesTypeListResponse

		if err1 := json.Unmarshal(body, &parsed); err1 != nil {
			return 0, len(result), fmt.Errorf("failed to unmarshal response: %w", err1)
		}

		result = append(result, parsed.Types...)

		return len(parsed.Types), len(result), nil
	}

	_, err := d.ListPaginated(pathEntitiesTypes, params, "EntityTypeList", addToResult)

	if err != nil {
		return nil, err
	}

	return result, err
}

type EntityListResponseRaw struct {
	Entities []json.RawMessage `json:"entities"`
}

func genTimeframeUnixMilliString(duration time.Duration) string {
	return strconv.FormatInt(time.Now().Add(duration).UnixMilli(), 10)
}

func (d *DynatraceClient) ListEntities(entitiesType EntitiesType) ([]string, error) {

	entityType := entitiesType.EntitiesTypeId
	log.Debug("Downloading all entities for entities Type %s", entityType)

	result := make([]string, 0)

	addToResult := func(body []byte) (int, int, error) {
		var parsedRaw EntityListResponseRaw

		if err1 := json.Unmarshal(body, &parsedRaw); err1 != nil {
			return 0, len(result), fmt.Errorf("failed to unmarshal response: %w", err1)
		}

		entitiesContentList := make([]string, len(parsedRaw.Entities))

		for idx, str := range parsedRaw.Entities {
			entitiesContentList[idx] = string(str)
		}

		result = append(result, entitiesContentList...)

		return len(parsedRaw.Entities), len(result), nil
	}

	runExtraction := true
	ignoreProperties := []string{}

	for runExtraction {
		params := GenListEntitiesParams(entityType, entitiesType, ignoreProperties)
		resp, err := d.ListPaginated(pathEntitiesObjects, params, entityType, addToResult)

		runExtraction, ignoreProperties, err = HandleListEntitiesError(entityType, resp, runExtraction, ignoreProperties, err)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (d *DynatraceClient) ListPaginated(urlPath string, params url.Values, logLabel string,
	addToResult func(body []byte) (int, int, error)) (rest.Response, error) {

	var resp rest.Response
	startTime := time.Now()
	receivedCount := 0
	totalReceivedCount := 0

	u, err := buildUrl(d.environmentUrl, urlPath, params)
	if err != nil {
		return resp, err
	}

	resp, receivedCount, totalReceivedCount, err = d.runAndProcessResponse(u, addToResult, receivedCount, totalReceivedCount, urlPath, logLabel)
	if err != nil {
		return resp, err
	}

	nbCalls := 1
	lastLogTime := time.Now()
	expectedTotalCount := resp.TotalCount
	nextPageKey := resp.NextPageKey
	emptyResponseRetryCount := 0

	for {

		if nextPageKey != "" {
			logLongRunningExtractionProgress(&lastLogTime, startTime, nbCalls, resp, logLabel)

			u = rest.AddNextPageQueryParams(u, nextPageKey)

			resp, receivedCount, totalReceivedCount, err = d.runAndProcessResponse(u, addToResult, receivedCount, totalReceivedCount, urlPath, logLabel)
			if err != nil {
				return resp, err
			}

			retry := false
			retry, emptyResponseRetryCount, err = isRetryOnEmptyResponse(receivedCount, emptyResponseRetryCount, resp)
			if err != nil {
				return resp, err
			}

			if retry {
				continue
			} else {
				validateWrongCountExtracted(resp, totalReceivedCount, expectedTotalCount, urlPath, logLabel, nextPageKey, params)

				nextPageKey = resp.NextPageKey
				nbCalls++
				emptyResponseRetryCount = 0
			}

		} else {

			break
		}
	}

	return resp, nil

}

func (d *DynatraceClient) DeleteSettings(objectID string) error {
	u, err := url.Parse(d.environmentUrl + pathSettingsObjects)
	if err != nil {
		return fmt.Errorf("failed to parse URL '%s': %w", d.environmentUrl+pathSettingsObjects, err)
	}

	return rest.DeleteConfig(d.client, u.String(), d.token, objectID)

}

const emptyResponseRetryMax = 10

func isRetryOnEmptyResponse(receivedCount int, emptyResponseRetryCount int, resp rest.Response) (bool, int, error) {
	if receivedCount == 0 {
		if emptyResponseRetryCount < emptyResponseRetryMax {
			emptyResponseRetryCount++
			rateLimitStrategy := rest.CreateRateLimitStrategy()
			rateLimitStrategy.ThrottleCallAfterError("Received empty array response, retrying with same nextPageKey (HTTP: %d) ", resp.StatusCode)
			return true, emptyResponseRetryCount, nil
		} else {
			return false, emptyResponseRetryCount, fmt.Errorf("received too many empty responses (=%d)", emptyResponseRetryCount)
		}
	}

	return false, emptyResponseRetryCount, nil
}

func (d *DynatraceClient) runAndProcessResponse(u *url.URL,
	addToResult func(body []byte) (int, int, error),
	receivedCount int, totalReceivedCount int, urlPath string, logLabel string) (rest.Response, int, int, error) {

	resp, err := rest.GetWithRetry(d.client, u.String(), d.token, d.retrySettings.Normal)
	err = validateRespErrors(err, resp, urlPath, logLabel)
	if err != nil {
		return resp, receivedCount, totalReceivedCount, err
	}

	receivedCount, totalReceivedCount, err = addToResult(resp.Body)

	return resp, receivedCount, totalReceivedCount, err
}

func buildUrl(environmentUrl, urlPath string, params url.Values) (*url.URL, error) {
	u, err := url.Parse(environmentUrl + urlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %w", environmentUrl+urlPath, err)
	}

	u.RawQuery = params.Encode()

	return u, nil
}

func validateRespErrors(err error, resp rest.Response, urlPath string, logLabel string) error {
	if err != nil {
		return err
	}

	if !success(resp) {
		return fmt.Errorf("Failed to get further data from api: %v for: %s (HTTP %d)!\n    response was: %s", urlPath, logLabel, resp.StatusCode, string(resp.Body))
	}
	return nil
}

func validateWrongCountExtracted(resp rest.Response, totalReceivedCount int, expectedTotalCount int, urlPath string, logLabel string, nextPageKey string, params url.Values) {
	if resp.NextPageKey == "" && totalReceivedCount != expectedTotalCount {
		log.Warn("Total count of items from api: %v for: %s does not match with count of actually downloaded items. Expected: %d Got: %d, last next page key received: %s \n   params: %v", urlPath, logLabel, expectedTotalCount, totalReceivedCount, nextPageKey, params)
	}
}

func logLongRunningExtractionProgress(lastLogTime *time.Time, startTime time.Time, nbCalls int, resp rest.Response, logLabel string) {
	if time.Since(*lastLogTime).Minutes() >= 1 {
		*lastLogTime = time.Now()
		nbItemsMessage := ""
		ETAMessage := ""
		runningMinutes := time.Since(startTime).Minutes()
		nbCallsPerMinute := (float64(nbCalls) / runningMinutes)
		if resp.PageSize > 0 && resp.TotalCount > 0 {
			nbProcessed := (nbCalls * resp.PageSize)
			nbLeft := resp.TotalCount - nbProcessed
			ETAMinutes := float64(nbLeft) / (nbCallsPerMinute * float64(resp.PageSize))
			nbItemsMessage = fmt.Sprintf(", processed %d of %d at %d items/call and", nbProcessed, resp.TotalCount, resp.PageSize)
			ETAMessage = fmt.Sprintf("ETA: %.1f minutes", (ETAMinutes))
		}

		log.Debug("Running extration of: %s for %.1f minutes%s %.1f call/minute. %s", logLabel, runningMinutes, nbItemsMessage, nbCallsPerMinute, ETAMessage)
	}
}
