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
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/supportarchive"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/accesstoken"
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

// VerifyEnvironmentAuthentication checks if the provided access token and platform credentials of the provided environment are valid.
func VerifyEnvironmentAuthentication(ctx context.Context, env manifest.EnvironmentDefinition) error {
	if env.Auth.AccessToken == nil && !env.HasPlatformCredentials() {
		return ErrorMissingAuth
	}

	classicUrl := env.URL.Value

	// check if the platform connection works and get the classicURL in order to check the access token authentication next if given
	if env.HasPlatformCredentials() {
		var err error
		if classicUrl, err = getDynatraceClassicURL(ctx, env.URL.Value, env.Auth.OAuth, env.Auth.PlatformToken); err != nil {
			return fmt.Errorf("could not authorize against environment '%s' (%s) using platform credentials: %w", env.Name, env.URL.Value, err)
		}
	}

	if env.Auth.AccessToken != nil {
		if err := checkClassicConnection(ctx, classicUrl, env.Auth.AccessToken.Value.Value()); err != nil {
			return fmt.Errorf("could not authorize against environment '%s' (%s) using access token authorization: %w", env.Name, classicUrl, err)
		}
	}
	return nil
}

// checkClassicConnection checks if a classic connection (via access token) can be established. Scopes are not validated.
func checkClassicConnection(ctx context.Context, classicURL string, accessToken string) error {
	additionalHeaders := environment.GetAdditionalHTTPHeadersFromEnv()
	factory := clients.Factory().
		WithClassicURL(classicURL).
		WithAccessToken(accessToken).
		WithRateLimiter(true).
		WithRetryOptions(&client.DefaultRetryOptions).
		WithCustomHeaders(additionalHeaders)

	if supportarchive.IsEnabled(ctx) {
		factory = factory.WithHTTPListener(&rest.HTTPListener{Callback: trafficlogs.GetInstance().LogToFiles})
	}

	client, err := factory.CreateClassicClientWithContext(ctx)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	_, err = accesstoken.GetAccessTokenMetadata(ctx, client, accessToken)
	return err
}

// CreateAccountClients gives back clients to use for specific accounts
func CreateAccountClients(ctx context.Context, manifestAccounts map[string]manifest.Account) (map[account.AccountInfo]*accounts.Client, error) {
	accClients := make(map[account.AccountInfo]*accounts.Client, len(manifestAccounts))
	for _, acc := range manifestAccounts {
		accClient, err := CreateAccountClient(ctx, acc)
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

// CreateAccountClient creates a client for the given account.
func CreateAccountClient(ctx context.Context, acc manifest.Account) (*accounts.Client, error) {
	oauthCreds := clientcredentials.Config{
		ClientID:     acc.OAuth.ClientID.Value.Value(),
		ClientSecret: acc.OAuth.ClientSecret.Value.Value(),
		TokenURL:     acc.OAuth.GetTokenEndpointValue(),
	}

	factory := clients.Factory().
		WithConcurrentRequestLimit(environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)).
		WithOAuthCredentials(oauthCreds).
		WithUserAgent(client.DefaultMonacoUserAgent).
		WithRateLimiter(true).
		WithRetryOptions(&rest.RetryOptions{DelayAfterRetry: 30 * time.Second, MaxRetries: 10, ShouldRetryFunc: rest.RetryIfTooManyRequests}).
		WithAccountURL(accountApiUrlOrDefault(acc.ApiUrl)).
		WithCustomHeaders(environment.GetAdditionalHTTPHeadersFromEnv())

	if supportarchive.IsEnabled(ctx) {
		factory = factory.WithHTTPListener(&rest.HTTPListener{Callback: trafficlogs.GetInstance().LogToFiles})
	}

	return factory.AccountClient(ctx)
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
func getDynatraceClassicURL(ctx context.Context, platformURL string, oauth *manifest.OAuth, platformToken *manifest.AuthSecret) (string, error) {
	if featureflags.BuildSimpleClassicURL.Enabled() {
		if classicURL, ok := findSimpleClassicURL(ctx, platformURL); ok {
			return classicURL, nil
		}
	}

	additionalHeaders := environment.GetAdditionalHTTPHeadersFromEnv()
	factory := clients.Factory().
		WithPlatformURL(platformURL).
		WithCustomHeaders(additionalHeaders)
	if platformToken != nil {
		factory = factory.WithPlatformToken(platformToken.Value.Value())
	}
	if oauth != nil {
		factory = factory.WithOAuthCredentials(clientcredentials.Config{
			ClientID:     oauth.ClientID.Value.Value(),
			ClientSecret: oauth.ClientSecret.Value.Value(),
			TokenURL:     oauth.GetTokenEndpointValue(),
		})
	}
	if supportarchive.IsEnabled(ctx) {
		factory = factory.WithHTTPListener(&rest.HTTPListener{Callback: trafficlogs.GetInstance().LogToFiles})
	}
	client, err := factory.CreatePlatformClient(ctx)
	if err != nil {
		return "", fmt.Errorf("could not create client: %w", err)
	}
	return metadata.GetDynatraceClassicURL(ctx, *client)
}

func findSimpleClassicURL(ctx context.Context, platformURL string) (classicUrl string, ok bool) {
	if !strings.Contains(platformURL, ".apps.") {
		log.DebugContext(ctx, "Environment URL not matching expected Platform URL pattern, unable to build Classic environment URL directly.")
		return "", false
	}

	additionalHeaders := environment.GetAdditionalHTTPHeadersFromEnv()
	classicUrl = strings.Replace(platformURL, ".apps.", ".live.", 1)

	client, err := clients.Factory().WithClassicURL(classicUrl).WithCustomHeaders(additionalHeaders).CreateClassicClientWithContext(ctx)
	if err != nil {
		return "", false
	}

	if classicheartbeat.TestClassic(ctx, *client) {
		log.DebugContext(ctx, "Found classic environment URL based on Platform URL: %s", classicUrl)
		return classicUrl, true
	}

	return "", false
}
