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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/cobra"
)

// SilenceUsageCommand gives back a command that is just configured to skip printing of usage info.
// We use it as a PreRun hook to enforce the behavior of printing usage info when the command structure
// given by the user is faulty
func SilenceUsageCommand() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	}
}

// VerifyClusterGen takes a manifestEnvironments map and tries to call the version endpoint of each environment
// in order to verify that the user has configured the environments correctly.
// Depending on the configured environment "type" the function tries to call the version endpoint of either
// 2nd gen cluster (classic) or 3rd gen cluster (platform). The function will return an error as soon as
// it receives an error from calling the version endpoint of an environment
func VerifyClusterGen(envs manifest.Environments) error {
	for _, env := range envs {
		// Assume 2nd gen cluster and check version endpoint
		if env.Type == manifest.Classic {
			if _, err := client.GetDynatraceVersion2ndGen(client.NewTokenAuthClient(env.Auth.Token.Value), env.Url.Value); err != nil {
				log.Error("Could not verify Dynatrace cluster generation of environment %q (%q). Please check the configured Auth credentials in the manifest", env.Name, env.Url)
				return fmt.Errorf("unable to call version endpoint of environment %q: %w", env.Name, err)
			}
			return nil
		}

		// Assume 3rd gen cluster an check version endpoint
		if env.Type == manifest.Platform {
			oauthCredentials := client.OauthCredentials{
				ClientID:     env.Auth.OAuth.ClientId.Value,
				ClientSecret: env.Auth.OAuth.ClientSecret.Value,
				TokenURL:     "https://sso-dev.dynatracelabs.com/sso/oauth2/token",
			}
			if _, err := client.GetDynatraceVersion2ndGen(client.NewOAuthClient(oauthCredentials), env.Url.Value); err != nil {
				log.Error("Could not verify Dynatrace cluster generation of environment %q (%q). Please check the configured Auth credentials in the manifest", env.Name, env.Url)
				return fmt.Errorf("unable to call version endpoint of environment %q: %w", env.Name, err)
			}
		}
	}
	return nil
}
