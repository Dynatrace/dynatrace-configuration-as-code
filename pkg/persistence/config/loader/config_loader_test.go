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

package loader

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/cache"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/list"
	ref "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func Test_parseConfigs(t *testing.T) {
	t.Setenv("ENV_VAR_SKIP_TRUE", "true")
	t.Setenv("ENV_VAR_SKIP_FALSE", "false")

	testLoaderContext := &LoaderContext{
		ProjectId: "project",
		Path:      "some-dir/",
		KnownApis: map[string]struct{}{"some-api": {}, api.DashboardShareSettings: {}},
		Environments: []manifest.EnvironmentDefinition{
			{
				Name:  "env name",
				URL:   manifest.URLDefinition{Type: manifest.ValueURLType, Value: "env url"},
				Group: "default",
				Auth: manifest.Auth{
					Token: manifest.AuthSecret{Name: "token var"},
				},
			},
		},
		TemplateCache:   cache.NoopCache[template.FileBasedTemplate]{},
		ParametersSerDe: config.DefaultParameterParsers,
	}

	tests := []struct {
		name              string
		filePathArgument  string
		filePathOnDisk    string
		fileContentOnDisk string
		wantConfigs       []config.Config
		wantErrorsContain []string
		envVars           map[string]string
	}{
		{
			name:              "reports error if file does not exist",
			filePathArgument:  "file1.yaml",
			filePathOnDisk:    "file2.yaml",
			wantErrorsContain: []string{"does not exist"},
		},
		{
			name:              "reports error with v1 warning when parsing v1 config",
			filePathArgument:  "test-file.yaml",
			filePathOnDisk:    "test-file.yaml",
			fileContentOnDisk: "config:\n  - profile: \"profile.json\"\n\nprofile:\n  - name: \"Star Trek Service\"",
			wantErrorsContain: []string{"config is not a valid v2 configuration"},
		},
		{
			name:              "reports error with v1 warning on broken v2 toplevel",
			filePathArgument:  "test-file.yaml",
			filePathOnDisk:    "test-file.yaml",
			fileContentOnDisk: "this_should_say_config:\n- id: profile\n  config:\n    name: Star Trek Service\n    skip: false\n",
			wantErrorsContain: []string{"failed to load config from file \"test-file.yaml"},
		},
		{
			name:              "reports detailed error for invalid v2 config if template is missing",
			filePathArgument:  "test-file.yaml",
			filePathOnDisk:    "test-file.yaml",
			fileContentOnDisk: "configs:\n- id: profile\n  config:\n    name: Star Trek Service\n    skip: false\n  type:\n    api: some-api",
			wantErrorsContain: []string{"missing property `template`"},
		},
		{
			name:              "reports detailed error for invalid v2 config if an unknown API is used",
			filePathArgument:  "test-file.yaml",
			filePathOnDisk:    "test-file.yaml",
			fileContentOnDisk: "configs:\n- id: profile\n  config:\n    name: Star Trek Service\n    skip: false\n  type:\n    api: another-api",
			wantErrorsContain: []string{"unknown API: another-api"},
		},
		{
			name:             "multiple types in one config is not allowed",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `configs:
- id: profile
  config:
    name: Star Trek Service
    skip: false
  type:
    api: another-api
    automation:
      resource: workflow
`,
			wantErrorsContain: []string{"only one config type is allowed at once"},
		},
		{
			name:             "integer as type definition",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `configs:
- id: profile
  config:
    name: Star Trek Service
    skip: false
  type: 1337
`,
			wantErrorsContain: []string{"cannot parse definition"},
		},
		{
			name:             "empty type definition",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile
  config:
    name: Star Trek Service
    skip: false
  type: {}
`,
			wantErrorsContain: []string{"no type is defined"},
		},
		{
			name:             "unknown type definition",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile
  config:
    name: Star Trek Service
  type:
    voyager: Captain Janeway
`,
			wantErrorsContain: []string{"unknown config-type"},
		},
		{
			name:             "Skip parameter is referenced to true",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						"name": &value.ValueParameter{Value: "Star Trek Service"},
					},
					Skip:        true,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "Skip parameter is referenced to false",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						"name": &value.ValueParameter{Value: "Star Trek Service"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "Skip parameter is defined (with default value) but omit",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						"name": &value.ValueParameter{Value: "Star Trek Service"},
					},
					Skip:        true,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "Skip parameter is defined as a value",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						"name": &value.ValueParameter{Value: "Star Trek Service"},
					},
					Skip:        true,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "Skip parameter is defined (w/o default value) but omit - should throw an error",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantErrorsContain: []string{"skip: cannot parse parameter definition in `test-file.yaml`: failed to resolve value: skip: cannot parse parameter: environment variable `ENV_VAR_SKIP_NOT_EXISTS` not set"},
		},
		{
			name:             "Skip parameter is defined with a wrong value - should throw an error",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantErrorsContain: []string{"resolved value can only be 'true' or 'false'"},
		},
		{
			name:             "Skip parameter is defined with a wrong value - should throw an error",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantErrorsContain: []string{"must be of type 'value' or 'environment'"},
		},
		{
			name:              "reports error for empty v2 config",
			filePathArgument:  "test-file.yaml",
			filePathOnDisk:    "test-file.yaml",
			wantErrorsContain: []string{"no configurations found in file"},
		},
		{
			name:             "loads settings 2.0 config with all properties",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						"name":                &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads settings 2.0 config with full value parameter as scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						"name":                &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: &value.ValueParameter{Value: "environment"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads settings 2.0 config with a full reference as scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: ref.New("project", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads settings 2.0 config with a reference as insertAfter",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
      scope: 'environment'
      insertAfter:
        type: reference
        configId: configId
        property: id
        configType: something`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:        &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter:       &value.ValueParameter{Value: "environment"},
						config.InsertAfterParameter: ref.New("project", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads settings 2.0 config with a reference as insertAfter but with wrong property",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
      scope: 'environment'
      insertAfter:
        type: reference
        configId: configId
        property: name
        configType: something`,
			wantErrorsContain: []string{`failed to parse insertAfter: property field of reference parameter "project:something:configId:name" must be "id"`},
		},
		{
			name:             "loads settings 2.0 config with a shorthand reference as scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: ref.New("project", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads settings 2.0 config with a full shorthand reference as scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: ref.New("proj2", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:              "loading a config without type content",
			filePathArgument:  "test-file.yaml",
			filePathOnDisk:    "test-file.yaml",
			fileContentOnDisk: "configs:\n- id: profile-id\n  config:\n    name: 'Star Trek > Star Wars'\n    template: 'profile.json'\n",
			wantErrorsContain: []string{"missing type definition"},
		},
		{
			name:             "fails to load with a compound as scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantErrorsContain: []string{compound.CompoundParameterType},
		},
		{
			name:             "fails to load with a list as scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantErrorsContain: []string{list.ListParameterType},
		},
		{
			name:             "loads with an environment parameter as scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: &environment.EnvironmentVariableParameter{Name: "TEST"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "load a workflow",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: workflow-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    automation:
      resource: workflow`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "workflow",
						ConfigId: "workflow-id",
					},
					Type: config.AutomationType{
						Resource: config.Workflow,
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter: &value.ValueParameter{Value: "Star Trek > Star Wars"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "load a business-calendar",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: bc-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    automation:
      resource: business-calendar`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "business-calendar",
						ConfigId: "bc-id",
					},
					Type: config.AutomationType{
						Resource: config.BusinessCalendar,
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter: &value.ValueParameter{Value: "Star Trek > Star Wars"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "load a scheduling rule",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: sr-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    automation:
      resource: scheduling-rule`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "scheduling-rule",
						ConfigId: "sr-id",
					},
					Type: config.AutomationType{
						Resource: config.SchedulingRule,
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter: &value.ValueParameter{Value: "Star Trek > Star Wars"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "load an unknown automation resource",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: automation-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    automation:
      resource: does-not-exist`,
			wantErrorsContain: []string{`unknown automation resource "does-not-exist"`},
		},
		{
			name:             "no automation resource specified",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: automation-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    automation: {}
`,
			wantErrorsContain: []string{`missing automation resource property`},
		},
		{
			name:             "empty automation resource specified",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: automation-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    automation:
      resource: ""
`,
			wantErrorsContain: []string{`missing automation resource property`},
		},
		{
			name:             "settings missing schemaid",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: automation-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    settings:
      scope: scope
`,
			wantErrorsContain: []string{`missing settings schemaId`},
		},
		{
			name:             "settings missing scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: automation-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    settings:
      schema: schema
`,
			wantErrorsContain: []string{`missing settings scope`},
		},
		{
			name:             "settings missing any property",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: automation-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    settings: {}
`,
			wantErrorsContain: []string{`missing settings`},
		},
		{
			name:             "fails to load with a parameter that is 'id'",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantErrorsContain: []string{config.IdParameter},
		},
		{
			name:             "fails to load with a parameter that is 'scope'",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
			wantErrorsContain: []string{config.ScopeParameter},
		},
		{
			name:             "fails to load with a parameter that is 'name'",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
    parameters:
      name: "some other name"
  type: some-api`,
			wantErrorsContain: []string{config.NameParameter},
		},
		{
			name:             "loads config with object id override",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
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
      scope: 'tenant'
  environmentOverrides:
    - environment: "env name"
      override:
        originObjectId: better-origin-object-id`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						"name":                &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "better-origin-object-id",
				},
			},
		},
		{
			name:             "reports error if some-api API is missing name",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    template: 'profile.json'
  type: some-api
`,
			wantErrorsContain: []string{"missing parameter `name`"},
		},
		{
			name:             "dashboard-share-settings do not require a name",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    template: 'profile.json'
  type:
    api:
      name: dashboard-share-settings
      scope:
        configId: 12345678-1234-1234-1234-123456789012
        configType: dashboard
        property: id
        type: reference`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "dashboard-share-settings",
						ConfigId: "profile-id",
					},
					Type: config.ClassicApiType{
						Api: "dashboard-share-settings",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.ScopeParameter: ref.New("project", "dashboard", "12345678-1234-1234-1234-123456789012", "id")},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "Settings do not require a name",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    template: 'profile.json'
  type:
    settings:
      schema: 'builtin:profile.test'
      scope: 'environment'
`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:profile.test",
						ConfigId: "profile-id",
					},
					Type: config.SettingsType{
						SchemaId: "builtin:profile.test",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.ScopeParameter: &value.ValueParameter{Value: "environment"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "Automations do not require a name",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    template: 'profile.json'
  type:
    automation:
      resource: workflow
`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "workflow",
						ConfigId: "profile-id",
					},
					Type: config.AutomationType{
						Resource: config.Workflow,
					},
					Template:    template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters:  config.Parameters{},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "Bucket config with FF on",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    template: 'profile.json'
  type: bucket
`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "bucket",
						ConfigId: "profile-id",
					},
					Type:        config.BucketType{},
					Template:    template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters:  config.Parameters{},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "Bucket written as api config",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    template: 'profile.json'
  type:
    api: bucket
`,
			wantErrorsContain: []string{"unknown API: bucket"},
		},
		{
			name:             "API without scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    api:
      name: 'some-api'
`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile-id",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						"name": &value.ValueParameter{Value: "Star Trek > Star Wars"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name:             "API with invalid structure",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
  type:
    api:
    - name: 'some-api'
`,
			wantErrorsContain: []string{"failed to unmarshal api-type"},
		},
		{
			name:             "loads complex api config with a full shorthand reference as scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    api:
      name: some-api
      scope: ["proj2", "something", "configId", "id"]`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile-id",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: ref.New("proj2", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads complex api config with the reference parameter",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    api:
      name: some-api
      scope:
        type: reference
        configId: configId
        property: id
        configType: something`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile-id",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: ref.New("project", "something", "configId", "id"),
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads complex api config environment parameter",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    api:
      name: some-api
      scope:
        type: environment
        name: TEST`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile-id",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: &environment.EnvironmentVariableParameter{Name: "TEST"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads complex api config with a value scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    api:
      name: some-api
      scope:
        type: value
        value: var`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "some-api",
						ConfigId: "profile-id",
					},
					Type: config.ClassicApiType{
						Api: "some-api",
					},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter:  &value.ValueParameter{Value: "Star Trek > Star Wars"},
						config.ScopeParameter: &value.ValueParameter{Value: "var"},
					},
					Skip:           false,
					Environment:    "env name",
					Group:          "default",
					OriginObjectId: "origin-object-id",
				},
			},
		},
		{
			name:             "loads complex api config with an invalid value scope",
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: profile-id
  config:
    name: 'Star Trek > Star Wars'
    template: 'profile.json'
    originObjectId: origin-object-id
  type:
    api:
      name: some-api
      scope:
        type: value
        wrong-field: var`,
			wantErrorsContain: []string{"missing property"},
		},
		{
			name: "Document dashboard config with FF on",
			envVars: map[string]string{
				featureflags.Temporary[featureflags.Documents].EnvName(): "true",
			},
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: dashboard-id
  config:
    name: Test dashboard
    originObjectId: ext-ID-123
    template: 'profile.json'
  type:
    document:
      kind: dashboard`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "document",
						ConfigId: "dashboard-id",
					},
					OriginObjectId: "ext-ID-123",
					Type:           config.DocumentType{Kind: config.DashboardKind, Private: false},
					Template:       template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter: &value.ValueParameter{Value: "Test dashboard"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name: "Document private dashboard config with FF on",
			envVars: map[string]string{
				featureflags.Temporary[featureflags.Documents].EnvName(): "true",
			},
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: dashboard-id
  config:
    name: Test dashboard
    originObjectId: ext-ID-123
    template: 'profile.json'
  type:
    document:
      kind: dashboard
      private: true`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "document",
						ConfigId: "dashboard-id",
					},
					OriginObjectId: "ext-ID-123",
					Type:           config.DocumentType{Kind: config.DashboardKind, Private: true},
					Template:       template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter: &value.ValueParameter{Value: "Test dashboard"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name: "Document notebook config with FF on",
			envVars: map[string]string{
				featureflags.Temporary[featureflags.Documents].EnvName(): "true",
			},
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: notebook-id
  config:
    name: Test notebook
    originObjectId: ext-ID-123
    template: 'profile.json'
  type:
    document:
      kind: notebook`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "document",
						ConfigId: "notebook-id",
					},
					OriginObjectId: "ext-ID-123",
					Type:           config.DocumentType{Kind: config.NotebookKind},
					Template:       template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter: &value.ValueParameter{Value: "Test notebook"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name: "Document config with invalid type with FF on",
			envVars: map[string]string{
				featureflags.Temporary[featureflags.Documents].EnvName(): "true",
			},
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: dashboard-id
  config:
    name: Test document
    originObjectId: ext-ID-123
    template: 'profile.json'
  type:
    document:
      kind: other`,
			wantErrorsContain: []string{
				"unknown document kind \"other\"",
			},
		},
		{
			name: "Document config with FF off",
			envVars: map[string]string{
				featureflags.Temporary[featureflags.Documents].EnvName(): "false",
			},
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: dashboard-id
  config:
    name: Test dashboard
    originObjectId: ext-ID-123
    template: 'profile.json'
  type:
    document:
      kind: dashboard`,
			wantErrorsContain: []string{
				"unknown config-type \"document\"",
			},
		},
		{
			name: "OpenPipeline config with FF off",
			envVars: map[string]string{
				featureflags.Temporary[featureflags.OpenPipeline].EnvName(): "false",
			},
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: openpipeline-id
  config:
    name: Test Pipeline
    template: 'profile.json'
  type:
    openpipeline:
      kind: bizevents`,
			wantErrorsContain: []string{
				"unknown config-type \"openpipeline\"",
			},
		},
		{
			name: "OpenPipeline config with FF on",
			envVars: map[string]string{
				featureflags.Temporary[featureflags.OpenPipeline].EnvName(): "true",
			},
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: bizevents-openpipeline-id
  config:
    name: Test Bizevents OpenPipeline
    template: 'profile.json'
  type:
    openpipeline:
      kind: bizevents`,
			wantConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "openpipeline",
						ConfigId: "bizevents-openpipeline-id",
					},
					Type:     config.OpenPipelineType{Kind: "bizevents"},
					Template: template.NewInMemoryTemplate("profile.json", "{}"),
					Parameters: config.Parameters{
						config.NameParameter: &value.ValueParameter{Value: "Test Bizevents OpenPipeline"},
					},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
				},
			},
		},
		{
			name: "OpenPipeline config with FF on and missing kind",
			envVars: map[string]string{
				featureflags.Temporary[featureflags.OpenPipeline].EnvName(): "true",
			},
			filePathArgument: "test-file.yaml",
			filePathOnDisk:   "test-file.yaml",
			fileContentOnDisk: `
configs:
- id: bizevents-openpipeline-id
  config:
    name: Test Bizevents OpenPipeline
    template: 'profile.json'
  type:
    openpipeline:`,
			wantErrorsContain: []string{
				"missing openpipeline kind property",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			testFs := afero.NewMemMapFs()
			_ = afero.WriteFile(testFs, tt.filePathOnDisk, []byte(tt.fileContentOnDisk), 0644)
			_ = afero.WriteFile(testFs, "profile.json", []byte("{}"), 0644)

			gotConfigs, gotErrors := LoadConfigFile(testFs, testLoaderContext, tt.filePathArgument)
			if len(tt.wantErrorsContain) != 0 {
				assert.Equal(t, len(tt.wantErrorsContain), len(gotErrors), "expected %v errors but got %v", len(tt.wantErrorsContain), len(gotErrors))

				for i, err := range gotErrors {
					assert.ErrorContains(t, err, tt.wantErrorsContain[i])
				}
				return
			}
			assert.Empty(t, gotErrors, "expected no errors but got: %v", gotErrors)

			// compare template contents
			assert.Empty(t, cmp.Diff(tt.wantConfigs, gotConfigs, cmp.Comparer(func(a, b template.Template) bool {
				cA, _ := a.Content()
				cA = strings.ReplaceAll(cA, " ", "")
				cA = strings.ReplaceAll(cA, "\n", "")

				cB, _ := b.Content()
				cB = strings.ReplaceAll(cB, " ", "")
				cB = strings.ReplaceAll(cB, "\n", "")

				return assert.Empty(t, cmp.Diff(cA, cB))
			})))
		})
	}
}

func Test_validateParameter(t *testing.T) {
	knownAPIs := map[string]struct{}{"some-api": {}, "other-api": {}}

	type given struct {
		configType string
		param      parameter.Parameter
	}
	tests := []struct {
		name    string
		given   given
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"valid reference between Config API types",
			given{
				"some-api",
				ref.New("project", "other-api", "config", "id"),
			},
			assert.NoError,
		},
		{
			"invalid reference from Config API to Setting ID",
			given{
				"some-api",
				ref.New("project", "builtin:some-setting", "config", "id"),
			},
			assert.Error,
		},
		{
			"valid reference from Config API to non-ID parameter of Setting",
			given{
				"some-api",
				ref.New("project", "builtin:some-setting", "config", "not-the-id-prop"),
			},
			assert.NoError,
		},
		{
			"valid reference between Settings",
			given{
				"builtin:some-setting",
				ref.New("project", "builtin:other-setting", "config", "not-the-id-prop"),
			},
			assert.NoError,
		},
		{
			"valid reference between Config API types in Compound Param",
			given{
				"some-api",
				makeCompoundParam(t, []parameter.ParameterReference{
					{
						Config:   coordinate.Coordinate{Project: "project", Type: "other-api", ConfigId: "config"},
						Property: "id",
					},
					{
						Config:   coordinate.Coordinate{Project: "project", Type: "some-api", ConfigId: "config"},
						Property: "some-value",
					},
				}),
			},
			assert.NoError,
		},
		{
			"invalid reference from Config API to Setting ID in Compound Param",
			given{
				"some-api",
				makeCompoundParam(t, []parameter.ParameterReference{
					{
						Config:   coordinate.Coordinate{Project: "project", Type: "builtin:some-setting", ConfigId: "config"},
						Property: "id",
					},
					{
						Config:   coordinate.Coordinate{Project: "project", Type: "some-api", ConfigId: "config"},
						Property: "some-value",
					},
				}),
			},
			assert.Error,
		},
		{
			"valid reference from Config API to non-ID parameter of Setting in Compound Param",
			given{
				"some-api",
				makeCompoundParam(t, []parameter.ParameterReference{
					{
						Config:   coordinate.Coordinate{Project: "project", Type: "other-api", ConfigId: "config"},
						Property: "id",
					},
					{
						Config:   coordinate.Coordinate{Project: "project", Type: "builtin:some-setting", ConfigId: "config"},
						Property: "some-value",
					},
				}),
			},
			assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := singleConfigEntryLoadContext{
				configFileLoaderContext: &configFileLoaderContext{
					LoaderContext: &LoaderContext{
						KnownApis:       knownAPIs,
						ParametersSerDe: config.DefaultParameterParsers,
					},
				},
				Type: tt.given.configType,
			}

			tt.wantErr(t, validateParameter(&ctx, "paramName", tt.given.param), fmt.Sprintf("validateParameter - given %s", tt.given))
		})
	}
}

func makeCompoundParam(t *testing.T, refs []parameter.ParameterReference) *compound.CompoundParameter {
	compoundParam, err := compound.New("param", "{}", refs)
	assert.NoError(t, err)
	return compoundParam
}
