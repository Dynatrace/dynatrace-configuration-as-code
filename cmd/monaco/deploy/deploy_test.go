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
	p "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"reflect"
	"testing"
)

func Test_filterProjectsByName(t *testing.T) {
	type args struct {
		projects []p.Project
		names    []string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"returns nothing if no names given",
			args{
				[]p.Project{
					{
						Id:           "project A",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project B",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
				},
				[]string{},
			},
			nil,
			false,
		},
		{
			"filters for project by name",
			args{
				[]p.Project{
					{
						Id:           "project A",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project B",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
				},
				[]string{"project A"},
			},
			[]string{"project A"},
			false,
		},
		{
			"filters for grouping projects by name",
			args{
				[]p.Project{
					{
						Id:           "project.a",
						GroupId:      "project",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project.b",
						GroupId:      "project",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project2",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project3.a",
						GroupId:      "project3",
						Configs:      nil,
						Dependencies: nil,
					},
				},
				[]string{"project"},
			},
			[]string{"project.a", "project.b"},
			false,
		},
		{
			"returns error if project of given name is not found",
			args{
				[]p.Project{
					{
						Id:           "project.a",
						GroupId:      "project",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project.b",
						GroupId:      "project",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project2",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project3.a",
						GroupId:      "project3",
						Configs:      nil,
						Dependencies: nil,
					},
				},
				[]string{"project", "UNDEFINED PROJECT"},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterProjectsByName(tt.args.projects, tt.args.names)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterProjectsByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterProjectsByName() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filterProjects(t *testing.T) {
	type args struct {
		projects             []p.Project
		specificProjects     []string
		specificEnvironments []string
	}
	tests := []struct {
		name    string
		args    args
		want    []p.Project
		wantErr bool
	}{
		{
			name: "empty projects",
			args: args{
				projects:             []p.Project{},
				specificProjects:     []string{"a-project"},
				specificEnvironments: []string{"an-env"},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "specific project not found",
			args: args{
				projects:             []p.Project{{Id: "a-project"}},
				specificProjects:     []string{"another-project"},
				specificEnvironments: []string{"an-env"},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "filter by specific project",
			args: args{
				projects:             []p.Project{{Id: "a-project"}, {Id: "another-project"}},
				specificProjects:     []string{"a-project"},
				specificEnvironments: []string{"an-env"},
			},
			want:    []p.Project{{Id: "a-project"}},
			wantErr: false,
		},
		{
			name: "filter by specific project and specific environment",
			args: args{
				projects: []p.Project{
					{
						Id:           "a-project",
						Dependencies: p.DependenciesPerEnvironment{"another-env": []string{"another-project"}},
					},
					{
						Id: "another-project",
					},
				},
				specificProjects:     []string{"a-project"},
				specificEnvironments: []string{"another-env"},
			},
			want: []p.Project{
				{
					Id:           "a-project",
					Dependencies: p.DependenciesPerEnvironment{"another-env": []string{"another-project"}},
				},
				{
					Id: "another-project",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterProjects(tt.args.projects, tt.args.specificProjects, tt.args.specificEnvironments)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterProjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterProjects() got = %v, want %v", got, tt.want)
			}
		})
	}
}

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
    type: classic
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
