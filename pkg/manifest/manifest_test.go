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
	environmentv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
	"reflect"
	"testing"
)

var sortStrings = cmpopts.SortSlices(func(a, b string) bool { return a < b })

func TestNewEnvironmentDefinitionFromV1(t *testing.T) {

	env := environmentv1.NewEnvironment("test", "name", "group", "http://google.com", "NAME")
	want := createValueEnvironmentDefinition()

	if got := NewEnvironmentDefinitionFromV1(env, "group"); !reflect.DeepEqual(got, want) {
		t.Errorf("NewEnvironmentDefinitionFromV1() = %v, want %v", got, want)
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

func TestGetEnvironmentsAsSlice(t *testing.T) {
	envs := map[string]EnvironmentDefinition{
		"Test":  {Name: "Test"},
		"Test2": {Name: "Test2"},
	}

	manifest := Manifest{
		Environments: envs,
	}

	actual := manifest.GetEnvironmentsAsSlice()

	assertEnvironmentsWithNames(t, actual, []string{"Test", "Test2"})
}

func TestManifestFilterEnvironmentsByNamesWithEmptyNames(t *testing.T) {
	envs := map[string]EnvironmentDefinition{
		"Test":  {Name: "Test"},
		"Test2": {Name: "Test2"},
	}

	manifest := Manifest{
		Environments: envs,
	}

	actual, err := manifest.FilterEnvironmentsByNames([]string{})
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

	actual, err := manifest.FilterEnvironmentsByNames(nil)
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

	actual, err := manifest.FilterEnvironmentsByNames([]string{"Test", "Test2"})
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

	actual, err := manifest.FilterEnvironmentsByNames([]string{"Test"})
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

	_, err := manifest.FilterEnvironmentsByNames([]string{"Test4"})
	assert.ErrorContains(t, err, "Test4", "Unknown environment should give an error")
}

func assertEnvironmentsWithNames(t *testing.T, environments []EnvironmentDefinition, expectedNames []string) {
	assert.Equal(t, len(environments), len(expectedNames), "Unexpected amount of environments")

	var environmentNames []string
	for _, env := range environments {
		environmentNames = append(environmentNames, env.Name)
	}

	assert.DeepEqual(t, environmentNames, expectedNames, sortStrings)
}
