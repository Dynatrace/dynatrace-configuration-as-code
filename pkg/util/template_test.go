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
	"gotest.tools/assert"
	"testing"
)

const testMatrixTemplateWithEnvVar = "Follow the {{.color}} {{ .Env.ANIMAL }}"
const testMatrixTemplateWithProperty = "Follow the {{.color}} {{ .ANIMAL }}"

func TestGetStringWithEnvVar(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithEnvVar)
	assert.NilError(t, err)

	SetEnv(t, "ANIMAL", "cow")
	result, err := template.ExecuteTemplate(getTemplateTestProperties())
	UnsetEnv(t, "ANIMAL")

	assert.NilError(t, err)
	assert.Equal(t, "Follow the white cow", result)
}

func TestGetStringWithEnvVarLeadsToErrorIfEnvVarNotPresent(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithEnvVar)
	assert.NilError(t, err)

	UnsetEnv(t, "ANIMAL")
	_, err = template.ExecuteTemplate(getTemplateTestProperties())

	assert.ErrorContains(t, err, "map has no entry for key \"ANIMAL\"")
}

func TestGetStringLeadsToErrorIfPropertyNotPresent(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithEnvVar)
	assert.NilError(t, err)

	SetEnv(t, "ANIMAL", "cow")
	_, err = template.ExecuteTemplate(make(map[string]string)) // empty map
	UnsetEnv(t, "ANIMAL")

	assert.ErrorContains(t, err, "map has no entry for key \"color\"")
}

func TestGetStringWithEnvVarAndProperty(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithProperty)
	assert.NilError(t, err)

	SetEnv(t, "ANIMAL", "cow")
	result, err := template.ExecuteTemplate(getTemplateTestPropertiesClashingWithEnvVars())
	UnsetEnv(t, "ANIMAL")

	assert.NilError(t, err)
	assert.Equal(t, "Follow the white rabbit", result)
}

func TestGetStringWithEnvVarIncludingEqualSigns(t *testing.T) {

	template, err := NewTemplateFromString("template_test", testMatrixTemplateWithEnvVar)
	assert.NilError(t, err)

	SetEnv(t, "ANIMAL", "cow=rabbit=chicken")
	result, err := template.ExecuteTemplate(getTemplateTestProperties())
	UnsetEnv(t, "ANIMAL")

	assert.NilError(t, err)
	assert.Equal(t, "Follow the white cow=rabbit=chicken", result)
}

func getTemplateTestProperties() map[string]string {

	m := make(map[string]string)

	m["color"] = "white"
	m["animalType"] = "rabbit"

	return m
}

func getTemplateTestPropertiesClashingWithEnvVars() map[string]string {

	m := make(map[string]string)

	m["color"] = "white"
	m["ANIMAL"] = "rabbit"

	return m
}
