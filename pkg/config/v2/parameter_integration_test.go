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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/compound"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/environment"
	listParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/list"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

func TestParametersAreLoadedAsExpected(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewOsFs())

	context := LoaderContext{
		Environments: []manifest.EnvironmentDefinition{
			{Name: "testEnv"},
		},
		KnownApis:       map[string]struct{}{"some-api": {}},
		ParametersSerDe: DefaultParameterParsers,
	}

	cfgs, errs := parseConfigs(fs, &context, "test-resources/parameter-type-test-config.yaml")
	assert.Check(t, len(errs) == 0, "Expected test config to load without error")
	assert.Check(t, len(cfgs) == 1, "Expected test config to contain a single definition")

	cfg := cfgs[0]
	assert.Equal(t, cfg.Parameters["simple_value"].GetType(), valueParam.ValueParameterType)
	assert.Equal(t, cfg.Parameters["full_value"].GetType(), valueParam.ValueParameterType)
	assert.Equal(t, cfg.Parameters["complex_value"].GetType(), valueParam.ValueParameterType)
	assert.Equal(t, cfg.Parameters["simple_reference"].GetType(), reference.ReferenceParameterType)
	assert.Equal(t, cfg.Parameters["multiline_reference"].GetType(), reference.ReferenceParameterType)
	assert.Equal(t, cfg.Parameters["full_reference"].GetType(), reference.ReferenceParameterType)
	assert.Equal(t, cfg.Parameters["environment"].GetType(), envParam.EnvironmentVariableParameterType)
	assert.Equal(t, cfg.Parameters["list"].GetType(), listParam.ListParameterType)
	assert.Equal(t, cfg.Parameters["list_array"].GetType(), listParam.ListParameterType)
	assert.Equal(t, cfg.Parameters["list_full_values"].GetType(), listParam.ListParameterType)
	assert.Equal(t, cfg.Parameters["list_complex_values"].GetType(), listParam.ListParameterType)
	assert.Equal(t, cfg.Parameters["compound_value"].GetType(), compound.CompoundParameterType)
	assert.Equal(t, cfg.Parameters["empty_compound"].GetType(), compound.CompoundParameterType)
	assert.Equal(t, cfg.Parameters["compound_on_compound"].GetType(), compound.CompoundParameterType)
}
