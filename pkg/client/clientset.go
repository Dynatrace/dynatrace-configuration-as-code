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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	clientAuth "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/auth"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"runtime"
)

// ClientSet composes a "full" set of sub-clients to access Dynatrace APIs
// Each field may be nil, if the ClientSet is partially initialized - e.g. no autClient will be part of a ClientSet
// created for a 'classic' Dynatrace environment, as Automations are a Platform feature
type ClientSet struct {
	// dtClient is the client capable of updating or creating settings and classic configs
	dtClient *dtclient.DynatraceClient
	// autClient is the client capable of updating or creating automation API configs
	autClient *automation.Client

	bucketClient *bucket.Client
}

func (s ClientSet) Classic() *dtclient.DynatraceClient {
	return s.dtClient
}

func (s ClientSet) Settings() *dtclient.DynatraceClient {
	return s.dtClient
}

func (s ClientSet) Automation() *automation.Client {
	return s.autClient
}

func (s ClientSet) Entities() *dtclient.DynatraceClient {
	return s.dtClient
}

func (s ClientSet) Bucket() *bucket.Client {
	return s.bucketClient
}

type ClientOptions struct {
	CustomUserAgent string
	SupportArchive  bool
	CachingDisabled bool
}

func (o ClientOptions) getUserAgentString() string {
	if o.CustomUserAgent == "" {
		return "Dynatrace Monitoring as Code/" + version.MonitoringAsCode + " " + (runtime.GOOS + " " + runtime.GOARCH)
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
		dtClient:  dtClient,
		autClient: nil,
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
	classicURL, err := metadata.GetDynatraceClassicURL(context.TODO(), rest.NewRestClient(oauthClient, nil, rest.CreateRateLimitStrategy()), url) //this will send the default user-agent
	if err != nil {
		return nil, err
	}

	var trafficLogger *trafficlogs.FileBasedLogger
	if opts.SupportArchive {
		trafficLogger = trafficlogs.NewFileBased()
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
	platformClient := rest.NewRestClient(clientAuth.NewOAuthClient(context.TODO(), oauthCredentials), trafficLogger, rest.CreateRateLimitStrategy())

	autClient := automation.NewClient(
		url,
		platformClient,
		automation.WithClientRequestLimiter(concurrency.NewLimiter(concurrentRequestLimit)),
		automation.WithCustomUserAgentString(opts.getUserAgentString()),
	)

	bucketClient := bucket.NewClient(url, platformClient)

	if err != nil {
		return nil, fmt.Errorf("unable to create API clients: %w", err)
	}

	return &ClientSet{
		dtClient:     dtClient,
		autClient:    autClient,
		bucketClient: bucketClient,
	}, nil
}
