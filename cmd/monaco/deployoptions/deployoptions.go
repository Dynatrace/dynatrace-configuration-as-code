/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package deployoptions

import (
	"context"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
)

// DeployConfigsOptions defines additional options used by DeployConfigs
type DeployConfigsOptions struct {
	// ContinueOnErr states that the deployment continues even when there happens to be an
	// error while deploying a certain configuration
	ContinueOnErr bool
	// DryRun states that the deployment shall just run in dry-run mode, meaning
	// that actual deployment of the configuration to a tenant will be skipped
	DryRun bool
	// ConcurrentDeploymentsLimiter limits the amount of concurrent deployments
	ConcurrentDeploymentsLimiter *rest.ConcurrentRequestLimiter
}

type ctxDeploymentOptionsKey struct{}

// NewContextWithDeployOptions creates a new child context that contains the deployment options
func NewContextWithDeployOptions(ctx context.Context, options DeployConfigsOptions) context.Context {
	return context.WithValue(ctx, ctxDeploymentOptionsKey{}, options)
}

func GetDeploymentOptionsFromContext(ctx context.Context) DeployConfigsOptions {
	if val, ok := ctx.Value(ctxDeploymentOptionsKey{}).(DeployConfigsOptions); ok {
		return val
	}
	return DeployConfigsOptions{}
}
