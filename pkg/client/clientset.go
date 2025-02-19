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

package client

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"time"

	"golang.org/x/oauth2/clientcredentials"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	automationApi "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/supportarchive"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
)

var (
	_ SettingsClient = (*dtclient.SettingsClient)(nil)
	_ ConfigClient   = (*dtclient.ConfigClient)(nil)
	_ SettingsClient = (*dtclient.DryRunSettingsClient)(nil)
	_ ConfigClient   = (*dtclient.DryRunConfigClient)(nil)
)

//go:generate mockgen -source=clientset.go -destination=client_mock.go -package=client ConfigClient

// ConfigClient is responsible for the classic Dynatrace configs. For settings objects, the [SettingsClient] is responsible.
// Each config endpoint is described by an [API] object to describe endpoints, structure, and behavior.
type ConfigClient interface {
	// Cache caches all config values for a given API.
	Cache(ctx context.Context, a api.API) error

	// List lists the available configs for an API.
	// It calls the underlying GET endpoint of the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	// The result is expressed using a list of Value (id and name tuples).
	List(ctx context.Context, a api.API) (values []dtclient.Value, err error)

	// Get reads a Dynatrace config identified by id from the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	Get(ctx context.Context, a api.API, id string) (json []byte, err error)

	// UpsertByName creates a given Dynatrace config if it doesn't exist and updates it otherwise using its name.
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	UpsertByName(ctx context.Context, a api.API, name string, payload []byte) (entity dtclient.DynatraceEntity, err error)

	// UpsertByNonUniqueNameAndId creates a given Dynatrace config if it doesn't exist and updates it based on specific rules if it does not
	// - if only one config with the name exist, behave like any other type and just update this entity
	// - if an exact match is found (same name and same generated UUID) update that entity
	// - if several configs exist, but non match the generated UUID create a new entity with generated UUID
	// It calls the underlying GET and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//	 GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//	 PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... with the given (or found by unique name) entity ID
	UpsertByNonUniqueNameAndId(ctx context.Context, a api.API, entityID string, name string, payload []byte, duplicate bool) (entity dtclient.DynatraceEntity, err error)

	// Delete removes a given config for a given API using its id.
	// It calls the DELETE endpoint for the API. E.g. for alerting profiles this would be:
	//    DELETE <environment-url>/api/config/v1/alertingProfiles/<id> ... to delete the config
	Delete(ctx context.Context, a api.API, id string) error

	// ExistsWithName checks if a config with the given name exists for the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	ExistsWithName(ctx context.Context, a api.API, name string) (exists bool, id string, err error)
}

//go:generate mockgen -source=clientset.go -destination=client_mock.go -package=client SettingsClient

// SettingsClient is the abstraction layer for CRUD operations on the Dynatrace Settings API.
// Its design is intentionally not dependent on Monaco objects.
//
// This interface exclusively accesses the [Settings API] of Dynatrace.
//
// The base mechanism for all methods is the same:
// We identify objects to be updated/deleted by their external-id. If an object can not be found using its external-id, we assume
// that it does not exist.
// More documentation is written in each method's documentation.
//
// [settings api]: https://www.dynatrace.com/support/help/dynatrace-api/environment-api/settings
type SettingsClient interface {
	// Cache caches all settings objects for a given schema.
	Cache(context.Context, string) error

	// Upsert either creates the supplied object, or updates an existing one.
	// First, we try to find the external-id of the object. If we can't find it, we create the object, if we find it, we
	// update the object.
	Upsert(context.Context, dtclient.SettingsObject, dtclient.UpsertSettingsOptions) (dtclient.DynatraceEntity, error)

	// ListSchemas returns all schemas that the Dynatrace environment reports
	ListSchemas(context.Context) (dtclient.SchemaList, error)

	// GetSchema returns the settings schema with the given schema ID
	GetSchema(context.Context, string) (dtclient.Schema, error)

	// List returns all settings objects for a given schema.
	List(context.Context, string, dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error)

	// Get returns the setting with the given object ID
	Get(context.Context, string) (*dtclient.DownloadSettingsObject, error)

	// Delete deletes a settings object giving its object ID
	Delete(context.Context, string) error
}

type AutomationClient interface {
	Get(ctx context.Context, resourceType automationApi.ResourceType, id string) (automation.Response, error)
	Create(ctx context.Context, resourceType automationApi.ResourceType, data []byte) (result automation.Response, err error)
	Update(ctx context.Context, resourceType automationApi.ResourceType, id string, data []byte) (automation.Response, error)
	List(ctx context.Context, resourceType automationApi.ResourceType) (automation.ListResponse, error)
	Upsert(ctx context.Context, resourceType automationApi.ResourceType, id string, data []byte) (result automation.Response, err error)
	Delete(ctx context.Context, resourceType automationApi.ResourceType, id string) (automation.Response, error)
}

type BucketClient interface {
	Get(ctx context.Context, bucketName string) (buckets.Response, error)
	List(ctx context.Context) (buckets.ListResponse, error)
	Create(ctx context.Context, bucketName string, data []byte) (buckets.Response, error)
	Update(ctx context.Context, bucketName string, data []byte) (buckets.Response, error)
	Upsert(ctx context.Context, bucketName string, data []byte) (buckets.Response, error)
	Delete(ctx context.Context, bucketName string) (buckets.Response, error)
}

type DocumentClient interface {
	Get(ctx context.Context, id string) (documents.Response, error)
	List(ctx context.Context, filter string) (documents.ListResponse, error)
	Create(ctx context.Context, name string, isPrivate bool, externalId string, data []byte, documentType documents.DocumentType) (coreapi.Response, error)
	Update(ctx context.Context, id string, name string, isPrivate bool, data []byte, documentType documents.DocumentType) (coreapi.Response, error)
	Delete(ctx context.Context, id string) (coreapi.Response, error)
}

type OpenPipelineClient interface {
	GetAll(ctx context.Context) ([]openpipeline.Response, error)
	Update(ctx context.Context, id string, data []byte) (openpipeline.Response, error)
}

type SegmentClient interface {
	List(ctx context.Context) (segments.Response, error)
	GetAll(ctx context.Context) ([]segments.Response, error)
	Delete(ctx context.Context, id string) (segments.Response, error)
	Create(ctx context.Context, data []byte) (segments.Response, error)
	Update(ctx context.Context, id string, data []byte) (segments.Response, error)
	Get(ctx context.Context, id string) (segments.Response, error)
}

type ServiceLevelObjectiveClient interface {
	List(ctx context.Context) (coreapi.PagedListResponse, error)
	Update(ctx context.Context, id string, body []byte) (coreapi.Response, error)
	Create(ctx context.Context, body []byte) (coreapi.Response, error)
}

var DefaultMonacoUserAgent = "Dynatrace Monitoring as Code/" + version.MonitoringAsCode + " " + (runtime.GOOS + " " + runtime.GOARCH)

var DefaultRetryOptions = corerest.RetryOptions{MaxRetries: 10, ShouldRetryFunc: corerest.RetryIfNotSuccess}

// ClientSet composes a "full" set of sub-clients to access Dynatrace APIs
// Each field may be nil, if the ClientSet is partially initialized - e.g. no autClient will be part of a ClientSet
// created for a 'classic' Dynatrace environment, as Automations are a Platform feature
type ClientSet struct {
	ConfigClient                ConfigClient
	SettingsClient              SettingsClient
	AutClient                   AutomationClient
	BucketClient                BucketClient
	DocumentClient              DocumentClient
	OpenPipelineClient          OpenPipelineClient
	SegmentClient               SegmentClient
	ServiceLevelObjectiveClient ServiceLevelObjectiveClient
}

type ClientOptions struct {
	CustomUserAgent string
	CachingDisabled bool
}

func (o ClientOptions) getUserAgentString() string {
	if o.CustomUserAgent == "" {
		return DefaultMonacoUserAgent
	}
	return o.CustomUserAgent
}

type PlatformAuth struct {
	OauthClientID, OauthClientSecret, OauthTokenURL string
	Token                                           string
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

func CreateClientSet(ctx context.Context, url string, auth manifest.Auth) (*ClientSet, error) {
	return CreateClientSetWithOptions(ctx, url, auth, ClientOptions{})
}

func CreateClientSetWithOptions(ctx context.Context, url string, auth manifest.Auth, opts ClientOptions) (*ClientSet, error) {
	var (
		configClient                ConfigClient
		settingsClient              SettingsClient
		bucketClient                BucketClient
		autClient                   AutomationClient
		documentClient              DocumentClient
		openPipelineClient          OpenPipelineClient
		segmentClient               SegmentClient
		serviceLevelObjectiveClient ServiceLevelObjectiveClient
		err                         error
	)
	concurrentReqLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)
	if err = validateURL(url); err != nil {
		return nil, err
	}

	cFactory := clients.Factory().
		WithConcurrentRequestLimit(concurrentReqLimit).
		WithUserAgent(opts.getUserAgentString()).
		WithRetryOptions(&DefaultRetryOptions).
		WithRateLimiter(true)

	if supportarchive.IsEnabled(ctx) {
		cFactory = cFactory.WithHTTPListener(&corerest.HTTPListener{Callback: trafficlogs.GetInstance().LogToFiles})
	}

	classicURL := url
	if auth.OAuth != nil {
		cFactory = cFactory.WithOAuthCredentials(
			clientcredentials.Config{
				ClientID:     auth.OAuth.ClientID.Value.Value(),
				ClientSecret: auth.OAuth.ClientSecret.Value.Value(),
				TokenURL:     auth.OAuth.GetTokenEndpointValue(),
			}).WithPlatformURL(url)
		client, err := cFactory.CreatePlatformClient()
		if err != nil {
			return nil, err
		}

		bucketClient, err = cFactory.BucketClientWithRetrySettings(15, time.Second, 5*time.Minute)
		if err != nil {
			return nil, err
		}

		autClient, err = cFactory.AutomationClient()
		if err != nil {
			return nil, err
		}

		documentClient, err = cFactory.DocumentClient()
		if err != nil {
			return nil, err
		}

		openPipelineClient, err = cFactory.OpenPipelineClient()
		if err != nil {
			return nil, err
		}

		segmentClient, err = cFactory.SegmentsClient()
		if err != nil {
			return nil, err
		}

		serviceLevelObjectiveClient, err = cFactory.SLOClient()
		if err != nil {
			return nil, err
		}

		settingsClient, err = dtclient.NewPlatformSettingsClient(client, dtclient.WithCachingDisabled(opts.CachingDisabled))
		if err != nil {
			return nil, err
		}

		classicURL, err = transformPlatformUrlToClassic(ctx, url, auth.OAuth, client)
		if err != nil {
			return nil, err
		}
	}

	if auth.Token != nil {
		cFactory = cFactory.WithAccessToken(auth.Token.Value.Value()).
			WithClassicURL(classicURL)
		client, err := cFactory.CreateClassicClient()
		if err != nil {
			return nil, err
		}

		configClient, err = dtclient.NewClassicConfigClient(client, dtclient.WithCachingDisabledForConfigClient(opts.CachingDisabled))
		if err != nil {
			return nil, err
		}

		if settingsClient == nil {
			settingsClient, err = dtclient.NewClassicSettingsClient(client, dtclient.WithCachingDisabled(opts.CachingDisabled), dtclient.WithAutoServerVersion())
			if err != nil {
				return nil, err
			}
		}
	}

	return &ClientSet{
		ConfigClient:                configClient,
		SettingsClient:              settingsClient,
		AutClient:                   autClient,
		BucketClient:                bucketClient,
		DocumentClient:              documentClient,
		OpenPipelineClient:          openPipelineClient,
		SegmentClient:               segmentClient,
		ServiceLevelObjectiveClient: serviceLevelObjectiveClient,
	}, nil
}

func transformPlatformUrlToClassic(ctx context.Context, url string, auth *manifest.OAuth, client *corerest.Client) (string, error) {
	classicUrl := url
	if auth != nil && client != nil {
		return metadata.GetDynatraceClassicURL(ctx, *client)
	}

	return classicUrl, nil
}
