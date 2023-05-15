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

package idutils

import (
	"encoding/base64"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestGenerateExternalIdIsStable(t *testing.T) {
	schemaId, id := "a", "b"

	output1, err := GenerateExternalID(coordinate.Coordinate{
		Type:     schemaId,
		ConfigId: id,
	})
	assert.NoError(t, err)
	output2, err := GenerateExternalID(coordinate.Coordinate{
		Type:     schemaId,
		ConfigId: id,
	})
	assert.NoError(t, err)
	assert.Equal(t, output1, output2)
}

func TestGenerateExternalIdGeneratesDifferentValuesForDifferentInput(t *testing.T) {
	output1, err := GenerateExternalID(coordinate.Coordinate{Type: "a", ConfigId: "a"})
	assert.NoError(t, err)
	output2, err := GenerateExternalID(coordinate.Coordinate{Type: "a", ConfigId: "b"})
	assert.NoError(t, err)
	output3, err := GenerateExternalID(coordinate.Coordinate{Type: "b", ConfigId: "b"})
	assert.NoError(t, err)

	assert.NotEqual(t, output1, output2)
	assert.NotEqual(t, output2, output3)
	assert.NotEqual(t, output1, output3)
}

func TestGenerateExternalIdWithOver500CharsCutsIt(t *testing.T) {
	output1, err := GenerateExternalID(coordinate.Coordinate{Type: strings.Repeat("a", 501)})
	assert.Zero(t, output1)
	assert.Error(t, err)
	output2, err := GenerateExternalID(coordinate.Coordinate{ConfigId: strings.Repeat("a", 501)})
	assert.Zero(t, output2)
	assert.Error(t, err)
	output3, err := GenerateExternalID(coordinate.Coordinate{Type: strings.Repeat("a", 250), ConfigId: strings.Repeat("a", 251)})
	assert.LessOrEqual(t, len(output3), 500)
	assert.NoError(t, err)

}

func TestGenerateExternalIdWithOver500CharactersProducesUniqueIDs(t *testing.T) {
	uniqueID1, err := GenerateExternalID(coordinate.Coordinate{Type: strings.Repeat("a", 250), ConfigId: strings.Repeat("a", 251)})
	assert.NoError(t, err)
	uniqueID2, err := GenerateExternalID(coordinate.Coordinate{Type: strings.Repeat("a", 250), ConfigId: strings.Repeat("a", 251)})
	assert.NoError(t, err)
	uniqueID3, err := GenerateExternalID(coordinate.Coordinate{Type: strings.Repeat("a", 250), ConfigId: strings.Repeat("a", 300)})
	assert.NoError(t, err)

	assert.Equal(t, uniqueID1, uniqueID2)
	assert.NotEqual(t, uniqueID1, uniqueID3)
}

func TestGenerateExternalIdStartsWithKnownPrefix(t *testing.T) {
	schemaId, id := "a", "b"

	extId, err := GenerateExternalID(coordinate.Coordinate{Type: schemaId, ConfigId: id})
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(extId, "monaco:"))
}

func TestGenerateExternalIdWithOther500CharsStartsWithKnownPrefix(t *testing.T) {
	extId, err := GenerateExternalID(coordinate.Coordinate{Type: strings.Repeat("a", 250), ConfigId: strings.Repeat("a", 251)})
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(extId, "monaco:"))
}

func TestGenerateExternalIdConsidersProjectName(t *testing.T) {
	expectIDWithoutProjectName := "monaco:c2NoZW1hLWlkJGNvbmZpZy1pZA=="
	expectIDWithProjectName := "monaco:cHJvamVjdC1uYW1lJHNjaGVtYS1pZCRjb25maWctaWQ="
	id1, err := GenerateExternalID(coordinate.Coordinate{
		Project:  "",
		Type:     "schema-id",
		ConfigId: "config-id",
	})
	assert.Equal(t, expectIDWithoutProjectName, id1)
	assert.NoError(t, err)
	id2, err := GenerateExternalID(coordinate.Coordinate{
		Project:  "project-name",
		Type:     "schema-id",
		ConfigId: "config-id",
	})
	assert.Equal(t, expectIDWithProjectName, id2)
	assert.NoError(t, err)
}

func TestGenerateExternalIdReturnsErrIfSchemaIDorConfigIDisMissing(t *testing.T) {

	id, err := GenerateExternalID(coordinate.Coordinate{ConfigId: "config-id"})
	assert.Zero(t, id)
	assert.Error(t, err)

	id, err = GenerateExternalID(coordinate.Coordinate{Type: "schema-id"})
	assert.Zero(t, id)
	assert.Error(t, err)

}

func TestGenerateExternalIdRawIdParts(t *testing.T) {
	id, _ := GenerateExternalID(coordinate.Coordinate{Project: "project-name", Type: "schema-id", ConfigId: "config-id"})
	decoded, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(id, "monaco:"))
	rawId := make([]byte, len(decoded))
	copy(rawId, decoded)
	assert.Equal(t, "project-name$schema-id$config-id", string(decoded))
}
