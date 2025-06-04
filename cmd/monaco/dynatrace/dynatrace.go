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
	"fmt"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/supportarchive"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/apitoken"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/classicheartbeat"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"

	"golang.org/x/oauth2/clientcredentials"
)

var ErrorMissingAuth = errors.New("no authentication credentials provided")

// VerifyEnvironmentsAuthentication verifies that all environments can be reached with the defined credentials.
// It returns the first error encountered
func VerifyEnvironmentsAuthentication(ctx context.Context, envs manifest.EnvironmentDefinitionsByName) error {
	for _, env := range envs {
		if err := VerifyEnvironmentAuthentication(ctx, env); err != nil {
			return err
		}
	}
	return nil
}

// VerifyEnvironmentAuthentication checks if the provided token and OAuth credentials of the provided environment are valid.
func VerifyEnvironmentAuthentication(ctx context.Context, env manifest.EnvironmentDefinition) error {
	if env.Auth.Token == nil && env.Auth.OAuth == nil {
		return ErrorMissingAuth
	}

	classicUrl := env.URL.Value

	// check if the OAuth connection works and get the classicURL in order to check the token authentication next if given
	if env.Auth.OAuth != nil {
		oauthCreds := clientcredentials.Config{
			ClientID:     env.Auth.OAuth.ClientID.Value.Value(),
			ClientSecret: env.Auth.OAuth.ClientSecret.Value.Value(),
			TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
		}
		var err error
		if classicUrl, err = getDynatraceClassicURL(ctx, env.URL.Value, oauthCreds); err != nil {
			return fmt.Errorf("could not authorize against environment '%s' (%s) using OAuth authorization: %w", env.Name, env.URL.Value, err)
		}
	}

	if env.Auth.Token != nil {
		if err := checkClassicConnection(ctx, classicUrl, env.Auth.Token.Value.Value()); err != nil {
			return fmt.Errorf("could not authorize against environment '%s' (%s) using token authorization: %w", env.Name, classicUrl, err)
		}
	}
	return nil
}

// checkClassicConnection checks if a classic connection (via token) can be established. Scopes are not validated.
func checkClassicConnection(ctx context.Context, classicURL string, token string) error {
	factory := clients.Factory().
		WithClassicURL(classicURL).
		WithAccessToken(token).
		WithRateLimiter(true).
		WithRetryOptions(&client.DefaultRetryOptions)

	if supportarchive.IsEnabled(ctx) {
		factory = factory.WithHTTPListener(&corerest.HTTPListener{Callback: trafficlogs.GetInstance().LogToFiles})
	}

	client, err := factory.CreateClassicClient()
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	_, err = apitoken.GetTokenMetadata(ctx, client, token)
	return err
}

// CreateAccountClients gives back clients to use for specific accounts
func CreateAccountClients(ctx context.Context, manifestAccounts map[string]manifest.Account) (map[account.AccountInfo]*accounts.Client, error) {
	concurrentRequestLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)
	accClients := make(map[account.AccountInfo]*accounts.Client, len(manifestAccounts))
	for _, acc := range manifestAccounts {
		oauthCreds := clientcredentials.Config{
			ClientID:     acc.OAuth.ClientID.Value.Value(),
			ClientSecret: acc.OAuth.ClientSecret.Value.Value(),
			TokenURL:     acc.OAuth.GetTokenEndpointValue(),
		}

		factory := clients.Factory().
			WithConcurrentRequestLimit(concurrentRequestLimit).
			WithOAuthCredentials(oauthCreds).
			WithUserAgent(client.DefaultMonacoUserAgent).
			WithRateLimiter(true).
			WithRetryOptions(&client.DefaultRetryOptions).
			WithAccountURL(accountApiUrlOrDefault(acc.ApiUrl))

		if supportarchive.IsEnabled(ctx) {
			factory = factory.WithHTTPListener(&corerest.HTTPListener{Callback: trafficlogs.GetInstance().LogToFiles})
		}

		accClient, err := factory.AccountClient(ctx)
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

// accountApiUrlOrDefault returns the API URL if available or the default.
func accountApiUrlOrDefault(apiUrl *manifest.URLDefinition) string {
	if apiUrl == nil || apiUrl.Value == "" {
		return "https://api.dynatrace.com"
	}

	return apiUrl.Value
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
func CreateEnvironmentClients(ctx context.Context, environments manifest.EnvironmentDefinitionsByName, dryRun bool) (EnvironmentClients, error) {
	clients := make(EnvironmentClients, len(environments))
	for _, env := range environments {
		if dryRun {
			clients[EnvironmentInfo{
				Name:  env.Name,
				Group: env.Group,
			}] = &client.DummyClientSet
			continue
		}

		clientSet, err := client.CreateClientSet(ctx, env.URL.Value, env.Auth)
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

// getDynatraceClassicURL transforms the platformURL to a classic URL either via string replacing or API call, depending on if the BuildSimpleClassicURL FF is enabled (default) or not
func getDynatraceClassicURL(ctx context.Context, platformURL string, oauthCreds clientcredentials.Config) (string, error) {
	if featureflags.BuildSimpleClassicURL.Enabled() {
		if classicURL, ok := findSimpleClassicURL(ctx, platformURL); ok {
			return classicURL, nil
		}
	}

	factory := clients.Factory().WithPlatformURL(platformURL).WithOAuthCredentials(oauthCreds)
	if supportarchive.IsEnabled(ctx) {
		factory = factory.WithHTTPListener(&corerest.HTTPListener{Callback: trafficlogs.GetInstance().LogToFiles})
	}
	client, err := factory.CreatePlatformClient(ctx)
	if err != nil {
		return "", fmt.Errorf("could not create client: %w", err)
	}
	return metadata.GetDynatraceClassicURL(ctx, *client)
}

func findSimpleClassicURL(ctx context.Context, platformURL string) (classicUrl string, ok bool) {
	if !strings.Contains(platformURL, ".apps.") {
		log.Debug("Environment URL not matching expected Platform URL pattern, unable to build Classic environment URL directly.")
		return "", false
	}

	classicUrl = strings.Replace(platformURL, ".apps.", ".live.", 1)

	client, err := clients.Factory().WithClassicURL(classicUrl).CreateClassicClient()
	if err != nil {
		return "", false
	}

	if classicheartbeat.TestClassic(ctx, *client) {
		log.Debug("Found classic environment URL based on Platform URL: %s", classicUrl)
		return classicUrl, true
	}

	return "", false
}
