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
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
	"testing"
)

var sortStrings = cmpopts.SortSlices(func(a, b string) bool { return a < b })

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
