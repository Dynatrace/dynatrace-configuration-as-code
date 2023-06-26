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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/auth"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/metadata"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"net/http"
)

// VerifyEnvironmentGeneration takes a manifestEnvironments map and tries to verify that each environment can be reached
// using the configured credentials
func VerifyEnvironmentGeneration(envs manifest.Environments) bool {
	if !featureflags.VerifyEnvironmentType().Enabled() {
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
	if _, err := version.GetDynatraceVersion(auth.NewTokenAuthClient(env.Auth.Token.Value), env.URL.Value); err != nil {
		var respErr clientErrors.RespError
		if errors.As(err, &respErr) {
			log.Error("Could not authorize against the environment with name %q (%s) using token authorization.", env.Name, env.URL.Value)
			if respErr.StatusCode != http.StatusForbidden && respErr.StatusCode != http.StatusUnauthorized {
				log.Error("Please verify that this environment is a Dynatrace Classic environment.")
			} else {
				log.Error(err.Error())
			}
		} else {
			log.Error("Could not connect to environment %q (%s): %v", env.Name, env.URL.Value, err)
		}
		return false
	}
	return true
}

func isPlatformEnvironment(env manifest.EnvironmentDefinition) bool {
	oauthCredentials := auth.OauthCredentials{
		ClientID:     env.Auth.OAuth.ClientID.Value,
		ClientSecret: env.Auth.OAuth.ClientSecret.Value,
		TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
	}
	if _, err := metadata.GetDynatraceClassicURL(auth.NewOAuthClient(context.TODO(), oauthCredentials), env.URL.Value); err != nil {
		var respErr clientErrors.RespError
		if errors.As(err, &respErr) {
			log.Error("Could not authorize against the environment with name %q (%s) using oAuth authorization.", env.Name, env.URL.Value)
			if respErr.StatusCode != http.StatusForbidden && respErr.StatusCode != http.StatusUnauthorized {
				log.Error("Please verify that this environment is a Dynatrace Platform environment.")
			} else {
				log.Error(err.Error())
			}
		} else {
			log.Error("Could not connect to environment %q (%s): %v", env.Name, env.URL.Value, err)
		}
		return false
	}
	return true
}

func CreateClientSet(url string, auth manifest.Auth) (*client.ClientSet, error) {
	if auth.OAuth == nil {
		return client.CreateClassicClientSet(url, auth.Token.Value)
	}
	return client.CreatePlatformClientSet(url, client.PlatformAuth{
		OauthClientID:     auth.OAuth.ClientID.Value,
		OauthClientSecret: auth.OAuth.ClientSecret.Value,
		Token:             auth.Token.Value,
		OauthTokenURL:     auth.OAuth.GetTokenEndpointValue(),
	})

}
