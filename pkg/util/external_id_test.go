// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"gotest.tools/assert"
	"strings"
	"testing"
)

func TestGenerateExternalIdIsStable(t *testing.T) {
	schemaId, id := "a", "b"

	output1 := GenerateExternalId(schemaId, id)
	output2 := GenerateExternalId(schemaId, id)

	assert.Equal(t, output1, output2)
}

func TestGenerateExternalIdGeneratesDifferentValuesForDifferentInput(t *testing.T) {
	output1 := GenerateExternalId("a", "a")
	output2 := GenerateExternalId("a", "b")
	output3 := GenerateExternalId("b", "b")

	assert.Assert(t, output1 != output2)
	assert.Assert(t, output2 != output3)
	assert.Assert(t, output1 != output3)
}

func TestGenerateExternalIdWithOver500CharsCutsIt(t *testing.T) {
	output1 := GenerateExternalId(strings.Repeat("a", 501), "")
	output2 := GenerateExternalId("", strings.Repeat("a", 501))
	output3 := GenerateExternalId(strings.Repeat("a", 250), strings.Repeat("a", 251))

	assert.Assert(t, len(output1) <= 500)
	assert.Assert(t, len(output2) <= 500)
	assert.Assert(t, len(output3) <= 500)
}
