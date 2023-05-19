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

package deploy

import (
	"context"
	"fmt"
	clientAuth "github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/auth"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
)

type clientSet struct {
	// dtClient is the client capable of updating or creating settings and classic configs
	dtClient dtclient.Client
	// autClient is the client capable of updating or creating automation API configs
	autClient automationClient
}

func (cs clientSet) automation() automationClient {
	return cs.autClient
}

func (cs clientSet) settings() dtclient.Client {
	return cs.dtClient
}

func (cs clientSet) classic() dtclient.Client {
	return cs.dtClient
}

func NewClientSet(dtClient dtclient.Client, autClient *automation.Client) *clientSet {
	return &clientSet{
		dtClient:  dtClient,
		autClient: autClient,
	}
}

func CreateClientSet(url string, auth manifest.Auth, dryRun bool) (*clientSet, error) {
	var dtClient dtclient.Client
	var autClient automationClient
	var err error

	switch {
	case dryRun:
		dtClient = &dtclient.DummyClient{}
		autClient = &dummyAutomationClient{}
	case auth.OAuth == nil:
		dtClient, err = dtclient.NewClassicClient(url, auth.Token.Value)
		autClient = &dummyAutomationClient{}
	default:
		oauthCredentials := clientAuth.OauthCredentials{
			ClientID:     auth.OAuth.ClientID.Value,
			ClientSecret: auth.OAuth.ClientSecret.Value,
			TokenURL:     auth.OAuth.GetTokenEndpointValue(),
		}
		dtClient, err = dtclient.NewPlatformClient(url, auth.Token.Value, oauthCredentials)
		autClient = automation.NewClient(url, clientAuth.NewOAuthClient(context.TODO(), oauthCredentials))
	}
	if err != nil {
		return nil, fmt.Errorf("unable to create API clients: %w", err)
	}

	return &clientSet{
		dtClient:  dtClient,
		autClient: autClient,
	}, nil

}
