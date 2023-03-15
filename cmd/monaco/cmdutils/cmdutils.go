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

var ssoTokenURL = "https://sso.dynatrace.com/sso/oauth2/token" //nolint:gosec

// VerifyClusterGen takes a manifestEnvironments map and tries to call the version endpoint of each environment
// in order to verify that the user has configured the environments correctly.
// Depending on the configured environment "type" the function tries to call the version endpoint of either
// classic gen or platform gen. The function will return an error as soon as
// it receives an error from calling the version endpoint of an environment
func VerifyClusterGen(envs manifest.Environments) error {
	for _, env := range envs {
		switch env.Type {
		case manifest.Classic:
			if _, err := client.GetDynatraceVersion(client.NewTokenAuthClient(env.Auth.Token.Value), client.Environment{URL: env.Url.Value, Type: client.Classic}); err != nil {
				return fmt.Errorf("could not verify Dynatrace cluster generation of environment %q (%q). Please check the configured Auth credentials in the manifest", env.Name, env.Url)
			}
		case manifest.Platform:
			oauthCredentials := client.OauthCredentials{
				ClientID:     env.Auth.OAuth.ClientId.Value,
				ClientSecret: env.Auth.OAuth.ClientSecret.Value,
				TokenURL:     ssoTokenURL,
			}
			if _, err := client.GetDynatraceVersion(client.NewOAuthClient(oauthCredentials), client.Environment{URL: env.Url.Value, Type: client.Platform}); err != nil {
				return fmt.Errorf("could not verify Dynatrace cluster generation of environment %q (%q). Please check the configured Auth credentials in the manifest", env.Name, env.Url)
			}
		default:
			return fmt.Errorf("invalid environment type")
		}
	}
	return nil
}
