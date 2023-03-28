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

//go:build unit

package v2

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/compound"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/list"
	ref "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

func Test_parseConfigs(t *testing.T) {
	t.Setenv("ENV_VAR_SKIP_TRUE", "true")
	t.Setenv("ENV_VAR_SKIP_FALSE", "false")

	testLoaderContext := &LoaderContext{
		ProjectId: "project",
		Path:      "some-dir/",
		KnownApis: map[string]struct{}{"some-api": {}},
		Environments: []manifest.EnvironmentDefinition{
			{
				Name:  "env name",
				Type:  manifest.Classic,
				URL:   manifest.URLDefinition{Type: manifest.ValueURLType, Value: "env url"},
				Group: "default",
				Auth: manifest.Auth{
					Token: manifest.AuthSecret{Name: "token var"},
				},
			},
		},
		ParametersSerDe: DefaultParameterParsers,
	}

	tests := []struct {
		name              string
		filePathArgument  string
		filePathOnDisk    string
		fileContentOnDisk string
		wantConfigs       []Config
		wantErrorsContain []string
	}{
		{
			"reports error if file does not exist",
			"file1.yaml",
			"file2.yaml",
			"",
			nil,
			[]string{"does not exist"},
		},
		{
			"reports error with v1 warning when parsing v1 config",
			"test-file.yaml",
			"test-file.yaml",
			"config:\n  - profile: \"profile.json\"\n\nprofile:\n  - name: \"Star Trek Service\"",
			nil,
			[]string{"is not valid v2 configuration"},
		},
		{
			"reports error with v1 warning on broken v2 toplevel",
			"test-file.yaml",
			"test-file.yaml",
			"this_should_say_config:\n- id: profile\n  config:\n    name: Star Trek Service\n    skip: false\n",
			nil,
			[]string{"failed to load config 'test-file.yaml"},
		},
		{
			"reports detailed error for invalid v2 config",
			"test-file.yaml",
			"test-file.yaml",
			"configs:\n- id: profile\n  config:\n    name: Star Trek Service\n    skip: false\n  type:\n    api: some-api",
			nil,
			[]string{"missing property `template`"},
		},
		{
			"reports detailed error for invalid v2 config",
			"test-file.yaml",
			"test-file.yaml",
			"configs:\n- id: profile\n  config:\n    name: Star Trek Service\n    skip: false\n  type:\n    api: another-api",
			nil,
			[]string{"unknown API: another-api"},
		},
		{
			"Skip parameter is referenced to true",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile
  config:
    name: Star Trek Service
    template: profile.json
    skip:
      type: environment
      name: ENV_VAR_SKIP_TRUE
      default: "false"
  type:
    api: some-api`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile",
					},
					Type: ClassicApiType{
						Api: "some-api",
					},
					Parameters: Parameters{
						"name": &value.ValueParameter{Value: "Star Trek Service"},
					},
					Skip:        true,
					Environment: "env name",
					Group:       "default",
				},
			},
			nil,
		},
		{
			"Skip parameter is referenced to false",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile
  config:
    name: Star Trek Service
    template: profile.json
    skip:
      type: environment
      name: ENV_VAR_SKIP_FALSE
  type:
    api: some-api`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile",
					},
					Type: ClassicApiType{
						Api: "some-api",
					},
					Parameters: Parameters{
						"name": &value.ValueParameter{Value: "Star Trek Service"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
			nil,
		},
		{
			"Skip parameter is defined (with default value) but omit",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile
  config:
    name: Star Trek Service
    template: profile.json
    skip:
      type: environment
      name: ENV_VAR_SKIP_NOT_EXISTS
      default: true
  type:
    api: some-api`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile",
					},
					Type: ClassicApiType{
						Api: "some-api",
					},
					Parameters: Parameters{
						"name": &value.ValueParameter{Value: "Star Trek Service"},
					},
					Skip:        true,
					Environment: "env name",
					Group:       "default",
				},
			},
			nil,
		},
		{
			"Skip parameter is defined as a value",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile
  config:
    name: Star Trek Service
    template: profile.json
    skip:
      type: value
      value: true
  type:
    api: some-api`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile",
					},
					Type: ClassicApiType{
						Api: "some-api",
					},
					Parameters: Parameters{
						"name": &value.ValueParameter{Value: "Star Trek Service"},
					},
					Skip:        true,
					Environment: "env name",
					Group:       "default",
				},
			},
			nil,
		},
		{
			"Skip parameter is defined (w/o default value) but omit - should throw an error",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile
  config:
    name: Star Trek Service
    template: profile.json
    skip:
      type: environment
      name: ENV_VAR_SKIP_NOT_EXISTS
  type:
    api: some-api`,
			nil,
			[]string{"skip: cannot parse parameter definition in `test-file.yaml`: failed to resolve value: skip: cannot parse parameter: environment variable `ENV_VAR_SKIP_NOT_EXISTS` not set"},
		},
		{
			"Skip parameter is defined with a wrong value - should throw an error",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile
  config:
    name: Star Trek Service
    template: profile.json
    skip:
      type: environment
      name: ENV_VAR_SKIP_NOT_EXISTS
      default: "wrong value"
  type:
    api: some-api`,
			nil,
			[]string{"resolved value can only be 'true' or 'false'"},
		},
		{
			"Skip parameter is defined with a wrong value - should throw an error",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile
  config:
    name: Star Trek Service
    template: profile.json
    skip:
        type: reference
        configId: configId
        property: id
        configType: something
  type:
    api: some-api`,
			nil,
			[]string{"must be of type 'value' or 'environment'"},
		},
		{
			"reports error for empty v2 config",
			"test-file.yaml",
			"test-file.yaml",
			"",
			nil,
			[]string{"no configurations found in file"},
		},
		{
			"loads settings 2.0 config with all properties",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope: 'tenant'`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Parameters: Parameters{
						"name":         &value.ValueParameter{Value: "Star Trek > Star Wars"},
						ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
			nil,
		},
		{
			"loads settings 2.0 config with full value parameter as scope",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope:
        type: value
        value: environment`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Parameters: Parameters{
						"name":         &value.ValueParameter{Value: "Star Trek > Star Wars"},
						ScopeParameter: &value.ValueParameter{Value: "environment"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
			nil,
		},
		{
			"loads settings 2.0 config with a full reference as scope",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope:
        type: reference
        configId: configId
        property: id
        configType: something`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Parameters: Parameters{
						NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						ScopeParameter: ref.New("project", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
			nil,
		},
		{
			"loads settings 2.0 config with a shorthand reference as scope",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope: ["something", "configId", "id"]`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Parameters: Parameters{
						NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						ScopeParameter: ref.New("project", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
			nil,
		},
		{
			"loads settings 2.0 config with a full shorthand reference as scope",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope: ["proj2", "something", "configId", "id"]`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Parameters: Parameters{
						NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						ScopeParameter: ref.New("proj2", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
			nil,
		},
		{
			"loading a config without type content",
			"test-file.yaml",
			"test-file.yaml",
			"configs:\n- id: profile-id\n  config:\n    name: 'Star Trek > Star Wars'\n    template: 'profile.json'\n",
			nil,
			[]string{"type configuration is missing"},
		},
		{
			"fails to load with a compound as scope",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope:
        type: compound
        format: "format"
        references: []`,
			nil,
			[]string{compound.CompoundParameterType},
		},
		{
			"fails to load with a list as scope",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope:
        type: list
        values: ["GEOLOCATTION-1234567", "GEOLOCATION-7654321"]`,
			nil,
			[]string{list.ListParameterType},
		},
		{
			"loads with an environment parameter as scope",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope:
        type: environment
        name: TEST`,
			[]Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Parameters: Parameters{
						NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						ScopeParameter: &environment.EnvironmentVariableParameter{Name: "TEST"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
			nil,
		},
		{
			"fails to load with a parameter that is 'id'",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id

    parameters:
      id: "test"
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope: validScope`,
			nil,
			[]string{IdParameter},
		},
		{
			"fails to load with a parameter that is 'scope'",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
    parameters:
      scope: "test"
  type:
    settings:
      schema: 'builtin:profile.test'
      schemaVersion: '1.0'
      scope: validScope`,
			nil,
			[]string{ScopeParameter},
		},
		{
			"fails to load with a parameter that is 'name'",
			"test-file.yaml",
			"test-file.yaml",
			`
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
    parameters:
      name: "some other name"
  type: some-api`,
			nil,
			[]string{NameParameter},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFs := afero.NewMemMapFs()
			_ = afero.WriteFile(testFs, tt.filePathOnDisk, []byte(tt.fileContentOnDisk), 0644)
			_ = afero.WriteFile(testFs, "profile.json", []byte("{}"), 0644)

			gotConfigs, gotErrors := parseConfigs(testFs, testLoaderContext, tt.filePathArgument)
			if len(tt.wantErrorsContain) != 0 {
				assert.Equal(t, len(tt.wantErrorsContain), len(gotErrors), "expected %v errors but got %v", len(tt.wantErrorsContain), len(gotErrors))

				for i, err := range gotErrors {
					assert.ErrorContains(t, err, tt.wantErrorsContain[i])
				}
				return
			}
			assert.Assert(t, len(gotErrors) == 0, "expected no errors but got: %v", gotErrors)
			assert.DeepEqual(t, gotConfigs, tt.wantConfigs, cmpopts.IgnoreInterfaces(struct{ template.Template }{}))
		})
	}
}
