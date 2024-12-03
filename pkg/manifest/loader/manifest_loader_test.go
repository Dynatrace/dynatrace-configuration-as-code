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
	monacoVersion "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func Test_extractUrlType(t *testing.T) {

	tests := []struct {
		name             string
		inputConfig      persistence.Environment
		givenEnvVarValue string
		want             manifest.URLDefinition
		wantErr          bool
	}{
		{
			name: "extracts_value_url",
			inputConfig: persistence.Environment{
				Name: "TEST ENV",
				URL:  persistence.TypedValue{Value: "TEST URL", Type: persistence.TypeValue},
				Auth: persistence.Auth{Token: &persistence.AuthSecret{Type: "environment", Name: "VAR"}},
			},
			want: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Value: "TEST URL",
			},
			wantErr: false,
		},
		{
			name: "extracts_value_if_type_empty",
			inputConfig: persistence.Environment{
				Name: "TEST ENV",
				URL:  persistence.TypedValue{Value: "TEST URL", Type: ""},
				Auth: persistence.Auth{Token: &persistence.AuthSecret{Type: "environment", Name: "VAR"}},
			},
			want: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Value: "TEST URL",
			},
			wantErr: false,
		},
		{
			name: "trims trailing slash from value url",
			inputConfig: persistence.Environment{
				Name: "TEST ENV",
				URL:  persistence.TypedValue{Value: "https://www.test.url/", Type: persistence.TypeValue},
				Auth: persistence.Auth{Token: &persistence.AuthSecret{Type: "environment", Name: "VAR"}},
			},
			want: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Value: "https://www.test.url",
			},
			wantErr: false,
		},
		{
			name: "extracts_environment_url",
			inputConfig: persistence.Environment{
				Name: "TEST ENV",
				URL:  persistence.TypedValue{Value: "TEST_TOKEN", Type: persistence.TypeEnvironment},
				Auth: persistence.Auth{Token: &persistence.AuthSecret{Type: "environment", Name: "VAR"}},
			},
			givenEnvVarValue: "resolved url value",
			want: manifest.URLDefinition{
				Type:  manifest.EnvironmentURLType,
				Name:  "TEST_TOKEN",
				Value: "resolved url value",
			},
			wantErr: false,
		},
		{
			name: "trims trailing slash from environment url",
			inputConfig: persistence.Environment{
				Name: "TEST ENV",
				URL:  persistence.TypedValue{Value: "TEST_TOKEN", Type: persistence.TypeEnvironment},
				Auth: persistence.Auth{Token: &persistence.AuthSecret{Type: "environment", Name: "VAR"}},
			},
			givenEnvVarValue: "https://www.test.url/",
			want: manifest.URLDefinition{
				Type:  manifest.EnvironmentURLType,
				Name:  "TEST_TOKEN",
				Value: "https://www.test.url",
			},
			wantErr: false,
		},
		{
			name: "fails_on_unknown_type",
			inputConfig: persistence.Environment{
				Name: "TEST ENV",
				URL:  persistence.TypedValue{Value: "TEST URL", Type: "this-is-not-a-type"},
				Auth: persistence.Auth{Token: &persistence.AuthSecret{Type: "environment", Name: "VAR"}},
			},
			want:    manifest.URLDefinition{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_TOKEN", tt.givenEnvVarValue)
			if got, gotErr := parseURLDefinition(&Context{}, tt.inputConfig.URL); got != tt.want || (!tt.wantErr && gotErr != nil) {
				t.Errorf("extractUrlType() = %v, %v, want %v, %v", got, gotErr, tt.want, tt.wantErr)
			}
		})
	}
}

func Test_parseProjectDefinition_SimpleType(t *testing.T) {
	tests := []struct {
		name  string
		given persistence.Project
		want  []manifest.ProjectDefinition
	}{
		{
			"parses_simple_project",
			persistence.Project{
				Name: "PROJ_NAME",
				Type: persistence.SimpleProjectType,
				Path: "PROJ_PATH",
			},
			[]manifest.ProjectDefinition{
				{
					Name: "PROJ_NAME",
					Path: "PROJ_PATH",
				},
			},
		},
		{
			"parses_simple_project_when_type_omitted",
			persistence.Project{
				Name: "PROJ_NAME",
				Path: "PROJ_PATH",
			},

			[]manifest.ProjectDefinition{
				{
					Name: "PROJ_NAME",
					Path: "PROJ_PATH",
				},
			},
		},
		{
			"sets_project_name_as_path_if_no_path_set",
			persistence.Project{
				Name: "PROJ_NAME",
			},

			[]manifest.ProjectDefinition{
				{
					Name: "PROJ_NAME",
					Path: "PROJ_NAME",
				},
			},
		},
		{
			"sets_project_name_as_path_if_no_path_set",
			persistence.Project{
				Name: "PROJ_NAME",
			},

			[]manifest.ProjectDefinition{
				{
					Name: "PROJ_NAME",
					Path: "PROJ_NAME",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErrs := parseProjectDefinition(nil, tt.given)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseProjectDefinition() got = %v, want %v", got, tt.want)
			}
			assert.Empty(t, gotErrs, "expected project %q to be valid", tt.given)
		})
	}
}

func Test_parseProjectDefinition_GroupingType(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = testFs.MkdirAll("PROJ_PATH", 0755)
	_ = testFs.MkdirAll("PROJ_PATH/a", 0755)
	_ = testFs.MkdirAll("PROJ_PATH/b", 0755)
	_ = afero.WriteFile(testFs, "PROJ_PATH/test_file", []byte("file should be ignored"), 0644)

	context := projectLoaderContext{
		fs:           testFs,
		manifestPath: ".",
	}
	project := persistence.Project{
		Name: "PROJ_NAME",
		Type: persistence.GroupProjectType,
		Path: "PROJ_PATH",
	}

	want := []manifest.ProjectDefinition{
		{
			Name:  "PROJ_NAME.a",
			Group: "PROJ_NAME",
			Path:  filepath.FromSlash("PROJ_PATH/a"),
		},
		{
			Name:  "PROJ_NAME.b",
			Group: "PROJ_NAME",
			Path:  filepath.FromSlash("PROJ_PATH/b"),
		},
	}
	got, gotErrs := parseProjectDefinition(&context, project)

	assert.Empty(t, gotErrs)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseProjectDefinition() got = %v, want %v", got, want)
	}
}

func Test_parseProjectDefinition_FailsOnUnknownType(t *testing.T) {
	context := projectLoaderContext{
		fs:           nil,
		manifestPath: ".",
	}
	project := persistence.Project{
		Name: "PROJ_NAME",
		Type: "not-a-project-type",
		Path: "PROJ_PATH",
	}

	_, gotErrs := parseProjectDefinition(&context, project)

	assert.Len(t, gotErrs, 1)
	assert.IsType(t, ProjectLoaderError{}, gotErrs[0])
}

func Test_parseProjectDefinition_FailsOnInvalidProjectDefinitions(t *testing.T) {
	context := projectLoaderContext{
		fs:           afero.NewMemMapFs(),
		manifestPath: ".",
	}

	_ = context.fs.Mkdir("./some/folder", 0777)
	_ = context.fs.Mkdir("./some/group", 0777)
	_ = context.fs.Mkdir("./some/group/project", 0777)

	tests := []struct {
		name    string
		project persistence.Project
	}{
		{
			"empty simple project",
			persistence.Project{
				Name: "",
				Path: "",
			},
		},
		{
			"simple project without a name",
			persistence.Project{
				Name: "",
				Path: "./some/folder",
			},
		},
		{
			"grouping dir that does not exist",
			persistence.Project{
				Name: "a grouping",
				Type: persistence.GroupProjectType,
				Path: "path/that/wont/be/found",
			},
		},
		{
			"grouping project without a name",
			persistence.Project{
				Name: "",
				Type: persistence.GroupProjectType,
				Path: "./some/group",
			},
		},
		{
			"name containing path separators",
			persistence.Project{
				Name: "names/must/not/be\\paths",
				Path: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErrs := parseProjectDefinition(&context, tt.project)

			assert.Len(t, gotErrs, 1)
			assert.IsType(t, ProjectLoaderError{}, gotErrs[0])
		})
	}

}

func Test_toProjectDefinitions(t *testing.T) {

	testFs := afero.NewMemMapFs()
	_ = testFs.MkdirAll("project/path/", 0755)
	_ = testFs.MkdirAll("project/path/a", 0755)
	_ = testFs.MkdirAll("project/path/b", 0755)
	_ = afero.WriteFile(testFs, "project/path/test_file", []byte("file should be ignored"), 0644)
	_ = testFs.MkdirAll("another/project/path/", 0755)
	_ = testFs.MkdirAll("another/project/path/one", 0755)
	_ = testFs.MkdirAll("another/project/path/two", 0755)
	_ = testFs.MkdirAll("empty/project/path", 0755)

	tests := []struct {
		name               string
		projectDefinitions []persistence.Project
		want               map[string]manifest.ProjectDefinition
		wantErrs           bool
	}{
		{
			"returns error on duplicate project id",
			[]persistence.Project{
				{
					Name: "project_id",
					Path: "project/path/",
				},
				{
					Name: "project_id",
					Path: "another/project/path/",
				},
			},
			nil,
			true,
		},
		{
			"returns error on duplicate project id between simple and grouping",
			[]persistence.Project{
				{
					Name: "project_id",
					Path: "project/path/",
				},
				{
					Name: "project_id",
					Type: persistence.GroupProjectType,
					Path: "another/project/path/",
				},
			},
			nil,
			true,
		},
		{
			"returns error on duplicate project id between grouping and grouping",
			[]persistence.Project{
				{
					Name: "project_id",
					Type: persistence.GroupProjectType,
					Path: "project/path/",
				},
				{
					Name: "project_id",
					Type: persistence.GroupProjectType,
					Path: "another/project/path/",
				},
			},
			nil,
			true,
		},
		{
			"returns error on duplicate project id between simple and sub-project in a group",
			[]persistence.Project{
				{
					Name: "project_id.a",
					Path: "some/project/path/",
				},
				{
					Name: "project_id", //this group will contain 'project_id.a' & 'project_id.b' projects
					Type: persistence.GroupProjectType,
					Path: "project/path/",
				},
			},
			nil,
			true,
		},
		{
			"returns error if grouping project path can not be read",
			[]persistence.Project{
				{
					Name: "project_id",
					Type: persistence.GroupProjectType,
					Path: "this/path/does/not/exist",
				},
			},
			nil,
			true,
		},
		{
			"returns error if project is invalid (empty)",
			[]persistence.Project{
				{
					Name: "",
					Path: "",
				},
			},
			nil,
			true,
		},
		{
			"returns error if project is invalid (path separators)",
			[]persistence.Project{
				{
					Name: "names/must/not/be\\paths",
					Path: "",
				},
			},
			nil,
			true,
		},
		{
			"returns error if a grouping project does not contain any projects",
			[]persistence.Project{
				{
					Name: "project_id",
					Type: persistence.GroupProjectType,
					Path: "empty/project/path/",
				},
			},
			nil,
			true,
		},
		{
			"correctly parses project definition",
			[]persistence.Project{
				{
					Name: "project_id_1",
					Path: filepath.FromSlash("project/path/"),
				},
				{
					Name: "project_id_2",
					Type: persistence.GroupProjectType,
					Path: filepath.FromSlash("another/project/path/"),
				},
			},
			map[string]manifest.ProjectDefinition{
				"project_id_1": {
					Name: "project_id_1",
					Path: filepath.FromSlash("project/path/"),
				},
				"project_id_2.one": {
					Name:  "project_id_2.one",
					Group: "project_id_2",
					Path:  filepath.FromSlash("another/project/path/one"),
				},
				"project_id_2.two": {
					Name:  "project_id_2.two",
					Group: "project_id_2",
					Path:  filepath.FromSlash("another/project/path/two"),
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := &projectLoaderContext{testFs, "path/to/a/manifest.yaml"}

			got, gotErrs := parseProjects(context, tt.projectDefinitions)

			numErrs := len(gotErrs)
			if (tt.wantErrs && numErrs <= 0) || (!tt.wantErrs && numErrs > 0) {
				t.Errorf("parseProjects() returned unexpected Errors = %v", gotErrs)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnmarshallingYAML(t *testing.T) {
	type expected struct {
		manifest persistence.Manifest
		wantErr  bool
	}
	var tests = []struct {
		name     string
		given    string
		expected expected
	}{
		{
			name: "unmarshall simple manifest",
			given: `
manifestVersion: "1.0"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url:
      type: environment
      value: ENV_URL
    auth:
      token:
        name: ENV_TOKEN
`,
			expected: expected{
				manifest: persistence.Manifest{
					ManifestVersion: "1.0",
					Projects: []persistence.Project{
						{
							Name: "project",
						},
					},
					EnvironmentGroups: []persistence.Group{
						{
							Name: "default",
							Environments: []persistence.Environment{
								{
									Name: "env",
									URL: persistence.TypedValue{
										Type:  persistence.TypeEnvironment,
										Value: "ENV_URL",
									},
									Auth: persistence.Auth{
										Token: &persistence.AuthSecret{
											Name: "ENV_TOKEN",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "can load shorthand url",
			given: `
manifestVersion: "1.0"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url: "https://www.dynatrace.com"
    auth:
      token:
        name: ENV_TOKEN
`,
			expected: expected{
				manifest: persistence.Manifest{
					ManifestVersion: "1.0",
					Projects: []persistence.Project{
						{
							Name: "project",
						},
					},
					EnvironmentGroups: []persistence.Group{
						{
							Name: "default",
							Environments: []persistence.Environment{
								{
									Name: "env",
									URL: persistence.TypedValue{
										Type:  persistence.TypeValue,
										Value: "https://www.dynatrace.com",
									},
									Auth: persistence.Auth{
										Token: &persistence.AuthSecret{
											Name: "ENV_TOKEN",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "fails on duplicate project definitions",
			given: `
projects:
- name: project
projects:
- name: project2
`,
			expected: expected{wantErr: true},
		},
		{
			name: "Load OAuth section",
			given: `
environmentGroups:
  - environments:
    - auth:
        oAuth:
          clientId:
            name: ENV_CLIENT_ID
          clientSecret:
            name: ENV_CLIENT_SECRET
          tokenEndpoint:
            value: "https://sso.token.endpoint"
`,
			expected: expected{
				manifest: persistence.Manifest{
					EnvironmentGroups: []persistence.Group{
						{
							Environments: []persistence.Environment{
								{
									Auth: persistence.Auth{
										OAuth: &persistence.OAuth{
											ClientID:      persistence.AuthSecret{Name: "ENV_CLIENT_ID"},
											ClientSecret:  persistence.AuthSecret{Name: "ENV_CLIENT_SECRET"},
											TokenEndpoint: &persistence.TypedValue{Value: "https://sso.token.endpoint"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "unmarshal manifest with parameters",
			given: `
manifestVersion: "1.0"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url:
      type: environment
      value: ENV_URL
    auth:
      token:
        name: ENV_TOKEN
parameters:
  owner: hansi
  department:
    type: environment
    name: ENV_VAR
`,
			expected: expected{
				manifest: persistence.Manifest{
					ManifestVersion: "1.0",
					Projects: []persistence.Project{
						{
							Name: "project",
						},
					},
					EnvironmentGroups: []persistence.Group{
						{
							Name: "default",
							Environments: []persistence.Environment{
								{
									Name: "env",
									URL: persistence.TypedValue{
										Type:  persistence.TypeEnvironment,
										Value: "ENV_URL",
									},
									Auth: persistence.Auth{
										Token: persistence.AuthSecret{
											Name: "ENV_TOKEN",
										},
									},
								},
							},
						},
					},
					Parameters: map[string]interface{}{
						"owner": "hansi",
						"department": map[interface{}]interface{}{
							"type": "environment",
							"name": "ENV_VAR",
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var actual persistence.Manifest
			err := yaml.UnmarshalStrict([]byte(tc.given), &actual)
			if tc.expected.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected.manifest, actual)
			}
		})
	}
}

func TestManifestVersionsCanBeParsedToVersionStruct(t *testing.T) {
	_, err := monacoVersion.ParseVersion(version.MinManifestVersion)
	assert.NoErrorf(t, err, "expected version.MinManifestVersion (%s) to parse to Version struct", version.MinManifestVersion)
	_, err = monacoVersion.ParseVersion(version.ManifestVersion)
	assert.NoErrorf(t, err, "expected version.ManifestVersion (%s) to parse to Version struct", version.ManifestVersion)
}

func Test_validateManifestVersion(t *testing.T) {
	tests := []struct {
		name            string
		manifestVersion string
		wantErr         bool
	}{
		{
			"no errs for current manifest version",
			version.ManifestVersion,
			false,
		},
		{
			"no errs for minimum supported manifest version",
			version.MinManifestVersion,
			false,
		},
		{
			"no error for short version",
			"1",
			false,
		},
		{
			"fails if version is garbage string",
			"just some random text that's not a version at all",
			true,
		},
		{
			"fails if semantic version is too long",
			"1.2.3.4.5",
			true,
		},
		{
			"fails if version is smaller than min supported",
			"0.0",
			true,
		},
		{
			"fails if version is large than current supported",
			fmt.Sprintf("%d.%d", math.MaxInt32, math.MaxInt32), //free bounds check for never overflowing version on 32bit binary
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateVersion(persistence.Manifest{ManifestVersion: tt.manifestVersion}); (err != nil) != tt.wantErr {
				t.Errorf("validateManifestVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadManifest(t *testing.T) {
	t.Setenv("e", "mock token")
	t.Setenv("token-env-var", "mock token")
	t.Setenv("empty-env-var", "")
	t.Setenv("client-id", "resolved-client-id")
	t.Setenv("client-secret", "resolved-client-secret")
	t.Setenv("ENV_OAUTH_ENDPOINT", "resolved-oauth-endpoint")

	tests := []struct {
		name             string
		manifestContent  string
		groups           []string
		envs             []string
		errsContain      []string
		expectedManifest manifest.Manifest
	}{
		{
			name:        "Everything missing",
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "Everything good",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}}]}]
`,
			errsContain: []string{},
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "Everything good with multiple environments in multiple groups",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"envA": {
						Name: "envA",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "Everything good with multiple environments in one group",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [
   {name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}},
   {name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}
  ]}
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"envA": {
						Name: "envA",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name:   "Only one env is loaded if group is loading restricted",
			groups: []string{"groupA"},
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"envA": {
						Name: "envA",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "Only one env is loaded if env is loading restricted",
			envs: []string{"envA"},
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"envA": {
						Name: "envA",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name:   "Two of three envs are loaded if env and group is loading restricted",
			envs:   []string{"envA"},
			groups: []string{"groupB"},
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupC, environments: [{name: envC, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"envA": {
						Name: "envA",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "Two of three envs are loaded if multiple envs restricted",
			envs: []string{"envA", "envB"},
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupC, environments: [{name: envC, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"envA": {
						Name: "envA",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name:   "Two of three envs are loaded if multiple groups restricted",
			groups: []string{"groupA", "groupB"},
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupC, environments: [{name: envC, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"envA": {
						Name: "envA",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name:   "Same configs in group and env restrictions",
			envs:   []string{"envA", "envB"},
			groups: []string{"groupA", "groupB"},
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupC, environments: [{name: envC, url: {value: "https://example.com"}, auth: {token: {name: token-does-not-exist-but-it-should-not-error-because-envC-is-not-loaded}}}]}
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"envA": {
						Name: "envA",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "Valid with parameters",
			manifestContent: `
manifestVersion: "1.0"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url: "https://test.test"
    auth:
      token:
        name: e
parameters:
  owner: hansi
  department:
    type: environment
    name: test-env-var
`,
			errsContain: []string{},
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"project": {
						Name: "project",
						Path: "project",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"env": {
						Group: "default",
						Name:  "env",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "https://test.test",
						},
						Auth: manifest.Auth{
							Token: manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
				Parameters: map[string]parameter.Parameter{
					"owner":      valueParam.New("hansi"),
					"department": environment.New("test-env-var"),
				},
			},
		},
		{
			name: "Invalid global parameter types produce error",
			manifestContent: `
manifestVersion: "1.0"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url: "https://test.test"
    auth:
      token:
        name: e
parameters:
  department:
    type: reference
    project: project
    configType: application-mobile
    configId: monaco-has-no-clue-I-exist-yet
`,
			errsContain:      []string{"invalid parameter type"},
			expectedManifest: manifest.Manifest{},
		},
		{
			name:   "Missing group errors",
			envs:   []string{"envA", "envB"},
			groups: []string{"groupA", "groupB", "doesnotexist"},
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupC, environments: [{name: envC, url: {value: "https://example.com"}, auth: {token: {name: token-does-not-exist-but-it-should-not-error-because-envC-is-not-loaded}}}]}
`,
			errsContain: []string{`requested group "doesnotexist" not found`},
		},
		{
			name:   "Missing env errors",
			envs:   []string{"envA", "envB", "doesnotexist"},
			groups: []string{"groupA", "groupB"},
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA, path: pathA}]
environmentGroups:
- {name: groupA, environments: [{name: envA, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupB, environments: [{name: envB, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
- {name: groupC, environments: [{name: envC, url: {value: "https://example.com"}, auth: {token: {name: token-env-var}}}]}
`,
			errsContain: []string{`requested environment "doesnotexist" not found`},
		},
		{
			name: "No manifestVersion",
			manifestContent: `
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}}]}]
`,
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "Invalid manifestVersion",
			manifestContent: `
manifestVersion: a
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
`,
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "Smaller version",
			manifestContent: `
manifestVersion: 0.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
`,
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "Larger Version",
			manifestContent: `
manifestVersion: 10000.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
`,
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "Projects are optional",
			manifestContent: `
manifestVersion: 1.0
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
`,
			expectedManifest: manifest.Manifest{
				Projects: manifest.ProjectDefinitionByProjectID{},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "environmentGroups are optional",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: projectA}]
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "projectA",
					},
				},
				Environments: nil,
				Accounts:     map[string]manifest.Account{},
			},
			errsContain: []string{},
		},
		{
			name: "Allow empty projects array",
			manifestContent: `
manifestVersion: 1.0
projects: []
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
`,
			expectedManifest: manifest.Manifest{
				Projects: manifest.ProjectDefinitionByProjectID{},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "Empty environments",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: []}]
`,
			errsContain: []string{"no environments"},
		},
		{
			name: "Duplicated environment names",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups:
  - {name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}
  - {name: f, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}
`,
			errsContain: []string{"duplicated environment name"},
		},
		{
			name: "Duplicated project names",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a},{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
`,
			errsContain: []string{"duplicated project name"},
		},
		{
			name: "Duplicated group names",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups:
  - {name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}
  - {name: b, environments: [{name: f, url: {value: d}, auth: {token: {name: e}}} ]}
`,
			errsContain: []string{"duplicated group name"},
		},
		{
			name: "Empty Groupname",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: '', environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
`,
			errsContain: []string{"missing group name"},
		},
		{
			name: "Invalid token-type",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e, type: f}}} ]}]
`,
			errsContain: []string{"type must be 'environment'"},
		},
		{
			name: "Empty token and no oauth",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: ''}}} ]}]
`,
			errsContain: []string{"failed to parse auth section: failed to parse token: no name given or empty"},
		},
		{
			name: "Empty url",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: ''}, auth: {token: {name: e}}} ]}]
`,
			errsContain: []string{"configured or value is blank"},
		},
		{
			name: "unknown url type",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d, type: f}, auth: {token: {name: e}}} ]}]
`,
			errsContain: []string{`"f" is not a valid URL type`},
		},
		{
			name: "env token not present",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: doesNotExist}}} ]}]
`,
			errsContain: []string{`environment-variable "doesNotExist" was not found`},
		},
		{
			name: "No errors with auth instead of token",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}}]}]
`,
			errsContain: []string{},
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "No errors with oAuth and token; OAuth token endpoint is not specified",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientId: {name: client-id}, clientSecret: {name: client-secret}}}}]}]
`,
			errsContain: []string{},
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
							OAuth: &manifest.OAuth{
								ClientID: manifest.AuthSecret{
									Name:  "client-id",
									Value: "resolved-client-id",
								},
								ClientSecret: manifest.AuthSecret{
									Name:  "client-secret",
									Value: "resolved-client-secret",
								},
								TokenEndpoint: nil,
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "No errors with oAuth and token; OAuth token endpoint is custom",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientId: {name: client-id}, clientSecret: {name: client-secret}, tokenEndpoint: {value: https://custom.sso.token.endpoint}}}}]}]
`,
			errsContain: []string{},
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
							OAuth: &manifest.OAuth{
								ClientID: manifest.AuthSecret{
									Name:  "client-id",
									Value: "resolved-client-id",
								},
								ClientSecret: manifest.AuthSecret{
									Name:  "client-secret",
									Value: "resolved-client-secret",
								},
								TokenEndpoint: &manifest.URLDefinition{
									Type:  manifest.ValueURLType,
									Value: "https://custom.sso.token.endpoint",
								},
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "No errors with oAuth and token; OAuth token endpoint is specified via environment variable",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientId: {name: client-id}, clientSecret: {name: client-secret}, tokenEndpoint: {type: environment, value: ENV_OAUTH_ENDPOINT}}}}]}]
`,
			errsContain: []string{},
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
							OAuth: &manifest.OAuth{
								ClientID: manifest.AuthSecret{
									Name:  "client-id",
									Value: "resolved-client-id",
								},
								ClientSecret: manifest.AuthSecret{
									Name:  "client-secret",
									Value: "resolved-client-secret",
								},
								TokenEndpoint: &manifest.URLDefinition{
									Type:  manifest.EnvironmentURLType,
									Name:  "ENV_OAUTH_ENDPOINT",
									Value: "resolved-oauth-endpoint",
								},
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "No errors with oAuth no token provided; OAuth token endpoint is specified via environment variable",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {oAuth: {clientId: {name: client-id}, clientSecret: {name: client-secret}, tokenEndpoint: {type: environment, value: ENV_OAUTH_ENDPOINT}}}}]}]
`,
			errsContain: []string{},
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: manifest.Auth{
							OAuth: &manifest.OAuth{
								ClientID: manifest.AuthSecret{
									Name:  "client-id",
									Value: "resolved-client-id",
								},
								ClientSecret: manifest.AuthSecret{
									Name:  "client-secret",
									Value: "resolved-client-secret",
								},
								TokenEndpoint: &manifest.URLDefinition{
									Type:  manifest.EnvironmentURLType,
									Name:  "ENV_OAUTH_ENDPOINT",
									Value: "resolved-oauth-endpoint",
								},
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
		},
		{
			name: "OAuth token endpoint is specified via environment variable that doesn't exists",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientId: {name: client-id}, clientSecret: {name: client-secret}, tokenEndpoint: {type: environment, value: ENV_NOT_EXISTS}}}}]}]
`,
			errsContain: []string{"environment variable \"ENV_NOT_EXISTS\" could not be found"},
		},
		{
			name: "OAuth token endpoint is specified with nonexistent type",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientId: {name: client-id}, clientSecret: {name: client-secret}, tokenEndpoint: {type: nonexistent, value: ENV_NOT_EXISTS}}}}]}]
`,
			errsContain: []string{"\"nonexistent\" is not a valid URL type"},
		},
		{
			name: "OAuth credentials are missing the ClientSecret",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientId: {name: client-id}}}}]}]
`,
			errsContain: []string{"failed to parse ClientSecret"},
		},
		{
			name: "No auth configured",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}}]}]
`,
			errsContain: []string{"no token or OAuth credentials provided"},
		},
		{
			name: "Unknown type",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {type: x}}}]}]
`,
			errsContain: []string{"type must be 'environment'"},
		},
		{
			name: "load url from env var",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {type: environment, value: e}, auth: {token: {name: e}}}]}]
`,
			expectedManifest: manifest.Manifest{
				Projects: map[string]manifest.ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]manifest.EnvironmentDefinition{
					"c": {
						Name: "c",
						URL: manifest.URLDefinition{
							Type:  manifest.EnvironmentURLType,
							Value: "mock token",
							Name:  "e",
						},
						Group: "b",
						Auth: manifest.Auth{
							Token: &manifest.AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
				Accounts: map[string]manifest.Account{},
			},
			errsContain: []string{},
		},
		{
			name: "load url from env var but value is empty",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {type: environment, value: empty-env-var}, auth: {token: {name: e}}}]}]
`,
			errsContain: []string{"is defined but has no value"},
		},
		{
			name: "load url from env var but not found",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {type: environment, value: not-found}, auth: {token: {name: e}}}]}]
`,
			errsContain: []string{"could not be found"},
		},
		{
			name: "token env var not found",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {type: environment, name: not-found}}}]}]
`,
			errsContain: []string{`environment-variable "not-found" was not found`},
		},
		{
			name: "token env var not set",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {type: environment, name: ""}}}]}]
`,
			errsContain: []string{"empty"},
		},
		{
			name: "ClientID empty var name",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientId: {type: environment, name: ""}, clientSecret: {name: client-secret}}}}]}]
`,
			errsContain: []string{"no name given or empty"},
		},
		{
			name: "ClientSecret empty var name",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientSecret: {type: environment, name: ""}, clientId: {name: client-id}}}}]}]
`,
			errsContain: []string{"no name given or empty"},
		},
		{
			name: "ClientID env var not found",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientId: {type: environment, name: "not-found"}, clientSecret: {name: client-secret}}}}]}]
`,
			errsContain: []string{`environment-variable "not-found" was not found`},
		},
		{
			name: "ClientSecret env var not found",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}, oAuth: {clientSecret: {type: environment, name: "not-found"}, clientId: {name: client-id}}}}]}]
`,
			errsContain: []string{`environment-variable "not-found" was not found`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			assert.NoError(t, afero.WriteFile(fs, "manifest.yaml", []byte(test.manifestContent), 0400))

			mani, errs := Load(&Context{
				Fs:           fs,
				ManifestPath: "manifest.yaml",
				Groups:       test.groups,
				Environments: test.envs,
			})

			if len(errs) == len(test.errsContain) {
				for i := range test.errsContain {
					assert.ErrorContains(t, errs[i], test.errsContain[i])
				}
			} else {
				t.Errorf("Unexpected amount of errors: %#v", errs)
			}

			assert.Equal(t, test.expectedManifest, mani)
		})
	}
}

func TestEnvVarResolutionCanBeDeactivated(t *testing.T) {
	e := persistence.Environment{
		Name: "TEST ENV",
		URL:  persistence.TypedValue{Value: "TEST_TOKEN", Type: persistence.TypeEnvironment},
		Auth: persistence.Auth{
			Token: &persistence.AuthSecret{Type: "environment", Name: "VAR"},
			OAuth: &persistence.OAuth{
				ClientID:     persistence.AuthSecret{Type: "environment", Name: "VAR_1"},
				ClientSecret: persistence.AuthSecret{Type: "environment", Name: "VAR_2"},
			},
		},
	}

	t.Run("URLs resolution produces error", func(t *testing.T) {
		_, gotErr := parseURLDefinition(&Context{}, e.URL)
		assert.Error(t, gotErr)
	})

	t.Run("URLs are not resolved if 'DoNotResolveEnvVars' option is set", func(t *testing.T) {
		_, gotErr := parseURLDefinition(&Context{Opts: Options{DoNotResolveEnvVars: true}}, e.URL)
		assert.NoError(t, gotErr)
	})

	t.Run("Auth token resolution produces error", func(t *testing.T) {
		_, gotErr := parseAuth(&Context{}, e.Auth)
		assert.Error(t, gotErr)
	})

	t.Run("Auth tokens are not resolved if 'DoNotResolveEnvVars' option is set", func(t *testing.T) {
		_, gotErr := parseAuth(&Context{Opts: Options{DoNotResolveEnvVars: true}}, e.Auth)
		assert.NoError(t, gotErr)
	})
}

func TestEnvironmentsAndAccountsAreOptionalUnlessDefined(t *testing.T) {
	tests := []struct {
		name                 string
		givenManifestContent string
		givenOptions         Options
		wantErr              bool
	}{
		{
			"optional by default",
			`
manifestVersion: 1.0
projects: [{name: a, path: p}]
`,
			Options{},
			false,
		},
		{
			"missing accounts produce error if required",
			`
manifestVersion: 1.0
projects: [{name: a, path: p}]
`,
			Options{RequireAccounts: true},
			true,
		},
		{
			"missing environmentGroups produce error if required",
			`
manifestVersion: 1.0
projects: [{name: a, path: p}]
`,
			Options{RequireEnvironmentGroups: true},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			assert.NoError(t, afero.WriteFile(fs, "manifest.yaml", []byte(tt.givenManifestContent), 0400))

			_, errs := Load(&Context{
				Fs:           fs,
				ManifestPath: "manifest.yaml",
				Opts:         tt.givenOptions,
			})

			if tt.wantErr {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
			}

		})
	}
}
