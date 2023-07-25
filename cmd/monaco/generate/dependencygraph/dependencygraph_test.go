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

package dependencygraph_test

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/generate/dependencygraph"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestInvalidCommandFlags(t *testing.T) {

	tests := []struct {
		name           string
		args           []string
		errMsgContains string
	}{
		{
			name:           "Fails loading non-existing default manifest",
			args:           []string{},
			errMsgContains: "manifest.yaml",
		},
		{
			name:           "Fails on unknown flag",
			args:           []string{"--project", "p"},
			errMsgContains: "unknown",
		},
		{
			name:           "Environment and group are mutually exclusive",
			args:           []string{"--environment", "e", "--group", "g"},
			errMsgContains: "flags in the group [environment group] are set none of the others can be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.MemMapFs{}

			cmd := dependencygraph.Command(&fs)

			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			assert.Error(t, err)
			assert.ErrorContains(t, err, tt.errMsgContains)
		})
	}
}

func TestGeneratesDOTFiles(t *testing.T) {

	t.Setenv("TOKEN", "some-value")

	fs := testutils.CreateTestFileSystem()

	outputFolder := "output-folder"

	cmd := dependencygraph.Command(fs)

	cmd.SetArgs([]string{
		"--manifest",
		"./test-resources/manifest.yaml",
		"-o",
		outputFolder,
	})
	err := cmd.Execute()
	assert.NoError(t, err)

	assertFileExists(t, fs, filepath.Join(outputFolder, "dependency_graph_env1.dot"))
	assertFileExists(t, fs, filepath.Join(outputFolder, "dependency_graph_env2.dot"))
}

func TestDoesNotOverwriteExistingFiles(t *testing.T) {

	t.Setenv("TOKEN", "some-value")

	// GIVEN pre-existing file overlapping with output name
	fs := testutils.CreateTestFileSystem()
	outputFolder := "output-folder"
	existingFile := "dependency_graph_env1.dot"

	absFolder, err := filepath.Abs(outputFolder)
	assert.NoError(t, err)
	err = fs.MkdirAll(absFolder, 0777)
	assert.NoError(t, err)

	existingPath := filepath.Join(outputFolder, existingFile)
	existingPath, err = filepath.Abs(existingPath)
	assert.NoError(t, err)

	err = afero.WriteFile(fs, existingPath, []byte{}, 0777)
	assert.NoError(t, err)

	// WHEN writing dependency graph
	cmd := dependencygraph.Command(fs)
	cmd.SetArgs([]string{
		"--manifest",
		"./test-resources/manifest.yaml",
		"-o",
		outputFolder,
		"--environment",
		"env1",
	})
	err = cmd.Execute()
	assert.NoError(t, err)

	// THEN existing file is untouched
	assertFileExists(t, fs, existingPath)
	existingContent, err := afero.ReadFile(fs, existingPath)
	assert.NoError(t, err)
	assert.Len(t, existingContent, 0, "expected pre-existing file to still be empty")

	// AND THEN new DOT file is created with timestamp appended
	time := timeutils.TimeAnchor().Format("20060102-150405")
	newFile := fmt.Sprintf("dependency_graph_env1_%s.dot", time)
	newPath := filepath.Join(outputFolder, newFile)
	newPath, err = filepath.Abs(newPath)

	assertFileExists(t, fs, newPath)
	newContent, err := afero.ReadFile(fs, newPath)
	assert.NoError(t, err)
	assert.Greater(t, len(newContent), 0, "expected pre-existing file to not be empty")
}

func assertFileExists(t *testing.T, fs afero.Fs, file string) {
	path, err := filepath.Abs(file)
	assert.NoError(t, err)

	exists, err := afero.Exists(fs, path)
	assert.NoError(t, err)
	assert.True(t, exists)
}
