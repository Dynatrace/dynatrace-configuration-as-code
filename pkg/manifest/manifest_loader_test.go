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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	monacoVersion "github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"math"
	"reflect"
	"testing"
)

func Test_extractUrlType(t *testing.T) {

	tests := []struct {
		name             string
		inputConfig      environment
		givenEnvVarValue string
		want             URLDefinition
		wantErr          bool
	}{
		{
			name: "extracts_value_url",
			inputConfig: environment{
				Name: "TEST ENV",
				URL:  url{Value: "TEST URL", Type: urlTypeValue},
				Auth: auth{Token: authSecret{Type: "environment", Name: "VAR"}},
			},
			want: URLDefinition{
				Type:  ValueURLType,
				Value: "TEST URL",
			},
			wantErr: false,
		},
		{
			name: "extracts_value_if_type_empty",
			inputConfig: environment{
				Name: "TEST ENV",
				URL:  url{Value: "TEST URL", Type: ""},
				Auth: auth{Token: authSecret{Type: "environment", Name: "VAR"}},
			},
			want: URLDefinition{
				Type:  ValueURLType,
				Value: "TEST URL",
			},
			wantErr: false,
		},
		{
			name: "trims trailing slash from value url",
			inputConfig: environment{
				Name: "TEST ENV",
				URL:  url{Value: "https://www.test.url/", Type: urlTypeValue},
				Auth: auth{Token: authSecret{Type: "environment", Name: "VAR"}},
			},
			want: URLDefinition{
				Type:  ValueURLType,
				Value: "https://www.test.url",
			},
			wantErr: false,
		},
		{
			name: "extracts_environment_url",
			inputConfig: environment{
				Name: "TEST ENV",
				URL:  url{Value: "TEST_TOKEN", Type: urlTypeEnvironment},
				Auth: auth{Token: authSecret{Type: "environment", Name: "VAR"}},
			},
			givenEnvVarValue: "resolved url value",
			want: URLDefinition{
				Type:  EnvironmentURLType,
				Name:  "TEST_TOKEN",
				Value: "resolved url value",
			},
			wantErr: false,
		},
		{
			name: "trims trailing slash from environment url",
			inputConfig: environment{
				Name: "TEST ENV",
				URL:  url{Value: "TEST_TOKEN", Type: urlTypeEnvironment},
				Auth: auth{Token: authSecret{Type: "environment", Name: "VAR"}},
			},
			givenEnvVarValue: "https://www.test.url/",
			want: URLDefinition{
				Type:  EnvironmentURLType,
				Name:  "TEST_TOKEN",
				Value: "https://www.test.url",
			},
			wantErr: false,
		},
		{
			name: "fails_on_unknown_type",
			inputConfig: environment{
				Name: "TEST ENV",
				URL:  url{Value: "TEST URL", Type: "this-is-not-a-type"},
				Auth: auth{Token: authSecret{Type: "environment", Name: "VAR"}},
			},
			want:    URLDefinition{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_TOKEN", tt.givenEnvVarValue)
			if got, gotErr := parseURLDefinition(tt.inputConfig.URL); got != tt.want || (!tt.wantErr && gotErr != nil) {
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
	project := project{
		Name: "PROJ_NAME",
		Type: "not-a-project-type",
		Path: "PROJ_PATH",
	}

	_, gotErrs := parseProjectDefinition(&context, project)

	assert.Len(t, gotErrs, 1)
	assert.IsType(t, projectLoaderError{}, gotErrs[0])
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

			assert.Len(t, gotErrs, 1)
			assert.IsType(t, projectLoaderError{}, gotErrs[0])
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

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerifyManifestYAML(t *testing.T) {
	type given struct {
		manifest manifest
	}
	type expected struct {
		errorMessage string
	}

	var tests = []struct {
		name     string
		given    given
		expected expected
	}{
		{
			name:     "fails on missing version",
			given:    given{manifest: manifest{}},
			expected: expected{errorMessage: "`manifestVersion` missing"},
		},
		{
			name:     "fails on missing projects",
			given:    given{},
			expected: expected{errorMessage: "no `projects` defined"},
		},
		{
			name:     "fails on missing environments",
			given:    given{},
			expected: expected{errorMessage: "no `environmentGroups` defined"},
		},
		{
			name:     "fails on missing version",
			given:    given{manifest: manifest{}},
			expected: expected{errorMessage: "`manifestVersion` missing"},
		},
		{
			name: "fails on no longer supported manifest version",
			given: given{
				manifest: manifest{
					ManifestVersion: "0.0",
				},
			},
			expected: expected{errorMessage: "`manifestVersion` 0.0 is no longer supported. Min required version is 1.0, please update manifest"},
		},
		{
			name: "fails on not yet supported manifest version",
			given: given{
				manifest: manifest{
					ManifestVersion: fmt.Sprintf("%d.%d", math.MaxInt32, math.MaxInt32),
				},
			},
			expected: expected{errorMessage: fmt.Sprintf("`manifestVersion` %d.%d is not supported by monaco 2.x. Max supported version is 1.0, please check manifest or update monaco", math.MaxInt32, math.MaxInt32)},
		},
		{
			name: "fails on malformed manifest version",
			given: given{
				manifest: manifest{
					ManifestVersion: "random text",
				},
			},
			expected: expected{errorMessage: "invalid `manifestVersion`: failed to parse version: format did not meet expected MAJOR.MINOR or MAJOR.MINOR.PATCH pattern: random text"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := verifyManifestYAML(tt.given.manifest)
			var errorMessages []string
			for _, err := range errors {
				errorMessages = append(errorMessages, err.Error())
			}
			fmt.Println(errors)
			assert.Contains(t, errorMessages, tt.expected.errorMessage)
		})
	}
}

func TestUnmarshallingYAML(t *testing.T) {
	type expected struct {
		manifest manifest
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
				manifest: manifest{
					ManifestVersion: "1.0",
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
									URL: url{
										Type:  urlTypeEnvironment,
										Value: "ENV_URL",
									},
									Auth: auth{
										Token: authSecret{
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
				manifest: manifest{
					EnvironmentGroups: []group{
						{
							Environments: []environment{
								{
									Auth: auth{
										OAuth: &oAuth{
											ClientID:      authSecret{Name: "ENV_CLIENT_ID"},
											ClientSecret:  authSecret{Name: "ENV_CLIENT_SECRET"},
											TokenEndpoint: &url{Value: "https://sso.token.endpoint"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var actual manifest
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
	t.Setenv("token-env-var", "mock token")
	t.Setenv("empty-env-var", "")
	t.Setenv("client-id", "resolved-client-id")
	t.Setenv("client-secret", "resolved-client-secret")
	t.Setenv("ENV_OAUTH_ENDPOINT", "resolved-oauth-endpoint")

	log.Default().SetLevel(log.LevelDebug)

	tests := []struct {
		name            string
		manifestContent string
		groups          []string
		envs            []string

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
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}}]}]
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
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
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
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"envA": {
						Name: "envA",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
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
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"envA": {
						Name: "envA",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
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
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"envA": {
						Name: "envA",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
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
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"envA": {
						Name: "envA",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
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
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"envA": {
						Name: "envA",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
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
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"envA": {
						Name: "envA",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
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
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"envA": {
						Name: "envA",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
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
			expectedManifest: Manifest{
				Projects: map[string]ProjectDefinition{
					"projectA": {
						Name: "projectA",
						Path: "pathA",
					},
				},
				Environments: map[string]EnvironmentDefinition{
					"envA": {
						Name: "envA",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupA",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
					"envB": {
						Name: "envB",
						Type: Classic,
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "https://example.com",
						},
						Group: "groupB",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "token-env-var",
								Value: "mock token",
							},
						},
					},
				},
			},
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
			name: "No projects",
			manifestContent: `
manifestVersion: 1.0
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
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
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: e}}} ]}]
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
			name: "Empty token",
			manifestContent: `
manifestVersion: 1.0
projects: [{name: a}]
environmentGroups: [{name: b, environments: [{name: c, url: {value: d}, auth: {token: {name: ''}}} ]}]
`,
			errsContain: []string{"no name given or empty"},
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
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
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
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
							OAuth: OAuth{
								ClientID: AuthSecret{
									Name:  "client-id",
									Value: "resolved-client-id",
								},
								ClientSecret: AuthSecret{
									Name:  "client-secret",
									Value: "resolved-client-secret",
								},
								TokenEndpoint: nil,
							},
						},
					},
				},
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
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
							OAuth: OAuth{
								ClientID: AuthSecret{
									Name:  "client-id",
									Value: "resolved-client-id",
								},
								ClientSecret: AuthSecret{
									Name:  "client-secret",
									Value: "resolved-client-secret",
								},
								TokenEndpoint: &URLDefinition{
									Type:  ValueURLType,
									Value: "https://custom.sso.token.endpoint",
								},
							},
						},
					},
				},
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
						URL: URLDefinition{
							Type:  ValueURLType,
							Value: "d",
						},
						Group: "b",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
							OAuth: OAuth{
								ClientID: AuthSecret{
									Name:  "client-id",
									Value: "resolved-client-id",
								},
								ClientSecret: AuthSecret{
									Name:  "client-secret",
									Value: "resolved-client-secret",
								},
								TokenEndpoint: &URLDefinition{
									Type:  EnvironmentURLType,
									Name:  "ENV_OAUTH_ENDPOINT",
									Value: "resolved-oauth-endpoint",
								},
							},
						},
					},
				},
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
			errsContain: []string{"failed to parse auth section"},
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
						URL: URLDefinition{
							Type:  EnvironmentURLType,
							Value: "mock token",
							Name:  "e",
						},
						Group: "b",
						Auth: Auth{
							Token: AuthSecret{
								Name:  "e",
								Value: "mock token",
							},
						},
					},
				},
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

			mani, errs := LoadManifest(&LoaderContext{
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
