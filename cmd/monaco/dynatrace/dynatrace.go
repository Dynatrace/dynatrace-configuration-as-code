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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"net/http"
)

// CreateDTClient is driven by data given through a manifest.EnvironmentDefinition to create an appropriate client.Client.
//
// In case when flag dryRun is true this factory returns the client.DummyClient.
func CreateDTClient(url string, a manifest.Auth, dryRun bool, opts ...func(dynatraceClient *dtclient.DynatraceClient)) (dtclient.Client, error) {
	switch {
	case dryRun:
		return dtclient.NewDummyClient(), nil
	case a.OAuth == nil:
		return dtclient.NewClassicClient(url, a.Token.Value, opts...)
	case a.OAuth != nil:
		oauthCredentials := client.OauthCredentials{
			ClientID:     a.OAuth.ClientID.Value,
			ClientSecret: a.OAuth.ClientSecret.Value,
			TokenURL:     a.OAuth.GetTokenEndpointValue(),
		}
		return dtclient.NewPlatformClient(url, a.Token.Value, oauthCredentials, opts...)
	default:
		return nil, fmt.Errorf("unable to create authorizing HTTP Client for environment %s - no oauth credentials given", url)
	}
}

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
	if _, err := client.GetDynatraceVersion(client.NewTokenAuthClient(env.Auth.Token.Value), env.URL.Value); err != nil {
		var respErr client.RespError
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
	oauthCredentials := client.OauthCredentials{
		ClientID:     env.Auth.OAuth.ClientID.Value,
		ClientSecret: env.Auth.OAuth.ClientSecret.Value,
		TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
	}
	if _, err := client.GetDynatraceClassicURL(client.NewOAuthClient(context.TODO(), oauthCredentials), env.URL.Value); err != nil {
		var respErr client.RespError
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

func CreateAutomation(url string, a manifest.Auth) (*deploy.Automation, error) {
	switch {
	//case dryRun:
	// TODO: do we need dry run?
	//case a.OAuth == nil:
	// TODO: just to print warning or return an error?
	case a.OAuth != nil:
		oauthCredentials := client.OauthCredentials{
			ClientID:     a.OAuth.ClientID.Value,
			ClientSecret: a.OAuth.ClientSecret.Value,
			TokenURL:     a.OAuth.GetTokenEndpointValue(),
		}
		c := automation.NewClient(url, client.NewOAuthClient(context.TODO(), oauthCredentials))
		return deploy.New(c)
	default:
		return nil, fmt.Errorf("unable to create authorizing HTTP Client for environment %s - no oauth credentials given", url)
	}
}
