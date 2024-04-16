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

package delete_test

import (
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestParseDeleteEntry(t *testing.T) {
	fileContent := []byte(`
delete:
- auto-tag/test entity
`)

	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	require.NoError(t, err)
	require.Len(t, actual, 1)
	require.Contains(t, actual, "auto-tag")
	require.Len(t, actual["auto-tag"], 1)
	require.Equal(t, "test entity", actual["auto-tag"][0].Identifier)
	require.Equal(t, "auto-tag", actual["auto-tag"][0].Type)
}

func TestParseSettingsDeleteEntry(t *testing.T) {
	fileContent := []byte(`
delete:
- builtin:tagging.auto/test entity
`)

	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	require.NoError(t, err)
	require.Len(t, actual, 1)
	require.Contains(t, actual, "builtin:tagging.auto")
	require.Len(t, actual["builtin:tagging.auto"], 1)
	require.Equal(t, "test entity", actual["builtin:tagging.auto"][0].Identifier)
	require.Equal(t, "builtin:tagging.auto", actual["builtin:tagging.auto"][0].Type)
}

func TestParseDeleteEntryWithMultipleSlashesShouldWork(t *testing.T) {
	fileContent := []byte(`
delete:
- auto-tag/test entity/entry
`)

	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	require.NoError(t, err)
	require.Len(t, actual, 1)
	require.Contains(t, actual, "auto-tag")
	require.Len(t, actual["auto-tag"], 1)
	require.Equal(t, "test entity/entry", actual["auto-tag"][0].Identifier)
	require.Equal(t, "auto-tag", actual["auto-tag"][0].Type)

}

func TestParseDeleteEntryInvalidEntryWithoutDelimiterShouldFail(t *testing.T) {
	fileContent := []byte(`
delete:
- auto-tag
`)

	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
	require.Error(t, err, "value `%s` should return error", "auto-tag")
	require.Empty(t, actual, "expected 0 results")

}

func TestParseDeleteFileDefinitions(t *testing.T) {
	fileContent := []byte(`
delete:
- auto-tag/test entity/entry
- management-zone/test entity/entry
`)

	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	require.NoError(t, err)
	require.Len(t, actual, 2)

	require.Contains(t, actual, "auto-tag")
	require.Len(t, actual["auto-tag"], 1)
	require.Equal(t, "test entity/entry", actual["auto-tag"][0].Identifier)
	require.Equal(t, "auto-tag", actual["auto-tag"][0].Type)

	require.Contains(t, actual, "management-zone")
	require.Len(t, actual["management-zone"], 1)
	require.Equal(t, "test entity/entry", actual["management-zone"][0].Identifier)
	require.Equal(t, "management-zone", actual["management-zone"][0].Type)
}

func TestParseDeleteFileDefinitionsWithInvalidDefinition(t *testing.T) {
	fileContent := []byte(`
delete:
- auto-tag/test entity/entry
- management-zone/test entity/entry
- invalid-definition
`)

	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	var e delete.ParseErrors
	require.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e), "expected 1 error")
	require.Empty(t, actual, "expected 0 results")
}

func TestLoadEntriesToDelete(t *testing.T) {

	tests := []struct {
		name             string
		givenFileContent string
		want             delete.DeleteEntries
	}{
		{
			"Loads simple file",
			`delete:
- management-zone/test entity/entities
- auto-tag/random tag
`,
			delete.DeleteEntries{
				"auto-tag": {
					{
						Type:       "auto-tag",
						Identifier: "random tag",
					},
				},
				"management-zone": {
					{
						Type:       "management-zone",
						Identifier: "test entity/entities",
					},
				},
			},
		},
		{
			"Loads Settings",
			`delete:
- management-zone/test entity/entities
- builtin:auto.tagging/random tag
`,
			delete.DeleteEntries{
				"builtin:auto.tagging": {
					{
						Type:       "builtin:auto.tagging",
						Identifier: "random tag",
					},
				},
				"management-zone": {
					{
						Type:       "management-zone",
						Identifier: "test entity/entities",
					},
				},
			},
		},
		{
			"Loads Full Format",
			`delete:
- project: "myProject"
  type: management-zone
  name: test entity/entities
- project: some-project
  type: builtin:auto.tagging
  id: my-tag
`,
			delete.DeleteEntries{
				"builtin:auto.tagging": {
					{
						Project:    "some-project",
						Type:       "builtin:auto.tagging",
						Identifier: "my-tag",
					},
				},
				"management-zone": {
					{
						Type:       "management-zone",
						Identifier: "test entity/entities",
					},
				},
			},
		},
		{
			"Loads Mixed Format",
			`delete:
- "management-zone/test entity/entities"
- project: some-project
  type: builtin:auto.tagging
  id: my-tag
`,
			delete.DeleteEntries{
				"builtin:auto.tagging": {
					{
						Project:    "some-project",
						Type:       "builtin:auto.tagging",
						Identifier: "my-tag",
					},
				},
				"management-zone": {
					{
						Type:       "management-zone",
						Identifier: "test entity/entities",
					},
				},
			},
		},
		{
			"Loads Subpath Entries",
			`delete:
- project: some-project
  type: key-user-actions-mobile
  scope: APPLICATION-MOBILE-1234
  name: my-action
`,
			delete.DeleteEntries{
				"key-user-actions-mobile": {
					{
						Type:       "key-user-actions-mobile",
						Scope:      "APPLICATION-MOBILE-1234",
						Identifier: "my-action",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := delete.LoadEntriesFromFile(createDeleteFile(t, []byte(tt.givenFileContent)))

			assert.NoError(t, err)
			assert.Equal(t, len(tt.want), len(result))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestLoadEntriesToDeleteFailsIfScopeIsUndefinedForSubPathAPI(t *testing.T) {
	fileContent := []byte(`delete:
- project: some-project
  type: key-user-actions-mobile
  name: my-action
`) // scope should be defined

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e), "expected 1 error")
	assert.Empty(t, result, "expected 0 results")
}

func TestLoadEntriesToDeleteFailsIfScopeIsDefinedForNonSubPathAPI(t *testing.T) {
	fileContent := []byte(`delete:
- project: some-project
  type: alerting-profile
  name: my-action
  scope: my-scope # scope should NOT be defined
`)

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e), "expected 1 error")
	assert.Empty(t, result, "expected 0 results")
}

func TestLoadEntriesToDeleteWithInvalidEntry(t *testing.T) {
	fileContent := []byte(`delete:
- management-zone/test entity/entities
- auto-invalid
`)

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e), "expected 1 error")
	assert.Empty(t, result, "expected 0 results")
}

func TestLoadEntriesToDeleteWithMultipleInvalidEntries(t *testing.T) {
	fileContent := []byte(`
delete:
- management-zone/test entity/entities
- auto-invalid
- type: unknown-api
  name: test
- type: alerting-profile
- type: alerting-profile
  name: my-name-2
  scope: no-scope-allowed
- type: key-user-actions-mobile
  name: test
  scope: ''
`)

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 5, len(e), "expected 5 errors")
	assert.Empty(t, result, "expected 0 results")
}

func TestLoadEntriesToDeleteNonExistingFile(t *testing.T) {
	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, nil))

	assert.Error(t, err)
	assert.Empty(t, result, "expected 0 results")
}

func TestLoadEntriesToDeleteWithMalformedFile(t *testing.T) {
	fileContent := []byte(`deleting:
- auto-invalid
`)

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))

	var typeError *yaml.TypeError
	assert.ErrorAs(t, err, &typeError)
	assert.Empty(t, result, "expected 0 results")
}

func TestLoadEntriesToDeleteWithEmptyFile(t *testing.T) {
	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, []byte("")))

	assert.ErrorContains(t, err, "is empty")
	assert.Empty(t, result, "expected 0 results")
}

func createDeleteFile(t testing.TB, content []byte) (afero.Fs, string) {
	t.Helper()

	workingDir := t.TempDir()
	deleteFileName := "delete.yaml"
	deleteFilePath := filepath.Join(workingDir, deleteFileName)
	fs := afero.NewMemMapFs()

	err := fs.MkdirAll(workingDir, 0777)
	assert.NoError(t, err)

	if content != nil {
		err = afero.WriteFile(fs, deleteFilePath, content, 0666)
		assert.NoError(t, err)
	}

	return fs, deleteFilePath
}
