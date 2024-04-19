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
	automationApi "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	lib "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	clientAuth "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/auth"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/useragent"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"golang.org/x/oauth2/clientcredentials"
	"runtime"
	"time"
)

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

var DefaultMonacoUserAgent = "Dynatrace Monitoring as Code/" + version.MonitoringAsCode + " " + (runtime.GOOS + " " + runtime.GOARCH)

// ClientSet composes a "full" set of sub-clients to access Dynatrace APIs
// Each field may be nil, if the ClientSet is partially initialized - e.g. no autClient will be part of a ClientSet
// created for a 'classic' Dynatrace environment, as Automations are a Platform feature
type ClientSet struct {
	// dtClient is the client capable of updating or creating settings and classic configs
	DTClient dtclient.Client
	// autClient is the client capable of updating or creating automation API configs
	AutClient AutomationClient
	// bucketClient is the client capable of updating or creating Grail Bucket configs
	BucketClient BucketClient
}

func (s ClientSet) Classic() dtclient.Client {
	return s.DTClient
}

func (s ClientSet) Settings() dtclient.Client {
	return s.DTClient
}

func (s ClientSet) Automation() AutomationClient {
	return s.AutClient
}

func (s ClientSet) Bucket() BucketClient {
	return s.BucketClient
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

	tokenClient := clientAuth.NewTokenAuthClient(token)
	var trafficLogger *trafficlogs.FileBasedLogger
	if opts.SupportArchive {
		trafficLogger = trafficlogs.NewFileBased()
	}

	restClient := rest.NewRestClient(tokenClient, trafficLogger, rest.CreateRateLimitStrategy())
	dtClient, err := dtclient.NewClassicClient(
		url,
		restClient,
		dtclient.WithCachingDisabled(opts.CachingDisabled),
		dtclient.WithAutoServerVersion(),
		dtclient.WithClientRequestLimiter(concurrency.NewLimiter(concurrentRequestLimit)),
		dtclient.WithCustomUserAgentString(opts.getUserAgentString()),
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

func CreatePlatformClientSet(url string, auth PlatformAuth, opts ClientOptions) (*ClientSet, error) {
	concurrentRequestLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)

	oauthCredentials := clientAuth.OauthCredentials{
		ClientID:     auth.OauthClientID,
		ClientSecret: auth.OauthClientSecret,
		TokenURL:     auth.OauthTokenURL,
	}

	tokenClient := clientAuth.NewTokenAuthClient(auth.Token)
	oauthClient := clientAuth.NewOAuthClient(context.TODO(), oauthCredentials)

	var trafficLogger *trafficlogs.FileBasedLogger
	if opts.SupportArchive {
		trafficLogger = trafficlogs.NewFileBased()
	}

	classicUrlClient := rest.NewRestClient(oauthClient, trafficLogger, rest.CreateRateLimitStrategy())
	classicUrlClient.Client().Transport = useragent.NewCustomUserAgentTransport(classicUrlClient.Client().Transport, opts.getUserAgentString())
	classicURL, err := metadata.GetDynatraceClassicURL(context.TODO(), classicUrlClient, url)
	if err != nil {
		return nil, err
	}

	client := rest.NewRestClient(oauthClient, trafficLogger, rest.CreateRateLimitStrategy())
	clientClassic := rest.NewRestClient(tokenClient, trafficLogger, rest.CreateRateLimitStrategy())

	dtClient, err := dtclient.NewPlatformClient(
		url,
		classicURL,
		client,
		clientClassic,
		dtclient.WithCachingDisabled(opts.CachingDisabled),
		dtclient.WithAutoServerVersion(),
		dtclient.WithClientRequestLimiter(concurrency.NewLimiter(concurrentRequestLimit)),
		dtclient.WithCustomUserAgentString(opts.getUserAgentString()),
	)
	if err != nil {
		return nil, err
	}

	clientFactory := clients.Factory().
		WithOAuthCredentials(clientcredentials.Config{
			ClientID:     auth.OauthClientID,
			ClientSecret: auth.OauthClientSecret,
			TokenURL:     auth.OauthTokenURL,
		}).
		WithEnvironmentURL(url).
		WithUserAgent(opts.getUserAgentString())

	if opts.SupportArchive {
		clientFactory = clientFactory.WithHTTPListener(&lib.HTTPListener{Callback: trafficLogger.LogToFiles})
	}

	bucketClient, err := clientFactory.BucketClientWithRetrySettings(15, time.Second, 5*time.Minute)
	if err != nil {
		return nil, err
	}

	autClient, err := clientFactory.AutomationClient()
	if err != nil {
		return nil, err
	}

	return &ClientSet{
		DTClient:     dtClient,
		AutClient:    autClient,
		BucketClient: bucketClient,
	}, nil
}
