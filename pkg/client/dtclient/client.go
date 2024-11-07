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
	"net/http"
	"net/url"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/cache"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	dtVersion "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/version"
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

type SettingsClient struct {
	// serverVersion is the Dynatrace server version the
	// client will be interacting with
	serverVersion version.Version

	// client is a rest client used to target platform enabled environments
	client *corerest.Client

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

	// limiter is used to limit parallel http requests
	limiter *concurrency.Limiter
}

type ClassicClient struct {
	client *corerest.Client

	// retrySettings are the settings to be used for retrying failed http requests
	retrySettings RetrySettings

	// classicConfigsCache caches classic settings values
	classicConfigsCache cache.Cache[[]Value]

	// limiter is used to limit parallel http requests
	limiter *concurrency.Limiter
}

func WithExternalIDGenerator(g idutils.ExternalIDGenerator) func(client *SettingsClient) {
	return func(d *SettingsClient) {
		d.generateExternalID = g
	}
}

// WithClientRequestLimiter specifies that a specifies the limiter to be used for
// limiting parallel client requests
func WithClientRequestLimiter(limiter *concurrency.Limiter) func(client *SettingsClient) {
	return func(d *SettingsClient) {
		d.limiter = limiter
	}
}

// WithRetrySettings sets the retry settings to be used by the DynatraceClient
func WithRetrySettings(retrySettings RetrySettings) func(*SettingsClient) {
	return func(d *SettingsClient) {
		d.retrySettings = retrySettings
	}
}

// WithClientRequestLimiter specifies that a specifies the limiter to be used for
// limiting parallel client requests
func WithClientRequestLimiterForClassic(limiter *concurrency.Limiter) func(client *ClassicClient) {
	return func(d *ClassicClient) {
		d.limiter = limiter
	}
}

// WithRetrySettings sets the retry settings to be used by the ClassicClient
func WithRetrySettingsForClassic(retrySettings RetrySettings) func(*ClassicClient) {
	return func(d *ClassicClient) {
		d.retrySettings = retrySettings
	}
}

// WithServerVersion sets the Dynatrace version of the Dynatrace server/tenant the client will be interacting with
func WithServerVersion(serverVersion version.Version) func(client *SettingsClient) {
	return func(d *SettingsClient) {
		d.serverVersion = serverVersion
	}
}

// WithAutoServerVersion can be used to let the client automatically determine the Dynatrace server version
// during creation using newDynatraceClient. If the server version is already known WithServerVersion should be used.
// Do not use this with NewPlatformSettingsClient() as the client will not work and cause an error to be logged.
func WithAutoServerVersion() func(client *SettingsClient) {
	return func(d *SettingsClient) {
		var serverVersion version.Version
		var err error

		d.serverVersion = version.UnknownVersion
		if d.client == nil {
			return
		}

		serverVersion, err = dtVersion.GetDynatraceVersion(context.TODO(), d.client) //this will send the default user-agent
		if err != nil {
			log.WithFields(field.Error(err)).Warn("Unable to determine Dynatrace server version: %v", err)
			return
		}
		d.serverVersion = serverVersion
	}
}

// WithCachingDisabledForClassic allows disabling the client's builtin caching mechanism for classic configs.
// Disabling the caching is recommended in situations where configs are fetched immediately after their creation (e.g. in test scenarios).
func WithCachingDisabledForClassic(disabled bool) func(client *ClassicClient) {
	return func(d *ClassicClient) {
		if disabled {
			d.classicConfigsCache = &cache.NoopCache[[]Value]{}
		}
	}
}

// WithCachingDisabled allows disabling the client's builtin caching mechanism for schema constraints and settings objects.
// Disabling the caching is recommended in situations where configs are fetched immediately after their creation (e.g. in test scenarios).
func WithCachingDisabled(disabled bool) func(client *SettingsClient) {
	return func(d *SettingsClient) {
		if disabled {
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

// NewPlatformSettingsClient creates a new settings client to be used for platform enabled environments
//
//nolint:dupl
func NewPlatformSettingsClient(client *corerest.Client, opts ...func(dynatraceClient *SettingsClient)) (*SettingsClient, error) {
	d := &SettingsClient{
		serverVersion:         version.Version{},
		client:                client,
		retrySettings:         DefaultRetrySettings,
		settingsSchemaAPIPath: settingsSchemaAPIPathPlatform,
		settingsObjectAPIPath: settingsObjectAPIPathPlatform,
		generateExternalID:    idutils.GenerateExternalIDForSettingsObject,
		settingsCache:         &cache.DefaultCache[[]DownloadSettingsObject]{},
		schemaCache:           &cache.DefaultCache[Schema]{},
		limiter:               concurrency.NewLimiter(5),
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

func NewClassicClient(client *corerest.Client, opts ...func(dynatraceClient *ClassicClient)) (*ClassicClient, error) {
	d := &ClassicClient{
		client:              client,
		retrySettings:       DefaultRetrySettings,
		classicConfigsCache: &cache.DefaultCache[[]Value]{},
		limiter:             concurrency.NewLimiter(5),
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

// NewClassicSettingsClient creates a new settings client to be used for classic environments.
//
//nolint:dupl
func NewClassicSettingsClient(client *corerest.Client, opts ...func(dynatraceClient *SettingsClient)) (*SettingsClient, error) {
	d := &SettingsClient{
		serverVersion:         version.Version{},
		client:                client,
		retrySettings:         DefaultRetrySettings,
		settingsSchemaAPIPath: settingsSchemaAPIPathClassic,
		settingsObjectAPIPath: settingsObjectAPIPathClassic,
		generateExternalID:    idutils.GenerateExternalIDForSettingsObject,
		settingsCache:         &cache.DefaultCache[[]DownloadSettingsObject]{},
		schemaCache:           &cache.DefaultCache[Schema]{},
		limiter:               concurrency.NewLimiter(5),
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

func (d *ClassicClient) CacheConfigs(ctx context.Context, api api.API) error {
	_, err := d.fetchExistingValues(ctx, api)
	return err
}

func (d *ClassicClient) ListConfigs(ctx context.Context, api api.API) (values []Value, err error) {
	return d.fetchExistingValues(ctx, api)
}

func (d *ClassicClient) ReadConfigById(ctx context.Context, api api.API, id string) (json []byte, err error) {
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

func (d *ClassicClient) DeleteConfigById(ctx context.Context, api api.API, id string) error {
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

func (d *ClassicClient) ConfigExistsByName(ctx context.Context, api api.API, name string) (exists bool, id string, err error) {
	if api.SingleConfiguration {
		// check that a single configuration is there by actually reading it.
		_, err := d.ReadConfigById(ctx, api, "")
		return err == nil, "", nil
	}

	existingObjectId, err := d.getExistingObjectId(ctx, name, api, nil)
	return existingObjectId != "", existingObjectId, err
}

func (d *ClassicClient) UpsertConfigByName(ctx context.Context, a api.API, name string, payload []byte) (entity DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		entity, err = d.upsertConfigByName(ctx, a, name, payload)
	})
	return
}

func (d *ClassicClient) upsertConfigByName(ctx context.Context, a api.API, name string, payload []byte) (entity DynatraceEntity, err error) {

	if a.ID == api.Extension {
		return d.uploadExtension(ctx, a, name, payload)
	}
	return d.upsertDynatraceObject(ctx, a, name, payload)
}

func (d *ClassicClient) UpsertConfigByNonUniqueNameAndId(ctx context.Context, api api.API, entityId string, name string, payload []byte, duplicate bool) (entity DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		entity, err = d.upsertDynatraceEntityByNonUniqueNameAndId(ctx, entityId, name, api, payload, duplicate)
	})
	return
}

func (d *SettingsClient) GetSettingById(ctx context.Context, objectId string) (res *DownloadSettingsObject, err error) {
	resp, err := coreapi.AsResponseOrError(d.client.GET(ctx, d.settingsObjectAPIPath+"/"+objectId, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		return nil, err
	}

	var result DownloadSettingsObject
	if err = json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal settings object: %w", err)
	}

	return &result, nil
}

func (d *SettingsClient) DeleteSettings(ctx context.Context, objectID string) error {
	_, err := coreapi.AsResponseOrError(d.client.DELETE(ctx, d.settingsObjectAPIPath+"/"+objectID, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		apiError := coreapi.APIError{}
		if errors.As(err, &apiError) && apiError.StatusCode == http.StatusNotFound {
			log.Debug("No settings object with id '%s' found to delete (HTTP 404 response)", objectID)
			return nil
		}
		return err
	}

	return nil
}
