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

package integrationtest

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"

	"github.com/spf13/afero"
)

var postfixFunc = func(s string) string {
	return s + "_postfix"
}

var prefixFunc = func(s string) string {
	return "prefix_" + s
}

var nameReplacingPostfixFunc = func(line string) string {
	return ReplaceName(line, postfixFunc)
}

var nameReplacingPostfixFuncForFileContent = func(fileContent string) string {

	var result = ""
	lines := strings.Split(fileContent, "\n")
	for i, line := range lines {
		result += ReplaceName(line, postfixFunc)
		if i < len(lines)-1 {
			result += "\n"
		}
	}
	return result
}

var nameReplacingPrefixFunc = func(line string) string {
	return ReplaceName(line, prefixFunc)
}

var idReplacingPostfixFunc = func(line string) string {
	return ReplaceId(line, postfixFunc)
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
	assert.Equal(t, "name: calc:synthetic.browser.delorean.speed_postfix", nameReplacingPostfixFunc("name: calc:synthetic.browser.delorean.speed"))
	assert.Equal(t, "  - name: calc:synthetic.browser.delorean.speed_postfix", nameReplacingPostfixFunc("  - name: calc:synthetic.browser.delorean.speed"))
}

func TestReplaceNameMatchingConfigV2(t *testing.T) {

	const configV2Config = `configs:
- id: profile
  config:
    name: Star Trek Service
    template: profile.json
    skip: false`

	const configV2ConfigExpected = `configs:
- id: profile
  config:
    name: Star Trek Service_postfix
    template: profile.json
    skip: false`

	result := nameReplacingPostfixFuncForFileContent(configV2Config)
	assert.Equal(t, configV2ConfigExpected, result)
}

func TestReplaceNameDependencyV2(t *testing.T) {
	assert.Equal(t, " name: [ \"project\",\"api\",\"test_postfix\",\"id\" ]", nameReplacingPostfixFunc(" name: [ \"project\",\"api\",\"test_postfix\",\"id\" ]"))
	assert.Equal(t, " name: [ \"project\",\"api\",\"test_postfix\",\"name\" ]", nameReplacingPostfixFunc(" name: [ \"project\",\"api\",\"test_postfix\",\"name\" ]"))
}

func TestRewriteConfigNames(t *testing.T) {
	tests := []struct {
		name              string
		givenTransformers []func(string) string
		expectedFile      string
	}{
		{
			"appends postfix to name",
			[]func(string) string{nameReplacingPostfixFunc},
			"expected-name-postfix.yaml",
		},
		{
			"surrounds name with pre- and postfix",
			[]func(string) string{nameReplacingPostfixFunc, nameReplacingPrefixFunc},
			"expected-name-full.yaml",
		},
		{
			"appends postfix to config IDs",
			[]func(string) string{idReplacingPostfixFunc},
			"expected-id-postfix.yaml",
		},
		{
			"appends postfix to name and config IDs",
			[]func(string) string{nameReplacingPostfixFunc, idReplacingPostfixFunc},
			"expected-both-postfix.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader = testutils.CreateTestFileSystem()
			err := RewriteConfigNames("rewrite-test-resources/given", reader, tt.givenTransformers)
			assert.NoError(t, err)

			got, err := afero.ReadFile(reader, "rewrite-test-resources/given/rewrite-test-config.yaml")
			assert.NoError(t, err)
			want, err := afero.ReadFile(reader, "rewrite-test-resources/want/"+tt.expectedFile)
			assert.NoError(t, err)
			want = bytes.ReplaceAll(want, []byte{'\r'}, []byte{})

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("RewriteConfigNames() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReplaceId(t *testing.T) {

	tests := []struct {
		name  string
		given string
		want  string
	}{
		{
			"leaves empty string unchanged",
			"",
			"",
		},
		{
			"leaves non-id string unchanged",
			"someproperty: value",
			"someproperty: value",
		},
		{
			"leaves non-id string with id unchanged",
			"someproperty: id",
			"someproperty: id",
		},
		{
			"replaces config property",
			"- id: theConfigId",
			"- id: theConfigId_postfix",
		},
		{
			"leaves id marked with no replace unchanged",
			"- id: theConfigId #monaco-test:no-replace",
			"- id: theConfigId #monaco-test:no-replace",
		},
		{
			"replaces configId reference prop",
			"   configId: theConfigId",
			"   configId: theConfigId_postfix",
		},
		{
			"replaces id with colors in value",
			"   id: extra:colons",
			"   id: extra:colons_postfix",
		},
		{
			"replaces configId in shorthand v2 reference #1",
			`someRef: ["project", "type", "theConfigId", "id"]`,
			`someRef: ["project", "type","theConfigId_postfix", "id"]`,
		},
		{
			"replaces configId in shorthand v2 reference #2",
			`someRef: ["theConfigId", "id" ]`,
			`someRef: ["theConfigId_postfix", "id" ]`,
		},
		{
			"replaces configId in shorthand v2 reference #3",
			`someRef: ["project","type","theConfigId","name"]`,
			`someRef: ["project","type","theConfigId_postfix","name"]`,
		},
		{
			"replaces configId in shorthand v2 reference #4",
			`someRef: ["theConfigId", "prop" ]`,
			`someRef: ["theConfigId_postfix", "prop" ]`,
		},
		{
			"replaces configId in shorthand v2 reference #5",
			`someRef: [project,type,theConfigId,name]`,
			`someRef: [project,type,"theConfigId_postfix",name]`,
		},
		{
			"replaces configId in shorthand v2 reference #6",
			`scope: ["builtin:tags.auto-tagging", "value-scope", "scope"]`,
			`scope: ["builtin:tags.auto-tagging","value-scope_postfix", "scope"]`,
		},
		{
			"does not replace in random array #1",
			`someArr: []`,
			`someArr: []`,
		},
		{
			"does not replace in random array #2",
			`someArr: [ project, type, configId, property, oneTooMany, twoTooMany]`,
			`someArr: [ project, type, configId, property, oneTooMany, twoTooMany]`,
		},
		{
			"does not replace in values (list type) array",
			`values: [ some_val, some_val2, some_val3]`,
			`values: [ some_val, some_val2, some_val3]`,
		},
		{
			"does not replace in values (list type) array #2",
			` values  : [ some_val, some_val2, some_val3]`,
			` values  : [ some_val, some_val2, some_val3]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ReplaceId(tt.given, postfixFunc))
		})
	}
}
