//go:build integration
// +build integration

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
	"strings"
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/assert"
)

var appendNameFunc = func(name string) string {
	return name + "_postfix"
}

var prependNameFunc = func(name string) string {
	return "prefix_" + name
}

var nameReplacingPostfixFunc = func(line string) string {
	return ReplaceName(line, appendNameFunc)
}

var nameReplacingPrefixFunc = func(line string) string {
	return ReplaceName(line, prependNameFunc)
}

func TestReplaceNameNotMatching(t *testing.T) {

	assert.Equal(t, "management-zone", nameReplacingPostfixFunc("management-zone"))
	assert.Equal(t, "config:", nameReplacingPostfixFunc("config:"))
}

func TestReplaceNameMatching(t *testing.T) {

	assert.Equal(t, "- name: test_postfix", nameReplacingPostfixFunc("- name: test"))
	assert.Equal(t, "   - name: test_postfix", nameReplacingPostfixFunc("   - name: test"))
	assert.Equal(t, "   -name: test_postfix", nameReplacingPostfixFunc("   -name: test"))
	assert.Equal(t, "	-name: test_postfix", nameReplacingPostfixFunc("	-name: test"))
	assert.Equal(t, "	-name: test_postfix  ", nameReplacingPostfixFunc("	-name: test  "))
	assert.Equal(t, "	-name: \"test_postfix\"  ", nameReplacingPostfixFunc("	-name: \"test\"  "))
	assert.Equal(t, "	-name: 'test_postfix'  ", nameReplacingPostfixFunc("	-name: 'test'  "))
}

func TestReplaceNameDependency(t *testing.T) {
	assert.Equal(t, "- name: test_postfix.id", nameReplacingPostfixFunc("- name: test_postfix.id"))
	assert.Equal(t, "- name: test_postfix.name", nameReplacingPostfixFunc("- name: test_postfix.name"))
}

func TestInMemoryReplaceNameSimpleMatching(t *testing.T) {

	transformers := []func(string) string{nameReplacingPostfixFunc}
	assertInMemoryReplace(t, transformers, "    - name: \"Test1_postfix\"")
}

func TestInMemoryReplaceNameAdvancedMatching(t *testing.T) {

	transformers := []func(string) string{nameReplacingPostfixFunc, nameReplacingPrefixFunc}
	assertInMemoryReplace(t, transformers, "    - name: \"prefix_Test1_postfix\"")
}

func assertInMemoryReplace(t *testing.T, transformers []func(string) string, expected string) {

	var reader = CreateTestFileSystem()
	err := RewriteConfigNames("test-resources", reader, transformers)
	assert.NilError(t, err)

	content, err := afero.ReadFile(reader, "test-resources/test-environments.yaml")
	assert.NilError(t, err)

	assert.Check(t, strings.Contains(string(content), expected), "content '%s' was invalid", content)
}
