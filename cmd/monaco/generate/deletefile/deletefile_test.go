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
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/generate/deletefile"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
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
			args:           []string{"manifest.yaml", "--specific-api", "auto-tag"},
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
	t.Setenv(featureflags.OpenPipeline.EnvName(), "1")
	t.Setenv(featureflags.Segments.EnvName(), "1")

	fs := testutils.CreateTestFileSystem()
	outputFolder := "output-folder"
	err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco generate deletefile ./test-resources/manifest.yaml --output-folder=%s", outputFolder))
	assert.NoError(t, err)

	expectedFile := filepath.Join(outputFolder, "delete.yaml")
	assertFileExists(t, fs, expectedFile)

	entries, errs := delete.LoadEntriesFromFile(fs, expectedFile)
	assert.NoError(t, errs)

	assertDeleteEntries(t, entries, "alerting-profile", "Star Trek Service", "Star Wars Service", "Star Gate Service", "Lord of the Rings Service", "A Song of Ice and Fire Service")
	assertDeleteEntries(t, entries, "dashboard", "Alpha Quadrant")
	assertDeleteEntries(t, entries, "builtin:alerting.maintenance-window", "maintenance-window-setting")
	assertDeleteEntries(t, entries, "management-zone", "mzone-1")
	assertDeleteEntries(t, entries, "builtin:management-zones", "management-zone-setting")
	assertDeleteEntries(t, entries, "notification", "Star Trek to #team-star-trek", "envOverride: Star Wars to #team-star-wars", "Captain's Log")
	assertDeleteEntries(t, entries, "application-mobile", "app-1", "app-2")
	assertDeleteEntries(t, entries, "application-web", "My first Web application")
	assertDeleteEntries(t, entries, "key-user-actions-web", "first-kua:My first Web application")
	assertDeleteEntries(t, entries, "user-action-and-session-properties-mobile", "property1:app-1", "property2:app-1", "property1:app-2")
	assertDeleteEntries(t, entries, "business-calendar", "ca-business-calendar")
	assertDeleteEntries(t, entries, "scheduling-rule", "ca-scheduling-rule")
	assertDeleteEntries(t, entries, "workflow", "ca-jira-issue-workflow")
	assertDeleteEntries(t, entries, "bucket", "my-bucket")
	assertDeleteEntries(t, entries, "document", "my-dashboard", "my-notebook")
	assertDeleteEntries(t, entries, "segment", "segmentID")

	assert.Empty(t, entries[api.DashboardShareSettings])
	assert.Empty(t, entries[string(config.OpenPipelineTypeID)])
}

func TestGeneratesValidDeleteFileWithCustomValues(t *testing.T) {
	t.Setenv("TOKEN", "some-value")
	t.Setenv(featureflags.OpenPipeline.EnvName(), "1")
	t.Setenv(featureflags.Segments.EnvName(), "1")

	fs := testutils.CreateTestFileSystem()
	outputFolder := "output-folder"
	err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco generate deletefile ./test-resources/manifest.yaml  --output-folder=%s", outputFolder))
	assert.NoError(t, err)
	require.NoError(t, err)

	expectedFile := filepath.Join(outputFolder, "delete.yaml")
	assertFileExists(t, fs, expectedFile)

	deleteFileContent := readFile(t, fs, expectedFile)

	var deleteEntries persistence.FullFileDefinition
	err = yaml.Unmarshal(deleteFileContent, &deleteEntries)
	require.NoError(t, err)

	assert.Contains(t, deleteEntries.DeleteEntries, persistence.DeleteEntry{
		Type:         "key-user-actions-web",
		ConfigName:   "first-kua",
		Scope:        "My first Web application",
		CustomValues: map[string]string{"actionType": "Load", "domain": "domain.com"},
	})
}

func TestGeneratesValidDeleteFileWithFilter(t *testing.T) {

	t.Setenv("TOKEN", "some-value")
	t.Setenv(featureflags.OpenPipeline.EnvName(), "1")
	t.Setenv(featureflags.Segments.EnvName(), "1")

	fs := testutils.CreateTestFileSystem()
	outputFolder := "output-folder"
	err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco generate deletefile ./test-resources/manifest.yaml --output-folder=%s --types=builtin:management-zones,notification --exclude-types=notification", outputFolder))
	assert.NoError(t, err)

	expectedFile := filepath.Join(outputFolder, "delete.yaml")
	assertFileExists(t, fs, expectedFile)

	entries, errs := delete.LoadEntriesFromFile(fs, expectedFile)
	assert.NoError(t, errs)

	assertDeleteEntries(t, entries, "builtin:management-zones", "management-zone-setting")
	assert.NotContains(t, entries, "notification")

}

func TestGeneratesValidDeleteFile_ForSpecificEnv(t *testing.T) {

	t.Setenv("TOKEN", "some-value")
	t.Setenv(featureflags.OpenPipeline.EnvName(), "1")
	t.Setenv(featureflags.Segments.EnvName(), "1")

	outputFolder := "output-folder"

	t.Run("env1 includes base notification name", func(t *testing.T) {
		fs := testutils.CreateTestFileSystem()
		err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco generate deletefile ./test-resources/manifest.yaml --environment=env1 --output-folder=%s", outputFolder))
		assert.NoError(t, err)

		expectedFile := filepath.Join(outputFolder, "delete.yaml")
		assertFileExists(t, fs, expectedFile)

		entries, errs := delete.LoadEntriesFromFile(fs, expectedFile)
		assert.NoError(t, errs)

		assertDeleteEntries(t, entries, "notification", "Star Trek to #team-star-trek", "Captain's Log")
	})

	t.Run("env2 includes over-written notification name", func(t *testing.T) {
		fs := testutils.CreateTestFileSystem()
		err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco generate deletefile ./test-resources/manifest.yaml --environment=env2 --output-folder=%s", outputFolder))
		assert.NoError(t, err)

		expectedFile := filepath.Join(outputFolder, "delete.yaml")
		assertFileExists(t, fs, expectedFile)

		entries, errs := delete.LoadEntriesFromFile(fs, expectedFile)
		assert.NoError(t, errs)

		assertDeleteEntries(t, entries, "notification", "envOverride: Star Wars to #team-star-wars", "Captain's Log")
	})

	t.Run("no specific env includes both notification names", func(t *testing.T) {
		fs := testutils.CreateTestFileSystem()
		err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco generate deletefile ./test-resources/manifest.yaml --output-folder=%s", outputFolder))
		assert.NoError(t, err)

		expectedFile := filepath.Join(outputFolder, "delete.yaml")
		assertFileExists(t, fs, expectedFile)

		entries, errs := delete.LoadEntriesFromFile(fs, expectedFile)
		assert.NoError(t, errs)

		assertDeleteEntries(t, entries, "notification", "Star Trek to #team-star-trek", "envOverride: Star Wars to #team-star-wars", "Captain's Log")
	})

}

func TestGeneratesValidDeleteFile_ForSingleProject(t *testing.T) {

	t.Setenv("TOKEN", "some-value")

	fs := testutils.CreateTestFileSystem()
	outputFolder := "output-folder"
	err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco generate deletefile ./test-resources/manifest.yaml --project=other-project --output-folder=%s", outputFolder))
	assert.NoError(t, err)

	expectedFile := filepath.Join(outputFolder, "delete.yaml")
	assertFileExists(t, fs, expectedFile)

	entries, errs := delete.LoadEntriesFromFile(fs, expectedFile)
	assert.NoError(t, errs)

	assertDeleteEntries(t, entries, "alerting-profile", "Lord of the Rings Service", "A Song of Ice and Fire Service")
}

func TestGeneratesValidDeleteFile_OmittingClassicConfigsWithNonStringNames(t *testing.T) {

	t.Setenv("TOKEN", "some-value")
	t.Setenv(featureflags.OpenPipeline.EnvName(), "1")
	t.Setenv(featureflags.Segments.EnvName(), "1")

	fs := testutils.CreateTestFileSystem()
	outputFolder := "output-folder"
	err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco generate deletefile ./test-resources/manifest_invalid_project.yaml --output-folder=%s", outputFolder))
	assert.NoError(t, err)

	expectedFile := filepath.Join(outputFolder, "delete.yaml")
	assertFileExists(t, fs, expectedFile)

	entries, errs := delete.LoadEntriesFromFile(fs, expectedFile)
	assert.NoError(t, errs)

	assertDeleteEntries(t, entries, "alerting-profile", "Star Trek Service", "Star Wars Service", "Star Gate Service", "Lord of the Rings Service", "A Song of Ice and Fire Service")
	assertDeleteEntries(t, entries, "dashboard", "Alpha Quadrant")
	assertDeleteEntries(t, entries, "builtin:alerting.maintenance-window", "maintenance-window-setting")
	assertDeleteEntries(t, entries, "management-zone", "mzone-1")
	assertDeleteEntries(t, entries, "builtin:management-zones", "management-zone-setting")
	assertDeleteEntries(t, entries, "notification", "Star Trek to #team-star-trek", "envOverride: Star Wars to #team-star-wars", "Captain's Log")
}

func assertDeleteEntries(t *testing.T, entries map[string][]pointer.DeletePointer, cfgType string, expectedCfgIdentifiers ...string) {
	vals, ok := entries[cfgType]
	assert.True(t, ok, "expected delete pointers for type %s", cfgType)

	assert.Len(t, vals, len(expectedCfgIdentifiers), "expected length of values to match that of expected cfg names")
	deleted := make([]string, len(vals))
	for i, v := range vals {
		deleted[i] = v.Identifier
		if v.Scope != "" {
			deleted[i] = deleted[i] + ":" + v.Scope
		}

	}
	assert.ElementsMatch(t, deleted, expectedCfgIdentifiers)
}

func TestDoesNotOverwriteExistingFiles(t *testing.T) {

	t.Setenv("TOKEN", "some-value")
	t.Setenv(featureflags.OpenPipeline.EnvName(), "1")
	t.Setenv(featureflags.Segments.EnvName(), "1")

	t.Run("default filename", func(t *testing.T) {
		time := timeutils.TimeAnchor().Format("20060102-150405")
		newFile := fmt.Sprintf("delete_%s.yaml", time)
		testPreexistingFileIsNotOverwritten(t, "delete.yaml", newFile, false)
	})

	t.Run("custom filename", func(t *testing.T) {
		time := timeutils.TimeAnchor().Format("20060102-150405")
		newFile := fmt.Sprintf("my-special-delete_file_%s.yaml", time)
		testPreexistingFileIsNotOverwritten(t, "my-special-delete_file.yaml", newFile, true)
	})

	t.Run("custom filename with dots", func(t *testing.T) {
		time := timeutils.TimeAnchor().Format("20060102-150405")
		newFile := fmt.Sprintf("my.special.delete.file_%s.yaml", time)
		testPreexistingFileIsNotOverwritten(t, "my.special.delete.file.yaml", newFile, true)
	})

	t.Run("custom filename with no file-ending", func(t *testing.T) {
		time := timeutils.TimeAnchor().Format("20060102-150405")
		newFile := fmt.Sprintf("my-special-delete_file_%s", time)
		testPreexistingFileIsNotOverwritten(t, "my-special-delete_file", newFile, true)
	})

}

func testPreexistingFileIsNotOverwritten(t *testing.T, existingFile string, expectedNewFile string, customFileName bool) {
	t.Helper()

	// GIVEN pre-existing file overlapping with output name
	fs := testutils.CreateTestFileSystem()
	outputFolder := "output-folder"

	absFolder, err := filepath.Abs(outputFolder)
	assert.NoError(t, err)
	err = fs.MkdirAll(absFolder, 0777)
	assert.NoError(t, err)

	existingPath := filepath.Join(outputFolder, existingFile)
	existingPath, err = filepath.Abs(existingPath)
	assert.NoError(t, err)

	err = afero.WriteFile(fs, existingPath, []byte{}, 0777)
	require.NoError(t, err)

	cmd := fmt.Sprintf("monaco generate deletefile ./test-resources/manifest.yaml --output-folder=%s", outputFolder)
	if customFileName {
		cmd = cmd + fmt.Sprintf(" --file=%s", existingFile)
	}
	err = monaco.RunWithFs(t, fs, cmd)
	require.NoError(t, err)

	// THEN existing file is untouched
	assertFileExists(t, fs, existingPath)
	existingContent, err := afero.ReadFile(fs, existingPath)
	assert.NoError(t, err)
	assert.Len(t, existingContent, 0, "expected pre-existing file to still be empty")

	// AND THEN new delete file is created with timestamp appended
	newPath := filepath.Join(outputFolder, expectedNewFile)
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

func readFile(t *testing.T, fs afero.Fs, file string) []byte {
	path, err := filepath.Abs(file)
	require.NoError(t, err)
	content, err := afero.ReadFile(fs, path)
	require.NoError(t, err)
	return content
}
