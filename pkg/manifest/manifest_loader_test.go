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
			Name: "PROJ_NAME.a",
			Path: "PROJ_PATH/a",
		},
		{
			Name: "PROJ_NAME.b",
			Path: "PROJ_PATH/b",
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
