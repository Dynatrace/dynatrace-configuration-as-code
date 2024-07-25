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

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	automationApi "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	_ SettingsClient  = (*dtclient.DynatraceClient)(nil)
	_ ConfigClient    = (*dtclient.DynatraceClient)(nil)
	_ DynatraceClient = (*dtclient.DynatraceClient)(nil)
	_ DynatraceClient = (*dtclient.DummyClient)(nil)
)

// ConfigClient is responsible for the classic Dynatrace configs. For settings objects, the [SettingsClient] is responsible.
// Each config endpoint is described by an [API] object to describe endpoints, structure, and behavior.
type ConfigClient interface {
	// ListConfigs lists the available configs for an API.
	// It calls the underlying GET endpoint of the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	// The result is expressed using a list of Value (id and name tuples).
	ListConfigs(ctx context.Context, a api.API) (values []dtclient.Value, err error)

	// ReadConfigById reads a Dynatrace config identified by id from the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	ReadConfigById(ctx context.Context, a api.API, id string) (json []byte, err error)

	// UpsertConfigByName creates a given Dynatrace config if it doesn't exist and updates it otherwise using its name.
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	UpsertConfigByName(ctx context.Context, a api.API, name string, payload []byte) (entity dtclient.DynatraceEntity, err error)

	// UpsertConfigByNonUniqueNameAndId creates a given Dynatrace config if it doesn't exist and updates it based on specific rules if it does not
	// - if only one config with the name exist, behave like any other type and just update this entity
	// - if an exact match is found (same name and same generated UUID) update that entity
	// - if several configs exist, but non match the generated UUID create a new entity with generated UUID
	// It calls the underlying GET and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//	 GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//	 PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... with the given (or found by unique name) entity ID
	UpsertConfigByNonUniqueNameAndId(ctx context.Context, a api.API, entityID string, name string, payload []byte, duplicate bool) (entity dtclient.DynatraceEntity, err error)

	// DeleteConfigById removes a given config for a given API using its id.
	// It calls the DELETE endpoint for the API. E.g. for alerting profiles this would be:
	//    DELETE <environment-url>/api/config/v1/alertingProfiles/<id> ... to delete the config
	DeleteConfigById(ctx context.Context, a api.API, id string) error

	// ConfigExistsByName checks if a config with the given name exists for the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	ConfigExistsByName(ctx context.Context, a api.API, name string) (exists bool, id string, err error)
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
	UpsertSettings(context.Context, dtclient.SettingsObject, dtclient.UpsertSettingsOptions) (dtclient.DynatraceEntity, error)

	// ListSchemas returns all schemas that the Dynatrace environment reports
	ListSchemas(context.Context) (dtclient.SchemaList, error)

	// GetSchemaById returns the settings schema with the given schema ID
	GetSchemaById(context.Context, string) (dtclient.Schema, error)

	// ListSettings returns all settings objects for a given schema.
	ListSettings(context.Context, string, dtclient.ListSettingsOptions) ([]dtclient.DownloadSettingsObject, error)

	// GetSettingById returns the setting with the given object ID
	GetSettingById(context.Context, string) (*dtclient.DownloadSettingsObject, error)

	// DeleteSettings deletes a settings object giving its object ID
	DeleteSettings(context.Context, string) error
}

//go:generate mockgen -source=clientset.go -destination=client_mock.go -package=client DynatraceClient

// DynatraceClient provides the functionality for performing basic CRUD operations on any Dynatrace API
// supported by monaco.
// It encapsulates the configuration-specific inconsistencies of certain APIs in one place to provide
// a common interface to work with. After all: A user of Client shouldn't care about the
// implementation details of each individual Dynatrace API.
// Its design is intentionally not dependent on the Config and Environment interfaces included in monaco.
// This makes sure, that Client can be used as a base for future tooling, which relies on
// a standardized way to access Dynatrace APIs.
type DynatraceClient interface {
	ConfigClient
	SettingsClient
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

var DefaultMonacoUserAgent = "Dynatrace Monitoring as Code/" + version.MonitoringAsCode + " " + (runtime.GOOS + " " + runtime.GOARCH)

// ClientSet composes a "full" set of sub-clients to access Dynatrace APIs
// Each field may be nil, if the ClientSet is partially initialized - e.g. no autClient will be part of a ClientSet
// created for a 'classic' Dynatrace environment, as Automations are a Platform feature
type ClientSet struct {
	// dtClient is the client capable of updating or creating settings and classic configs
	DTClient DynatraceClient
	// autClient is the client capable of updating or creating automation API configs
	AutClient AutomationClient
	// bucketClient is the client capable of updating or creating Grail Bucket configs
	BucketClient BucketClient
	// DocumentClient is a client capable of manipulating documents
	DocumentClient DocumentClient
	// OpenPipelineClient is a client capable of manipulating openPipeline configs
	OpenPipelineClient OpenPipelineClient
}

func (s ClientSet) Classic() ConfigClient {
	return s.DTClient
}

func (s ClientSet) Settings() SettingsClient {
	return s.DTClient
}

func (s ClientSet) Automation() AutomationClient {
	return s.AutClient
}

func (s ClientSet) Bucket() BucketClient {
	return s.BucketClient
}

func (s ClientSet) Document() DocumentClient {
	return s.DocumentClient
}

func (s ClientSet) OpenPipeline() OpenPipelineClient {
	return s.OpenPipelineClient
}

type ClientOptions struct {
	CustomUserAgent string
	SupportArchive  bool
	CachingDisabled bool
}

func (o ClientOptions) getUserAgentString() string {
	if o.CustomUserAgent == "" {
		return DefaultMonacoUserAgent
	}
	return o.CustomUserAgent
}

func CreateClassicClientSet(url string, token string, opts ClientOptions) (*ClientSet, error) {
	concurrentRequestLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)

	if err := validateURL(url); err != nil {
		return nil, err
	}

	clientFactory := clients.Factory().
		WithConcurrentRequestLimit(concurrentRequestLimit).
		WithAccessToken(token).
		WithClassicURL(url).
		WithUserAgent(opts.getUserAgentString())

	var trafficLogger *trafficlogs.FileBasedLogger
	if opts.SupportArchive {
		trafficLogger = trafficlogs.NewFileBased()
		clientFactory = clientFactory.WithHTTPListener(&corerest.HTTPListener{Callback: trafficLogger.LogToFiles})
	}

	classicClient, err := clientFactory.CreateClassicClient()
	if err != nil {
		return nil, err
	}

	dtClient, err := dtclient.NewClassicClient(
		classicClient,
		dtclient.WithCachingDisabled(opts.CachingDisabled),
		dtclient.WithAutoServerVersion(),
	)
	if err != nil {
		return nil, err
	}

	return &ClientSet{
		DTClient: dtClient,
	}, nil
}

type PlatformAuth struct {
	OauthClientID, OauthClientSecret, OauthTokenURL string
	Token                                           string
}

func CreatePlatformClientSet(platformURL string, auth PlatformAuth, opts ClientOptions) (*ClientSet, error) {
	concurrentRequestLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)

	if err := validateURL(platformURL); err != nil {
		return nil, err
	}

	clientFactory := clients.Factory().
		WithOAuthCredentials(clientcredentials.Config{
			ClientID:     auth.OauthClientID,
			ClientSecret: auth.OauthClientSecret,
			TokenURL:     auth.OauthTokenURL,
		}).
		WithConcurrentRequestLimit(concurrentRequestLimit).
		WithPlatformURL(platformURL).
		WithUserAgent(opts.getUserAgentString())

	if opts.SupportArchive {
		trafficLogger := trafficlogs.NewFileBased()
		clientFactory = clientFactory.WithHTTPListener(&corerest.HTTPListener{Callback: trafficLogger.LogToFiles})
	}

	client, err := clientFactory.CreatePlatformClient()
	if err != nil {
		return nil, err
	}

	classicURL, err := metadata.GetDynatraceClassicURL(context.TODO(), *client)
	if err != nil {
		return nil, err
	}

	clientFactory = clientFactory.WithClassicURL(classicURL).WithAccessToken(auth.Token)

	classicClient, err := clientFactory.CreateClassicClient()
	if err != nil {
		return nil, err
	}

	dtClient, err := dtclient.NewPlatformClient(
		client,
		classicClient,
		dtclient.WithCachingDisabled(opts.CachingDisabled),
		dtclient.WithAutoServerVersion(),
	)
	if err != nil {
		return nil, err
	}

	bucketClient, err := clientFactory.BucketClientWithRetrySettings(15, time.Second, 5*time.Minute)
	if err != nil {
		return nil, err
	}

	autClient, err := clientFactory.AutomationClient()
	if err != nil {
		return nil, err
	}

	documentClient, err := clientFactory.DocumentClient()
	if err != nil {
		return nil, err
	}

	openPipelineClient, err := clientFactory.OpenPipelineClient()
	if err != nil {
		return nil, err
	}

	return &ClientSet{
		DTClient:           dtClient,
		AutClient:          autClient,
		BucketClient:       bucketClient,
		DocumentClient:     documentClient,
		OpenPipelineClient: openPipelineClient,
	}, nil
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
