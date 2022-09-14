//go:build unit
// +build unit

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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/afero"
	"reflect"
	"testing"

	"gotest.tools/assert"
)

var testTokenCfg = tokenConfig{Type: "environment", Config: map[string]interface{}{"name": "VAR"}}

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
