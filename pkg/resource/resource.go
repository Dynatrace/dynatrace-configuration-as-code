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
	// Verify verifies given auth types and configs:
	// 1. if the needed Auth is set (OAuth/token/platform)
	// 2. if the needed Auth for a certain config is set (e.g., permissions in settings require a platform connection)
	// 3. if the configs are valid (e.g., open pipeline: only one kind in one environment)
	Verify(ev manifest.EnvironmentDefinition, cfg []config.Config) (bool, error)
	// Deploy deploys a given resource and returns the resolved entity
	Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error)

	// Cache caches any needed data for a given configType and identifier (schema, api)
	Cache(c config.Type, identifier string)
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

// DefaultPlatformVerify returns if there are any configs to a given type and sets an error in case platform auth is not set
// if there aren't any related configs, the auth is not validated
func DefaultPlatformVerify(ev manifest.EnvironmentDefinition, configs []config.Config, cfg config.TypeID) (bool, error) {
	hasConfigs := HasConfigType(configs, cfg)
	if !hasConfigs {
		return hasConfigs, nil
	}
	if err := CheckPlatformSetInManifest(ev, cfg); err != nil {
		return hasConfigs, err
	}
	return hasConfigs, nil
}
