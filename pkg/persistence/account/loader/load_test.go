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

package loader

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("Load single file", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "test-resources/valid.yaml")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Account.Policies, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Account.Permissions, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment[0].Policies, 2)
		assert.Len(t, maps.Values(loaded.Groups)[0].Environment[0].Permissions, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].ManagementZone, 1)
		assert.Len(t, maps.Values(loaded.Groups)[0].ManagementZone[0].Permissions, 1)

	})

	t.Run("Load single file - with refs", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "test-resources/valid-with-refs.yaml")
		assert.NoError(t, err)
		assert.Len(t, loaded.Groups, 1)
		assert.NotNil(t, loaded.Groups["monaco-group"].Account)
		assert.Len(t, loaded.Groups["monaco-group"].Account.Policies, 2)
		assert.IsType(t, account.Reference{}, loaded.Groups["monaco-group"].Account.Policies[0])
		assert.IsType(t, "", loaded.Groups["monaco-group"].Account.Policies[1])
		assert.NotNil(t, loaded.Groups["monaco-group"].Environment)
		assert.Len(t, loaded.Groups["monaco-group"].Environment, 1)
		assert.Equal(t, "vsy13800", loaded.Groups["monaco-group"].Environment[0].Name)
		assert.Len(t, loaded.Groups["monaco-group"].Environment[0].Policies, 2)
		assert.IsType(t, account.Reference{}, loaded.Groups["monaco-group"].Environment[0].Policies[0])
		assert.IsType(t, "", loaded.Groups["monaco-group"].Environment[0].Policies[1])
		assert.Len(t, loaded.Policies, 2)
	})

	t.Run("Load multiple files", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "test-resources/multi")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
	})

	t.Run("Duplicate group", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "test-resources/duplicate-group.yaml")
		assert.Error(t, err)
	})

	t.Run("Duplicate user", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "test-resources/duplicate-user.yaml")
		assert.Error(t, err)
	})

	t.Run("Duplicate policy", func(t *testing.T) {
		_, err := Load(afero.NewOsFs(), "test-resources/duplicate-policy.yaml")
		assert.Error(t, err)
	})

	t.Run("root folder not found", func(t *testing.T) {
		result, err := Load(afero.NewOsFs(), "test-resources/non-existent-folder")
		assert.Equal(t, &account.AMResources{
			Policies: make(map[string]account.Policy, 0),
			Groups:   make(map[string]account.Group, 0),
			Users:    make(map[string]account.User, 0),
		}, result)
		assert.NoError(t, err)
	})
}
