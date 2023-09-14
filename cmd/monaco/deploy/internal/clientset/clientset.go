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

package clientset

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

func NewEnvironmentClients(environments manifest.Environments, dryRun bool) (deploy.EnvironmentClients, error) {
	clients := make(deploy.EnvironmentClients, len(environments))
	for _, env := range environments {
		clientSet, err := NewClientSet(env, dryRun)
		if err != nil {
			return deploy.EnvironmentClients{}, err
		}

		clients[deploy.EnvironmentInfo{
			Name:  env.Name,
			Group: env.Group,
		}] = clientSet
	}

	return clients, nil
}

func NewClientSet(env manifest.EnvironmentDefinition, dryRun bool) (deploy.ClientSet, error) {
	if dryRun {
		return deploy.DummyClientSet, nil
	}

	cl, err := dynatrace.CreateClientSet(env.URL.Value, env.Auth)
	if err != nil {
		return deploy.ClientSet{}, err
	}

	return deploy.ClientSet{
		Classic:    cl.Classic(),
		Settings:   cl.Settings(),
		Automation: cl.Automation(),
		Bucket:     cl.Bucket(),
	}, nil
}
