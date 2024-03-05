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
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
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
			name:           "Manifest argument is required",
			args:           []string{},
			errMsgContains: "accepts 1 arg(s), received 0",
		},
		{
			name:           "Fails on unknown flag",
			args:           []string{"manifest.yaml", "--project", "p"},
			errMsgContains: "unknown",
		},
		{
			name:           "Environment and group are mutually exclusive",
			args:           []string{"manifest.yaml", "--environment", "e", "--group", "g"},
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

	expectedGraph := map[string][]string{
		"project:reports:report":                                                 {},
		"project:alerting-profile:profile":                                       {"project:notification:slack", "project:notification:email", "project:notification:email_single_receiver", "project:notification:email_list_as_array"},
		"project:dashboard:dashboard":                                            {"project:reports:report"},
		"project:dashboard:dashboard-with-settings-reference":                    {},
		"project:maintenance-window:maintenancewindow":                           {},
		"project:builtin:alerting.maintenance-window:maintenance-window-setting": {},
		"project:management-zone:zone":                                           {"project:dashboard:dashboard", "project:maintenance-window:maintenancewindow"},
		"project:builtin:management-zones:management-zone-setting":               {"project:dashboard:dashboard-with-settings-reference", "project:builtin:alerting.maintenance-window:maintenance-window-setting"},
		"project:notification:slack":                                             {},
		"project:notification:email":                                             {},
		"project:notification:email_single_receiver":                             {},
		"project:notification:email_list_as_array":                               {},
	}

	fs := testutils.CreateTestFileSystem()

	outputFolder := "output-folder"

	cmd := dependencygraph.Command(fs)

	cmd.SetArgs([]string{
		"./test-resources/manifest.yaml",
		"-o",
		outputFolder,
	})
	err := cmd.Execute()
	assert.NoError(t, err)

	f1 := filepath.Join(outputFolder, "dependency_graph_env1.dot")
	assertFileExists(t, fs, f1)
	assertCreatedDOTGraph(t, fs, f1, expectedGraph)

	f2 := filepath.Join(outputFolder, "dependency_graph_env2.dot")
	assertFileExists(t, fs, f2)
	assertCreatedDOTGraph(t, fs, f2, expectedGraph)
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
	require.NoError(t, err)

	exists, err := afero.Exists(fs, path)
	require.NoError(t, err)
	require.True(t, exists)
}

func assertCreatedDOTGraph(t *testing.T, fs afero.Fs, file string, expectedGraph map[string][]string) {
	path, err := filepath.Abs(file)
	require.NoError(t, err)

	content, err := afero.ReadFile(fs, path)
	require.NoError(t, err)

	// the loaded graph does not contain node names, so we only unmarshal this to check it's a valid/loadable DOT representation and ignore the content
	err = dot.Unmarshal(content, simple.NewDirectedGraph())
	assert.NoError(t, err)

	// then check the string directly to ensure it contains the nodes and edges we expect
	cS := string(content)
	for node, edges := range expectedGraph {
		assert.Contains(t, cS, fmt.Sprintf("%q;", node))
		for _, e := range edges {
			assert.Contains(t, cS, fmt.Sprintf("%q -> %q;", node, e))
		}
	}
}
