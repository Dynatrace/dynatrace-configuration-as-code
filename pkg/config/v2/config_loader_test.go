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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

func Test_parseConfigs(t *testing.T) {
	testLoaderContext := &LoaderContext{
		ProjectId: "project",
		Path:      "some-dir/",
		KnownApis: map[string]struct{}{"some-api": {}},
		Environments: []manifest.EnvironmentDefinition{
			manifest.NewEnvironmentDefinition(
				"env name",
				manifest.UrlDefinition{Type: manifest.ValueUrlType, Value: "env url"},
				"default",
				&manifest.EnvironmentVariableToken{EnvironmentVariableName: "token var"},
			),
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
					Type: Type{
						SchemaId:      "builtin:profile.test",
						SchemaVersion: "1.0",
						Scope:         "tenant",
					},
					Parameters:  Parameters{"name": &value.ValueParameter{Value: string("Star Trek > Star Wars")}},
					References:  []coordinate.Coordinate{},
					Skip:        false,
					Environment: "env name",
					Group:       "default",
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
