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

package dtclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/throttle"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ConfigClient is responsible for the classic Dynatrace configs. For settings objects, the [SettingsClient] is responsible.
// Each config endpoint is described by an [API] object to describe endpoints, structure, and behavior.
type ConfigClient interface {
	// ListConfigs lists the available configs for an API.
	// It calls the underlying GET endpoint of the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	// The result is expressed using a list of Value (id and name tuples).
	ListConfigs(a api.API) (values []Value, err error)

	// ReadConfigById reads a Dynatrace config identified by id from the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	ReadConfigById(a api.API, id string) (json []byte, err error)

	// UpsertConfigByName creates a given Dynatrace config if it doesn't exist and updates it otherwise using its name.
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	UpsertConfigByName(a api.API, name string, payload []byte) (entity DynatraceEntity, err error)

	// UpsertConfigByNonUniqueNameAndId creates a given Dynatrace config if it doesn't exist and updates it based on specific rules if it does not
	// - if only one config with the name exist, behave like any other type and just update this entity
	// - if an exact match is found (same name and same generated UUID) update that entity
	// - if several configs exist, but non match the generated UUID create a new entity with generated UUID
	// It calls the underlying GET and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//	 GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//	 PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... with the given (or found by unique name) entity ID
	UpsertConfigByNonUniqueNameAndId(a api.API, entityID string, name string, payload []byte) (entity DynatraceEntity, err error)

	// DeleteConfigById removes a given config for a given API using its id.
	// It calls the DELETE endpoint for the API. E.g. for alerting profiles this would be:
	//    DELETE <environment-url>/api/config/v1/alertingProfiles/<id> ... to delete the config
	DeleteConfigById(a api.API, id string) error

	// ConfigExistsByName checks if a config with the given name exists for the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	ConfigExistsByName(a api.API, name string) (exists bool, id string, err error)
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

	// ListEntitiesTypes returns all entities types
	ListEntitiesTypes() ([]EntitiesType, error)

	// ListEntities returns all entities objects for a given type.
	ListEntities(EntitiesType) ([]string, error)
}

//go:generate mockgen -source=client.go -destination=client_mock.go -package=dtclient DynatraceClient

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
	// environmentURL is the base URL of the Dynatrace server the
	// client will be interacting with
	environmentURL string
	// environmentURLClassic is the base URL of the classic generation
	// Dynatrace tenant
	environmentURLClassic string
	// client is the underlying HTTP client that will be used to communicate
	// with platform gen environments
	client *http.Client

	// client is the underlying HTTP client that will be used to communicate
	// with second gen environments
	clientClassic *http.Client
	// retrySettings specify the retry behavior of the dynatrace client in case something goes wrong

	// settingsSchemaAPIPath is the API path to use for accessing settings schemas
	settingsSchemaAPIPath string

	//  settingsObjectAPIPath is the API path to use for accessing settings objects
	settingsObjectAPIPath string

	// retrySettings are the settings to be used for retrying failed http requests
	retrySettings rest.RetrySettings

	// limiter is used to limit parallel http requests
	limiter *concurrency.Limiter
}

var (
	_ EntitiesClient = (*DynatraceClient)(nil)
	_ SettingsClient = (*DynatraceClient)(nil)
	_ ConfigClient   = (*DynatraceClient)(nil)
	_ Client         = (*DynatraceClient)(nil)
)

// WithClientRequestLimiter specifies that a specifies the limiter to be used for
// limiting parallel client requests
func WithClientRequestLimiter(limiter *concurrency.Limiter) func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		d.limiter = limiter
	}
}

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

// WithClient sets the underlying http client to a specific one.
// Useful for testing
func WithClient(client *http.Client) func(d *DynatraceClient) {
	return func(d *DynatraceClient) {
		d.client = client
		d.clientClassic = client
	}
}

// WithAutoServerVersion can be used to let the client automatically determine the Dynatrace server version
// during creation using newDynatraceClient. If the server version is already known WithServerVersion should be used
func WithAutoServerVersion() func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		var serverVersion version.Version
		var err error
		if _, ok := d.client.Transport.(*oauth2.Transport); ok {
			// for platform enabled tenants there is no dedicated version endpoint
			// so this call would need to be "redirected" to the second gen URL, which do not currently resolve
			d.serverVersion = version.UnknownVersion
		} else {
			serverVersion, err = client.GetDynatraceVersion(d.clientClassic, d.environmentURLClassic)
		}
		if err != nil {
			log.Warn("Unable to determine Dynatrace server version: %v", err)
			d.serverVersion = version.UnknownVersion
		} else {
			d.serverVersion = serverVersion
		}
	}
}

const (
	settingsSchemaAPIPathClassic  = "/api/v2/settings/schemas"
	settingsSchemaAPIPathPlatform = "/platform/classic/environment-api/v2/settings/schemas"
	settingsObjectAPIPathClassic  = "/api/v2/settings/objects"
	settingsObjectAPIPathPlatform = "/platform/classic/environment-api/v2/settings/objects"
)

// NewPlatformClient creates a new dynatrace client to be used for platform enabled environments
func NewPlatformClient(dtURL string, token string, oauthCredentials client.OauthCredentials, opts ...func(dynatraceClient *DynatraceClient)) (*DynatraceClient, error) {
	dtURL = strings.TrimSuffix(dtURL, "/")
	if err := validateURL(dtURL); err != nil {
		return nil, err
	}

	tokenClient := client.NewTokenAuthClient(token)
	oauthClient := client.NewOAuthClient(context.TODO(), oauthCredentials)

	classicURL, err := client.GetDynatraceClassicURL(oauthClient, dtURL)
	if err != nil {
		log.Error("Unable to determine Dynatrace classic environment URL: %v", err)
		return nil, err
	}

	d := &DynatraceClient{
		serverVersion:         version.Version{},
		environmentURL:        dtURL,
		environmentURLClassic: classicURL,
		client:                oauthClient,
		clientClassic:         tokenClient,
		retrySettings:         rest.DefaultRetrySettings,
		settingsSchemaAPIPath: settingsSchemaAPIPathPlatform,
		settingsObjectAPIPath: settingsObjectAPIPathPlatform,
		limiter:               concurrency.NewLimiter(5),
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

// NewClassicClient creates a new dynatrace client to be used for classic environments
func NewClassicClient(dtURL string, token string, opts ...func(dynatraceClient *DynatraceClient)) (*DynatraceClient, error) {
	dtURL = strings.TrimSuffix(dtURL, "/")
	if err := validateURL(dtURL); err != nil {
		return nil, err
	}

	tokenClient := client.NewTokenAuthClient(token)

	d := &DynatraceClient{
		serverVersion:         version.Version{},
		environmentURL:        dtURL,
		environmentURLClassic: dtURL,
		client:                tokenClient,
		clientClassic:         tokenClient,
		retrySettings:         rest.DefaultRetrySettings,
		settingsSchemaAPIPath: settingsSchemaAPIPathClassic,
		settingsObjectAPIPath: settingsObjectAPIPathClassic,
		limiter:               concurrency.NewLimiter(5),
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

func validateURL(dtURL string) error {
	parsedUrl, err := url.ParseRequestURI(dtURL)
	if err != nil {
		return fmt.Errorf("environment url %q was not valid: %w", dtURL, err)
	}

	if parsedUrl.Host == "" {
		return fmt.Errorf("no host specified in the url %q", dtURL)
	}

	if parsedUrl.Scheme != "https" {
		log.Warn("You are using an insecure connection (%s). Consider switching to HTTPS.", parsedUrl.Scheme)
	}
	return nil
}

func (d *DynatraceClient) UpsertSettings(obj SettingsObject) (result DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		result, err = d.upsertSettings(obj)
	})
	return
}

func (d *DynatraceClient) upsertSettings(obj SettingsObject) (DynatraceEntity, error) {

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
		return DynatraceEntity{}, fmt.Errorf("failed to build settings object: %w", err)
	}

	requestUrl := d.environmentURL + d.settingsObjectAPIPath

	resp, err := rest.SendWithRetryWithInitialTry(d.client, rest.Post, obj.Id, requestUrl, payload, d.retrySettings.Normal)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to create or update dynatrace obj: %w", err)
	}

	if !success(resp) {
		return DynatraceEntity{}, fmt.Errorf("failed to create or update settings object with externalId %s (HTTP %d)!\n\tResponse was: %s", externalId, resp.StatusCode, string(resp.Body))
	}

	entity, err := parsePostResponse(resp)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Debug("\tCreated/Updated object %s (%s) with externalId %s", obj.Id, obj.SchemaId, externalId)
	return entity, nil

}
func (d *DynatraceClient) ListConfigs(api api.API) (values []Value, err error) {
	d.limiter.ExecuteBlocking(func() {
		values, err = d.listConfigs(api)
	})
	return
}
func (d *DynatraceClient) listConfigs(api api.API) (values []Value, err error) {

	fullUrl := api.CreateURL(d.environmentURLClassic)
	values, err = getExistingValuesFromEndpoint(d.clientClassic, api, fullUrl, d.retrySettings)
	return values, err
}

func (d *DynatraceClient) ReadConfigById(api api.API, id string) (json []byte, err error) {
	d.limiter.ExecuteBlocking(func() {
		json, err = d.readConfigById(api, id)
	})
	return
}

func (d *DynatraceClient) readConfigById(api api.API, id string) (json []byte, err error) {
	var dtUrl string
	isSingleConfigurationApi := api.SingleConfiguration

	if isSingleConfigurationApi {
		dtUrl = api.CreateURL(d.environmentURLClassic)
	} else {
		dtUrl = api.CreateURL(d.environmentURLClassic) + "/" + url.PathEscape(id)
	}

	response, err := rest.Get(d.clientClassic, dtUrl)

	if err != nil {
		return nil, err
	}

	if !success(response) {
		return nil, fmt.Errorf("failed to get existing config for api %v (HTTP %v)!\n    Response was: %v", api.ID, response.StatusCode, string(response.Body))
	}

	return response.Body, nil
}

func (d *DynatraceClient) DeleteConfigById(api api.API, id string) (err error) {
	d.limiter.ExecuteBlocking(func() {
		err = d.deleteConfigById(api, id)
	})
	return
}

func (d *DynatraceClient) deleteConfigById(api api.API, id string) error {

	return rest.DeleteConfig(d.clientClassic, api.CreateURL(d.environmentURLClassic), id)
}

func (d *DynatraceClient) ConfigExistsByName(api api.API, name string) (exists bool, id string, err error) {
	d.limiter.ExecuteBlocking(func() {
		exists, id, err = d.configExistsByName(api, name)
	})
	return
}

func (d *DynatraceClient) configExistsByName(api api.API, name string) (exists bool, id string, err error) {
	apiURL := api.CreateURL(d.environmentURLClassic)
	existingObjectId, err := getObjectIdIfAlreadyExists(d.clientClassic, api, apiURL, name, d.retrySettings)
	return existingObjectId != "", existingObjectId, err
}

func (d *DynatraceClient) UpsertConfigByName(api api.API, name string, payload []byte) (entity DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		entity, err = d.upsertConfigByName(api, name, payload)
	})
	return
}

func (d *DynatraceClient) upsertConfigByName(api api.API, name string, payload []byte) (entity DynatraceEntity, err error) {

	if api.ID == "extension" {
		fullUrl := api.CreateURL(d.environmentURLClassic)
		return uploadExtension(d.clientClassic, fullUrl, name, payload)
	}
	return upsertDynatraceObject(d.clientClassic, d.environmentURLClassic, name, api, payload, d.retrySettings)
}

func (d *DynatraceClient) UpsertConfigByNonUniqueNameAndId(api api.API, entityId string, name string, payload []byte) (entity DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		entity, err = d.upsertConfigByNonUniqueNameAndId(api, entityId, name, payload)
	})
	return
}

func (d *DynatraceClient) upsertConfigByNonUniqueNameAndId(api api.API, entityId string, name string, payload []byte) (entity DynatraceEntity, err error) {
	return upsertDynatraceEntityByNonUniqueNameAndId(d.clientClassic, d.environmentURLClassic, entityId, name, api, payload, d.retrySettings)
}

// SchemaListResponse is the response type returned by the ListSchemas operation
type SchemaListResponse struct {
	Items      SchemaList `json:"items"`
	TotalCount int        `json:"totalCount"`
}
type SchemaList []struct {
	SchemaId string `json:"schemaId"`
}

func (d *DynatraceClient) ListSchemas() (schemas SchemaList, err error) {
	d.limiter.ExecuteBlocking(func() {
		schemas, err = d.listSchemas()
	})
	return
}

func (d *DynatraceClient) listSchemas() (SchemaList, error) {
	u, err := url.Parse(d.environmentURL + d.settingsSchemaAPIPath)
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

func (d *DynatraceClient) GetSettingById(objectId string) (res *DownloadSettingsObject, err error) {
	d.limiter.ExecuteBlocking(func() {
		res, err = d.getSettingById(objectId)
	})
	return
}

func (d *DynatraceClient) getSettingById(objectId string) (*DownloadSettingsObject, error) {
	u, err := url.Parse(d.environmentURL + d.settingsObjectAPIPath + "/" + objectId)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %w", d.environmentURL+d.settingsObjectAPIPath, err)
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

func (d *DynatraceClient) ListSettings(schemaId string, opts ListSettingsOptions) (res []DownloadSettingsObject, err error) {
	d.limiter.ExecuteBlocking(func() {
		res, err = d.listSettings(schemaId, opts)
	})
	return
}
func (d *DynatraceClient) listSettings(schemaId string, opts ListSettingsOptions) ([]DownloadSettingsObject, error) {
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

	_, err := d.listPaginated(d.settingsObjectAPIPath, params, schemaId, addToResult)

	if err != nil {
		return nil, err
	}

	return result, nil
}

type EntitiesTypeListResponse struct {
	Types []EntitiesType `json:"types"`
}

type EntitiesType struct {
	EntitiesTypeId  string                   `json:"type"`
	ToRelationships []map[string]interface{} `json:"toRelationships"`
	Properties      []map[string]interface{} `json:"properties"`
}

func (e EntitiesType) String() string {
	return e.EntitiesTypeId
}

func (d *DynatraceClient) ListEntitiesTypes() (res []EntitiesType, err error) {
	d.limiter.ExecuteBlocking(func() {
		res, err = d.listEntitiesTypes()
	})
	return
}

func (d *DynatraceClient) listEntitiesTypes() ([]EntitiesType, error) {

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

	_, err := d.listPaginated(pathEntitiesTypes, params, "EntityTypeList", addToResult)

	if err != nil {
		return nil, err
	}

	return result, nil
}

type EntityListResponseRaw struct {
	Entities []json.RawMessage `json:"entities"`
}

func genTimeframeUnixMilliString(duration time.Duration) string {
	return strconv.FormatInt(time.Now().Add(duration).UnixMilli(), 10)
}

func (d *DynatraceClient) ListEntities(entitiesType EntitiesType) (res []string, err error) {
	d.limiter.ExecuteBlocking(func() {
		res, err = d.listEntities(entitiesType)
	})
	return
}

func (d *DynatraceClient) listEntities(entitiesType EntitiesType) ([]string, error) {

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
	var ignoreProperties []string

	for runExtraction {
		params := genListEntitiesParams(entityType, entitiesType, ignoreProperties)
		resp, err := d.listPaginated(pathEntitiesObjects, params, entityType, addToResult)

		runExtraction, ignoreProperties, err = handleListEntitiesError(entityType, resp, runExtraction, ignoreProperties, err)

		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (d *DynatraceClient) listPaginated(urlPath string, params url.Values, logLabel string,
	addToResult func(body []byte) (int, int, error)) (rest.Response, error) {

	var resp rest.Response
	startTime := time.Now()
	receivedCount := 0
	totalReceivedCount := 0

	u, err := buildUrl(d.environmentURL, urlPath, params)
	if err != nil {
		return resp, client.RespError{
			Err:        err,
			StatusCode: 0,
		}
	}

	resp, receivedCount, totalReceivedCount, _, err = d.runAndProcessResponse(false, u, addToResult, receivedCount, totalReceivedCount, urlPath)
	if err != nil {
		return resp, client.RespError{
			Err:        err,
			StatusCode: resp.StatusCode,
		}
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

			var isLastAvailablePage bool
			resp, receivedCount, totalReceivedCount, isLastAvailablePage, err = d.runAndProcessResponse(true, u, addToResult, receivedCount, totalReceivedCount, urlPath)
			if err != nil {
				return resp, client.RespError{
					Err:        err,
					StatusCode: resp.StatusCode,
				}
			}
			if isLastAvailablePage {
				break
			}

			retry := false
			retry, emptyResponseRetryCount, err = isRetryOnEmptyResponse(receivedCount, emptyResponseRetryCount, resp)
			if err != nil {
				return resp, client.RespError{
					Err:        err,
					StatusCode: resp.StatusCode,
				}
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

func (d *DynatraceClient) DeleteSettings(objectID string) (err error) {
	d.limiter.ExecuteBlocking(func() {
		err = d.deleteSettings(objectID)
	})
	return
}

func (d *DynatraceClient) deleteSettings(objectID string) error {
	u, err := url.Parse(d.environmentURL + d.settingsObjectAPIPath)
	if err != nil {
		return fmt.Errorf("failed to parse URL '%s': %w", d.environmentURL+d.settingsObjectAPIPath, err)
	}

	return rest.DeleteConfig(d.client, u.String(), objectID)

}

const emptyResponseRetryMax = 10

func isRetryOnEmptyResponse(receivedCount int, emptyResponseRetryCount int, resp rest.Response) (bool, int, error) {
	if receivedCount == 0 {
		if emptyResponseRetryCount < emptyResponseRetryMax {
			emptyResponseRetryCount++
			throttle.ThrottleCallAfterError(emptyResponseRetryCount, "Received empty array response, retrying with same nextPageKey (HTTP: %d) ", resp.StatusCode)
			return true, emptyResponseRetryCount, nil
		} else {
			return false, emptyResponseRetryCount, fmt.Errorf("received too many empty responses (=%d)", emptyResponseRetryCount)
		}
	}

	return false, emptyResponseRetryCount, nil
}

func (d *DynatraceClient) runAndProcessResponse(isNextCall bool, u *url.URL,
	addToResult func(body []byte) (int, int, error),
	receivedCount int, totalReceivedCount int, urlPath string) (rest.Response, int, int, bool, error) {
	isLastAvailablePage := false

	resp, err := rest.GetWithRetry(d.client, u.String(), d.retrySettings.Normal)
	isLastAvailablePage, err = validateRespErrors(isNextCall, err, resp, urlPath)
	if err != nil || isLastAvailablePage {
		return resp, receivedCount, totalReceivedCount, isLastAvailablePage, err
	}

	receivedCount, totalReceivedCount, err = addToResult(resp.Body)

	return resp, receivedCount, totalReceivedCount, isLastAvailablePage, err
}

func buildUrl(environmentUrl, urlPath string, params url.Values) (*url.URL, error) {
	u, err := url.Parse(environmentUrl + urlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %w", environmentUrl+urlPath, err)
	}

	u.RawQuery = params.Encode()

	return u, nil
}

func validateRespErrors(isNextCall bool, err error, resp rest.Response, urlPath string) (bool, error) {
	if err != nil {
		return false, err
	}
	isLastAvailablePage := false
	if success(resp) {
		return false, nil

	} else if isNextCall {
		if resp.StatusCode == http.StatusBadRequest {
			isLastAvailablePage = true
			log.Warn("Failed to get additional data from paginated API %s - pages may have been removed during request.\n    Response was: %s", urlPath, string(resp.Body))
			return isLastAvailablePage, nil
		} else {
			return isLastAvailablePage, fmt.Errorf("failed to get further data from paginated API %s (HTTP %d)!\n    Response was: %s", urlPath, resp.StatusCode, string(resp.Body))
		}
	} else {
		return isLastAvailablePage, fmt.Errorf("failed to get data from paginated API %s (HTTP %d)!\n    Response was: %s", urlPath, resp.StatusCode, string(resp.Body))
	}

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
		nbCallsPerMinute := float64(nbCalls) / runningMinutes
		if resp.PageSize > 0 && resp.TotalCount > 0 {
			nbProcessed := nbCalls * resp.PageSize
			nbLeft := resp.TotalCount - nbProcessed
			ETAMinutes := float64(nbLeft) / (nbCallsPerMinute * float64(resp.PageSize))
			nbItemsMessage = fmt.Sprintf(", processed %d of %d at %d items/call and", nbProcessed, resp.TotalCount, resp.PageSize)
			ETAMessage = fmt.Sprintf("ETA: %.1f minutes", ETAMinutes)
		}

		log.Debug("Running extraction of: %s for %.1f minutes%s %.1f call/minute. %s", logLabel, runningMinutes, nbItemsMessage, nbCallsPerMinute, ETAMessage)
	}
}
