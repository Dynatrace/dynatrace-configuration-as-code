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

package delete

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/assert"
)

func TestParseDeleteEntry(t *testing.T) {
	api := "auto-tag"
	name := "test entity"

	entry, err := parseDeleteEntry(0, api+deleteDelimiter+name)

	assert.NilError(t, err)
	assert.Equal(t, api, entry.Type)
	assert.Equal(t, name, entry.ConfigId)
}

func TestParseSettingsDeleteEntry(t *testing.T) {
	cfgType := "builtin:tagging.auto"
	name := "test entity"

	entry, err := parseDeleteEntry(0, cfgType+deleteDelimiter+name)

	assert.NilError(t, err)
	assert.Equal(t, cfgType, entry.Type)
	assert.Equal(t, name, entry.ConfigId)
}

func TestParseDeleteEntryWithMultipleSlashesShouldWork(t *testing.T) {
	api := "auto-tag"
	name := "test entity/entry"

	entry, err := parseDeleteEntry(0, api+deleteDelimiter+name)

	assert.NilError(t, err)
	assert.Equal(t, api, entry.Type)
	assert.Equal(t, name, entry.ConfigId)
}

func TestParseDeleteEntryInvalidEntryWithoutDelimiterShouldFail(t *testing.T) {
	value := "auto-tag"

	_, err := parseDeleteEntry(0, value)

	assert.Assert(t, err != nil, "value `%s` should return error", value)
}

func TestParseDeleteFileDefinitions(t *testing.T) {
	api := "auto-tag"
	name := "test entity/entry"
	entity := api + deleteDelimiter + name

	api2 := "management-zone"
	name2 := "test entity/entry"
	entity2 := api2 + deleteDelimiter + name2

	result, errors := parseDeleteFileDefinition(deleteFileDefinition{
		DeleteEntries: []string{
			entity,
			entity2,
		},
	})

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 2, len(result))

	apiEntities := result[api]

	assert.Equal(t, 1, len(apiEntities))
	assert.Equal(t, DeletePointer{
		Type:     api,
		ConfigId: name,
	}, apiEntities[0])

	api2Entities := result[api2]

	assert.Equal(t, 1, len(api2Entities))
	assert.Equal(t, DeletePointer{
		Type:     api2,
		ConfigId: name2,
	}, api2Entities[0])
}

func TestParseDeleteFileDefinitionsWithInvalidDefinition(t *testing.T) {
	api := "auto-tag"
	name := "test entity/entry"
	entity := api + deleteDelimiter + name

	api2 := "management-zone"
	name2 := "test entity/entry"
	entity2 := api2 + deleteDelimiter + name2

	result, errors := parseDeleteFileDefinition(deleteFileDefinition{
		DeleteEntries: []string{
			entity,
			entity2,
			"invalid-definition",
		},
	})

	assert.Equal(t, 1, len(errors))
	assert.Equal(t, 0, len(result))
}

func TestLoadEntriesToDelete(t *testing.T) {
	fileContent := `delete:
- management-zone/test entity/entities
- auto-tag/random tag
`

	workingDir := filepath.FromSlash("/home/test/monaco")
	deleteFile := "delete.yaml"

	fs := afero.NewMemMapFs()
	err := fs.MkdirAll(workingDir, 0777)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, filepath.Join(workingDir, deleteFile), []byte(fileContent), 0666)
	assert.NilError(t, err)

	knownApis := []string{
		"management-zone",
		"auto-tag",
	}

	result, errors := LoadEntriesToDelete(fs, knownApis, workingDir, deleteFile)

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 2, len(result))

	apiEntities := result["management-zone"]

	assert.Equal(t, 1, len(apiEntities))
	assert.Equal(t, DeletePointer{
		Type:     "management-zone",
		ConfigId: "test entity/entities",
	}, apiEntities[0])

	api2Entities := result["auto-tag"]

	assert.Equal(t, 1, len(api2Entities))
	assert.Equal(t, DeletePointer{
		Type:     "auto-tag",
		ConfigId: "random tag",
	}, api2Entities[0])
}

func TestLoadEntriesToDeleteWithInvalidEntry(t *testing.T) {
	fileContent := `delete:
- management-zone/test entity/entities
- auto-invalid
`

	workingDir := filepath.FromSlash("/home/test/monaco")
	deleteFile := "delete.yaml"

	fs := afero.NewMemMapFs()
	err := fs.MkdirAll(workingDir, 0777)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, filepath.Join(workingDir, deleteFile), []byte(fileContent), 0666)
	assert.NilError(t, err)

	knownApis := []string{
		"management-zone",
		"auto-tag",
	}

	result, errors := LoadEntriesToDelete(fs, knownApis, workingDir, deleteFile)

	assert.Equal(t, 1, len(errors))
	assert.Equal(t, 0, len(result))
}

func TestLoadEntriesToDeleteNonExistingFile(t *testing.T) {
	workingDir := filepath.FromSlash("/home/test/monaco")

	fs := afero.NewMemMapFs()
	err := fs.MkdirAll(workingDir, 0777)

	assert.NilError(t, err)

	knownApis := []string{
		"management-zone",
		"auto-tag",
	}

	result, errors := LoadEntriesToDelete(fs, knownApis, workingDir, "delete.yaml")

	assert.Equal(t, 1, len(errors))
	assert.Equal(t, 0, len(result))
}

func TestLoadEntriesToDeleteWithMalformedFile(t *testing.T) {
	fileContent := `deleting:
- auto-invalid
`

	workingDir := filepath.FromSlash("/home/test/monaco")
	deleteFile := "delete.yaml"

	fs := afero.NewMemMapFs()
	err := fs.MkdirAll(workingDir, 0777)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, filepath.Join(workingDir, deleteFile), []byte(fileContent), 0666)
	assert.NilError(t, err)

	knownApis := []string{
		"management-zone",
		"auto-tag",
	}

	result, errors := LoadEntriesToDelete(fs, knownApis, workingDir, deleteFile)

	assert.Equal(t, 1, len(errors))
	assert.Equal(t, 0, len(result))
}

func TestLoadEntriesToDeleteWithEmptyFile(t *testing.T) {
	workingDir := filepath.FromSlash("/home/test/monaco")
	deleteFile := "delete.yaml"

	fs := afero.NewMemMapFs()
	err := fs.MkdirAll(workingDir, 0777)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, filepath.Join(workingDir, deleteFile), []byte{}, 0666)
	assert.NilError(t, err)

	knownApis := []string{
		"management-zone",
		"auto-tag",
	}

	result, errors := LoadEntriesToDelete(fs, knownApis, workingDir, deleteFile)

	assert.Equal(t, 1, len(errors))
	assert.Equal(t, 0, len(result))
}
