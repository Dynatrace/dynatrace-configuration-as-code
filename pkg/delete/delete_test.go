//go:build unit
// +build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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
	"gotest.tools/assert"
	"testing"
)

const testYamlList = `
delete:
- Han Solo
- Chewbacca
- Darth Maul
- "Count - Doku"
`

func TestUnmarshalDeleteYaml(t *testing.T) {

	result, e := unmarshalDeleteYaml(testYamlList, "test-yaml")
	assert.NilError(t, e)

	assert.Check(t, len(result) == 4)
	assert.Equal(t, "Han Solo", result[0])
	assert.Equal(t, "Chewbacca", result[1])
	assert.Equal(t, "Darth Maul", result[2])
	assert.Equal(t, "Count - Doku", result[3])
}

func TestSplitValidConfigLine(t *testing.T) {

	configType, name, err := splitConfigToDelete("dashboard/my-dashboard")
	assert.NilError(t, err)

	assert.Equal(t, "dashboard", configType)
	assert.Equal(t, "my-dashboard", name)
}

func TestSplitConfigLineWithTooManyDelimiters(t *testing.T) {

	_, _, err := splitConfigToDelete("dashboard/my/dashboard")
	assert.ErrorContains(t, err, "more than one '/' delimiter")
}

func TestSplitConfigLineWithNoDelimiter(t *testing.T) {

	_, _, err := splitConfigToDelete("dashboard-my-dashboard")
	assert.ErrorContains(t, err, "does not contain '/' delimiter")
}
