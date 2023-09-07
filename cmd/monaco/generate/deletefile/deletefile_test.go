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

package deletefile_test

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/generate/deletefile"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestInvalidCommandUsage(t *testing.T) {

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
			args:           []string{"manifest.yaml", "--environment", "e"},
			errMsgContains: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.MemMapFs{}

			cmd := deletefile.Command(&fs)

			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			assert.Error(t, err)
			assert.ErrorContains(t, err, tt.errMsgContains)
		})
	}
}

func TestGeneratesValidDeleteFile(t *testing.T) {

	t.Setenv("TOKEN", "some-value")

	fs := testutils.CreateTestFileSystem()

	outputFolder := "output-folder"

	cmd := deletefile.Command(fs)

	cmd.SetArgs([]string{
		"./test-resources/manifest.yaml",
		"-o",
		outputFolder,
	})
	err := cmd.Execute()
	assert.NoError(t, err)

	expectedFile := filepath.Join(outputFolder, "delete.yaml")
	assertFileExists(t, fs, expectedFile)

	entries, errs := delete.LoadEntriesToDelete(fs, expectedFile)
	assert.Len(t, errs, 0)

	assertDeleteEntries(t, entries, "alerting-profile", "Star Trek Service", "Star Wars Service", "Star Gate Service", "Lord of the Rings Service", "A Song of Ice and Fire Service")
	assertDeleteEntries(t, entries, "dashboard", "Alpha Quadrant")
	assertDeleteEntries(t, entries, "builtin:alerting.maintenance-window", "maintenance-window-setting")
	assertDeleteEntries(t, entries, "management-zone", "mzone-1")
	assertDeleteEntries(t, entries, "builtin:management-zones", "management-zone-setting")
	assertDeleteEntries(t, entries, "notification", "Star Trek to #team-star-trek", "Captain's Log")
}

func TestGeneratesValidDeleteFile_ForSingleProject(t *testing.T) {

	t.Setenv("TOKEN", "some-value")

	fs := testutils.CreateTestFileSystem()

	outputFolder := "output-folder"

	cmd := deletefile.Command(fs)

	cmd.SetArgs([]string{
		"./test-resources/manifest.yaml",
		"--project",
		"other-project",
		"-o",
		outputFolder,
	})
	err := cmd.Execute()
	assert.NoError(t, err)

	expectedFile := filepath.Join(outputFolder, "delete.yaml")
	assertFileExists(t, fs, expectedFile)

	entries, errs := delete.LoadEntriesToDelete(fs, expectedFile)
	assert.Len(t, errs, 0)

	assertDeleteEntries(t, entries, "alerting-profile", "Lord of the Rings Service", "A Song of Ice and Fire Service")
}

func TestGeneratesValidDeleteFile_OmittingClassicConfigsWithNonStringNames(t *testing.T) {

	t.Setenv("TOKEN", "some-value")

	fs := testutils.CreateTestFileSystem()

	outputFolder := "output-folder"

	cmd := deletefile.Command(fs)

	cmd.SetArgs([]string{
		"./test-resources/manifest_invalid_project.yaml",
		"-o",
		outputFolder,
	})
	err := cmd.Execute()
	assert.NoError(t, err)

	expectedFile := filepath.Join(outputFolder, "delete.yaml")
	assertFileExists(t, fs, expectedFile)

	entries, errs := delete.LoadEntriesToDelete(fs, expectedFile)
	assert.Len(t, errs, 0)

	assertDeleteEntries(t, entries, "alerting-profile", "Star Trek Service", "Star Wars Service", "Star Gate Service", "Lord of the Rings Service", "A Song of Ice and Fire Service")
	assertDeleteEntries(t, entries, "dashboard", "Alpha Quadrant")
	assertDeleteEntries(t, entries, "builtin:alerting.maintenance-window", "maintenance-window-setting")
	assertDeleteEntries(t, entries, "management-zone", "mzone-1")
	assertDeleteEntries(t, entries, "builtin:management-zones", "management-zone-setting")
	assertDeleteEntries(t, entries, "notification", "Star Trek to #team-star-trek", "Captain's Log")
}

func assertDeleteEntries(t *testing.T, entries map[string][]delete.DeletePointer, cfgType string, expectedCfgIdentifiers ...string) {
	vals, ok := entries[cfgType]
	assert.True(t, ok, "expected delete pointers for type %s", cfgType)

	assert.Len(t, vals, len(expectedCfgIdentifiers), "expected length of values to match that of expected cfg names")
	deleted := make([]string, len(vals))
	for i, v := range vals {
		deleted[i] = v.Identifier
	}
	assert.ElementsMatch(t, deleted, expectedCfgIdentifiers)
}

func TestDoesNotOverwriteExistingFiles(t *testing.T) {

	t.Setenv("TOKEN", "some-value")

	// GIVEN pre-existing file overlapping with output name
	fs := testutils.CreateTestFileSystem()
	outputFolder := "output-folder"
	existingFile := "delete.yaml"

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
	cmd := deletefile.Command(fs)
	cmd.SetArgs([]string{
		"./test-resources/manifest.yaml",
		"-o",
		outputFolder,
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
	newFile := fmt.Sprintf("delete.yaml_%s", time)
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
