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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/environment"
	clientAuth "github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/auth"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
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

func CreateClassicClientSet(url string, token string) (*ClientSet, error) {
	concurrentRequestLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)

	dtClient, err := dtclient.NewClassicClient(
		url,
		token,
		dtclient.WithAutoServerVersion(),
		dtclient.WithClientRequestLimiter(concurrency.NewLimiter(concurrentRequestLimit)),
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

func CreatePlatformClientSet(url string, auth PlatformAuth) (*ClientSet, error) {
	concurrentRequestLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)

	oauthCredentials := clientAuth.OauthCredentials{
		ClientID:     auth.OauthClientID,
		ClientSecret: auth.OauthClientSecret,
		TokenURL:     auth.OauthTokenURL,
	}
	dtClient, err := dtclient.NewPlatformClient(
		url,
		auth.Token,
		oauthCredentials,
		dtclient.WithAutoServerVersion(),
		dtclient.WithClientRequestLimiter(concurrency.NewLimiter(concurrentRequestLimit)),
	)
	autClient := automation.NewClient(
		url,
		clientAuth.NewOAuthClient(context.TODO(), oauthCredentials),
		automation.WithClientRequestLimiter(concurrency.NewLimiter(concurrentRequestLimit)),
		automation.WithCustomUserAgentString("Dynatrace Monitoring as Code/"+version.MonitoringAsCode+" "+(runtime.GOOS+" "+runtime.GOARCH)),
	)

	if err != nil {
		return nil, fmt.Errorf("unable to create API clients: %w", err)
	}

	return &ClientSet{
		dtClient:  dtClient,
		autClient: autClient,
	}, nil
}
