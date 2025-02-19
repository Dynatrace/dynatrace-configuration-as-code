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
	"strings"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/supportarchive"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/classicheartbeat"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"

	"golang.org/x/oauth2/clientcredentials"
)

// VerifyEnvironmentGeneration takes a manifestEnvironments map and tries to verify that each environment can be reached
// using the configured credentials
func VerifyEnvironmentGeneration(ctx context.Context, envs manifest.Environments) bool {
	if !featureflags.VerifyEnvironmentType.Enabled() {
		return true
	}
	for _, env := range envs {
		if !isValidEnvironment(ctx, env) {
			return false
		}
	}
	return true
}

func isValidEnvironment(ctx context.Context, env manifest.EnvironmentDefinition) bool {
	if env.Auth.Token == nil && env.Auth.OAuth == nil {
		log.Error("No token and oAuth provided in manifest")
		return false
	}

	if env.Auth.OAuth == nil {
		return isClassicEnvironment(ctx, env)
	}

	return isPlatformEnvironment(ctx, env)
}

func isClassicEnvironment(ctx context.Context, env manifest.EnvironmentDefinition) bool {
	client, err := clients.Factory().
		WithClassicURL(env.URL.Value).
		WithAccessToken(env.Auth.Token.Value.Value()).
		WithRateLimiter(true).
		WithRetryOptions(&client.DefaultRetryOptions).
		CreateClassicClient()
	if err != nil {
		log.Error("Could not create client %q (%s): %v", env.Name, env.URL.Value, err)
		return false
	}

	if _, err := version.GetDynatraceVersion(ctx, client); err != nil {
		var apiErr coreapi.APIError
		if errors.As(err, &apiErr) {
			log.WithFields(field.Error(err)).Error("Could not authorize against the environment with name %q (%s) using token authorization: %v", env.Name, env.URL.Value, err)
		} else {
			log.WithFields(field.Error(err)).Error("Could not connect to environment %q (%s): %v", env.Name, env.URL.Value, err)
		}
		log.Error("Please verify that this environment is a Dynatrace Classic environment.")
		return false
	}
	return true
}

func isPlatformEnvironment(ctx context.Context, env manifest.EnvironmentDefinition) bool {
	oauthCreds := clientcredentials.Config{
		ClientID:     env.Auth.OAuth.ClientID.Value.Value(),
		ClientSecret: env.Auth.OAuth.ClientSecret.Value.Value(),
		TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
	}

	if _, err := getDynatraceClassicURL(ctx, env.URL.Value, oauthCreds); err != nil {
		var apiError coreapi.APIError
		if errors.As(err, &apiError) {
			log.WithFields(field.Error(err)).Error("Could not authorize against the environment with name %q (%s) using oAuth authorization: %v", env.Name, env.URL.Value, err)
		} else {
			log.WithFields(field.Error(err)).Error("Could not connect to environment %q (%s): %v", env.Name, env.URL.Value, err)
		}
		log.Error("Please verify that this environment is a Dynatrace Platform environment.")
		return false
	}
	return true
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

		accClient, err := factory.AccountClient()
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
func CreateEnvironmentClients(ctx context.Context, environments manifest.Environments, dryRun bool) (EnvironmentClients, error) {
	clients := make(EnvironmentClients, len(environments))
	for _, env := range environments {
		if dryRun {
			clients[EnvironmentInfo{
				Name:  env.Name,
				Group: env.Group,
			}] = &client.DryRunClientSet
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

func getDynatraceClassicURL(ctx context.Context, platformURL string, oauthCreds clientcredentials.Config) (string, error) {
	if featureflags.BuildSimpleClassicURL.Enabled() {
		if classicURL, ok := findSimpleClassicURL(ctx, platformURL); ok {
			return classicURL, nil
		}
	}

	client, err := clients.Factory().WithPlatformURL(platformURL).WithOAuthCredentials(oauthCreds).CreatePlatformClient()
	if err != nil {
		return "", err
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
