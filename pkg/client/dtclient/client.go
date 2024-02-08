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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/cache"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/useragent"
	dtVersion "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"strings"
)

// ConfigClient is responsible for the classic Dynatrace configs. For settings objects, the [SettingsClient] is responsible.
// Each config endpoint is described by an [API] object to describe endpoints, structure, and behavior.
type ConfigClient interface {
	// ListConfigs lists the available configs for an API.
	// It calls the underlying GET endpoint of the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	// The result is expressed using a list of Value (id and name tuples).
	ListConfigs(ctx context.Context, a APIData) (values []Value, err error)

	// ReadConfigById reads a Dynatrace config identified by id from the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	ReadConfigById(a APIData, id string) (json []byte, err error)

	// UpsertConfigByName creates a given Dynatrace config if it doesn't exist and updates it otherwise using its name.
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	UpsertConfigByName(ctx context.Context, a APIData, name string, payload []byte) (entity DynatraceEntity, err error)

	// UpsertConfigByNonUniqueNameAndId creates a given Dynatrace config if it doesn't exist and updates it based on specific rules if it does not
	// - if only one config with the name exist, behave like any other type and just update this entity
	// - if an exact match is found (same name and same generated UUID) update that entity
	// - if several configs exist, but non match the generated UUID create a new entity with generated UUID
	// It calls the underlying GET and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//	 GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//	 PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... with the given (or found by unique name) entity ID
	UpsertConfigByNonUniqueNameAndId(ctx context.Context, a APIData, entityID string, name string, payload []byte, duplicate bool) (entity DynatraceEntity, err error)

	// DeleteConfigById removes a given config for a given API using its id.
	// It calls the DELETE endpoint for the API. E.g. for alerting profiles this would be:
	//    DELETE <environment-url>/api/config/v1/alertingProfiles/<id> ... to delete the config
	DeleteConfigById(a APIData, id string) error

	// ConfigExistsByName checks if a config with the given name exists for the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	ConfigExistsByName(ctx context.Context, a APIData, name string) (exists bool, id string, err error)
}

// DownloadSettingsObject is the response type for the ListSettings operation
type DownloadSettingsObject struct {
	ExternalId       string                    `json:"externalId"`
	SchemaVersion    string                    `json:"schemaVersion"`
	SchemaId         string                    `json:"schemaId"`
	ObjectId         string                    `json:"objectId"`
	Scope            string                    `json:"scope"`
	Value            json.RawMessage           `json:"value"`
	ModificationInfo *SettingsModificationInfo `json:"modificationInfo"`
}

type SettingsModificationInfo struct {
	Deletable          bool          `json:"deletable"`
	Modifiable         bool          `json:"modifiable"`
	Movable            bool          `json:"movable"`
	ModifiablePaths    []interface{} `json:"modifiablePaths"`
	NonModifiablePaths []interface{} `json:"nonModifiablePaths"`
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
	UpsertSettings(context.Context, SettingsObject, UpsertSettingsOptions) (DynatraceEntity, error)

	// ListSchemas returns all schemas that the Dynatrace environment reports
	ListSchemas() (SchemaList, error)

	FetchSchemasConstraints(schemaID string) (SchemaConstraints, error)

	// ListSettings returns all settings objects for a given schema.
	ListSettings(context.Context, string, ListSettingsOptions) ([]DownloadSettingsObject, error)

	// GetSettingById returns the setting with the given object ID
	GetSettingById(string) (*DownloadSettingsObject, error)

	// DeleteSettings deletes a settings object giving its object ID
	DeleteSettings(string) error
}

type UpsertSettingsOptions struct {
	OverrideRetry *rest.RetrySetting
}

// defaultListSettingsFields  are the fields we are interested in when getting setting objects
const defaultListSettingsFields = "objectId,value,externalId,schemaVersion,schemaId,scope,modificationInfo"

// reducedListSettingsFields are the fields we are interested in when getting settings objects but don't care about the
// actual value payload
const reducedListSettingsFields = "objectId,externalId,schemaVersion,schemaId,scope,modificationInfo"
const defaultPageSize = "500"

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

	// platformClient is a rest client used to target platform enabled environments
	platformClient *rest.Client

	// classicClient is a rest client used to target classic environments (e.g.,for downloading classic configs)
	classicClient *rest.Client

	// settingsSchemaAPIPath is the API path to use for accessing settings schemas
	settingsSchemaAPIPath string

	//  settingsObjectAPIPath is the API path to use for accessing settings objects
	settingsObjectAPIPath string

	// retrySettings are the settings to be used for retrying failed http requests
	retrySettings rest.RetrySettings

	// limiter is used to limit parallel http requests
	limiter *concurrency.Limiter

	// generateExternalID is used to generate an external id for settings 2.0 objects
	generateExternalID idutils.ExternalIDGenerator

	// settingsCache caches settings objects
	settingsCache cache.Cache[[]DownloadSettingsObject]

	// schemaConstraintsCache caches schema constraints
	schemaConstraintsCache cache.Cache[SchemaConstraints]

	// classicConfigsCache caches classic settings values
	classicConfigsCache cache.Cache[[]Value]
}

var (
	_ SettingsClient = (*DynatraceClient)(nil)
	_ ConfigClient   = (*DynatraceClient)(nil)
	_ Client         = (*DynatraceClient)(nil)
)

func WithExternalIDGenerator(g idutils.ExternalIDGenerator) func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		d.generateExternalID = g
	}
}

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

// WithAutoServerVersion can be used to let the client automatically determine the Dynatrace server version
// during creation using newDynatraceClient. If the server version is already known WithServerVersion should be used
func WithAutoServerVersion() func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		var serverVersion version.Version
		var err error

		if _, ok := d.platformClient.Client().Transport.(*oauth2.Transport); ok {
			// for platform enabled tenants there is no dedicated version endpoint
			// so this call would need to be "redirected" to the second gen URL, which do not currently resolve
			d.serverVersion = version.UnknownVersion
		} else {
			serverVersion, err = dtVersion.GetDynatraceVersion(context.TODO(), d.classicClient, d.environmentURLClassic) //this will send the default user-agent
		}
		if err != nil {
			log.WithFields(field.Error(err)).Warn("Unable to determine Dynatrace server version: %v", err)
			d.serverVersion = version.UnknownVersion
		} else {
			d.serverVersion = serverVersion
		}
	}
}

// WithCachingDisabled allows disabling the client's builtin caching mechanism for
// classic configs, schema constraints and settings objects. Disabling the caching
// is recommended in situations where configs are fetched immediately after their creation (e.g. in test scenarios)
func WithCachingDisabled(disabled bool) func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		if disabled {
			d.classicConfigsCache = &cache.NoopCache[[]Value]{}
			d.schemaConstraintsCache = &cache.NoopCache[SchemaConstraints]{}
			d.settingsCache = &cache.NoopCache[[]DownloadSettingsObject]{}
		}
	}
}

// WithCustomUserAgentString allows to configure a custom user-agent string that the Client will send with each HTTP request
// If none is set, the default Monaco CLI specific user-agent is sent.
func WithCustomUserAgentString(userAgent string) func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		if d.platformClient != nil && d.platformClient.Client() != nil {
			d.platformClient.Client().Transport = useragent.NewCustomUserAgentTransport(d.platformClient.Client().Transport, userAgent)
		}
		if d.classicClient != nil && d.classicClient.Client() != nil {
			d.classicClient.Client().Transport = useragent.NewCustomUserAgentTransport(d.classicClient.Client().Transport, userAgent)
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
func NewPlatformClient(dtURL string, classicURL string, client *rest.Client, classicClient *rest.Client, opts ...func(dynatraceClient *DynatraceClient)) (*DynatraceClient, error) {
	dtURL = strings.TrimSuffix(dtURL, "/")
	if err := validateURL(dtURL); err != nil {
		return nil, err
	}

	classicURL = strings.TrimSuffix(classicURL, "/")
	if err := validateURL(classicURL); err != nil {
		return nil, err
	}

	d := &DynatraceClient{
		serverVersion:          version.Version{},
		environmentURL:         dtURL,
		environmentURLClassic:  classicURL,
		platformClient:         client,
		classicClient:          classicClient,
		retrySettings:          rest.DefaultRetrySettings,
		settingsSchemaAPIPath:  settingsSchemaAPIPathPlatform,
		settingsObjectAPIPath:  settingsObjectAPIPathPlatform,
		limiter:                concurrency.NewLimiter(5),
		generateExternalID:     idutils.GenerateExternalID,
		settingsCache:          &cache.DefaultCache[[]DownloadSettingsObject]{},
		classicConfigsCache:    &cache.DefaultCache[[]Value]{},
		schemaConstraintsCache: &cache.DefaultCache[SchemaConstraints]{},
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

// NewClassicClient creates a new dynatrace client to be used for classic environments
func NewClassicClient(dtURL string, client *rest.Client, opts ...func(dynatraceClient *DynatraceClient)) (*DynatraceClient, error) {
	dtURL = strings.TrimSuffix(dtURL, "/")
	if err := validateURL(dtURL); err != nil {
		return nil, err
	}
	d := &DynatraceClient{
		serverVersion:          version.Version{},
		environmentURL:         dtURL,
		environmentURLClassic:  dtURL,
		platformClient:         client,
		classicClient:          client,
		retrySettings:          rest.DefaultRetrySettings,
		settingsSchemaAPIPath:  settingsSchemaAPIPathClassic,
		settingsObjectAPIPath:  settingsObjectAPIPathClassic,
		limiter:                concurrency.NewLimiter(5),
		generateExternalID:     idutils.GenerateExternalID,
		settingsCache:          &cache.DefaultCache[[]DownloadSettingsObject]{},
		classicConfigsCache:    &cache.DefaultCache[[]Value]{},
		schemaConstraintsCache: &cache.DefaultCache[SchemaConstraints]{},
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

func (d *DynatraceClient) ListConfigs(ctx context.Context, api APIData) (values []Value, err error) {
	d.limiter.ExecuteBlocking(func() {
		values, err = d.listConfigs(ctx, api)
	})
	return
}
func (d *DynatraceClient) listConfigs(ctx context.Context, api APIData) (values []Value, err error) {

	fullUrl := api.CreateURL(d.environmentURLClassic)
	values, err = d.getExistingValuesFromEndpoint(ctx, api, fullUrl)
	return values, err
}

func (d *DynatraceClient) ReadConfigById(api APIData, id string) (json []byte, err error) {
	d.limiter.ExecuteBlocking(func() {
		json, err = d.readConfigById(context.TODO(), api, id)
	})
	return
}

func (d *DynatraceClient) readConfigById(ctx context.Context, api APIData, id string) (json []byte, err error) {
	var dtUrl string
	isSingleConfigurationApi := api.SingleConfiguration

	if isSingleConfigurationApi {
		dtUrl = api.CreateURL(d.environmentURLClassic)
	} else {
		dtUrl = api.CreateURL(d.environmentURLClassic) + "/" + url.PathEscape(id)
	}

	response, err := d.classicClient.Get(ctx, dtUrl)

	if err != nil {
		return nil, err
	}

	if !response.IsSuccess() {
		return nil, rest.NewRespErr(fmt.Sprintf("failed to get existing config for api %v (HTTP %v)!\n    Response was: %v", api.ID, response.StatusCode, string(response.Body)), response).WithRequestInfo(http.MethodGet, dtUrl)
	}

	return response.Body, nil
}

func (d *DynatraceClient) DeleteConfigById(api APIData, id string) (err error) {
	d.limiter.ExecuteBlocking(func() {
		err = d.deleteConfigById(context.TODO(), api, id)
	})
	return
}

func (d *DynatraceClient) deleteConfigById(ctx context.Context, api APIData, id string) error {

	u := api.CreateURL(d.environmentURLClassic)
	parsedURL, err := url.Parse(u)
	if err != nil {
		return err
	}
	parsedURL = parsedURL.JoinPath(id)

	resp, err := d.classicClient.Delete(ctx, parsedURL.String())
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		log.Debug("No config with id '%s' found to delete (HTTP 404 response)", id)
		return nil
	}

	if !resp.IsSuccess() {
		return rest.NewRespErr(fmt.Sprintf("failed call to DELETE %s (HTTP %d)!\n Response was:\n %s", api.CreateURL(d.environmentURLClassic)+"/"+id, resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodDelete, u)
	}
	return nil
}

func (d *DynatraceClient) ConfigExistsByName(ctx context.Context, api APIData, name string) (exists bool, id string, err error) {
	d.limiter.ExecuteBlocking(func() {
		exists, id, err = d.configExistsByName(ctx, api, name)
	})
	return
}

func (d *DynatraceClient) configExistsByName(ctx context.Context, api APIData, name string) (exists bool, id string, err error) {
	apiURL := api.CreateURL(d.environmentURLClassic)
	existingObjectId, err := d.getObjectIdIfAlreadyExists(ctx, api, apiURL, name)
	return existingObjectId != "", existingObjectId, err
}

func (d *DynatraceClient) UpsertConfigByName(ctx context.Context, api APIData, name string, payload []byte) (entity DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		entity, err = d.upsertConfigByName(ctx, api, name, payload)
	})
	return
}

func (d *DynatraceClient) upsertConfigByName(ctx context.Context, api APIData, name string, payload []byte) (entity DynatraceEntity, err error) {
	if api.ID == "extension" {
		return d.uploadExtension(ctx, api, name, payload)
	}
	return d.upsertDynatraceObject(ctx, api, name, payload)
}

func (d *DynatraceClient) UpsertConfigByNonUniqueNameAndId(ctx context.Context, api APIData, entityId string, name string, payload []byte, duplicate bool) (entity DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		entity, err = d.upsertConfigByNonUniqueNameAndId(ctx, api, entityId, name, payload, duplicate)
	})
	return
}

func (d *DynatraceClient) upsertConfigByNonUniqueNameAndId(ctx context.Context, api APIData, entityId string, name string, payload []byte, duplicate bool) (entity DynatraceEntity, err error) {
	return d.upsertDynatraceEntityByNonUniqueNameAndId(ctx, entityId, name, api, payload, duplicate)
}

func (d *DynatraceClient) GetSettingById(objectId string) (res *DownloadSettingsObject, err error) {
	d.limiter.ExecuteBlocking(func() {
		res, err = d.getSettingById(context.TODO(), objectId)
	})
	return
}

func (d *DynatraceClient) getSettingById(ctx context.Context, objectId string) (*DownloadSettingsObject, error) {
	u, err := url.Parse(d.environmentURL + d.settingsObjectAPIPath + "/" + objectId)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %w", d.environmentURL+d.settingsObjectAPIPath, err)
	}

	resp, err := d.platformClient.Get(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to GET settings object with object id %q: %w", objectId, err)
	}

	if !resp.IsSuccess() {
		// special case of settings API: If you give it any object ID for a setting object that does not exist, it will give back 400 BadRequest instead
		// of 404 Not found. So we interpret both status codes, 400 and 404, as "not found"
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
			return nil, rest.NewRespErr(ErrSettingNotFound.Error(), resp).WithRequestInfo(http.MethodGet, u.String()).WithErr(ErrSettingNotFound)
		}
		return nil, rest.NewRespErr(fmt.Sprintf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodGet, u.String())
	}

	var result DownloadSettingsObject
	if err = json.Unmarshal(resp.Body, &result); err != nil {
		return nil, rest.NewRespErr("failed to unmarshal response", resp).WithRequestInfo(http.MethodGet, u.String()).WithErr(err)
	}

	return &result, nil
}

func (d *DynatraceClient) DeleteSettings(objectID string) (err error) {
	d.limiter.ExecuteBlocking(func() {
		err = d.deleteSettings(context.TODO(), objectID)
	})
	return
}

func (d *DynatraceClient) deleteSettings(ctx context.Context, objectID string) error {
	u, err := url.Parse(d.environmentURL + d.settingsObjectAPIPath)
	if err != nil {
		return fmt.Errorf("failed to parse URL '%s': %w", d.environmentURL+d.settingsObjectAPIPath, err)
	}

	u = u.JoinPath(objectID)
	resp, err := d.platformClient.Delete(ctx, u.String())
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		log.Debug("No config with id '%s' found to delete (HTTP 404 response)", objectID)
		return nil
	}

	if !resp.IsSuccess() {
		return rest.NewRespErr(fmt.Sprintf("failed call to DELETE %s (HTTP %d)!\n Response was:\n %s", u.String()+"/"+objectID, resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodDelete, u.String())
	}
	return nil
}

func buildUrl(environmentUrl, urlPath string, params url.Values) (*url.URL, error) {
	u, err := url.Parse(environmentUrl + urlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %w", environmentUrl+urlPath, err)
	}

	u.RawQuery = params.Encode()

	return u, nil
}
