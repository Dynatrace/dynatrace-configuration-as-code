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

package resource_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource"
)

var (
	oAuthEnv = manifest.EnvironmentDefinition{
		Name: "env",
		Auth: manifest.Auth{
			OAuth: &manifest.OAuth{
				ClientID:     manifest.AuthSecret{},
				ClientSecret: manifest.AuthSecret{},
			},
		},
	}
	tokenEnv = manifest.EnvironmentDefinition{
		Name: "env",
		Auth: manifest.Auth{
			Token: &manifest.AuthSecret{},
		},
	}
	sloConfigs = []config.Config{
		{
			Type: config.ServiceLevelObjective{},
		},
	}
)

func TestCheckPlatformSetInManifest(t *testing.T) {
	t.Run("returns error if platform connection is not set", func(t *testing.T) {
		err := resource.CheckPlatformSetInManifest(tokenEnv, config.ServiceLevelObjectiveID)
		assert.Error(t, err)
	})

	t.Run("returns nil if platform connection is set", func(t *testing.T) {
		err := resource.CheckPlatformSetInManifest(oAuthEnv, config.ServiceLevelObjectiveID)
		assert.NoError(t, err)
	})
}

func TestHasConfigType(t *testing.T) {
	skippedSloConfigs := []config.Config{
		{
			Type: config.ServiceLevelObjective{},
			Skip: true,
		},
	}
	t.Run("returns true if there are matching configs", func(t *testing.T) {
		hasConfigs := resource.HasConfigType(sloConfigs, config.ServiceLevelObjectiveID)
		assert.True(t, hasConfigs)
	})

	t.Run("returns false if there are no matching configs", func(t *testing.T) {
		hasConfigs := resource.HasConfigType(sloConfigs, config.SegmentID)
		assert.False(t, hasConfigs)
	})

	t.Run("returns false if there are matching configs but they are skipped", func(t *testing.T) {
		hasConfigs := resource.HasConfigType(skippedSloConfigs, config.ServiceLevelObjectiveID)
		assert.False(t, hasConfigs)
	})
}

func TestDefaultPlatformVerify(t *testing.T) {
	t.Run("returns false and no error if there aren't any related configs even if auth is invalid", func(t *testing.T) {
		hasConfigs, err := resource.DefaultPlatformVerify(tokenEnv, sloConfigs, config.SegmentID)

		assert.NoError(t, err)
		assert.False(t, hasConfigs)
		assert.Error(t, resource.CheckPlatformSetInManifest(tokenEnv, config.ServiceLevelObjectiveID))
	})

	t.Run("returns true and error if there are matching configs, but auth is invalid", func(t *testing.T) {
		hasConfigs, err := resource.DefaultPlatformVerify(tokenEnv, sloConfigs, config.ServiceLevelObjectiveID)

		assert.Error(t, err)
		assert.True(t, hasConfigs)
	})

	t.Run("returns true and no error if auth is valid and there are matching configs", func(t *testing.T) {
		hasConfigs, err := resource.DefaultPlatformVerify(oAuthEnv, sloConfigs, config.ServiceLevelObjectiveID)

		assert.NoError(t, err)
		assert.True(t, hasConfigs)
	})
}
