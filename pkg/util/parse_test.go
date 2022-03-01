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

package util

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

const testYaml = `
light:
    - Han: "Solo"
    - Chew: "Baca"
dark:
    - Darth: "Maul"
    - Count: "Doku"
`

func TestUnmarshalYaml(t *testing.T) {

	e, result := UnmarshalYaml(testYaml, "test-yaml")
	assert.NilError(t, e)

	assert.Check(t, len(result) == 2)

	light := result["light"]
	dark := result["dark"]

	assert.Check(t, light != nil)
	assert.Check(t, dark != nil)

	assert.Equal(t, "Solo", light["Han"])
	assert.Equal(t, "Baca", light["Chew"])

	assert.Equal(t, "Maul", dark["Darth"])
	assert.Equal(t, "Doku", dark["Count"])
}

const yamlTestPathSeparators = `
config:
    - application-tagging: "application-tagging.json"
arbitraryPaths:
    - p1: "// represents a comment maybe"
    - p2: "\\ only back slashes \\"
    - p3: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36"
    - p4: "/absolute/path/dashboard.id"
    - p5: "relative/path/dashboard.name"
    - p6: "\\absolute\\backslash\\dashboard.id"
    - p7: "relative\\backslash\\dashboard.name"
retainURLs:
    - url: "https://dynatrace.com/"
someExtension:
    - path: "/this/is\\a/path/with\\slashes/and\\backslashes/to\\extension.json"
`

func TestUnmarshalYamlNormalizesPathSeparatorsIfValueIsReferencingVariableInAnotherYaml(t *testing.T) {
	e, result := UnmarshalYaml(yamlTestPathSeparators, "test-yaml-path-separators")
	assert.NilError(t, e)

	config := result["config"]
	arbitraryPaths := result["arbitraryPaths"]
	url := result["retainURLs"]

	// Shorthand 'ps' for platform-dependant path separator so that less code is needed in assertions below
	ps := string(os.PathSeparator)

	assert.Equal(t, arbitraryPaths["p4"], fmt.Sprintf("%sabsolute%spath%sdashboard.id", ps, ps, ps))
	assert.Equal(t, arbitraryPaths["p5"], fmt.Sprintf("relative%spath%sdashboard.name", ps, ps))
	assert.Equal(t, arbitraryPaths["p6"], fmt.Sprintf("%sabsolute%sbackslash%sdashboard.id", ps, ps, ps))
	assert.Equal(t, arbitraryPaths["p7"], fmt.Sprintf("relative%sbackslash%sdashboard.name", ps, ps))

	assert.Equal(t, url["url"], "https://dynatrace.com/")
	assert.Equal(t, config["application-tagging"], "application-tagging.json")
}

func TestUnmarshalYamlDoesNotNormalizePathSeparatorsIfValueIsNotReferencingVariableInAnotherYaml(t *testing.T) {
	e, result := UnmarshalYaml(yamlTestPathSeparators, "test-yaml-path-separators")
	assert.NilError(t, e)

	config := result["config"]
	arbitraryPaths := result["arbitraryPaths"]
	url := result["retainURLs"]

	fmt.Println(arbitraryPaths)

	assert.Equal(t, arbitraryPaths["p1"], "// represents a comment maybe")
	assert.Equal(t, arbitraryPaths["p2"], "\\ only back slashes \\")
	assert.Equal(t, arbitraryPaths["p3"], "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36")

	assert.Equal(t, url["url"], "https://dynatrace.com/")
	assert.Equal(t, config["application-tagging"], "application-tagging.json")
}

func TestUnmarshalYamlDoesNotReplaceSlashesAndBackslashesInJsonReferenceInSectionOtherThanConfigSection(t *testing.T) {
	e, result := UnmarshalYaml(yamlTestPathSeparators, "test-yaml-path-separators")
	assert.NilError(t, e)

	someExtension := result["someExtension"]

	assert.Equal(t, someExtension["path"], "/this/is\\a/path/with\\slashes/and\\backslashes/to\\extension.json")
}

const yamlTestEnvVar = `
envVars:
    - env-var: "{{ .Env.TEST_ENV_VAR }}"
    - env-var-with-content: "{{ .Env.TEST_ENV_VAR }} Or am I?"
`

func TestReplaceEnvVarWhenVarIsPresent(t *testing.T) {

	SetEnv(t, "TEST_ENV_VAR", "I'm the king of the World!")

	e, result := UnmarshalYaml(yamlTestEnvVar, "test-yaml-test-env-var")
	assert.NilError(t, e)

	testMap := result["envVars"]
	assert.Equal(t, "I'm the king of the World!", testMap["env-var"])
	assert.Equal(t, "I'm the king of the World! Or am I?", testMap["env-var-with-content"])

	UnsetEnv(t, "TEST_ENV_VAR")
}

func TestReplaceEnvVarWhenVarIsNotPresent(t *testing.T) {

	// just in case:
	UnsetEnv(t, "TEST_ENV_VAR")

	err, _ := UnmarshalYaml(yamlTestEnvVar, "test-yaml-test-env-var")
	assert.ErrorContains(t, err, "map has no entry for key \"TEST_ENV_VAR\"")
}

const testYamlParsingIssueOnLevel1 = `
light: dark
`

func TestUnmarshalConvertYamlHasParsingIssuesOnLevel1(t *testing.T) {
	// Given
	m := make(map[string]interface{})
	yaml.Unmarshal([]byte(testYamlParsingIssueOnLevel1), &m)

	// When
	e, _ := convert(m)

	// Then
	assert.ErrorContains(t, e, "cannot convert YAML on level 1: value of key 'light' has unexpected type")
}

const testYamlParsingIssueOnLevel2 = `
light:
 - test
   - test2
`

func TestUnmarshalConvertYamlHasParsingIssuesOnLevel2(t *testing.T) {
	// Given
	m := make(map[string]interface{})
	yaml.Unmarshal([]byte(testYamlParsingIssueOnLevel2), &m)

	// When
	e, _ := convert(m)

	// Then
	assert.ErrorContains(t, e, "cannot convert YAML on level 2: test - test2")
}

const testYamlParsingIssueOnLevel3 = `
light:
 - 123: test2
`

func TestUnmarshalConvertYamlHasParsingIssuesOnLevel3(t *testing.T) {
	// Given
	m := make(map[string]interface{})
	yaml.Unmarshal([]byte(testYamlParsingIssueOnLevel3), &m)

	// When
	e, _ := convert(m)

	// Then
	assert.ErrorContains(t, e, "cannot convert YAML on level 3: invalid key type '%!s(int=123)'")
}

const testYamlParsingIssueOnLevel4 = `
light:
    - Han:
       - test
`

func TestUnmarshalConvertYamlHasParsingIssuesOnLevel4(t *testing.T) {
	// Given
	m := make(map[string]interface{})
	yaml.Unmarshal([]byte(testYamlParsingIssueOnLevel4), &m)

	// When
	e, _ := convert(m)

	// Then
	assert.ErrorContains(t, e, "cannot convert YAML on level 4: value of key 'Han' has unexpected type")
}
