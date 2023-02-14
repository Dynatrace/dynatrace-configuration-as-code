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

package template

import (
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"reflect"
	"testing"
)

func TestCreateTemplateFromString(t *testing.T) {
	type args struct {
		path    string
		content string
	}
	tests := []struct {
		name string
		args args
		want Template
	}{
		{
			"simple template created",
			args{
				"a/file/path.json",
				" { file: content } ",
			},
			&fileBasedTemplate{
				path:    "a/file/path.json",
				content: " { file: content } ",
			},
		},
		{
			"works on empty inputs",
			args{
				"",
				"",
			},
			&fileBasedTemplate{
				path:    "",
				content: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateTemplateFromString(tt.args.path, tt.args.content)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateTemplateFromString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadTemplate(t *testing.T) {
	testFilepath := "proj/api/template.json"

	testFs := afero.NewMemMapFs()
	_ = testFs.MkdirAll("proj/api/", 0755)
	_ = afero.WriteFile(testFs, testFilepath, []byte("{ template: from_file }"), 0644)

	got, gotErr := LoadTemplate(testFs, testFilepath)
	assert.NilError(t, gotErr)
	assert.Equal(t, got.Id(), testFilepath)
	assert.Equal(t, got.Name(), testFilepath)
	assert.Equal(t, got.(FileBasedTemplate).FilePath(), testFilepath)
	assert.Equal(t, got.Content(), "{ template: from_file }")
}

func TestLoadTemplate_ReturnsErrorIfFileDoesNotExist(t *testing.T) {
	testFilepath := "proj/api/template.json"

	testFs := afero.NewMemMapFs()

	_, gotErr := LoadTemplate(testFs, testFilepath)
	assert.ErrorContains(t, gotErr, testFilepath)
}

func Test_fileBasedTemplate_Id_Returns_Path(t *testing.T) {
	template := fileBasedTemplate{
		path:    "PATH",
		content: "CONT",
	}

	assert.Equal(t, template.Id(), "PATH")
}

func Test_fileBasedTemplate_Name_Returns_Path(t *testing.T) {
	template := fileBasedTemplate{
		path:    "PATH",
		content: "CONT",
	}

	assert.Equal(t, template.Name(), "PATH")
}

func Test_fileBasedTemplate_Filepath_Returns_Path(t *testing.T) {
	template := fileBasedTemplate{
		path:    "PATH",
		content: "CONT",
	}

	assert.Equal(t, template.FilePath(), "PATH")
}

func Test_fileBasedTemplate_Content_Returns_Content(t *testing.T) {
	template := fileBasedTemplate{
		path:    "PATH",
		content: "CONT",
	}

	assert.Equal(t, template.Content(), "CONT")
}
