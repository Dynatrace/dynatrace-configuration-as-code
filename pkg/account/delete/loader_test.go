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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/delete"
)

func TestLoadEntriesToDelete(t *testing.T) {
	tests := []struct {
		name                   string
		givenDeleteFileContent string
		want                   delete.Resources
		wantErr                bool
	}{
		{
			"basic all types file",
			`delete:
  - type: user
    email: test.account.1@ruxitlabs.com
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
      environment: vuc`,
			delete.Resources{
				Users: []delete.User{
					{
						Email: "test.account.1@ruxitlabs.com",
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
			},
			false,
		},
		{
			"empty delete entries are ok",
			"delete:",
			delete.Resources{},
			false,
		},
		{
			"empty delete file is wrong",
			"",
			delete.Resources{},
			true,
		},
		{
			"config delete file produces error",
			`delete:
- "management-zone/test entity/entities"
- project: some-project
  type: builtin:auto.tagging
  id: my-tag
`,
			delete.Resources{},
			true,
		},
		{
			"unknown type produces error",
			`delete:
  - type: magic
    email: there-are-no-service-users@yet.com`,
			delete.Resources{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			filename, _ := filepath.Abs("delete.yaml")
			err := afero.WriteFile(fs, filename, []byte(tt.givenDeleteFileContent), 0777)
			assert.NoError(t, err)

			got, err := delete.LoadResourcesToDelete(fs, filename)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.ElementsMatch(t, tt.want.Users, got.Users)
			assert.ElementsMatch(t, tt.want.AccountPolicies, got.AccountPolicies)
			assert.ElementsMatch(t, tt.want.EnvironmentPolicies, got.EnvironmentPolicies)
			assert.ElementsMatch(t, tt.want.Groups, got.Groups)
		})
	}
}
