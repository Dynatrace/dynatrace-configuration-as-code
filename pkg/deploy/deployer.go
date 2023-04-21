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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
)

type Deployer struct {
	dtClient dtclient.Client

	// automation is nil in case of dealing with a classical DT instance
	automation *Automation

	apis api.APIs
	// continueOnErr states that the deployment continues even when there happens to be an
	// error while deploying a certain configuration
	continueOnErr bool
	// dryRun states that the deployment shall just run in dry-run mode, meaning
	// that actual deployment of the configuration to a tenant will be skipped
	dryRun bool
}

type DeployerOptions func(d *Deployer)

func NewDeployer(dtClient dtclient.Client, automation *Automation, opts ...DeployerOptions) *Deployer {
	d := new(Deployer)

	d.dtClient = dtClient
	d.automation = automation
	d.apis = api.NewAPIs()

	for _, o := range opts {
		o(d)
	}

	return d
}

func WithContinueOnErr(b ...bool) DeployerOptions {
	return func(d *Deployer) {
		if len(b) > 0 {
			d.continueOnErr = b[0]
		}
		d.continueOnErr = true
	}
}

func WithDryRun(b ...bool) DeployerOptions {
	return func(d *Deployer) {
		if len(b) > 0 {
			d.dryRun = b[0]
		}
		d.dryRun = true
	}
}

func WithAPIs(apis api.APIs) DeployerOptions {
	return func(d *Deployer) {
		d.apis = apis
	}
}
