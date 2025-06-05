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

package resource

import (
	"context"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

type Deployable interface {
	// Enabled returns true if the Resource is enabled (FF is not false)
	Enabled() bool
	// VerifyAuth checks if the needed auth for the given configs is set (classic/platform)
	VerifyAuth(auth manifest.EnvironmentDefinition, configs []config.Config) error
	// VerifyConfigs checks if the given configs are valid for a resource (e.g., open pipeline: only one kind in one environment)
	VerifyConfigs(configs []config.Config) (bool, error)
	// Deploy deploys a given resource and returns the resolved entity
	Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error)
}

type Cachable interface {
	// Cache caches any needed data for a given configType
	Cache(c config.Type)
	// ClearCache Clears the cache that is filled by Cache
	ClearCache()
}

// HasConfigType returns true if the given config Type ID exists inside the configs slice
func HasConfigType(configs []config.Config, cfg config.TypeID) bool {
	for _, c := range configs {
		if !c.Skip && c.Type.ID() == cfg {
			return true
		}
	}
	return false
}

// CheckPlatformSetInManifest checks if any platform related authentication is set in the provided manifest definition.
// The config ID is used for adding context to possible errors
func CheckPlatformSetInManifest(ev manifest.EnvironmentDefinition, c config.TypeID) error {
	if err := ev.Auth.CheckPlatformSet(); err != nil {
		return fmt.Errorf("API of type '%s' for environment '%s' requires platform authentication: %w", c, ev.Name, err)
	}
	return nil
}
