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
	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/cache"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	dtVersion "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/version"
	"net/http"
	"net/url"
)

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

var retryIfNotStatusNotFound = func(resp *http.Response) bool { return resp.StatusCode != http.StatusNotFound }
var retryIfNotStatusBadRequest = func(resp *http.Response) bool { return resp.StatusCode != http.StatusBadRequest }

type UpsertSettingsOptions struct {
	OverrideRetry *RetrySetting
	InsertAfter   string
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

// DynatraceClient is the default implementation of the HTTP
// client targeting the relevant Dynatrace APIs for Monaco
type DynatraceClient struct {
	// serverVersion is the Dynatrace server version the
	// client will be interacting with
	serverVersion version.Version

	// platformClient is a rest client used to target platform enabled environments
	platformClient *corerest.Client

	// classicClient is a rest client used to target classic environments (e.g.,for downloading classic configs)
	classicClient *corerest.Client

	// settingsSchemaAPIPath is the API path to use for accessing settings schemas
	settingsSchemaAPIPath string

	//  settingsObjectAPIPath is the API path to use for accessing settings objects
	settingsObjectAPIPath string

	// retrySettings are the settings to be used for retrying failed http requests
	retrySettings RetrySettings

	// generateExternalID is used to generate an external id for settings 2.0 objects
	generateExternalID idutils.ExternalIDGenerator

	// settingsCache caches settings objects
	settingsCache cache.Cache[[]DownloadSettingsObject]

	// schemaCache caches schema constraints
	schemaCache cache.Cache[Schema]

	// classicConfigsCache caches classic settings values
	classicConfigsCache cache.Cache[[]Value]
}

func WithExternalIDGenerator(g idutils.ExternalIDGenerator) func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		d.generateExternalID = g
	}
}

// WithRetrySettings sets the retry settings to be used by the DynatraceClient
func WithRetrySettings(retrySettings RetrySettings) func(*DynatraceClient) {
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

		d.serverVersion = version.UnknownVersion
		if d.classicClient == nil {
			return
		}

		serverVersion, err = dtVersion.GetDynatraceVersion(context.TODO(), d.classicClient) //this will send the default user-agent
		if err != nil {
			log.WithFields(field.Error(err)).Warn("Unable to determine Dynatrace server version: %v", err)
			return
		}
		d.serverVersion = serverVersion
	}
}

// WithCachingDisabled allows disabling the client's builtin caching mechanism for
// classic configs, schema constraints and settings objects. Disabling the caching
// is recommended in situations where configs are fetched immediately after their creation (e.g. in test scenarios)
func WithCachingDisabled(disabled bool) func(client *DynatraceClient) {
	return func(d *DynatraceClient) {
		if disabled {
			d.classicConfigsCache = &cache.NoopCache[[]Value]{}
			d.schemaCache = &cache.NoopCache[Schema]{}
			d.settingsCache = &cache.NoopCache[[]DownloadSettingsObject]{}
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
//
//nolint:dupl
func NewPlatformClient(client *corerest.Client, classicClient *corerest.Client, opts ...func(dynatraceClient *DynatraceClient)) (*DynatraceClient, error) {
	d := &DynatraceClient{
		serverVersion:         version.Version{},
		platformClient:        client,
		classicClient:         classicClient,
		retrySettings:         DefaultRetrySettings,
		settingsSchemaAPIPath: settingsSchemaAPIPathPlatform,
		settingsObjectAPIPath: settingsObjectAPIPathPlatform,
		generateExternalID:    idutils.GenerateExternalIDForSettingsObject,
		settingsCache:         &cache.DefaultCache[[]DownloadSettingsObject]{},
		classicConfigsCache:   &cache.DefaultCache[[]Value]{},
		schemaCache:           &cache.DefaultCache[Schema]{},
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

// NewClassicClient creates a new dynatrace client to be used for classic environments
//
//nolint:dupl
func NewClassicClient(client *corerest.Client, opts ...func(dynatraceClient *DynatraceClient)) (*DynatraceClient, error) {
	d := &DynatraceClient{
		serverVersion:         version.Version{},
		platformClient:        client,
		classicClient:         client,
		retrySettings:         DefaultRetrySettings,
		settingsSchemaAPIPath: settingsSchemaAPIPathClassic,
		settingsObjectAPIPath: settingsObjectAPIPathClassic,
		generateExternalID:    idutils.GenerateExternalIDForSettingsObject,
		settingsCache:         &cache.DefaultCache[[]DownloadSettingsObject]{},
		classicConfigsCache:   &cache.DefaultCache[[]Value]{},
		schemaCache:           &cache.DefaultCache[Schema]{},
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

func (d *DynatraceClient) CacheConfigs(ctx context.Context, api api.API) error {
	_, err := d.fetchExistingValues(ctx, api)
	return err
}

func (d *DynatraceClient) ListConfigs(ctx context.Context, api api.API) (values []Value, err error) {
	return d.fetchExistingValues(ctx, api)
}

func (d *DynatraceClient) ReadConfigById(ctx context.Context, api api.API, id string) (json []byte, err error) {
	var dtUrl = api.URLPath
	if !api.SingleConfiguration {
		dtUrl = dtUrl + "/" + url.PathEscape(id)
	}

	response, err := coreapi.AsResponseOrError(d.classicClient.GET(ctx, dtUrl, corerest.RequestOptions{}))
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (d *DynatraceClient) DeleteConfigById(ctx context.Context, api api.API, id string) error {
	parsedURL, err := url.Parse(api.URLPath)
	if err != nil {
		return err
	}
	parsedURL = parsedURL.JoinPath(id)

	requestRetrier := corerest.RequestRetrier{
		MaxRetries:      d.retrySettings.Normal.MaxRetries,
		DelayAfterRetry: d.retrySettings.Normal.WaitTime,
		ShouldRetryFunc: retryIfNotStatusNotFound,
	}

	_, err = coreapi.AsResponseOrError(d.classicClient.DELETE(ctx, parsedURL.String(), corerest.RequestOptions{CustomRetrier: &requestRetrier}))
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

func (d *DynatraceClient) ConfigExistsByName(ctx context.Context, api api.API, name string) (exists bool, id string, err error) {
	if api.SingleConfiguration {
		// check that a single configuration is there by actually reading it.
		_, err := d.ReadConfigById(ctx, api, "")
		return err == nil, "", nil
	}

	existingObjectId, err := d.getExistingObjectId(ctx, name, api, nil)
	return existingObjectId != "", existingObjectId, err
}

func (d *DynatraceClient) UpsertConfigByName(ctx context.Context, a api.API, name string, payload []byte) (entity DynatraceEntity, err error) {
	if a.ID == api.Extension {
		return d.uploadExtension(ctx, a, name, payload)
	}
	return d.upsertDynatraceObject(ctx, a, name, payload)
}

func (d *DynatraceClient) UpsertConfigByNonUniqueNameAndId(ctx context.Context, api api.API, entityId string, name string, payload []byte, duplicate bool) (entity DynatraceEntity, err error) {
	return d.upsertDynatraceEntityByNonUniqueNameAndId(ctx, entityId, name, api, payload, duplicate)
}
