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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	listParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/list"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

func TestParametersAreLoadedAsExpected(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewOsFs())

	loaderContext := LoaderContext{
		Environments: manifest.Environments{
			SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
				"testEnv": {Name: "testEnv"},
			},
			AllEnvironmentNames: map[string]struct{}{
				"testenv": {},
			},
		},
		KnownApis:       map[string]struct{}{"some-api": {}},
		ParametersSerDe: config.DefaultParameterParsers,
	}

	cfgs, errs := LoadConfigFile(t.Context(), fs, &loaderContext, "testdata/parameter-type-test-config.yaml")
	require.Empty(t, errs, "Expected test config to load without error")
	assert.Len(t, cfgs, 1, "Expected test config to contain a single definition")

	cfg := cfgs[0]
	assert.Equal(t, valueParam.ValueParameterType, cfg.Parameters["simple_value"].GetType())
	assert.Equal(t, valueParam.ValueParameterType, cfg.Parameters["full_value"].GetType())
	assert.Equal(t, valueParam.ValueParameterType, cfg.Parameters["complex_value"].GetType())
	assert.Equal(t, reference.ReferenceParameterType, cfg.Parameters["simple_reference"].GetType())
	assert.Equal(t, reference.ReferenceParameterType, cfg.Parameters["multiline_reference"].GetType())
	assert.Equal(t, reference.ReferenceParameterType, cfg.Parameters["full_reference"].GetType())
	assert.Equal(t, envParam.EnvironmentVariableParameterType, cfg.Parameters["environment"].GetType())
	assert.Equal(t, listParam.ListParameterType, cfg.Parameters["list"].GetType())
	assert.Equal(t, listParam.ListParameterType, cfg.Parameters["list_array"].GetType())
	assert.Equal(t, listParam.ListParameterType, cfg.Parameters["list_full_values"].GetType())
	assert.Equal(t, listParam.ListParameterType, cfg.Parameters["list_complex_values"].GetType())
	assert.Equal(t, compound.CompoundParameterType, cfg.Parameters["compound_value"].GetType())
	assert.Equal(t, compound.CompoundParameterType, cfg.Parameters["empty_compound"].GetType())
	assert.Equal(t, compound.CompoundParameterType, cfg.Parameters["compound_on_compound"].GetType())
}
