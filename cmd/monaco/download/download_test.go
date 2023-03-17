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

package download

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_validateOutputFolder(t *testing.T) {
	type args struct {
		fs           afero.Fs
		outputFolder string
		project      string
	}
	tests := []struct {
		name       string
		args       args
		wantErrors bool
	}{
		{
			"no error if output does not exist yet",
			args{
				getTestFs([]string{}, []string{}),
				"output",
				"project",
			},
			false,
		},
		{
			"no error if output exists as folder",
			args{
				getTestFs([]string{"output"}, []string{}),
				"output",
				"project",
			},
			false,
		},
		{
			"no error if project exists as folder",
			args{
				getTestFs([]string{"output/project"}, []string{}),
				"output",
				"project",
			},
			false,
		},
		{
			"error if output exists as file",
			args{
				getTestFs([]string{}, []string{"output"}),
				"output",
				"project",
			},
			true,
		},
		{
			"error if project exists as file",
			args{
				getTestFs([]string{}, []string{"output/project"}),
				"output",
				"project",
			},
			true,
		},
		{
			"error if everything exists",
			args{
				getTestFs([]string{"output", "output/project"}, []string{"output", "output/project"}),
				"output",
				"project",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotErrs := validateOutputFolder(tt.args.fs, tt.args.outputFolder, tt.args.project); !tt.wantErrors && len(gotErrs) > 0 {
				t.Errorf("validateOutputFolder() encountered unexpted errors: %v", gotErrs)
			}
		})
	}
}

func getTestFs(existingFolderPaths []string, existingFilePaths []string) afero.Fs {
	fs := afero.NewMemMapFs()
	for _, p := range existingFolderPaths {
		_ = fs.MkdirAll(p, 0777)
	}
	for _, p := range existingFilePaths {
		_ = afero.WriteFile(fs, p, []byte{}, 0777)
	}
	return fs
}

func Test_checkForCircularDependencies(t *testing.T) {
	type args struct {
		proj project.Project
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"writes nothing if no configs are downloaded",
			args{
				project.Project{},
			},
			false,
		}, {
			"return errors if cyclic dependency in downloaded configs",
			args{
				project.Project{
					Id: "test_project",
					Configs: map[string]project.ConfigsPerType{
						"test_project": {
							"dashboard": []config.Config{
								{
									Template: template.CreateTemplateFromString("some/path", "{}"),
									Parameters: map[string]parameter.Parameter{
										"name": &valueParam.ValueParameter{Value: "name A"},
										"ref":  parameter.NewDummy(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "b"}),
									},
									Coordinate: coordinate.Coordinate{
										Project:  "test",
										Type:     "dashboard",
										ConfigId: "a",
									},
								},
								{
									Template: template.CreateTemplateFromString("some/path", "{}"),
									Parameters: map[string]parameter.Parameter{
										"name": &valueParam.ValueParameter{Value: "name A"},
										"ref":  parameter.NewDummy(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "a"}),
									},
									Coordinate: coordinate.Coordinate{
										Project:  "test",
										Type:     "dashboard",
										ConfigId: "b",
									},
								},
							},
						},
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reportForCircularDependencies(tt.args.proj)
			if tt.wantErr {
				assert.ErrorContains(t, err, "there are circular dependencies")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetApisToDownload(t *testing.T) {
	type given struct {
		apis         api.APIs
		specificAPIs []string
	}
	type expected struct {
		apis []string
	}
	tests := []struct {
		name     string
		given    given
		expected expected
	}{
		{
			name: "filter all specific defined api",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				specificAPIs: []string{"api_1"},
			},
			expected: expected{
				apis: []string{"api_1"},
			},
		}, {
			name: "if deprecated api is defined, do not filter it",
			given: given{
				apis: api.APIs{
					"api_1":          api.API{ID: "api_1"},
					"api_2":          api.API{ID: "api_2"},
					"deprecated_api": api.API{ID: "deprecated_api", DeprecatedBy: "new_api"},
				},
				specificAPIs: []string{"api_1", "deprecated_api"},
			},
			expected: expected{
				apis: []string{"api_1", "deprecated_api"},
			},
		},
		{
			name: "if specific api is not requested, filter deprecated apis",
			given: given{
				apis: api.APIs{
					"api_1":          api.API{ID: "api_1"},
					"api_2":          api.API{ID: "api_2"},
					"deprecated_api": api.API{ID: "deprecated_api", DeprecatedBy: "new_api"},
				},
				specificAPIs: []string{},
			},
			expected: expected{
				apis: []string{"api_1", "api_2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getApisToDownload(tt.given.apis, tt.given.specificAPIs)
			for _, e := range tt.expected.apis {
				assert.Contains(t, actual, e)
			}
		})
	}
}
