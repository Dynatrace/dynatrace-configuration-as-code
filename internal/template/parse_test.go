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

package template

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"gopkg.in/yaml.v2"
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

	result, e := UnmarshalYaml(testYaml, "test-yaml")
	require.NoError(t, e)

	require.Len(t, result, 2)

	light := result["light"]
	dark := result["dark"]

	assert.NotNil(t, light)
	assert.NotNil(t, dark)

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
	result, e := UnmarshalYaml(yamlTestPathSeparators, "test-yaml-path-separators")
	require.NoError(t, e)

	config := result["config"]
	arbitraryPaths := result["arbitraryPaths"]
	url := result["retainURLs"]

	// Shorthand 'ps' for platform-dependant path separator so that less code is needed in assertions below
	ps := string(os.PathSeparator)

	assert.Equal(t, fmt.Sprintf("%sabsolute%spath%sdashboard.id", ps, ps, ps), arbitraryPaths["p4"])
	assert.Equal(t, fmt.Sprintf("relative%spath%sdashboard.name", ps, ps), arbitraryPaths["p5"])
	assert.Equal(t, fmt.Sprintf("%sabsolute%sbackslash%sdashboard.id", ps, ps, ps), arbitraryPaths["p6"])
	assert.Equal(t, fmt.Sprintf("relative%sbackslash%sdashboard.name", ps, ps), arbitraryPaths["p7"])

	assert.Equal(t, "https://dynatrace.com/", url["url"])
	assert.Equal(t, "application-tagging.json", config["application-tagging"])
}

func TestUnmarshalYamlDoesNotNormalizePathSeparatorsIfValueIsNotReferencingVariableInAnotherYaml(t *testing.T) {
	result, e := UnmarshalYaml(yamlTestPathSeparators, "test-yaml-path-separators")
	require.NoError(t, e)

	config := result["config"]
	arbitraryPaths := result["arbitraryPaths"]
	url := result["retainURLs"]

	assert.Equal(t, "// represents a comment maybe", arbitraryPaths["p1"])
	assert.Equal(t, "\\ only back slashes \\", arbitraryPaths["p2"])
	assert.Equal(t, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36", arbitraryPaths["p3"])

	assert.Equal(t, "https://dynatrace.com/", url["url"])
	assert.Equal(t, "application-tagging.json", config["application-tagging"])
}

func TestUnmarshalYamlDoesNotReplaceSlashesAndBackslashesInJsonReferenceInSectionOtherThanConfigSection(t *testing.T) {
	result, e := UnmarshalYaml(yamlTestPathSeparators, "test-yaml-path-separators")
	require.NoError(t, e)

	someExtension := result["someExtension"]

	assert.Equal(t, "/this/is\\a/path/with\\slashes/and\\backslashes/to\\extension.json", someExtension["path"])
}

const yamlTestEnvVar = `
envVars:
    - env-var: "{{ .Env.TEST_ENV_VAR }}"
    - env-var-with-content: "{{ .Env.TEST_ENV_VAR }} Or am I?"
`

func TestReplaceEnvVarWhenVarIsPresent(t *testing.T) {

	t.Setenv("TEST_ENV_VAR", "I'm the king of the World!")

	result, e := UnmarshalYaml(yamlTestEnvVar, "test-yaml-test-env-var")
	require.NoError(t, e)

	testMap := result["envVars"]
	assert.Equal(t, "I'm the king of the World!", testMap["env-var"])
	assert.Equal(t, "I'm the king of the World! Or am I?", testMap["env-var-with-content"])
}

func TestReplaceEnvVarWhenVarIsNotPresent(t *testing.T) {

	_, err := UnmarshalYaml(yamlTestEnvVar, "test-yaml-test-env-var")
	require.ErrorContains(t, err, "map has no entry for key \"TEST_ENV_VAR\"")
}

func TestUnmarshalYamlWithoutTemplatingDoesNotReplaceVariables(t *testing.T) {

	t.Setenv("TEST_ENV_VAR", "I'm the king of the World!")

	result, e := UnmarshalYamlWithoutTemplating(yamlTestEnvVar, "test-yaml-test-env-var")
	require.NoError(t, e)

	testMap := result["envVars"]
	assert.Equal(t, "{{ .Env.TEST_ENV_VAR }}", testMap["env-var"])
	assert.Equal(t, "{{ .Env.TEST_ENV_VAR }} Or am I?", testMap["env-var-with-content"])
}

func TestUnmarshalYamlWithoutTemplatingDoesNotFailIfVariablesAreMissing(t *testing.T) {
	_, e := UnmarshalYamlWithoutTemplating(yamlTestEnvVar, "test-yaml-test-env-var")
	require.NoError(t, e)
}

const testYamlParsingIssueOnLevel1 = `
light: dark
`

func TestUnmarshalConvertYamlHasParsingIssuesOnLevel1(t *testing.T) {
	// Given
	m := make(map[string]interface{})
	yaml.Unmarshal([]byte(testYamlParsingIssueOnLevel1), &m)

	// When
	_, e := convert(m)

	// Then
	require.ErrorContains(t, e, "cannot convert YAML on level 1: value of key 'light' has unexpected type")
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
	_, e := convert(m)

	// Then
	require.ErrorContains(t, e, "cannot convert YAML on level 2: test - test2")
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
	_, e := convert(m)

	// Then
	require.ErrorContains(t, e, "cannot convert YAML on level 3: invalid key type '%!s(int=123)'")
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
	_, e := convert(m)

	// Then
	require.ErrorContains(t, e, "cannot convert YAML on level 4: value of key 'Han' has unexpected type")
}

func Test_ensureAnyTemplateStringsAreInQuotes(t *testing.T) {

	tests := []struct {
		given string
		want  string
	}{
		{
			"random string",
			"random string",
		},
		{
			`value: "{{ something in quotes }}"`,
			`value: "{{ something in quotes }}"`,
		},
		{
			`- url: {{ no quotes }}`,
			`- url: "{{ no quotes }}"`,
		},
		{
			`- url: "  {{ end quote  missing YAML error is unchanged }}`,
			`- url: "  {{ end quote  missing YAML error is unchanged }}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.given, func(t *testing.T) {
			if got := ensureAnyTemplateStringsAreInQuotes(tt.given); got != tt.want {
				t.Errorf("ensureAnyTemplateStringsAreInQuotes() = %v, want %v", got, tt.want)
			}
		})
	}
}
