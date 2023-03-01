//go:build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package manifest

import (
	"fmt"
	version2 "github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/afero"
	"math"
	"reflect"
	"testing"

	"gotest.tools/assert"
)

var testTokenCfg = tokenConfig{Type: "environment", Name: "VAR"}

func Test_extractUrlType(t *testing.T) {
	tests := []struct {
		name        string
		inputConfig environment
		want        UrlType
		wantErr     bool
	}{
		{
			"extracts_value_url",
			environment{
				Name:  "TEST ENV",
				Url:   url{Value: "TEST URL", Type: "value"},
				Token: testTokenCfg,
			},
			ValueUrlType,
			false,
		},
		{
			"extracts_value_if_type_empty",
			environment{
				Name:  "TEST ENV",
				Url:   url{Value: "TEST URL", Type: ""},
				Token: testTokenCfg,
			},
			ValueUrlType,
			false,
		},
		{
			"extracts_environment_url",
			environment{
				Name:  "TEST ENV",
				Url:   url{Value: "TEST URL", Type: "environment"},
				Token: testTokenCfg,
			},
			EnvironmentUrlType,
			false,
		},
		{
			"fails_on_unknown_type",
			environment{
				Name:  "TEST ENV",
				Url:   url{Value: "TEST URL", Type: "this-is-not-a-type"},
				Token: testTokenCfg,
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, gotErr := extractUrlType(tt.inputConfig); got != tt.want || (!tt.wantErr && gotErr != nil) {
				t.Errorf("extractUrlType() = %v, %v, want %v, %v", got, gotErr, tt.want, tt.wantErr)
			}
		})
	}
}

func Test_parseProjectDefinition_SimpleType(t *testing.T) {
	type args struct {
		context *projectLoaderContext
		project project
	}
	tests := []struct {
		name     string
		args     args
		want     []ProjectDefinition
		wantErrs []error
	}{
		{
			"parses_simple_project",
			args{
				context: nil,
				project: project{
					Name: "PROJ_NAME",
					Type: simpleProjectType,
					Path: "PROJ_PATH",
				},
			},
			[]ProjectDefinition{
				{
					Name: "PROJ_NAME",
					Path: "PROJ_PATH",
				},
			},
			nil,
		},
		{
			"parses_simple_project_when_type_omitted",
			args{
				context: nil,
				project: project{
					Name: "PROJ_NAME",
					Path: "PROJ_PATH",
				},
			},
			[]ProjectDefinition{
				{
					Name: "PROJ_NAME",
					Path: "PROJ_PATH",
				},
			},
			nil,
		},
		{
			"sets_project_name_as_path_if_no_path_set",
			args{
				context: nil,
				project: project{
					Name: "PROJ_NAME",
				},
			},
			[]ProjectDefinition{
				{
					Name: "PROJ_NAME",
					Path: "PROJ_NAME",
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErrs := parseProjectDefinition(tt.args.context, tt.args.project)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseProjectDefinition() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(gotErrs, tt.wantErrs) {
				t.Errorf("parseProjectDefinition() gotErrs = %v, wantErrs %v", gotErrs, tt.wantErrs)
			}
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
	project := project{
		Name: "PROJ_NAME",
		Type: groupProjectType,
		Path: "PROJ_PATH",
	}

	want := []ProjectDefinition{
		{
			Name:  "PROJ_NAME.a",
			Group: "PROJ_NAME",
			Path:  "PROJ_PATH/a",
		},
		{
			Name:  "PROJ_NAME.b",
			Group: "PROJ_NAME",
			Path:  "PROJ_PATH/b",
		},
	}
	got, gotErrs := parseProjectDefinition(&context, project)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseProjectDefinition() got = %v, want %v", got, want)
	}

	assert.Assert(t, len(gotErrs) == 0)
}

func Test_parseProjectDefinition_FailsOnUnknownType(t *testing.T) {
	context := projectLoaderContext{
		fs:           nil,
		manifestPath: ".",
	}
	project := project{
		Name: "PROJ_NAME",
		Type: "not-a-project-type",
		Path: "PROJ_PATH",
	}

	_, gotErrs := parseProjectDefinition(&context, project)

	assert.Assert(t, len(gotErrs) == 1)
	assert.ErrorType(t, gotErrs[0], ManifestProjectLoaderError{})
}

func Test_parseProjectDefinition_FailsOnInvalidProjectDefinitions(t *testing.T) {
	context := projectLoaderContext{
		fs:           afero.NewMemMapFs(),
		manifestPath: ".",
	}

	tests := []struct {
		name    string
		project project
	}{
		{
			"invalid simple project",
			project{
				Name: "",
				Path: "",
			},
		},
		{
			"grouping dir that does not exist",
			project{
				Name: "a grouping",
				Type: groupProjectType,
				Path: "path/that/wont/be/found",
			},
		},
		{
			"name containing path separators",
			project{
				Name: "names/must/not/be\\paths",
				Path: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErrs := parseProjectDefinition(&context, tt.project)

			assert.Assert(t, len(gotErrs) == 1)
			assert.ErrorType(t, gotErrs[0], ManifestProjectLoaderError{})
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
		projectDefinitions []project
		want               map[string]ProjectDefinition
		wantErrs           bool
	}{
		{
			"returns error on duplicate project id",
			[]project{
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
			[]project{
				{
					Name: "project_id",
					Path: "project/path/",
				},
				{
					Name: "project_id",
					Type: groupProjectType,
					Path: "another/project/path/",
				},
			},
			nil,
			true,
		},
		{
			"returns error on duplicate project id between grouping and grouping",
			[]project{
				{
					Name: "project_id",
					Type: groupProjectType,
					Path: "project/path/",
				},
				{
					Name: "project_id",
					Type: groupProjectType,
					Path: "another/project/path/",
				},
			},
			nil,
			true,
		},
		{
			"returns error on duplicate project id between simple and sub-project in a group",
			[]project{
				{
					Name: "project_id.a",
					Path: "some/project/path/",
				},
				{
					Name: "project_id", //this group will contain 'project_id.a' & 'project_id.b' projects
					Type: groupProjectType,
					Path: "project/path/",
				},
			},
			nil,
			true,
		},
		{
			"returns error if grouping project path can not be read",
			[]project{
				{
					Name: "project_id",
					Type: groupProjectType,
					Path: "this/path/does/not/exist",
				},
			},
			nil,
			true,
		},
		{
			"returns error if project is invalid (empty)",
			[]project{
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
			[]project{
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
			[]project{
				{
					Name: "project_id",
					Type: groupProjectType,
					Path: "empty/project/path/",
				},
			},
			nil,
			true,
		},
		{
			"correctly parses project definition",
			[]project{
				{
					Name: "project_id_1",
					Path: "project/path/",
				},
				{
					Name: "project_id_2",
					Type: groupProjectType,
					Path: "another/project/path/",
				},
			},
			map[string]ProjectDefinition{
				"project_id_1": {
					Name: "project_id_1",
					Path: "project/path/",
				},
				"project_id_2.one": {
					Name:  "project_id_2.one",
					Group: "project_id_2",
					Path:  "another/project/path/one",
				},
				"project_id_2.two": {
					Name:  "project_id_2.two",
					Group: "project_id_2",
					Path:  "another/project/path/two",
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := &projectLoaderContext{testFs, "path/to/a/manifest.yaml"}

			got, gotErrs := toProjectDefinitions(context, tt.projectDefinitions)

			numErrs := len(gotErrs)
			if (tt.wantErrs && numErrs <= 0) || (!tt.wantErrs && numErrs > 0) {
				t.Errorf("toProjectDefinitions() returned unexpected Errors = %v", gotErrs)
			}

			assert.DeepEqual(t, got, tt.want, cmpopts.SortSlices(func(a, b ProjectDefinition) bool { return a.Name < b.Name }))
		})
	}
}

func Test_parseManifestFile(t *testing.T) {
	tests := []struct {
		name     string
		context  *ManifestLoaderContext
		data     string
		want     manifest
		wantErrs bool
	}{
		{
			"parses simple manifest",
			&ManifestLoaderContext{},
			fmt.Sprintf(
				`manifestVersion: "%s"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url:
      type: environment
      value: ENV_URL
    token:
      name: ENV_TOKEN
`, version.ManifestVersion),
			manifest{
				ManifestVersion: version.ManifestVersion,
				Projects: []project{
					{
						Name: "project",
					},
				},
				EnvironmentGroups: []group{
					{
						Name: "default",
						Environments: []environment{
							{
								Name: "env",
								Url: url{
									Type:  "environment",
									Value: "ENV_URL",
								},
								Token: tokenConfig{
									Name: "ENV_TOKEN",
								},
							},
						},
					},
				},
			},
			false,
		},
		{
			"fails on missing version",
			&ManifestLoaderContext{},
			`projects:
- name: project
environments:
- group: default
  entries:
  - name: env
    url:
      type: environment
      value: ENV_URL
    token:
      name: ENV_TOKEN
`,
			manifest{},
			true,
		},
		{
			"fails on missing projects",
			&ManifestLoaderContext{},
			fmt.Sprintf(
				`manifestVersion: "%s"
environments:
- group: default
  entries:
  - name: env
    url:
      type: environment
      value: ENV_URL
    token:
      name: ENV_TOKEN
`, version.ManifestVersion),
			manifest{},
			true,
		},
		{
			"fails on missing environments",
			&ManifestLoaderContext{},
			fmt.Sprintf(
				`manifestVersion: "%s"
projects:
- name: project
`, version.ManifestVersion),
			manifest{},
			true,
		},
		{
			"fails on duplicate project definitions",
			&ManifestLoaderContext{},
			fmt.Sprintf(
				`manifestVersion: "%s"
projects:
- name: project
projects:
- name: project2
environments:
- group: default
  entries:
  - name: env
    url:
      type: environment
      value: ENV_URL
    token:
      name: ENV_TOKEN
`, version.ManifestVersion),
			manifest{},
			true,
		},
		{
			"fails on no longer supported manifest version",
			&ManifestLoaderContext{},
			`manifestVersion: 0.0
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url:
      type: environment
      value: ENV_URL
    token:
      name: ENV_TOKEN
`,
			manifest{},
			true,
		},
		{
			"fails on not yet supported manifest version",
			&ManifestLoaderContext{},
			fmt.Sprintf(
				`manifestVersion: "%s"
projects:
- name: project
projects:
- name: project2
environmentGroupss:
- name: default
  environments:
  - name: env
    url:
      type: environment
      value: ENV_URL
    token:
      name: ENV_TOKEN
`, fmt.Sprintf("%d.%d", math.MaxInt32, math.MaxInt32)),
			manifest{},
			true,
		},
		{
			"fails on malformed manifest version",
			&ManifestLoaderContext{},
			`manifestVersion: "random text"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url:
      type: environment
      value: ENV_URL
    token:
      name: ENV_TOKEN
`,
			manifest{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErrs := parseManifestFile(tt.context, []byte(tt.data))
			if (tt.wantErrs && len(gotErrs) < 1) || (!tt.wantErrs && len(gotErrs) > 0) {
				t.Errorf("parseManifest() gotErrs = %v, wantErrs = %v", gotErrs, tt.wantErrs)
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseManifest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManifestVersionsCanBeParsedToVersionStruct(t *testing.T) {
	_, err := version2.ParseVersion(version.MinManifestVersion)
	assert.NilError(t, err, "expected version.MinManifestVersion (%s) to parse to Version struct", version.MinManifestVersion)
	_, err = version2.ParseVersion(version.ManifestVersion)
	assert.NilError(t, err, "expected version.ManifestVersion (%s) to parse to Version struct", version.ManifestVersion)
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
			"fails if semantic version is too short",
			"1",
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
			if err := validateManifestVersion(tt.manifestVersion); (err != nil) != tt.wantErr {
				t.Errorf("validateManifestVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadManifest(t *testing.T) {
	t.Setenv("e", "mock token")

	tests := []struct {
		name             string
		manifestContent  string
		errsContain      []string
		expectedManifest Manifest
	}{
		{
			name:        "Everything missing",
			errsContain: []string{"manifestVersion", "project", "environmentGroups"},
		},
		{
			name: "Everything good",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{},
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"c": {
						Name: "c",
						Type: Classic,
						url: UrlDefinition{
							Type:  "value",
							Value: "d",
						},
						Group: "b",
						Token: Token{
							Name:  "e",
							Value: "mock token",
						},
					},
				},
			},
		},
		{
			name: "No errors with type = Platform",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}, type: Platform}]}]
`,
			errsContain: []string{},
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"c": {
						Name: "c",
						Type: Platform,
						url: UrlDefinition{
							Type:  "value",
							Value: "d",
						},
						Group: "b",
						Token: Token{
							Name:  "e",
							Value: "mock token",
						},
					},
				},
			},
		},
		{
			name: "No errors with type = Classic",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a, path: p}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}, type: Classic}]}]
`,
			errsContain: []string{},
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"a": {
						Name: "a",
						Path: "p",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"c": {
						Name: "c",
						Type: Classic,
						url: UrlDefinition{
							Type:  "value",
							Value: "d",
						},
						Group: "b",
						Token: Token{
							Name:  "e",
							Value: "mock token",
						},
					},
				},
			},
		},
		{
			name: "No manifestVersion",
			manifestContent: `
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "Invalid manifestVersion",
			manifestContent: `
manifestVersion: a
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "Smaller version",
			manifestContent: `
manifestVersion: 0.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "Larger Version",
			manifestContent: `
manifestVersion: 10000.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{"manifestVersion"},
		},
		{
			name: "No projects",
			manifestContent: `
manifestVersion: 1.0
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{"projects"},
		},
		{
			name: "No environmentGroups",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
`,
			errsContain: []string{"environmentGroups"},
		},
		{
			name: "Empty projects",
			manifestContent: `
manifestVersion: 1.0
projects: []
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{"projects"},
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
  - {name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}
  - {name: f, environments: [{name: c, url: {value: d}, token: {name: e}}]}
`,
			errsContain: []string{"duplicated environment name"},
		},
		{
			name: "Duplicated project names",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a},{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{"duplicated project name"},
		},
		{
			name: "Duplicated group names",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups:
  - {name: b, environments: [{name: c, url: {value: d}, token: {name: e}}]}
  - {name: b, environments: [{name: f, url: {value: d}, token: {name: e}}]}
`,
			errsContain: []string{"duplicated group name"},
		},
		{
			name: "Empty Groupname",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: '', environments: [{name: c, url: {value: d}, token: {name: e}}]}]
`,
			errsContain: []string{"missing group name"},
		},
		{
			name: "Invalid token-type",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e, type: f}}]}]
`,
			errsContain: []string{"unknown token type"},
		},
		{
			name: "Empty token",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: ''}}]}]
`,
			errsContain: []string{"missing or empty"},
		},
		{
			name: "Empty url",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: ''}, token: {name: e}}]}]
`,
			errsContain: []string{"configured or value is blank"},
		},
		{
			name: "unknown url type",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d, type: f}, token: {name: e}}]}]
`,
			errsContain: []string{"f is not a valid"},
		},
		{
			name: "env token not present",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: doesNotExist}}]}]
`,
			errsContain: []string{`no environment variable found for token "doesNotExist"`},
		},
		{
			name: "config type is invalid",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, token: {name: e}, type: doesNotExist}]}]
`,
			errsContain: []string{`invalid environment-type "doesNotExist"`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			assert.NilError(t, afero.WriteFile(fs, "manifest.yaml", []byte(test.manifestContent), 0400))

			mani, errs := LoadManifest(&ManifestLoaderContext{
				Fs:           fs,
				ManifestPath: "manifest.yaml",
			})

			if len(errs) == len(test.errsContain) {
				for i := range test.errsContain {
					assert.ErrorContains(t, errs[i], test.errsContain[i])
				}
			} else {
				t.Errorf("Unexpected amount of errors: %#v", errs)
			}

			assert.DeepEqual(t, test.expectedManifest, mani, cmp.AllowUnexported(EnvironmentDefinition{}))

		})
	}
}
