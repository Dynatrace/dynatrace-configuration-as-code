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
	"errors"
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

// VerifyEnvironmentGeneration takes a manifestEnvironments map and tries to verify that each environment can be reached
// using the configured credentials
func VerifyEnvironmentGeneration(envs manifest.Environments) bool {
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
	return true
}

func isClassicEnvironment(env manifest.EnvironmentDefinition) bool {
	if _, err := client.GetDynatraceVersion(client.NewTokenAuthClient(env.Auth.Token.Value), env.URL.Value); err != nil {
		var respErr client.RespError
		if errors.As(err, &respErr) {
			log.Error("Could not authorize against the environment with name %q (%s) using token authorization.", env.Name, env.URL.Value)
			if respErr.StatusCode != http.StatusForbidden {
				log.Error("Please verify that this environment is a Dynatrace Classic environment.")
			} else {
				log.Error(err.Error())
			}
		} else {
			log.Error("Could not connect to environment %q (%s)", env.Name, env.URL.Value)
		}
		return false
	}
	return true
}

func isPlatformEnvironment(env manifest.EnvironmentDefinition) bool {
	oauthCredentials := client.OauthCredentials{
		ClientID:     env.Auth.OAuth.ClientID.Value,
		ClientSecret: env.Auth.OAuth.ClientSecret.Value,
		TokenURL:     env.Auth.OAuth.TokenEndpoint.Value,
	}
	if _, err := client.GetDynatraceClassicURL(client.NewOAuthClient(oauthCredentials), env.URL.Value); err != nil {
		var respErr client.RespError
		if errors.As(err, &respErr) {
			log.Error("Could not authorize against the environment with name %q (%s) using oAuth authorization.", env.Name, env.URL.Value)
			if respErr.StatusCode != http.StatusForbidden {
				log.Error("Please verify that this environment is a Dynatrace Platform environment.")
			} else {
				log.Error(err.Error())
			}
		} else {
			log.Error("Could not connect to environment %q (%s)", env.Name, env.URL.Value)
		}
		return false
	}
	return true
}
