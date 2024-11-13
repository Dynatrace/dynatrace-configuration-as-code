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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
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

func Test_checkEnvironments(t *testing.T) {

	env1Id := "env1"
	env1Definition :=
		manifest.EnvironmentDefinition{
			Name: env1Id,
			Auth: manifest.Auth{OAuth: &manifest.OAuth{ClientID: manifest.AuthSecret{Name: "id", Value: "value"}, ClientSecret: manifest.AuthSecret{Name: "id", Value: "value"}}},
		}

	env1DefinitionWithoutPlatform :=
		manifest.EnvironmentDefinition{
			Name: env1Id,
		}

	env2Id := "env2"
	env2Definition :=
		manifest.EnvironmentDefinition{
			Name: env2Id,
			Auth: manifest.Auth{OAuth: &manifest.OAuth{ClientID: manifest.AuthSecret{Name: "id", Value: "value"}, ClientSecret: manifest.AuthSecret{Name: "id", Value: "value"}}},
		}

	project1Id := "project1"
	project2Id := "project2"

	t.Run("defined environment in project succeeds", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{},
					},
				},
			},
			manifest.Environments{
				env1Id: env1Definition,
			})
		assert.NoError(t, err)
	})

	t.Run("undefined environment in project fails", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						"unknown_env": project.ConfigsPerType{},
					},
				},
			},
			manifest.Environments{
				env1Id: env1Definition,
			})
		assert.ErrorContains(t, err, "undefined environment")
	})

	t.Run("platform config with platform environment succeeds", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{createOpenPipelineConfigForTest("bizevents-openpipeline-id", "bizevents", project1Id)},
						},
					},
				},
			},
			manifest.Environments{
				env1Id: env1Definition,
			})
		assert.NoError(t, err)
	})

	t.Run("platform config without platform environment fails", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{createOpenPipelineConfigForTest("bizevents-openpipeline-id", "bizevents", project1Id)},
						},
					},
				},
			},
			manifest.Environments{
				env1Id: env1DefinitionWithoutPlatform,
			})
		assert.ErrorContains(t, err, "environment \"env1\" is not configured to access platform")
	})

	t.Run("two different openpipeline configs in same project succceed", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents-openpipeline-id", "bizevents", project1Id),
								createOpenPipelineConfigForTest("events-openpipeline-id", "events", project1Id),
							},
						},
					},
				},
			},
			manifest.Environments{env1Id: env1Definition})
		assert.NoError(t, err)
	})

	t.Run("two different openpipeline configs in different projects succceed", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents-openpipeline-id", "bizevents", project1Id),
							},
						},
					},
				},
				{
					Id: project2Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("events-openpipeline-id", "events", project2Id),
							},
						},
					},
				},
			},
			manifest.Environments{env1Id: env1Definition})
		assert.NoError(t, err)
	})

	t.Run("two identical openpipeline configs in same project but different environments succceed", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents1-openpipeline-id", "bizevents", project1Id),
							},
						},
						env2Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents2-openpipeline-id", "bizevents", project1Id),
							},
						},
					},
				},
			},
			manifest.Environments{
				env1Id: env1Definition,
				env2Id: env2Definition,
			})
		assert.NoError(t, err)
	})

	t.Run("two identical openpipeline configs in different projects and environments succceed", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents1-openpipeline-id", "bizevents", project1Id),
							},
						},
					},
				},
				{
					Id: project2Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env2Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents2-openpipeline-id", "bizevents", project2Id),
							},
						},
					},
				},
			},
			manifest.Environments{
				env1Id: env1Definition,
				env2Id: env2Definition,
			})
		assert.NoError(t, err)
	})

	t.Run("two identical openpipeline configs in same project and environments fail", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents1-openpipeline-id", "bizevents", project1Id),
								createOpenPipelineConfigForTest("bizevents2-openpipeline-id", "bizevents", project1Id),
							},
						},
					},
				},
			},
			manifest.Environments{
				env1Id: env1Definition,
				env2Id: env2Definition,
			})
		assert.ErrorContains(t, err, "has multiple openpipeline configurations of kind")
	})

	t.Run("two identical openpipeline configs in different projects and same environments fail", func(t *testing.T) {
		err := validateProjectsWithEnvironments(
			[]project.Project{
				{
					Id: project1Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents1-openpipeline-id", "bizevents", project1Id),
							},
						},
					},
				},
				{
					Id: project2Id,
					Configs: project.ConfigsPerTypePerEnvironments{
						env1Id: project.ConfigsPerType{
							"openpipeline": []config.Config{
								createOpenPipelineConfigForTest("bizevents2-openpipeline-id", "bizevents", project2Id),
							},
						},
					},
				},
			},
			manifest.Environments{
				env1Id: env1Definition,
			})
		assert.ErrorContains(t, err, "has multiple openpipeline configurations of kind")
	})

}

func createOpenPipelineConfigForTest(configId string, kind string, project string) config.Config {
	return config.Config{
		Template: template.NewInMemoryTemplateWithPath("a.json", ""),
		Coordinate: coordinate.Coordinate{
			Project:  project,
			Type:     "openpipeline",
			ConfigId: configId,
		},
		Type: config.OpenPipelineType{Kind: kind},
	}
}
func Test_ValidateAuthenticationWithProjectConfigs(t *testing.T) {
	envId := "environmentId"
	token := manifest.AuthSecret{Name: "token", Value: "value"}
	oAuth := manifest.OAuth{
		ClientID:     manifest.AuthSecret{Name: "id", Value: "value"},
		ClientSecret: manifest.AuthSecret{Name: "id", Value: "value"}}
	documentConf := config.Config{
		Type: config.DocumentType{},
		Skip: false,
	}
	classicConf := config.Config{
		Type: config.ClassicApiType{},
		Skip: false,
	}
	classicConfSkip := classicConf
	classicConfSkip.Skip = true
	documentConfSkip := documentConf
	documentConfSkip.Skip = true

	success_tests := []struct {
		name                 string
		environments         manifest.Environments
		configs              project.ConfigsPerType
		expectedErrorMessage string
	}{
		{
			"oAuth manifest with document api",
			manifest.Environments{
				envId: manifest.EnvironmentDefinition{
					Name: envId,
					Auth: manifest.Auth{
						OAuth: &oAuth},
				}},
			project.ConfigsPerType{
				string(config.DocumentTypeId): []config.Config{documentConf}},
			"",
		},
		{
			"token manifest with classic api",
			manifest.Environments{
				envId: manifest.EnvironmentDefinition{
					Name: envId,
					Auth: manifest.Auth{
						Token: &token},
				}},
			project.ConfigsPerType{
				string(config.ClassicApiTypeId): []config.Config{classicConf}},
			"",
		},
		{
			"token and oAuth manifest with classic and document api",
			manifest.Environments{
				envId: manifest.EnvironmentDefinition{
					Name: envId,
					Auth: manifest.Auth{
						Token: &token,
						OAuth: &oAuth,
					},
				}},
			project.ConfigsPerType{
				string(config.DocumentTypeId):   []config.Config{documentConf},
				string(config.ClassicApiTypeId): []config.Config{classicConf, classicConfSkip},
			},
			"",
		},
		{
			"token manifest with document api expect validation error",
			manifest.Environments{
				envId: manifest.EnvironmentDefinition{
					Name: envId,
					Auth: manifest.Auth{
						Token: &token},
				}},
			project.ConfigsPerType{
				string(config.DocumentTypeId): []config.Config{documentConf}},
			"requires OAuth for environment",
		},
		{
			"oAuth manifest with document and classic api expect validation error",
			manifest.Environments{
				envId: manifest.EnvironmentDefinition{
					Name: envId,
					Auth: manifest.Auth{
						OAuth: &oAuth},
				}},
			project.ConfigsPerType{
				string(config.DocumentTypeId):   []config.Config{documentConf},
				string(config.ClassicApiTypeId): []config.Config{classicConf},
			},
			"requires a token for environment",
		},
		{
			"oAuth manifest with document and classic api classic api with skip true, expect no error",
			manifest.Environments{
				envId: manifest.EnvironmentDefinition{
					Name: envId,
					Auth: manifest.Auth{
						OAuth: &oAuth},
				}},
			project.ConfigsPerType{
				"dashboard": []config.Config{classicConfSkip, documentConf}},
			"",
		},
		{
			"token manifest with document and classic api document api with skip true, expect no error",
			manifest.Environments{
				envId: manifest.EnvironmentDefinition{
					Name: envId,
					Auth: manifest.Auth{
						Token: &token},
				}},
			project.ConfigsPerType{
				string(config.DocumentTypeId):   []config.Config{documentConfSkip},
				string(config.ClassicApiTypeId): []config.Config{classicConf},
			},
			"",
		},
	}

	for _, tc := range success_tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAuthenticationWithProjectConfigs(
				[]project.Project{
					{
						Id: "some id",
						Configs: project.ConfigsPerTypePerEnvironments{
							envId: tc.configs,
						},
					},
				},
				tc.environments)
			if tc.expectedErrorMessage != "" {
				assert.ErrorContains(t, err, tc.expectedErrorMessage)
				return
			}
			assert.NoError(t, err)
		})
	}
}
