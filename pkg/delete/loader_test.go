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
	"go.yaml.in/yaml/v2"
)

// This test should contain all possible entry types (except the legacy one)
func TestMixOfClassicAndSettingsEntry(t *testing.T) {
	fileContent := []byte(`delete:
- type: management-zone
  name: test entity/entities
- project: some-project
  type: builtin:auto.tagging
  id: my-tag
`)
	want := delete.DeleteEntries{
		"builtin:auto.tagging": {{
			Project:    "some-project",
			Type:       "builtin:auto.tagging",
			Identifier: "my-tag",
		}},
		"management-zone": {{
			Type:       "management-zone",
			Identifier: "test entity/entities",
		}}}
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
	require.NoError(t, err)
	require.Equal(t, want, actual)
}

func TestClassicEntry(t *testing.T) {
	fileContent := []byte(`delete:
- type: management-zone
  name: test entity/entities
`)
	want := delete.DeleteEntries{
		"management-zone": {{
			Type:       "management-zone",
			Identifier: "test entity/entities",
		}}}
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
	require.NoError(t, err)
	require.Equal(t, want, actual)
}

func TestClassicEntryWithOriginID(t *testing.T) {
	fileContent := []byte(`delete:
- type: management-zone
  objectId: origin-object-ID
`)
	want := delete.DeleteEntries{
		"management-zone": {{
			Type:           "management-zone",
			OriginObjectId: "origin-object-ID",
		}}}
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
	require.NoError(t, err)
	require.Equal(t, want, actual)
}

func TestClassicEntryFailsIfNameAndOriginIdCoexists(t *testing.T) {
	fileContent := []byte(`delete:
- type: management-zone
  name: config-name
  objectId: origin-object-ID
`)
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e))
	assert.Empty(t, actual)
}

func TestClassicEntryFailsWithoutNameOrOriginId(t *testing.T) {
	fileContent := []byte(`delete:
- type: management-zone
`)
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e))
	assert.Empty(t, actual)
}

func TestClassicKUAMobileEntry(t *testing.T) {
	given := []byte(`delete:
- type: key-user-actions-mobile
  name: my-action
  scope: parent-name
`)
	want := delete.DeleteEntries{
		"key-user-actions-mobile": {{
			Type:       "key-user-actions-mobile",
			Scope:      "parent-name",
			Identifier: "my-action",
		}}}
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
	require.NoError(t, err)
	require.Equal(t, want, actual)
}

func TestClassicKUAMobileEntryFailsIfScopeIsUndefined(t *testing.T) {
	given := []byte(`delete:
- type: key-user-actions-mobile
  name: my-action
`) // scope should be defined

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))

	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e), "expected 1 error")
	assert.Empty(t, result, "expected 0 results")
}

func TestClassicEntries(t *testing.T) {
	given := []byte(`delete:
- type: alerting-profile
  name: my-action
  scope: my-scope # scope should NOT be defined
`)

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))

	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e), "expected 1 error")
	assert.Empty(t, result, "expected 0 results")
}

func TestSettingsEntry(t *testing.T) {
	given := []byte(`delete:
- project: some-project
  type: builtin:auto.tagging
  id: my-tag
`)
	want := delete.DeleteEntries{
		"builtin:auto.tagging": {{
			Project:    "some-project",
			Type:       "builtin:auto.tagging",
			Identifier: "my-tag",
		}}}
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
	require.NoError(t, err)
	require.Equal(t, want, actual)
}

func TestSettingsEntryWithOriginID(t *testing.T) {
	fileContent := []byte(`delete:
- type: builtin:auto.tagging
  objectId: origin-object-ID
`)
	want := delete.DeleteEntries{
		"builtin:auto.tagging": {{
			Type:           "builtin:auto.tagging",
			OriginObjectId: "origin-object-ID",
		}}}
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
	require.NoError(t, err)
	require.Equal(t, want, actual)
}

func TestSettingsEntryFailsIfIdAndOriginIdCoexists(t *testing.T) {
	given := []byte(`delete:
- type: builtin:auto.tagging
  id: my-tag
  objectId: origin-object-ID
`)
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e))
	assert.Empty(t, actual)
}
func TestSettingsEntryFailsIfProjectAndOriginIdCoexists(t *testing.T) {
	given := []byte(`delete:
- type: builtin:auto.tagging
  project: some-project
  objectId: origin-object-ID
`)
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e))
	assert.Empty(t, actual)
}

func TestSettingsEntryFailsIfNotValid(t *testing.T) {
	given := []byte(`delete:
- type: builtin:auto.tagging
`)
	actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 1, len(e))
	assert.Empty(t, actual)
}

func TestLegacy(t *testing.T) {
	t.Run("all legacy entry types", func(t *testing.T) {
		given := []byte(`delete:
- management-zone/test entity/entities
- builtin:auto.tagging/random tag
`)
		want := delete.DeleteEntries{
			"builtin:auto.tagging": {{
				Type:       "builtin:auto.tagging",
				Identifier: "random tag",
			}},
			"management-zone": {{
				Type:       "management-zone",
				Identifier: "test entity/entities",
			}}}

		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
		require.NoError(t, err)
		require.Equal(t, want, actual)
	})

	t.Run("legacy classic entry", func(t *testing.T) {
		given := []byte(`
delete:
- auto-tag/test entity
`)
		want := delete.DeleteEntries{
			"auto-tag": {
				{
					Type:       "auto-tag",
					Identifier: "test entity",
				}}}

		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
		require.NoError(t, err)
		require.Equal(t, want, actual)
	})

	t.Run("legacy classic entry with multiple slashes", func(t *testing.T) {
		given := []byte(`
delete:
- auto-tag/test entity/entry
- management-zone/test entity/entry
`)
		want := delete.DeleteEntries{
			"auto-tag": {{
				Type:       "auto-tag",
				Identifier: "test entity/entry",
			}},
			"management-zone": {{
				Type:       "management-zone",
				Identifier: "test entity/entry",
			}}}

		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
		require.NoError(t, err)
		require.Equal(t, want, actual)
	})

	t.Run("legacy settings entry", func(t *testing.T) {
		given := []byte(`
delete:
- builtin:tagging.auto/test entity
`)
		want := delete.DeleteEntries{
			"builtin:tagging.auto": {{
				Type:       "builtin:tagging.auto",
				Identifier: "test entity",
			}}}

		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
		require.NoError(t, err)
		require.Equal(t, want, actual)
	})

	t.Run("legacy entry with invalid definition", func(t *testing.T) {
		given := []byte(`
delete:
- auto-tag/test entity/entry
- management-zone/test entity/entry
- invalid-definition
`)

		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))

		var e delete.ParseErrors
		require.ErrorAs(t, err, &e)
		assert.Equal(t, 1, len(e), "expected 1 error")
		require.Empty(t, actual, "expected 0 results")
	})

	t.Run("legacy entry without delimiter (slash) should fail", func(t *testing.T) {
		given := []byte(`
delete:
- auto-tag
`)
		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
		require.Error(t, err, "value `%s` should return error", "auto-tag")
		require.Empty(t, actual, "expected 0 results")
	})

	t.Run("mix of legacy and new format", func(t *testing.T) {
		given := []byte(`delete:
- "management-zone/legacy entity/entities"
- type: management-zone
  name: actual_entry_definition
`)
		want := delete.DeleteEntries{
			"management-zone": {
				{
					Type:       "management-zone",
					Identifier: "legacy entity/entities",
				},
				{
					Type:       "management-zone",
					Identifier: "actual_entry_definition",
				},
			},
		}

		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))
		require.NoError(t, err)
		require.Equal(t, want, actual)
	})
}

func TestLoadMultipleInvalidEntries(t *testing.T) {
	given := []byte(`
delete:
- type: invalid-api_1
  name: test
- type: alerting-profile
- type: alerting-profile
  name: my-name-2
  scope: no-scope-allowed
`)

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))

	var e delete.ParseErrors
	assert.ErrorAs(t, err, &e)
	assert.Equal(t, 3, len(e))
	assert.Empty(t, result)
}

func TestLoadMalformedFile(t *testing.T) {
	given := []byte(`wrong:
- auto-invalid
`)

	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, given))

	var typeError *yaml.TypeError
	assert.ErrorAs(t, err, &typeError)
	assert.Empty(t, result)
}

func TestLoadNonExistingFile(t *testing.T) {
	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, nil))

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestEmptyFileFails(t *testing.T) {
	result, err := delete.LoadEntriesFromFile(createDeleteFile(t, []byte("")))

	assert.ErrorContains(t, err, "is empty")
	assert.Empty(t, result)
}

func TestLoad_DocumentsEntry(t *testing.T) {
	t.Run("identify via project-id pair'", func(t *testing.T) {
		fileContent := []byte(`delete:
- type: document
  project: project
  id: monaco-config-id
`)
		want := delete.DeleteEntries{
			"document": {{
				Type:       "document",
				Project:    "project",
				Identifier: "monaco-config-id",
			}}}
		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
		require.NoError(t, err)
		require.Equal(t, want, actual)
	})

	t.Run("identify via 'objectId'", func(t *testing.T) {
		fileContent := []byte(`delete:
- type: document
  objectId: origin-object-ID
`)
		want := delete.DeleteEntries{
			"document": {{
				Type:           "document",
				OriginObjectId: "origin-object-ID",
			}}}
		actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, fileContent))
		require.NoError(t, err)
		require.Equal(t, want, actual)
	})
}

func TestLoad_Automation(t *testing.T) {
	tests := []testCase{
		{
			name: "declare via coordinate",
			given: []byte(`delete:
- type:    workflow
  project: my-project
  id:      my-workflow
- type:    scheduling-rule
  project: my-project
  id:      my-rule
- type:    business-calendar
  project: my-project
  id:      my-calendar
`),
			want: delete.DeleteEntries{
				"workflow": {{
					Project:    "my-project",
					Type:       "workflow",
					Identifier: "my-workflow",
				}},
				"scheduling-rule": {{
					Project:    "my-project",
					Type:       "scheduling-rule",
					Identifier: "my-rule",
				}},
				"business-calendar": {{
					Project:    "my-project",
					Type:       "business-calendar",
					Identifier: "my-calendar",
				}},
			},
		},
		{
			name: "declare via originId",
			given: []byte(`delete:
- type:     workflow
  objectId: workflow-id
- type:     scheduling-rule
  objectId: rule-id
- type:     business-calendar
  objectId: calendar-id
`),
			want: delete.DeleteEntries{
				"workflow": {{
					OriginObjectId: "workflow-id",
					Type:           "workflow",
				}},
				"scheduling-rule": {{
					OriginObjectId: "rule-id",
					Type:           "scheduling-rule",
				}},
				"business-calendar": {{
					OriginObjectId: "calendar-id",
					Type:           "business-calendar",
				}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := delete.LoadEntriesFromFile(createDeleteFile(t, tc.given))
			if tc.want != nil {
				require.NoError(t, err)
				require.Equal(t, tc.want, actual)
			} else {
				require.Error(t, err)
				assert.Empty(t, actual)
			}
		})
	}
}

type testCase struct {
	name  string
	given []byte
	want  delete.DeleteEntries
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
