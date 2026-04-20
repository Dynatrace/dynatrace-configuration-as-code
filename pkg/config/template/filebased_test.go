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

package template_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTemplate(t *testing.T) {
	testFilepath := filepath.FromSlash("proj/api/template.json")

	testFs := afero.NewMemMapFs()
	_ = testFs.MkdirAll("proj/api/", 0755)
	_ = afero.WriteFile(testFs, testFilepath, []byte("{ template: from_file }"), 0644)

	got, gotErr := template.NewFileTemplate(testFs, testFilepath)
	require.NoError(t, gotErr)
	assert.Equal(t, testFilepath, got.ID())
	assert.Equal(t, testFilepath, got.(*template.FileBasedTemplate).FilePath())
	gotContent, err := got.Content()
	assert.NoError(t, err)
	assert.Equal(t, "{ template: from_file }", gotContent)
}

func TestLoadTemplate_ReturnsErrorIfFileDoesNotExist(t *testing.T) {
	testFilepath := filepath.FromSlash("proj/api/template.json")

	testFs := afero.NewMemMapFs()

	_, gotErr := template.NewFileTemplate(testFs, testFilepath)
	require.ErrorContains(t, gotErr, testFilepath)
}

func TestLoadTemplate_WorksWithAnyPathSeparator(t *testing.T) {

	testFs := afero.NewReadOnlyFs(afero.NewOsFs())
	tests := []struct {
		name          string
		givenFilepath string
	}{
		{
			"windows path",
			`test-resources\template.json`,
		},
		{
			"relative windows path",
			`..\template\test-resources\template.json`,
		},
		{
			"unix path",
			`test-resources/template.json`,
		},
		{
			"relative unix path",
			`../template/test-resources/template.json`,
		},
		{
			"mixed path",
			`..\template\test-resources/template.json`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErr := template.NewFileTemplate(testFs, tt.givenFilepath)
			require.NoError(t, gotErr)
		})
	}
}

func TestLoadTemplate_RejectsSymlink(t *testing.T) {
	dir := t.TempDir()

	targetPath := filepath.Join(dir, "target.json")
	require.NoError(t, os.WriteFile(targetPath, []byte(`{"key": "value"}`), 0644))

	symlinkPath := filepath.Join(dir, "link.json")
	require.NoError(t, os.Symlink(targetPath, symlinkPath))

	testFs := afero.NewBasePathFs(afero.NewOsFs(), dir)

	_, err := template.NewFileTemplate(testFs, "link.json")
	assert.ErrorContains(t, err, "symbolic link")
}

func TestLoadTemplate_AllowsRegularFileOnOsFs(t *testing.T) {
	dir := t.TempDir()

	filePath := filepath.Join(dir, "regular.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{"key": "value"}`), 0644))

	testFs := afero.NewBasePathFs(afero.NewOsFs(), dir)

	tmpl, err := template.NewFileTemplate(testFs, "regular.json")
	require.NoError(t, err)

	content, err := tmpl.Content()
	require.NoError(t, err)
	assert.Equal(t, `{"key": "value"}`, content)
}

func TestLoadTemplate_WorksWithMemMapFs(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "template.json", []byte("content"), 0644)

	tmpl, err := template.NewFileTemplate(testFs, "template.json")
	require.NoError(t, err)

	content, err := tmpl.Content()
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}
