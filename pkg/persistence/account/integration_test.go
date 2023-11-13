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

package account_test

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/writer"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestLoadAndReWriteAccountResources(t *testing.T) {
	testResources := "loader/test-resources/multi"
	fs := afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())

	// LOAD RESOURCES FROM DISK
	resources, err := loader.Load(fs, testResources)
	assert.NoError(t, err)

	assert.NotEmpty(t, resources.Groups)
	assert.NotEmpty(t, resources.Policies)
	assert.NotEmpty(t, resources.Users)

	// TRANSFORM TO IN_MEMORY
	//TODO: remove when loader returns in memory model
	inMemResources := toInMemRepresentation(*resources)

	// WRITE IN-MEMORY REPRESENTATION TO DISK
	c := writer.Context{
		Fs:            fs,
		OutputFolder:  "test-folder",
		ProjectFolder: "test-project",
	}
	err = writer.WriteAccountResources(c, inMemResources)
	assert.NoError(t, err)

	// ASSERT FILES WRITTEN AS EXPECTED
	expectedOutputFolder, err := filepath.Abs(filepath.Join(c.OutputFolder, c.ProjectFolder))
	assert.NoError(t, err)
	assertFileExists(t, c.Fs, filepath.Join(expectedOutputFolder, "users.yaml"))
	assertFileExists(t, c.Fs, filepath.Join(expectedOutputFolder, "groups.yaml"))
	assertFileExists(t, c.Fs, filepath.Join(expectedOutputFolder, "policies.yaml"))

	// ASSERT WRITTEN FILES MATCH ORIGINALS AFTER LOADING THEM FROM DISK
	writtenResources, err := loader.Load(fs, expectedOutputFolder)
	assert.NoError(t, err)
	assert.Equal(t, resources.Groups, writtenResources.Groups)
	assert.Equal(t, resources.Policies, writtenResources.Policies)
	assert.Equal(t, resources.Users, writtenResources.Users)
}

func toInMemRepresentation(resources persistence.AMResources) account.Resources {
	inMemResources := account.Resources{
		Policies: make(map[account.PolicyId]account.Policy),
		Groups:   make(map[account.GroupId]account.Group),
		Users:    make(map[account.UserId]account.User),
	}
	for id, v := range resources.Policies {
		inMemResources.Policies[id] = account.Policy{
			ID:          v.ID,
			Name:        v.Name,
			Level:       v.Level,
			Description: v.Description,
			Policy:      v.Policy,
		}
	}
	for id, v := range resources.Groups {
		var acc *account.Account
		if v.Account != nil {
			acc = &account.Account{
				Permissions: v.Account.Permissions,
				Policies:    v.Account.Policies,
			}
		}
		env := make([]account.Environment, len(v.Environment))
		for i, e := range v.Environment {
			env[i] = account.Environment{
				Name:        e.Name,
				Permissions: e.Permissions,
				Policies:    e.Policies,
			}
		}
		mz := make([]account.ManagementZone, len(v.ManagementZone))
		for i, m := range v.ManagementZone {
			mz[i] = account.ManagementZone{
				Environment:    m.Environment,
				ManagementZone: m.ManagementZone,
				Permissions:    m.Permissions,
			}
		}
		inMemResources.Groups[id] = account.Group{
			ID:             v.ID,
			Name:           v.Name,
			Description:    v.Description,
			Account:        acc,
			Environment:    env,
			ManagementZone: mz,
		}
	}
	for id, v := range resources.Users {
		inMemResources.Users[id] = account.User{
			Email:  v.Email,
			Groups: v.Groups,
		}
	}
	return inMemResources
}

func assertFileExists(t *testing.T, fs afero.Fs, path string) {
	exists, err := afero.Exists(fs, path)
	assert.NoError(t, err)
	assert.True(t, exists, "expected file to exist %v", path)
}
