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
	"gotest.tools/assert"
	"strings"
	"testing"
)

func TestGenerateExternalIdIsStable(t *testing.T) {
	schemaId, id := "a", "b"

	output1 := GenerateExternalID(schemaId, id)
	output2 := GenerateExternalID(schemaId, id)

	assert.Equal(t, output1, output2)
}

func TestGenerateExternalIdGeneratesDifferentValuesForDifferentInput(t *testing.T) {
	output1 := GenerateExternalID("a", "a")
	output2 := GenerateExternalID("a", "b")
	output3 := GenerateExternalID("b", "b")

	assert.Assert(t, output1 != output2)
	assert.Assert(t, output2 != output3)
	assert.Assert(t, output1 != output3)
}

func TestGenerateExternalIdWithOver500CharsCutsIt(t *testing.T) {
	output1 := GenerateExternalID(strings.Repeat("a", 501), "")
	output2 := GenerateExternalID("", strings.Repeat("a", 501))
	output3 := GenerateExternalID(strings.Repeat("a", 250), strings.Repeat("a", 251))

	assert.Assert(t, len(output1) <= 500)
	assert.Assert(t, len(output2) <= 500)
	assert.Assert(t, len(output3) <= 500)
}

func TestGenerateExternalIdWithOther500CharsIsStable(t *testing.T) {
	output1 := GenerateExternalID(strings.Repeat("a", 250), strings.Repeat("a", 251))
	output2 := GenerateExternalID(strings.Repeat("a", 250), strings.Repeat("a", 251))
	output3 := GenerateExternalID(strings.Repeat("a", 250), strings.Repeat("a", 300))

	assert.Equal(t, output1, output2)
	assert.Assert(t, output1 != output3)
}

func TestGenerateExternalIdStartsWithKnownPrefix(t *testing.T) {
	schemaId, id := "a", "b"

	extId := GenerateExternalID(schemaId, id)

	assert.Assert(t, strings.HasPrefix(extId, "monaco:"))
}

func TestGenerateExternalIdWithOther500CharsStartsWithKnownPrefix(t *testing.T) {
	extId := GenerateExternalID(strings.Repeat("a", 250), strings.Repeat("a", 251))

	assert.Assert(t, strings.HasPrefix(extId, "monaco:"))
}
