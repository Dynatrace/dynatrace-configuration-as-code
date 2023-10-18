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

package account

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("Load single file", func(t *testing.T) {
		loaded, err := Load(afero.NewOsFs(), "test-resources/valid.yaml")
		assert.NoError(t, err)
		assert.Len(t, loaded.Users, 1)
		assert.Len(t, loaded.Groups, 1)
		assert.Len(t, loaded.Policies, 1)
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
		assert.Equal(t, &AMResources{
			Policies: make(map[string]Policy, 0),
			Groups:   make(map[string]Group, 0),
			Users:    make(map[string]User, 0),
		}, result)
		assert.NoError(t, err)
	})

}
