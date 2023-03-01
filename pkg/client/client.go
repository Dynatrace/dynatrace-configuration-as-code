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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	version2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	. "github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
)

// ConfigClient is responsible for the classic Dynatrace configs. For settings objects, the [SettingsClient] is responsible.
// Each config endpoint is described by an [Api] object to describe endpoints, structure, and behavior.
type ConfigClient interface {
	// ListConfigs lists the available configs for an API.
	// It calls the underlying GET endpoint of the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	// The result is expressed using a list of Value (id and name tuples).
	ListConfigs(a Api) (values []Value, err error)

	// ReadConfigById reads a Dynatrace config identified by id from the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	ReadConfigById(a Api, id string) (json []byte, err error)

	// UpsertConfigByName creates a given Dynatrace config if it doesn't exist and updates it otherwise using its name.
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	UpsertConfigByName(a Api, name string, payload []byte) (entity DynatraceEntity, err error)

	// UpsertConfigByNonUniqueNameAndId creates a given Dynatrace config if it doesn't exist and updates it based on specific rules if it does not
	// - if only one config with the name exist, behave like any other type and just update this entity
	// - if an exact match is found (same name and same generated UUID) update that entity
	// - if several configs exist, but non match the generated UUID create a new entity with generated UUID
	// It calls the underlying GET and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//	 GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//	 PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... with the given (or found by unique name) entity ID
	UpsertConfigByNonUniqueNameAndId(a Api, entityId string, name string, payload []byte) (entity DynatraceEntity, err error)

	// DeleteConfigById removes a given config for a given API using its id.
	// It calls the DELETE endpoint for the API. E.g. for alerting profiles this would be:
	//    DELETE <environment-url>/api/config/v1/alertingProfiles/<id> ... to delete the config
	DeleteConfigById(a Api, id string) error

	// ConfigExistsByName checks if a config with the given name exists for the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	ConfigExistsByName(a Api, name string) (exists bool, id string, err error)
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

// ErrSettingNotFound is returned when no settings 2.0 object could be found
var ErrSettingNotFound = errors.New("settings object not found")

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

	// GetSettingById returns the setting with the given object ID
	GetSettingById(string) (*DownloadSettingsObject, error)

	// DeleteSettings deletes a settings object giving its object ID
	DeleteSettings(string) error
}

// defaultListSettingsFields  are the fields we are interested in when getting setting objects
const defaultListSettingsFields = "objectId,value,externalId,schemaVersion,schemaId,scope"
const defaultListEntitiesFields = "+lastSeenTms,+firstSeenTms,+tags,+managementZones,+toRelationships,+fromRelationships,+icon,+properties"

// reducedListSettingsFields are the fields we are interested in when getting settings objects but don't care about the
// actual value payload
const reducedListSettingsFields = "objectId,externalId,schemaVersion,schemaId,scope"
const defaultPageSize = "500"
const defaultEntityRelativeTimeframe = "now-5w"

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

	// ListEntitiesTypes returns all entities types
	ListEntitiesTypes() (EntitiesTypeList, error)

	// ListEntities returns all entities objects for a given type.
	ListEntities(string) ([]string, error)
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
	// serverVersion is the Dynatrace server version the
	// client will be interacting with
	serverVersion version.Version
	// environmentUrl is the base URL of the Dynatrace server the
	// client will be interacting with
	environmentUrl string
	// client is the HTTP client that will be used by the dynatrace client
	client *http.Client
	// retrySettings specify the retry behavior of the dynatrace client in case something goes wrong
	retrySettings rest.RetrySettings
}

// OauthCredentials holds information for authenticating to Dynatrace
// using Oauth2.0 client credential flow
type OauthCredentials struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
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

// WithServerVersion sets the Dynatrace version of the Dynatrace server/tenant the client will be interacting with
func WithServerVersion(serverVersion version.Version) func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		d.serverVersion = serverVersion
	}
}

// WithAutoServerVersion can be used to let the client automatically determine the Dynatrace server version
// during creation using NewDynatraceClient. If the server version is already known WithServerVersion should be used
func WithAutoServerVersion() func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		serverVersion, err := GetDynatraceVersion(d.client, d.environmentUrl)
		if err != nil {
			log.Error("Unable to determine Dynatrace server version: %v", err)
			d.serverVersion = version.UnknownVersion
		} else {
			d.serverVersion = serverVersion
		}
	}
}

// TokenAuthTransport should be used to enable a client
// to use dynatrace token authorization
type TokenAuthTransport struct {
	http.RoundTripper
	header http.Header
}

// NewTokenAuthTransport creates a new http transport to be used for
// token authorization
func NewTokenAuthTransport(baseTransport http.RoundTripper, token string) *TokenAuthTransport {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	t := &TokenAuthTransport{
		RoundTripper: baseTransport,
		header:       http.Header{},
	}
	t.setHeader("Authorization", "Api-Token "+token)
	t.setHeader("Content-type", "application/json")
	t.setHeader("User-Agent", "Dynatrace Monitoring as Code/"+version2.MonitoringAsCode+" "+(runtime.GOOS+" "+runtime.GOARCH))
	return t
}

func (t *TokenAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add the custom headers to the request
	for k, v := range t.header {
		req.Header[k] = v
	}
	return t.RoundTripper.RoundTrip(req)
}

func (t *TokenAuthTransport) setHeader(key, value string) {
	t.header.Set(key, value)
}

// NewTokenAuthClient creates a new HTTP client that supports token based authorization
func NewTokenAuthClient(token string) *http.Client {
	if !isNewDynatraceTokenFormat(token) {
		log.Warn("You used an old token format. Please consider switching to the new 1.205+ token format.")
		log.Warn("More information: https://www.dynatrace.com/support/help/dynatrace-api/basics/dynatrace-api-authentication")
	}
	return &http.Client{Transport: NewTokenAuthTransport(nil, token)}
}

// NewOAuthClient creates a new HTTP client that supports OAuth2 client credentials based authorization
func NewOAuthClient(oauthConfig OauthCredentials) *http.Client {
	config := clientcredentials.Config{
		ClientID:     oauthConfig.ClientID,
		ClientSecret: oauthConfig.ClientSecret,
		TokenURL:     oauthConfig.TokenURL,
		Scopes:       oauthConfig.Scopes,
	}
	return config.Client(context.TODO())
}

// NewDynatraceClient creates a new DynatraceClient.
// It takes a http.Client to do its HTTP communication, a URL to the targeting Dynatrace
// environment and an optional list of options to further configure the behavior of the client
// It is also allowed to pass nil as httpClient
func NewDynatraceClient(httpClient *http.Client, environmentURL string, opts ...func(dynatraceClient *DynatraceClient)) (*DynatraceClient, error) {
	environmentURL = strings.TrimSuffix(environmentURL, "/")
	parsedUrl, err := url.ParseRequestURI(environmentURL)
	if err != nil {
		return nil, fmt.Errorf("environment url %q was not valid: %w", environmentURL, err)
	}

	if parsedUrl.Host == "" {
		return nil, fmt.Errorf("no host specified in the url %q", environmentURL)
	}

	if parsedUrl.Scheme != "https" {
		log.Warn("You are using an insecure connection (%s). Consider switching to HTTPS.", parsedUrl.Scheme)
	}

	if httpClient == nil {
		httpClient = &http.Client{}
	}

	dtClient := &DynatraceClient{
		environmentUrl: environmentURL,
		client:         httpClient,
		retrySettings:  rest.DefaultRetrySettings,
		serverVersion:  version.Version{},
	}

	for _, o := range opts {
		o(dtClient)
	}

	if dtClient.serverVersion.Invalid() {
		log.Warn("Dynatrace Client was created without specifying a version of the targeting Dynatrace server. This can result in faulty behavior.")
	}

	return dtClient, nil
}

func isNewDynatraceTokenFormat(token string) bool {
	return strings.HasPrefix(token, "dt0c01.") && strings.Count(token, ".") == 2
}

func (d *DynatraceClient) UpsertSettings(obj SettingsObject) (DynatraceEntity, error) {

	// special handling for updating settings 2.0 objects on tenants with version pre 1.262.0
	// Tenants with versions < 1.262 are not able to handle updates of existing
	// settings 2.0 objects that are non-deletable.
	// So we check if the object with originObjectID already exists, if yes and the tenant is older than 1.262
	// then we cannot perform the upsert operation
	if !d.serverVersion.Invalid() && d.serverVersion.SmallerThan(version.Version{Major: 1, Minor: 262, Patch: 0}) {
		fetchedSettingObj, err := d.GetSettingById(obj.OriginObjectId)
		if err != nil && !errors.Is(err, ErrSettingNotFound) {
			return DynatraceEntity{}, fmt.Errorf("unable to fetch settings object with object id %q: %w", obj.OriginObjectId, err)
		}
		if fetchedSettingObj != nil {
			log.Warn("Unable to update Settings 2.0 object of schema %q and object id %q on Dynatrace environment with a version < 1.262.0", obj.SchemaId, obj.OriginObjectId)
			return DynatraceEntity{
				Id:   fetchedSettingObj.ObjectId,
				Name: fetchedSettingObj.ObjectId,
			}, nil
		}
	}

	externalId := idutils.GenerateExternalID(obj.SchemaId, obj.Id)
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

	resp, err := rest.SendWithRetryWithInitialTry(d.client, rest.Post, obj.Id, requestUrl, payload, d.retrySettings.Normal)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to upsert dynatrace obj: %w", err)
	}

	if !success(resp) {
		return DynatraceEntity{}, fmt.Errorf("failed to upsert settings object with externalId %s (HTTP %d)!\n\tResponse was: %s", externalId, resp.StatusCode, string(resp.Body))
	}

	entity, err := parsePostResponse(resp)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Debug("\tUpserted object %s (%s) with externalId %s", obj.Id, obj.SchemaId, externalId)
	return entity, nil
}

func (d *DynatraceClient) ListConfigs(api Api) (values []Value, err error) {

	fullUrl := api.GetUrl(d.environmentUrl)
	values, err = getExistingValuesFromEndpoint(d.client, api, fullUrl, d.retrySettings)
	return values, err
}

func (d *DynatraceClient) ReadConfigById(api Api, id string) (json []byte, err error) {
	var dtUrl string
	isSingleConfigurationApi := api.IsSingleConfigurationApi()

	if isSingleConfigurationApi {
		dtUrl = api.GetUrl(d.environmentUrl)
	} else {
		dtUrl = api.GetUrl(d.environmentUrl) + "/" + url.PathEscape(id)
	}

	response, err := rest.Get(d.client, dtUrl)

	if err != nil {
		return nil, err
	}

	if !success(response) {
		return nil, fmt.Errorf("Failed to get existing config for api %v (HTTP %v)!\n    Response was: %v", api.GetId(), response.StatusCode, string(response.Body))
	}

	return response.Body, nil
}

func (d *DynatraceClient) DeleteConfigById(api Api, id string) error {

	return rest.DeleteConfig(d.client, api.GetUrl(d.environmentUrl), id)
}

func (d *DynatraceClient) ConfigExistsByName(api Api, name string) (exists bool, id string, err error) {
	apiURL := api.GetUrl(d.environmentUrl)
	existingObjectId, err := getObjectIdIfAlreadyExists(d.client, api, apiURL, name, d.retrySettings)
	return existingObjectId != "", existingObjectId, err
}

func (d *DynatraceClient) UpsertConfigByName(api Api, name string, payload []byte) (entity DynatraceEntity, err error) {

	if api.GetId() == "extension" {
		fullUrl := api.GetUrl(d.environmentUrl)
		return uploadExtension(d.client, fullUrl, name, payload)
	}
	return upsertDynatraceObject(d.client, d.environmentUrl, name, api, payload, d.retrySettings)
}

func (d *DynatraceClient) UpsertConfigByNonUniqueNameAndId(api Api, entityId string, name string, payload []byte) (entity DynatraceEntity, err error) {
	return upsertDynatraceEntityByNonUniqueNameAndId(d.client, d.environmentUrl, entityId, name, api, payload, d.retrySettings)
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
	resp, err := rest.Get(d.client, u.String())
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

func (d *DynatraceClient) GetSettingById(objectId string) (*DownloadSettingsObject, error) {
	u, err := url.Parse(d.environmentUrl + pathSettingsObjects + "/" + objectId)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %w", d.environmentUrl+pathSettingsObjects, err)
	}

	resp, err := rest.Get(d.client, u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to GET settings object with object id %q: %w", objectId, err)
	}

	if !success(resp) {
		// special case of settings API: If you give it any object ID for a setting object that does not exist, it will give back 400 BadRequest instead
		// of 404 Not found. So we interpret both status codes, 400 and 404, as "not found"
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
			return nil, ErrSettingNotFound
		}
		return nil, fmt.Errorf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body))
	}

	var result DownloadSettingsObject
	if err = json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (d *DynatraceClient) ListSettings(schemaId string, opts ListSettingsOptions) ([]DownloadSettingsObject, error) {

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

	addToResult := func(body []byte) error {
		var parsed struct {
			Items []DownloadSettingsObject `json:"items"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
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

		return nil
	}

	err := d.ListPaginated(pathSettingsObjects, params, addToResult)

	if err != nil {
		return nil, err
	}

	return result, err
}

type EntitiesTypeListResponse struct {
	Types EntitiesTypeList `json:"types"`
}
type EntitiesTypeList []struct {
	EntitiesType string `json:"type"`
}

func (d *DynatraceClient) ListEntitiesTypes() (EntitiesTypeList, error) {

	params := url.Values{
		"pageSize": []string{defaultPageSize},
	}

	result := make(EntitiesTypeList, 0)

	addToResult := func(body []byte) error {
		var parsed EntitiesTypeListResponse

		if err1 := json.Unmarshal(body, &parsed); err1 != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err1)
		}

		result = append(result, parsed.Types...)

		return nil
	}

	err := d.ListPaginated(pathEntitiesTypes, params, addToResult)

	if err != nil {
		return nil, err
	}

	return result, err
}

type EntityListResponseRaw struct {
	Entities []json.RawMessage `json:"entities"`
}

func (d *DynatraceClient) ListEntities(entityType string) ([]string, error) {

	params := url.Values{
		"entitySelector": []string{"type(\"" + entityType + "\")"},
		"pageSize":       []string{defaultPageSize},
		"fields":         []string{defaultListEntitiesFields},
		"from":           []string{defaultEntityRelativeTimeframe},
	}

	result := make([]string, 0)

	addToResult := func(body []byte) error {
		var parsedRaw EntityListResponseRaw

		if err1 := json.Unmarshal(body, &parsedRaw); err1 != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err1)
		}

		entitiesContentList := make([]string, len(parsedRaw.Entities))

		for idx, str := range parsedRaw.Entities {
			entitiesContentList[idx] = string(str)
		}

		result = append(result, entitiesContentList...)

		return nil
	}

	err := d.ListPaginated(pathEntitiesObjects, params, addToResult)

	if err != nil {
		return nil, err
	}

	return result, err
}

func (d *DynatraceClient) ListPaginated(urlPath string, params url.Values, addToResult func(body []byte) error) error {
	u, err := url.Parse(d.environmentUrl + urlPath)
	if err != nil {
		return fmt.Errorf("failed to parse URL '%s': %w", d.environmentUrl+urlPath, err)
	}

	u.RawQuery = params.Encode()

	resp, err := rest.GetWithRetry(d.client, u.String(), d.retrySettings.Normal)
	if err != nil {
		return err
	}

	if !success(resp) {
		return fmt.Errorf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body))
	}

	for {
		err = addToResult(resp.Body)
		if err != nil {
			return err
		}

		if resp.NextPageKey != "" {
			u = rest.AddNextPageQueryParams(u, resp.NextPageKey)

			resp, err = rest.GetWithRetry(d.client, u.String(), d.retrySettings.Normal)

			if err != nil {
				return err
			}

			if !success(resp) && resp.StatusCode != http.StatusBadRequest {
				return fmt.Errorf("Failed to get further data from paginated API %s (HTTP %d)!\n    Response was: %s", urlPath, resp.StatusCode, string(resp.Body))
			} else if resp.StatusCode == http.StatusBadRequest {
				log.Warn("Failed to get additional data from paginated API %s - pages may have been removed during request.\n    Response was: %s", urlPath, string(resp.Body))
				break
			}

		} else {
			break
		}
	}

	return nil

}

func (d *DynatraceClient) DeleteSettings(objectID string) error {
	u, err := url.Parse(d.environmentUrl + pathSettingsObjects)
	if err != nil {
		return fmt.Errorf("failed to parse URL '%s': %w", d.environmentUrl+pathSettingsObjects, err)
	}

	return rest.DeleteConfig(d.client, u.String(), objectID)

}
