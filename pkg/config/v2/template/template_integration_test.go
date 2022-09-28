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

package template

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

const test_yaml = "test-resources/templating-integration-test-config.yaml"
const test_json = "test-resources/templating-integration-test-template.json"

func TestConfigurationTemplatingFromFilesProducesValidJson(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewOsFs())

	template, err := LoadTemplate(fs, test_json)
	assert.NilError(t, err, "Expected template json (%s) to be loaded without error", test_json)

	properties := loadPropertiesFromYaml(fs, t)

	rendered, err := Render(template, properties)
	assert.NilError(t, err, "Expected template to render without error:\n %s", rendered)

	err = util.ValidateJson(rendered, util.Location{})
	assert.NilError(t, err, "Expected rendered template to be valid JSON:\n %s", rendered)
}

func loadPropertiesFromYaml(fs afero.Fs, t *testing.T) map[string]interface{} {
	bytes, err := afero.ReadFile(fs, test_yaml)
	assert.NilError(t, err, "Expected config yaml (%s) to be read without error", test_yaml)

	err, properties := util.UnmarshalYamlWithoutTemplating(string(bytes), test_yaml)
	assert.NilError(t, err, "Expected config yaml (%s) to be parsed without error", test_yaml)

	props := map[string]interface{}{}
	for k, v := range properties["properties"] {
		props[k] = v
	}
	return props
}

func TestTemplatesWithSpecialCharactersProduceValidJson(t *testing.T) {
	tests := []struct {
		name           string
		templateString string
		properties     map[string]interface{}
		want           string
	}{
		{
			"empty test should work",
			`{}`,
			map[string]interface{}{},
			`{}`,
		},
		{
			"newlines are escaped",
			`{ "key": "{{ .value }}", "object": { "o_key": "{{ .object_value}}" } }`,
			map[string]interface{}{
				"value":        "A string\nwith several lines\n\n - here's one\n\n - and another",
				"object_value": "and\none\nmore",
			},
			`{ "key": "A string\nwith several lines\n\n - here's one\n\n - and another", "object": { "o_key": "and\none\nmore" } }`,
		},
		{
			"regular slashes are not escaped",
			`{ "userAgent": "{{ .value }}" }`,
			map[string]interface{}{
				"value": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36",
			},
			`{ "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36" }`,
		},
		{
			"a v1 list definition does not get quotes escaped",
			`{ "list": [ {{ .entries }} ] }`,
			map[string]interface{}{
				"entries": `"element a", "element b", "element c"`,
			},
			`{ "list": [ "element a", "element b", "element c" ] }`,
		},
		{
			"a v1 list definition can still contain newlines",
			`{ "list": [ {{ .entries }} ] }`,
			map[string]interface{}{
				"entries": `"element a",
"element b",
"element c"`,
			},
			`{ "list": [ "element a",
"element b",
"element c" ] }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := CreateTemplateFromString("template_test", tt.templateString)

			result, err := Render(template, tt.properties)
			assert.NilError(t, err)
			assert.Equal(t, result, tt.want)

			err = util.ValidateJson(result, util.Location{})
			assert.NilError(t, err)
		})
	}
}
