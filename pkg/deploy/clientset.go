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

package deploy

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

func createClientSet(env manifest.EnvironmentDefinition, opts DeployConfigsOptions) (ClientSet, error) {

	if opts.DryRun {
		return DummyClientSet, nil
	}

	cl, err := createClients(env.URL.Value, env.Auth, opts)
	if err != nil {
		return ClientSet{}, err
	}

	return ClientSet{
		Classic:    cl.Classic(),
		Settings:   cl.Settings(),
		Automation: cl.Automation(),
	}, nil
}

func createClients(url string, auth manifest.Auth, opts DeployConfigsOptions) (*client.ClientSet, error) {

	if auth.OAuth == nil {
		return client.CreateClassicClientSet(url, auth.Token.Value, client.ClientOptions{
			SupportArchive: opts.SupportArchive,
		})
	}

	return client.CreatePlatformClientSet(url, client.PlatformAuth{
		OauthClientID:     auth.OAuth.ClientID.Value,
		OauthClientSecret: auth.OAuth.ClientSecret.Value,
		Token:             auth.Token.Value,
		OauthTokenURL:     auth.OAuth.GetTokenEndpointValue(),
	}, client.ClientOptions{
		SupportArchive: opts.SupportArchive,
	})
}
