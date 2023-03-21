//go:build unit

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

package v2

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

const test_yaml = "test-resources/templating-integration-test-config.yaml"

func TestConfigurationTemplatingFromFilesProducesValidJson(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewOsFs())

	context := LoaderContext{
		Environments: []manifest.EnvironmentDefinition{
			{Name: "testEnv"},
		},
		KnownApis:       map[string]struct{}{"some-api": {}},
		ParametersSerDe: DefaultParameterParsers,
	}

	cfgs, errs := parseConfigs(fs, &context, test_yaml)
	assert.Check(t, len(errs) == 0, "Expected test config to load without error")
	assert.Check(t, len(cfgs) == 1, "Expected test config to contain a single definition")

	testCfg := cfgs[0]
	properties := getProperties(t, testCfg)

	rendered, err := template.Render(testCfg.Template, properties)
	assert.NilError(t, err, "Expected template to render without error:\n %s", rendered)

	err = json.ValidateJson(rendered, json.Location{})
	assert.NilError(t, err, "Expected rendered template to be valid JSON:\n %s", rendered)
}

func getProperties(t *testing.T, cfg Config) map[string]interface{} {
	emptyResolveCtxt := parameter.ResolveContext{}
	props := map[string]interface{}{}
	for k, p := range cfg.Parameters {
		val, err := p.ResolveValue(emptyResolveCtxt)
		assert.NilError(t, err, "Expected simple string Parameter to resolve without error")
		props[k] = val
	}
	return props
}
