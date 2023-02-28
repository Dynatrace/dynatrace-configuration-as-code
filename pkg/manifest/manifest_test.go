//go:build unit

// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package manifest

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/converter/v1environment"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
	"reflect"
	"testing"
)

var sortStrings = cmpopts.SortSlices(func(a, b string) bool { return a < b })

func TestNewEnvironmentDefinitionFromV1(t *testing.T) {
	type args struct {
		env   environmentV1
		group string
	}
	tests := []struct {
		name string
		args args
		want EnvironmentDefinition
	}{
		{
			"simple v1 environment is converted",
			args{
				v1environment.NewEnvironmentV1("test", "name", "group", "http://google.com", "NAME"),
				"group",
			},
			createValueEnvironmentDefinition(),
		},
		{
			"v1 environment with env var is converted",
			args{
				v1environment.NewEnvironmentV1("test", "name", "group", "{{ .Env.ENV_VAR }}", "NAME"),
				"group",
			},
			createEnvEnvironmentDefinition(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEnvironmentDefinitionFromV1(tt.args.env, tt.args.group); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEnvironmentDefinitionFromV1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvironmentDefinitionGetUrl(t *testing.T) {

	definition := createValueEnvironmentDefinition()
	url, err := definition.GetUrl()

	assert.NilError(t, err)
	assert.Equal(t, url, "http://google.com")
}

func TestEnvironmentDefinitionGetUrlMissingEnvVar(t *testing.T) {

	definition := createEnvEnvironmentDefinition()
	_, err := definition.GetUrl()

	assert.ErrorContains(t, err, "no environment variable set for ENV_VAR")
}

func TestEnvironmentDefinitionGetUrlResolveEnvVar(t *testing.T) {
	t.Setenv("ENV_VAR", "http://monaco-is-great.com")

	definition := createEnvEnvironmentDefinition()

	url, err := definition.GetUrl()

	assert.NilError(t, err)
	assert.Equal(t, url, "http://monaco-is-great.com")

}

func TestEnvironmentDefinitionGetUrlFailsOnUnkownType(t *testing.T) {

	definition := EnvironmentDefinition{
		Name: "test",
		url: UrlDefinition{
			Type:  "!!!THIS IS NOT A TYPE!!!",
			Value: "http://google.com",
		},
		Group: "group",
		Token: &EnvironmentVariableToken{EnvironmentVariableName: "NAME"},
	}
	_, err := definition.GetUrl()

	assert.ErrorContains(t, err, "url.type `!!!THIS IS NOT A TYPE!!!` is not a valid type")
}

func createEnvEnvironmentDefinition() EnvironmentDefinition {
	return EnvironmentDefinition{
		Name: "test",
		url: UrlDefinition{
			Type:  EnvironmentUrlType,
			Value: "ENV_VAR",
		},
		Group: "group",
		Token: &EnvironmentVariableToken{EnvironmentVariableName: "NAME"},
	}
}

func createValueEnvironmentDefinition() EnvironmentDefinition {
	return EnvironmentDefinition{
		Name: "test",
		url: UrlDefinition{
			Type:  ValueUrlType,
			Value: "http://google.com",
		},
		Group: "group",
		Token: &EnvironmentVariableToken{EnvironmentVariableName: "NAME"},
	}
}

func TestManifestFilterEnvironmentsByNamesWithEmptyNames(t *testing.T) {
	envs := map[string]EnvironmentDefinition{
		"Test":  {Name: "Test"},
		"Test2": {Name: "Test2"},
	}

	manifest := Manifest{
		Environments: envs,
	}

	actual, err := manifest.Environments.FilterByNames([]string{})
	assert.NilError(t, err, "empty array should not be an error")

	assertEnvironmentsWithNames(t, actual, []string{"Test", "Test2"})
}

func TestManifestFilterEnvironmentsByNamesWithNil(t *testing.T) {
	envs := map[string]EnvironmentDefinition{
		"Test":  {Name: "Test"},
		"Test2": {Name: "Test2"},
	}

	manifest := Manifest{
		Environments: envs,
	}

	actual, err := manifest.Environments.FilterByNames(nil)
	assert.NilError(t, err, "empty array should not be an error")

	assertEnvironmentsWithNames(t, actual, []string{"Test", "Test2"})
}

func TestManifestFilterEnvironmentsByNamesWithAllNames(t *testing.T) {
	envs := map[string]EnvironmentDefinition{
		"Test":  {Name: "Test"},
		"Test2": {Name: "Test2"},
	}

	manifest := Manifest{
		Environments: envs,
	}

	actual, err := manifest.Environments.FilterByNames([]string{"Test", "Test2"})
	assert.NilError(t, err, "empty array should not be an error")

	assertEnvironmentsWithNames(t, actual, []string{"Test", "Test2"})
}

func TestManifestFilterEnvironmentsByNamesWithOneName(t *testing.T) {
	envs := map[string]EnvironmentDefinition{
		"Test":  {Name: "Test"},
		"Test2": {Name: "Test2"},
	}

	manifest := Manifest{
		Environments: envs,
	}

	actual, err := manifest.Environments.FilterByNames([]string{"Test"})
	assert.NilError(t, err, "empty array should not be an error")

	assertEnvironmentsWithNames(t, actual, []string{"Test"})
}

func TestManifestFilterEnvironmentsByNamesWithAnUnknownName(t *testing.T) {
	envs := map[string]EnvironmentDefinition{
		"Test":  {Name: "Test"},
		"Test2": {Name: "Test2"},
	}

	manifest := Manifest{
		Environments: envs,
	}

	_, err := manifest.Environments.FilterByNames([]string{"Test4"})
	assert.ErrorContains(t, err, "Test4", "Unknown environment should give an error")
}

func assertEnvironmentsWithNames(t *testing.T, environments Environments, expectedNames []string) {
	assert.Equal(t, len(environments), len(expectedNames), "Unexpected amount of environments")

	var environmentNames []string
	for k := range environments {
		environmentNames = append(environmentNames, k)
	}

	assert.DeepEqual(t, environmentNames, expectedNames, sortStrings)
}

func TestFilterByGroup(t *testing.T) {
	tests := []struct {
		name  string
		envs  Environments
		group string
		exp   Environments
	}{
		{
			name:  "empty",
			envs:  Environments{},
			group: "a",
			exp:   Environments{},
		},
		{
			name:  "simple",
			envs:  Environments{"a": EnvironmentDefinition{Group: "b"}},
			group: "b",
			exp:   Environments{"a": EnvironmentDefinition{Group: "b"}},
		},
		{
			name: "filter",
			envs: Environments{
				"a": EnvironmentDefinition{Group: "b"},
				"b": EnvironmentDefinition{Group: "c"},
			},
			group: "b",
			exp:   Environments{"a": EnvironmentDefinition{Group: "b"}},
		},
		{
			name: "empty",
			envs: Environments{
				"a": EnvironmentDefinition{Group: "b"},
				"b": EnvironmentDefinition{Group: "c"},
			},
			group: "",
			exp:   Environments{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := test.envs.FilterByGroup(test.group)

			assert.DeepEqual(t, test.exp, r, cmpopts.IgnoreUnexported(EnvironmentDefinition{}))
		})
	}
}
