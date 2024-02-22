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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"testing"
)

const test_yaml = "test-resources/templating-integration-test-config.yaml"
const test_json = "test-resources/templating-integration-test-template.json"

// Note: this tests v1 templating paths - for v2 see pkg/config/v2/template/template_integration_test.go
func TestConfigurationTemplatingFromFilesProducesValidJson(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewOsFs())
	bytes, err := afero.ReadFile(fs, test_yaml)
	require.NoErrorf(t, err, "Expected config yaml (%s) to be read without error", test_yaml)

	properties, err := UnmarshalYaml(string(bytes), test_yaml)
	require.NoErrorf(t, err, "Expected config yaml (%s) to be parsed without error", test_yaml)

	template, err := NewTemplate(fs, test_json)
	require.NoErrorf(t, err, "Expected template json (%s) to be loaded without error", test_json)

	rendered, err := template.ExecuteTemplate(properties["properties"])
	require.NoError(t, err, "Expected template to render without error\n %s", rendered)

	err = json.ValidateJson(rendered, json.Location{})
	require.NoError(t, err, "Expected rendered template to be valid JSON:\n %s", rendered)
}
