//go:build unit

// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deploy

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func Test_DoDeploy_InvalidManifest(t *testing.T) {
	t.Setenv("ENV_TOKEN", "mock env token")
	t.Setenv("ENV_URL", "https://example.com")

	manifestYaml := `manifestVersion: "1.0"`

	configYaml := `configs:
- id: profile
  config:
    name: alerting-profile
    template: profile.json
    skip: false
  type:
    api: alerting-profile
`
	testFs := afero.NewMemMapFs()
	// Create v1 configuration
	configPath, _ := filepath.Abs("project/alerting-profile/profile.yaml")
	_ = afero.WriteFile(testFs, configPath, []byte(configYaml), 0644)
	templatePath, _ := filepath.Abs("project/alerting-profile/profile.json")
	_ = afero.WriteFile(testFs, templatePath, []byte("{}"), 0644)
	manifestPath, _ := filepath.Abs("manifest.yaml")
	_ = afero.WriteFile(testFs, manifestPath, []byte(manifestYaml), 0644)

	err := deployConfigs(testFs, manifestPath, []string{}, []string{}, []string{}, true, true)
	assert.Error(t, err)
}

func Test_DoDeploy(t *testing.T) {
	t.Setenv("ENV_TOKEN", "mock env token")

	manifestYaml := `manifestVersion: "1.0"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: project
    url:
      value: https://abcde.dev.dynatracelabs.com
    auth:
      token:
        type: environment
        name: ENV_TOKEN
`
	configYaml := `configs:
- id: profile
  config:
    name: alerting-profile
    template: profile.json
    skip: false
  type:
    api: alerting-profile
`
	testFs := afero.NewMemMapFs()
	// Create v1 configuration
	configPath, _ := filepath.Abs("project/alerting-profile/profile.yaml")
	_ = afero.WriteFile(testFs, configPath, []byte(configYaml), 0644)
	templatePath, _ := filepath.Abs("project/alerting-profile/profile.json")
	_ = afero.WriteFile(testFs, templatePath, []byte("{}"), 0644)

	manifestPath, _ := filepath.Abs("manifest.yaml")
	_ = afero.WriteFile(testFs, manifestPath, []byte(manifestYaml), 0644)

	t.Run("Wrong environment group", func(t *testing.T) {
		err := deployConfigs(testFs, manifestPath, []string{"NOT_EXISTING_GROUP"}, []string{}, []string{}, true, true)
		assert.Error(t, err)
	})
	t.Run("Wrong environment name", func(t *testing.T) {
		err := deployConfigs(testFs, manifestPath, []string{"default"}, []string{"NOT_EXISTING_ENV"}, []string{}, true, true)
		assert.Error(t, err)
	})

	t.Run("Wrong project name", func(t *testing.T) {
		err := deployConfigs(testFs, manifestPath, []string{"default"}, []string{"project"}, []string{"NON_EXISTING_PROJECT"}, true, true)
		assert.Error(t, err)
	})

	t.Run("no parameters", func(t *testing.T) {
		err := deployConfigs(testFs, manifestPath, []string{}, []string{}, []string{}, true, true)
		assert.NoError(t, err)
	})

	t.Run("correct parameters", func(t *testing.T) {
		err := deployConfigs(testFs, manifestPath, []string{"default"}, []string{"project"}, []string{"project"}, true, true)
		assert.NoError(t, err)
	})

}
