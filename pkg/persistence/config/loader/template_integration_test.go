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

package loader

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

const test_yaml = "testdata/templating-integration-test-config.yaml"

func TestConfigurationTemplatingFromFilesProducesValidJson(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewOsFs())

	loaderContext := LoaderContext{
		Environments: []manifest.EnvironmentDefinition{
			{Name: "testEnv"},
		},
		KnownApis:       map[string]struct{}{"some-api": {}},
		ParametersSerDe: config.DefaultParameterParsers,
	}

	cfgs, errs := LoadConfigFile(context.TODO(), fs, &loaderContext, test_yaml)
	require.Empty(t, errs, "Expected test config to load without error")
	require.Len(t, cfgs, 1, "Expected test config to contain a single definition")

	testCfg := cfgs[0]
	properties := getProperties(t, testCfg)

	rendered, err := template.Render(testCfg.Template, properties)
	require.NoError(t, err, "Expected template to render without error:\n %s", rendered)

	err = json.ValidateJson(rendered, json.Location{})
	require.NoError(t, err, "Expected rendered template to be valid JSON:\n %s", rendered)
}

func getProperties(t *testing.T, cfg config.Config) map[string]interface{} {
	emptyResolveCtxt := parameter.ResolveContext{}
	props := map[string]interface{}{}
	for k, p := range cfg.Parameters {
		val, err := p.ResolveValue(emptyResolveCtxt)
		assert.NoError(t, err, "Expected simple string Parameter to resolve without error")
		props[k] = val
	}
	return props
}
