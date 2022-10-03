//go:build unit
// +build unit

// @license
// Copyright 2022 Dynatrace LLC
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

package v2

import (
	"fmt"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"testing"

	"gotest.tools/assert"
)

func Test_checkDuplicatedId(t *testing.T) {
	assert.Equal(t, len(findDuplicatedConfigIdentifiers(nil)), 0)
	assert.Equal(t, len(findDuplicatedConfigIdentifiers(singleElementList())), 0)
	assert.Equal(t, len(findDuplicatedConfigIdentifiers(listOfDifferentElements())), 0)
	assert.Equal(t, len(findDuplicatedConfigIdentifiers(oneDuplicatedElement())), 1)
	assert.Equal(t, findDuplicatedConfigIdentifiers(oneDuplicatedElement())[0], "project:api:id")
}

func Test_reportsOneDuplicateId(t *testing.T) {
	assert.Equal(t, len(findDuplicatedConfigIdentifiers(twiceDuplicatedElement())), 1)
}

func Test_notADuplicateIfFullCoordinateIsDifferent(t *testing.T) {
	assert.Equal(t, len(findDuplicatedConfigIdentifiers(duplicatedIdInDifferentProjects())), 0)
	assert.Equal(t, len(findDuplicatedConfigIdentifiers(duplicatedIdInDifferentApis())), 0)
}

func singleElementList() []config.Config {
	return []config.Config{{Coordinate: coordinate.Coordinate{Config: "id"}}}
}

func listOfDifferentElements() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project1", Api: "api1", Config: "id1"}}}
}

func oneDuplicatedElement() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id1"}}}
}

func twiceDuplicatedElement() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id1"}}}
}

func duplicatedIdInDifferentProjects() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project1", Api: "api", Config: "id"}}}
}

func duplicatedIdInDifferentApis() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "aws-credentials", Config: "credential"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "azure-credentials", Config: "credential"}}}
}

func TestLoadProjects_LoadsSimpleProject(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 1, "Expected a single loaded project")

	c, found := got[0].Configs["env"]
	assert.Assert(t, found, "Expected configs loaded for test environment")
	assert.Equal(t, len(c), 2, "Expected a dashboard and alerting-profile configs in loaded project")

	db, found := c["dashboard"]
	assert.Assert(t, found, "Expected configs loaded for dashboard api")
	assert.Equal(t, len(db), 1, "Expected a one config to be loaded for dashboard")

	a, found := c["alerting-profile"]
	assert.Assert(t, found, "Expected configs loaded for dashboard api")
	assert.Equal(t, len(a), 1, "Expected a one config to be loaded for alerting-profile")
}

func TestLoadProjects_AllowsOverlappingIdsInDifferentApis(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Dashboard\n    template: board.json"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 1, "Expected a single loaded project")
}

func TestLoadProjects_AllowsOverlappingIdsInDifferentProjects(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project2/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project2/alerting-profile/profile.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project", "project2"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 2, "Expected two loaded project")
}

func TestLoadProjects_AllowsOverlappingIdsInEnvironmentOverride(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte(`
configs:
- id: profile
  config:
    name: Test Profile
    template: profile.json
  environmentOverrides:
    - environment: env1
      override:
        name: Some Special Name
    - environment: env2
      override:
        skip: true`), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)

	context := getFullProjectLoaderContext([]string{"alerting-profile"}, []string{"project"}, []string{"env1", "env2"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 1, "Expected a single loaded project")
	assert.Equal(t, len(got[0].Configs["env1"]), 1, "Expected one config for env1")
	assert.Equal(t, len(got[0].Configs["env2"]), 1, "Expected one config for env2")

	env1ConfCoordinate := got[0].Configs["env1"]["alerting-profile"][0].Coordinate
	env2ConfCoordinate := got[0].Configs["env2"]["alerting-profile"][0].Coordinate
	assert.Equal(t, env1ConfCoordinate, env2ConfCoordinate, "Expected coordinates to be the same between environments")
}

func TestLoadProjects_ContainsCoordinateWhenReturningErrorForDuplicates(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile2.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/config.yaml", []byte("configs:\n- id: DASH_OVERLAP\n  config:\n    name: Test Dash\n    template: dash.json\n- id: DASH_OVERLAP\n  config:\n    name: Test Dash 2\n    template: dash.json"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/dash.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 1, "Expected to fail on overlapping coordinates")
	err := gotErrs[0]
	assert.ErrorContains(t, err, "project:alerting-profile:OVERLAP")
	assert.ErrorContains(t, err, "project:dashboard:DASH_OVERLAP")
}

func TestLoadProjects_ReturnsErrOnOverlappingCoordinate_InDifferentFiles(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile2.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 1, "Expected to fail on overlapping coordinates")
}

func TestLoadProjects_ReturnsErrOnOverlappingCoordinate_InSameFile(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml",
		[]byte(`configs:
- id: OVERLAP
  config:
    name: Test Profile
    template: profile.json
- id: OVERLAP
  config:
    name: Some Other Profile
    template: profile.json`), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 1, "Expected to fail on overlapping coordinates")
}

func getSimpleProjectLoaderContext(projects []string) ProjectLoaderContext {
	return getTestProjectLoaderContext([]string{"alerting-profile", "dashboard"}, projects)
}

func getTestProjectLoaderContext(apis []string, projects []string) ProjectLoaderContext {
	return getFullProjectLoaderContext(apis, projects, []string{"env"})
}

func getFullProjectLoaderContext(apis []string, projects []string, environments []string) ProjectLoaderContext {

	projectDefinitions := make(manifest.ProjectDefinitionByProjectId, len(projects))
	for _, p := range projects {
		projectDefinitions[p] = manifest.ProjectDefinition{
			Name: p,
			Path: p + "/",
		}
	}

	envDefinitions := make(map[string]manifest.EnvironmentDefinition, len(environments))
	for _, e := range environments {
		envDefinitions[e] = manifest.EnvironmentDefinition{
			Name:  e,
			Token: &manifest.EnvironmentVariableToken{EnvironmentVariableName: fmt.Sprintf("%s_VAR", e)},
		}
	}

	return ProjectLoaderContext{
		Apis:       apis,
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects:     projectDefinitions,
			Environments: envDefinitions,
		},
		ParametersSerde: config.DefaultParameterParsers,
	}
}
