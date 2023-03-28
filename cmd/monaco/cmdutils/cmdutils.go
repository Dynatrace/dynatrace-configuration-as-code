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

package cmdutils

import (
	"context"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/cobra"
	"net/http"
)

// SilenceUsageCommand gives back a command that is just configured to skip printing of usage info.
// We use it as a PreRun hook to enforce the behavior of printing usage info when the command structure
// given by the user is faulty
func SilenceUsageCommand() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	}
}

// CreateDTClient is driven by data given through a manifest.EnvironmentDefinition to create an appropriate client.Client.
//
// In case when flag dryRun is true this factory returns the client.DummyClient.
func CreateDTClient(env manifest.EnvironmentDefinition, dryRun bool) (client.Client, error) {
	switch {
	case dryRun:
		return client.NewDummyClient(), nil
	case env.Type == manifest.Classic:
		return client.NewClassicClient(env.URL.Value, env.Auth.Token.Value)
	case env.Type == manifest.Platform:
		oauthCredentials := client.OauthCredentials{
			ClientID:     env.Auth.OAuth.ClientID.Value,
			ClientSecret: env.Auth.OAuth.ClientSecret.Value,
			TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
		}
		return client.NewPlatformClient(env.URL.Value, env.Auth.Token.Value, oauthCredentials)
	default:
		return nil, fmt.Errorf("unable to create authorizing HTTP Client for environment %s - no oauth credentials given", env.URL.Value)
	}
}

// VerifyEnvironmentGeneration takes a manifestEnvironments map and tries to verify that each environment can be reached
// using the configured credentials
func VerifyEnvironmentGeneration(envs manifest.Environments) bool {
	if featureflags.VerifyEnvironmentType().Enabled() {
		for _, env := range envs {
			switch env.Type {
			case manifest.Classic:
				return isClassicEnvironment(env)
			case manifest.Platform:
				return isPlatformEnvironment(env)
			default:
				log.Error("Could not authorize against the environment with name %q (%s). Unknown environment type.", env.Name, env.URL.Value)
				return false
			}
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
