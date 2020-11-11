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
	"testing"

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
pathFromLinux:
    - id: "here/dir/file.id"
    - name: "here\\dir/file.name"
    - dir: "here/dir/"
    - abs: "/home/here/"
pathFromWindows:
    - id: "here\\dir\\file.id"
    - name: "here\\dir/file.name"
    - dir: "here\\dir\\"
    - abs: "\\home\\here\\"
retainURLs:
    - url: "https://dynatrace.com/"
`

func TestReplacePathSeparators(t *testing.T) {
	e, result := UnmarshalYaml(yamlTestPathSeparators, "test-yaml-path-separators")
	assert.NilError(t, e)

	fromLinux := result["pathFromLinux"]
	fromWindows := result["pathFromWindows"]
	url := result["retainURLs"]

	assert.Equal(t, fromLinux["id"], fromWindows["id"])
	assert.Equal(t, fromLinux["name"], fromWindows["name"])
	assert.Equal(t, fromLinux["dir"], fromWindows["dir"])
	assert.Equal(t, fromLinux["abs"], fromLinux["abs"])
	assert.Equal(t, url["url"], "https://dynatrace.com/")
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
