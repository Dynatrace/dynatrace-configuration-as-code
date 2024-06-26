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

package dynatrace

import (
	"context"
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	lib "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/support"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/auth"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"golang.org/x/oauth2/clientcredentials"
)

// VerifyEnvironmentGeneration takes a manifestEnvironments map and tries to verify that each environment can be reached
// using the configured credentials
func VerifyEnvironmentGeneration(envs manifest.Environments) bool {
	if !featureflags.Permanent[featureflags.VerifyEnvironmentType].Enabled() {
		return true
	}
	for _, env := range envs {
		if (env.Auth.OAuth == nil && !isClassicEnvironment(env)) || (env.Auth.OAuth != nil && !isPlatformEnvironment(env)) {
			return false
		}
	}
	return true
}

func isClassicEnvironment(env manifest.EnvironmentDefinition) bool {
	if _, err := version.GetDynatraceVersion(context.TODO(), rest.NewRestClient(auth.NewTokenAuthClient(env.Auth.Token.Value.Value()), nil, rest.CreateRateLimitStrategy()), env.URL.Value); err != nil {
		var respErr rest.RespError
		if errors.As(err, &respErr) {
			log.WithFields(field.Error(err)).Error("Could not authorize against the environment with name %q (%s) using token authorization: %v", env.Name, env.URL.Value, err)
		} else {
			log.WithFields(field.Error(err)).Error("Could not connect to environment %q (%s): %v", env.Name, env.URL.Value, err)
		}
		log.Error("Please verify that this environment is a Dynatrace Classic environment.")
		return false
	}
	return true
}

func isPlatformEnvironment(env manifest.EnvironmentDefinition) bool {
	oauthCredentials := auth.OauthCredentials{
		ClientID:     env.Auth.OAuth.ClientID.Value.Value(),
		ClientSecret: env.Auth.OAuth.ClientSecret.Value.Value(),
		TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
	}
	if _, err := metadata.GetDynatraceClassicURL(context.TODO(), rest.NewRestClient(auth.NewOAuthClient(context.TODO(), oauthCredentials), nil, rest.CreateRateLimitStrategy()), env.URL.Value); err != nil {
		var respErr rest.RespError
		if errors.As(err, &respErr) {
			log.WithFields(field.Error(err)).Error("Could not authorize against the environment with name %q (%s) using oAuth authorization: %v", env.Name, env.URL.Value, err)
		} else {
			log.WithFields(field.Error(err)).Error("Could not connect to environment %q (%s): %v", env.Name, env.URL.Value, err)
		}
		log.Error("Please verify that this environment is a Dynatrace Platform environment.")
		return false
	}
	return true
}

// CreateClients creates a new client set based on the provided URL and authentication information.
func CreateClients(url string, auth manifest.Auth) (*client.ClientSet, error) {
	if auth.OAuth == nil {
		return client.CreateClassicClientSet(url, auth.Token.Value.Value(), client.ClientOptions{
			SupportArchive: support.SupportArchive,
		})
	}
	return client.CreatePlatformClientSet(url, client.PlatformAuth{
		OauthClientID:     auth.OAuth.ClientID.Value.Value(),
		OauthClientSecret: auth.OAuth.ClientSecret.Value.Value(),
		Token:             auth.Token.Value.Value(),
		OauthTokenURL:     auth.OAuth.GetTokenEndpointValue(),
	}, client.ClientOptions{
		SupportArchive: support.SupportArchive,
	})
}

// CreateAccountClients gives back clients to use for specific accounts
func CreateAccountClients(manifestAccounts map[string]manifest.Account) (map[account.AccountInfo]*accounts.Client, error) {
	concurrentRequestLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)
	accClients := make(map[account.AccountInfo]*accounts.Client, len(manifestAccounts))
	for _, acc := range manifestAccounts {
		oauthCreds := clientcredentials.Config{
			ClientID:     acc.OAuth.ClientID.Value.Value(),
			ClientSecret: acc.OAuth.ClientSecret.Value.Value(),
			TokenURL:     acc.OAuth.GetTokenEndpointValue(),
		}

		var apiUrl string
		if acc.ApiUrl == nil || acc.ApiUrl.Value == "" {
			apiUrl = "https://api.dynatrace.com"
		} else {
			apiUrl = acc.ApiUrl.Value
		}

		factory := clients.Factory()
		if support.SupportArchive {
			factory = factory.WithHTTPListener(&lib.HTTPListener{Callback: trafficlogs.NewFileBased().LogToFiles})
		}

		accClient, err := factory.
			WithConcurrentRequestLimit(concurrentRequestLimit).
			WithOAuthCredentials(oauthCreds).
			WithUserAgent(client.DefaultMonacoUserAgent).
			AccountClient(apiUrl)

		if err != nil {
			return accClients, err
		}
		accClients[account.AccountInfo{
			Name:        acc.Name,
			AccountUUID: acc.AccountUUID.String(),
		}] = accClient
	}
	return accClients, nil
}

type (
	// EnvironmentInfo environment specific information
	EnvironmentInfo struct {
		Name  string
		Group string
	}
	// EnvironmentClients is a collection of clients to use for specific environments
	EnvironmentClients map[EnvironmentInfo]*client.ClientSet
)

// Names gives back all environment Names for which the EnvironmentClients has a client sets
func (e EnvironmentClients) Names() []string {
	n := make([]string, 0, len(e))
	for k := range e {
		n = append(n, k.Name)
	}
	return n
}

// CreateEnvironmentClients gives back clients to use for specific environments
func CreateEnvironmentClients(environments manifest.Environments) (EnvironmentClients, error) {
	clients := make(EnvironmentClients, len(environments))
	for _, env := range environments {

		clientSet, err := CreateClients(env.URL.Value, env.Auth)
		if err != nil {
			return EnvironmentClients{}, err
		}
		clients[EnvironmentInfo{
			Name:  env.Name,
			Group: env.Group,
		}] = clientSet
	}

	return clients, nil
}
