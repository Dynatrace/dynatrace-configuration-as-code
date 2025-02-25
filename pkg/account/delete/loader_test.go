//go:build unit

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

package delete_test

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/delete"
)

func TestLoader_BasicAllTypesSucceeds(t *testing.T) {
	t.Setenv(featureflags.ServiceUsers.EnvName(), "true")
	fs, deleteFilename := newMemMapFsWithDeleteFile(t, `delete:
  - type: user
    email: test.account.1@user.com
  - type: service-user
    name: my-service-user
  - type: group
    name: Log viewer
  - type: policy
    name: AppEngine - Admin
    level:
      type: account
  - type: policy
    name: AppEngine - Admin
    level:
      type: environment
      environment: vuc`)

	resources, err := delete.LoadResourcesToDelete(fs, deleteFilename)
	assert.NoError(t, err)
	expectedResources := delete.Resources{
		Users: []delete.User{
			{
				Email: "test.account.1@user.com",
			},
		},
		ServiceUsers: []delete.ServiceUser{
			delete.ServiceUser{
				Name: "my-service-user",
			},
		},
		Groups: []delete.Group{
			{
				Name: "Log viewer",
			},
		},
		AccountPolicies: []delete.AccountPolicy{
			{
				Name: "AppEngine - Admin",
			},
		},
		EnvironmentPolicies: []delete.EnvironmentPolicy{
			{
				Name:        "AppEngine - Admin",
				Environment: "vuc",
			},
		},
	}
	assert.Equal(t, expectedResources, resources)
}

func TestLoader_ServiceUserProducesErrorWithoutFeatureFlag(t *testing.T) {
	fs, deleteFilename := newMemMapFsWithDeleteFile(t, `delete:
  - type: service-user
    name: my-service-user`)

	_, err := delete.LoadResourcesToDelete(fs, deleteFilename)
	assert.Error(t, err)
}

func TestLoader_NoEntriesSucceeds(t *testing.T) {
	fs, deleteFilename := newMemMapFsWithDeleteFile(t, `delete:`)

	resources, err := delete.LoadResourcesToDelete(fs, deleteFilename)
	assert.NoError(t, err)
	assert.Empty(t, resources)
}

func TestLoader_EmptyFileProducesError(t *testing.T) {
	fs, deleteFilename := newMemMapFsWithDeleteFile(t, ``)

	_, err := delete.LoadResourcesToDelete(fs, deleteFilename)
	assert.Error(t, err)
}

func TestLoader_ConfigDeleteFileProducesError(t *testing.T) {
	fs, deleteFilename := newMemMapFsWithDeleteFile(t, `delete:
- "management-zone/test entity/entities"
- project: some-project
  type: builtin:auto.tagging
  id: my-tag
`)

	_, err := delete.LoadResourcesToDelete(fs, deleteFilename)
	assert.Error(t, err)
}

func TestLoader_UnknownTypeProducesError(t *testing.T) {
	fs, deleteFilename := newMemMapFsWithDeleteFile(t, `delete:
  - type: unknown-type
    email: not.theres@yet.com`)

	_, err := delete.LoadResourcesToDelete(fs, deleteFilename)
	assert.Error(t, err)
}

func newMemMapFsWithDeleteFile(t *testing.T, contents string) (afero.Fs, string) {
	fs := afero.NewMemMapFs()
	filename, _ := filepath.Abs("delete.yaml")
	err := afero.WriteFile(fs, filename, []byte(contents), 0777)
	assert.NoError(t, err)
	return fs, filename
}
