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
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"testing"

	"gotest.tools/assert"
)

func Test_checkDuplicatedId(t *testing.T) {
	assert.Equal(t, len(getDuplicatedId(nil)), 0)
	assert.Equal(t, len(getDuplicatedId(singleElementList())), 0)
	assert.Equal(t, len(getDuplicatedId(listOfDifferentElements())), 0)
	assert.Equal(t, len(getDuplicatedId(oneDuplicatedElement())), 1)
	assert.Equal(t, getDuplicatedId(oneDuplicatedElement())[0], "id")
}

func Test_reportsOneDuplicateId(t *testing.T) {
	assert.Equal(t, len(getDuplicatedId(twiceDuplicatedElement())), 1)
}

func Test_notADuplicateIfFullCoordinateIsDifferent(t *testing.T) {
	assert.Equal(t, len(getDuplicatedId(duplicatedIdInDifferentProjects())), 0)
	assert.Equal(t, len(getDuplicatedId(duplicatedIdInDifferentApis())), 0)
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

	context := getTestProjectLoaderContext([]string{"dashboard", "alerting-profile"}, []string{"project"})

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

	context := getTestProjectLoaderContext([]string{"dashboard", "alerting-profile"}, []string{"project"})

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

	context := getTestProjectLoaderContext([]string{"alerting-profile"}, []string{"project", "project2"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 2, "Expected two loaded project")
}

func TestLoadProjects_ReturnsErrOnOverlappingCoordinate_InDifferentFiles(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile2.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)

	context := getTestProjectLoaderContext([]string{"alerting-profile"}, []string{"project"})

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

	context := getTestProjectLoaderContext([]string{"alerting-profile"}, []string{"project"})

	_, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 1, "Expected to fail on overlapping coordinates")
}

func getTestProjectLoaderContext(apis []string, projects []string) ProjectLoaderContext {

	projectDefinitions := make(manifest.ProjectDefinitionByProjectId, len(projects))
	for _, p := range projects {
		projectDefinitions[p] = manifest.ProjectDefinition{
			Name: p,
			Path: p + "/",
		}
	}

	return ProjectLoaderContext{
		Apis:       apis,
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: projectDefinitions,
			Environments: map[string]manifest.EnvironmentDefinition{
				"env": {
					Name:  "env",
					Token: &manifest.EnvironmentVariableToken{EnvironmentVariableName: "ENV_VAR"},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}
}
