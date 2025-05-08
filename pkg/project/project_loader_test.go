//go:build unit

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

package project

import (
	"bytes"
	"fmt"
	"io/fs"
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

const testDirectoryFileMode = fs.FileMode(0755)
const testFileFileMode = fs.FileMode(0644)

func Test_findDuplicatedConfigIdentifiers(t *testing.T) {
	tests := []struct {
		name         string
		input        []config.Config
		want         []error
		wantErrorMap int
	}{
		{
			"nil input produces empty output",
			nil,
			nil,
			0,
		},
		{
			"no duplicates in single config",
			[]config.Config{{Coordinate: coordinate.Coordinate{ConfigId: "id"}}},
			nil,
			0,
		},
		{
			"no duplicates if project differs",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project1", Type: "api", ConfigId: "id"}},
			},
			nil,
			0,
		},
		{
			"no duplicates if api differs",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "aws-credentials", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "azure-credentials", ConfigId: "id"}},
			},
			nil,
			0,
		},
		{
			"no duplicates in list of disparate configs",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project1", Type: "api1", ConfigId: "id1"}},
			},
			nil,
			0,
		},
		{
			"finds duplicate configs",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"}},
			},
			[]error{newDuplicateConfigIdentifierError(config.Config{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}})},
			1,
		},
		{
			"finds each duplicate",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"}},
			},
			[]error{
				newDuplicateConfigIdentifierError(config.Config{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}}),
				newDuplicateConfigIdentifierError(config.Config{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}}),
			},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorMap := make(map[coordinate.Coordinate]struct{})
			got := findDuplicatedConfigIdentifiers(t.Context(), tt.input, errorMap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findDuplicatedConfigIdentifiers() got = %v, want %v", got, tt.want)
			}
			assert.Equal(t, len(errorMap), tt.wantErrorMap)
		})
	}
}

func Test_checkKeyUserActionScope(t *testing.T) {
	tests := []struct {
		name  string
		input []config.Config
		want  []error
	}{
		{
			"nil input produces empty output",
			nil,
			nil,
		},
		{
			"does not return any errors if valid",
			[]config.Config{
				{
					Coordinate: coordinate.Coordinate{Project: "project", Type: "key-user-actions-web", ConfigId: "id"},
					Parameters: config.Parameters{
						"scope": &reference.ReferenceParameter{
							ParameterReference: parameter.ParameterReference{
								Config: coordinate.Coordinate{
									Project:  "project",
									Type:     "key-user-actions-web",
									ConfigId: "id",
								},
								Property: "prop",
							},
						},
					},
				},
			},
			nil,
		},
		{
			"errors with missing or wrong scope",
			[]config.Config{
				{
					Coordinate: coordinate.Coordinate{Project: "project", Type: "key-user-actions-web", ConfigId: "id1"},
					Parameters: config.Parameters{
						"scope": &value.ValueParameter{Value: "some-value"},
					},
				},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "key-user-actions-web", ConfigId: "id2"}},
			},
			[]error{
				fmt.Errorf("scope parameter of config of type 'key-user-actions-web' with ID 'id1' needs to be a reference parameter to another web-application config"),
				fmt.Errorf("scope parameter of config of type 'key-user-actions-web' with ID 'id2' needs to be a reference parameter to another web-application config"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorMap := make(map[coordinate.Coordinate]struct{})
			got := checkKeyUserActionScope(t.Context(), tt.input, errorMap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkKeyUserActionScope() got = %v, want %v", got, tt.want)
			}
			assert.Equal(t, len(errorMap), len(tt.want))
		})
	}
}

func TestLoadProjects_RejectsManifestsWithNoProjects(t *testing.T) {
	testFs := testutils.TempFs(t)
	loaderContext := getSimpleProjectLoaderContext([]string{})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, got, 0, "Expected no project loaded")
	assert.Len(t, gotErrs, 1, "Expected to fail with no projects")
	assert.ErrorContains(t, gotErrs[0], "no projects")
}

func TestLoadProjects_LoadsSimpleProject(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, testFs.MkdirAll("project/dashboard", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)
	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")

	dashboards := findConfigs(t, got[0], "env", "dashboard")
	assert.Len(t, dashboards, 1, "Expected a one config to be loaded for dashboard")

	alertingProfiles := findConfigs(t, got[0], "env", "alerting-profile")
	assert.Len(t, alertingProfiles, 1, "Expected a one config to be loaded for alerting-profile")
}

func TestLoadProjects_LoadsSimpleProjectInFoldersNotMatchingApiName(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, testFs.MkdirAll("project/not-dashboard-dir", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/not-dashboard-dir/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/not-dashboard-dir/board.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)
	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")

	db := findConfigs(t, got[0], "env", "dashboard")
	assert.Len(t, db, 1, "Expected a one config to be loaded for dashboard")

	a := findConfigs(t, got[0], "env", "alerting-profile")
	assert.Len(t, a, 1, "Expected a one config to be loaded for alerting-profile")
}

func TestLoadProjects_LoadsProjectInRootDir(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/board.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")

	db := findConfigs(t, got[0], "env", "dashboard")
	assert.Len(t, db, 1, "Expected a one config to be loaded for dashboard")

	a := findConfigs(t, got[0], "env", "alerting-profile")
	assert.Len(t, a, 1, "Expected a one config to be loaded for ")
}

func TestLoadProjects_LoadsProjectInManyDirs(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/a/b/c", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/a/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: ../profile.json\n  type:\n    api: alerting-profile\n- id: profile2\n  config:\n    name: Test Profile\n    template: b/c/profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/a/b/c/profile.json", []byte("{}"), 0644))

	require.NoError(t, afero.WriteFile(testFs, "project/a/b/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: ../../board.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/board.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	errutils.PrintErrors(gotErrs)
	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")

	db := findConfigs(t, got[0], "env", "dashboard")
	assert.Len(t, db, 1, "Expected a one config to be loaded for dashboard")

	a := findConfigs(t, got[0], "env", "alerting-profile")
	assert.Len(t, a, 2, "Expected a one config to be loaded for alerting-profile")
}

func TestLoadProjects_LoadsProjectInHiddenDirDoesNotLoad(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/.a", 0755))
	require.NoError(t, testFs.MkdirAll("project/b", 0755))
	require.NoError(t, testFs.MkdirAll("project/a/.b/c", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/.a/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: ../b/profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/b/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/b/profile.json", []byte("{}"), 0644))

	require.NoError(t, afero.WriteFile(testFs, "project/a/.b/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: ../../board.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/a/.b/c/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: ../../board.json\n  type:\n    api: dashboard"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	errutils.PrintErrors(gotErrs)
	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")

	db := findConfigs(t, got[0], "env", "dashboard")
	assert.Len(t, db, 0, "Expected zero config to be loaded for dashboard")

	a := findConfigs(t, got[0], "env", "alerting-profile")
	assert.Len(t, a, 1, "Expected a one config to be loaded for alerting-profile")
}

func TestLoadProjects_NameDuplicationParameterShouldNotBePresentForOneEnvironment(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.Mkdir("project", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/profile.json", []byte("{}"), 0644))

	loaderContext := getFullProjectLoaderContext(
		[]string{"alerting-profile", "dashboard"},
		[]string{"project"},
		[]string{"env"})

	projects, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)
	assert.Empty(t, gotErrs)
	assert.Len(t, projects, 1, "expected one project")

	envProfile := findConfig(t, projects[0], "env", "alerting-profile", 0)
	assert.NotContains(t, envProfile.Parameters, config.NonUniqueNameConfigDuplicationParameter, "name duplication parameter should not be present")
}

func TestLoadProjects_NameDuplicationParameterShouldNotBePresentForTwoEnvironments(t *testing.T) {
	// the name duplication check should find names that are duplicated in the configs **in the same env**
	// it is valid that configs have the same name if they're deployed to separate environments.

	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.Mkdir("project", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/profile.json", []byte("{}"), 0644))

	loaderContext := getFullProjectLoaderContext(
		[]string{"alerting-profile", "dashboard"},
		[]string{"project"},
		[]string{"env", "env2"})

	projects, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)
	assert.Empty(t, gotErrs)
	assert.Len(t, projects, 1, "expected one project")

	envProfile := findConfig(t, projects[0], "env", "alerting-profile", 0)
	env2Profile := findConfig(t, projects[0], "env2", "alerting-profile", 0)

	assert.NotContains(t, envProfile.Parameters, config.NonUniqueNameConfigDuplicationParameter, "name duplication parameter should not be present")
	assert.NotContains(t, env2Profile.Parameters, config.NonUniqueNameConfigDuplicationParameter, "name duplication parameter should not be present")
}

func TestLoadProjects_NameDuplicationParameterShouldBePresentIfNameIsDuplicatedTwoEnvironments(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.Mkdir("project", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard.yaml", []byte("configs:\n- id: dashboard\n  config:\n    name: Dashboard\n    template: dashboard.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard2.yaml", []byte("configs:\n- id: dashboard2\n  config:\n    name: Dashboard\n    template: dashboard.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard.json", []byte("{}"), 0644))

	loaderContext := getFullProjectLoaderContext(
		[]string{"alerting-profile", "dashboard"},
		[]string{"project"},
		[]string{"env", "env2"})

	projects, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)
	assert.Empty(t, gotErrs)
	assert.Len(t, projects, 1, "expected one project")

	envProfile := findConfig(t, projects[0], "env", "dashboard", 0)
	env2Profile := findConfig(t, projects[0], "env2", "dashboard", 0)

	assert.Contains(t, envProfile.Parameters, config.NonUniqueNameConfigDuplicationParameter, "name duplication parameter should be present")
	assert.Contains(t, env2Profile.Parameters, config.NonUniqueNameConfigDuplicationParameter, "name duplication parameter should be present")
}

func TestLoadProjects_NameDuplicationParameterShouldBePresentIfNameIsDuplicatedOneEnvironment(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.Mkdir("project", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard.yaml", []byte("configs:\n- id: dashboard\n  config:\n    name: Dashboard\n    template: dashboard.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard2.yaml", []byte("configs:\n- id: dashboard2\n  config:\n    name: Dashboard\n    template: dashboard.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard.json", []byte("{}"), 0644))

	loaderContext := getFullProjectLoaderContext(
		[]string{"alerting-profile", "dashboard"},
		[]string{"project"},
		[]string{"env"})

	projects, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)
	assert.Empty(t, gotErrs)
	assert.Len(t, projects, 1, "expected one project")

	envProfile := findConfig(t, projects[0], "env", "dashboard", 0)

	assert.Contains(t, envProfile.Parameters, config.NonUniqueNameConfigDuplicationParameter, "name duplication parameter should be present")
}

func findConfigs(t *testing.T, p Project, e, a string) []config.Config {
	assert.Containsf(t, p.Configs, e, "Expected to find environment '%s'", e)

	env := p.Configs[e]

	return env[a]
}

func findConfig(t *testing.T, p Project, e, a string, cIndex int) config.Config {
	configs := findConfigs(t, p, e, a)
	assert.NotEmpty(t, configs)
	assert.True(t, len(configs) > cIndex, "Config on index %d does not exist. Configs loaded: %d", cIndex, len(configs))

	return configs[cIndex]
}

func TestLoadProjects_LoadsKnownAndUnknownApiNames(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, testFs.MkdirAll("project/not-dashboard-dir", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/not-dashboard-dir/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: unknown-api"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/not-dashboard-dir/board.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 1, "Expected to load project with an error")
	assert.ErrorContains(t, gotErrs[0], "unknown API: unknown-api")
	assert.Len(t, got, 0, "Expected no loaded projects")
}

func TestLoadProjects_LoadsProjectWithConfigAndSettingsConfigurations(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, testFs.MkdirAll("project/dashboard", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/settings.yaml", []byte(`
configs:
- id: setting_one
  type:
    settings:
      schema: "builtin:super.special.schema"
      schemaVersion: "1.42.14"
      scope: "tenant"
  config:
    name: Setting One
    template: my_first_setting.json
- id: setting_two
  type:
    settings:
      schema: "builtin:other.cool.schema"
      scope: "HOST-1234567"
  config:
    name: Setting Two
    template: my_second_setting.json
`), 0644))
	require.NoError(t, testFs.MkdirAll("project", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/my_first_setting.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/my_second_setting.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")

	db := findConfigs(t, got[0], "env", "dashboard")
	assert.Len(t, db, 1, "Expected a one config to be loaded for dashboard")

	a := findConfigs(t, got[0], "env", "alerting-profile")
	assert.Len(t, a, 1, "Expected a one config to be loaded for alerting-profile")

	s1 := findConfigs(t, got[0], "env", "builtin:super.special.schema")
	assert.Len(t, s1, 1, "Expected a one config to be loaded for 'builtin:super.special.schema'")

	s2 := findConfigs(t, got[0], "env", "builtin:other.cool.schema")
	assert.Len(t, s2, 1, "Expected a one config to be loaded for 'builtin:other.cool.schema'")
}

func TestLoadProjects_LoadsProjectConfigsWithCorrectTypeInformation(t *testing.T) {

	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, testFs.MkdirAll("project/dashboard", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/settings.yaml", []byte(`
configs:
- id: setting_one
  type:
    settings:
      schema: "builtin:super.special.schema"
      schemaVersion: "1.42.14"
      scope: "tenant"
  config:
    name: Setting One
    template: my_first_setting.json
- id: setting_two
  type:
    settings:
      schema: "builtin:other.cool.schema"
      scope: "HOST-1234567"
  config:
    name: Setting Two
    template: my_second_setting.json
`), 0644))
	require.NoError(t, testFs.MkdirAll("project", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/my_first_setting.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/my_second_setting.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")

	db := findConfigs(t, got[0], "env", "dashboard")
	assert.Equal(t, db[0].Type, config.ClassicApiType{
		Api: "dashboard",
	})

	a := findConfigs(t, got[0], "env", "alerting-profile")
	assert.Equal(t, a[0].Type, config.ClassicApiType{
		Api: "alerting-profile",
	})

	s1 := findConfigs(t, got[0], "env", "builtin:super.special.schema")
	assert.Equal(t, s1[0].Type, config.SettingsType{
		SchemaId:      "builtin:super.special.schema",
		SchemaVersion: "1.42.14",
	})
	assert.Equal(t, s1[0].Parameters[config.ScopeParameter], &value.ValueParameter{Value: "tenant"})

	s2 := findConfigs(t, got[0], "env", "builtin:other.cool.schema")
	assert.Equal(t, s2[0].Type, config.SettingsType{
		SchemaId:      "builtin:other.cool.schema",
		SchemaVersion: "",
	})
	assert.Equal(t, s2[0].Parameters[config.ScopeParameter], &value.ValueParameter{Value: "HOST-1234567"})
}

func TestLoadProjects_AllowsOverlappingIdsInDifferentApis(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, testFs.MkdirAll("project/dashboard", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")
}

func TestLoadProjects_AllowsOverlappingIdsInDifferentProjects(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, testFs.MkdirAll("project2/alerting-profile", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project2/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project2/alerting-profile/profile.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project", "project2"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 2, "Expected two loaded project")
}

func TestLoadProjects_AllowsOverlappingIdsInEnvironmentOverride(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte(`
configs:
- id: profile
  config:
    name: Test Profile
    template: profile.json
  type:
    api: alerting-profile
  environmentOverrides:
    - environment: env1
      override:
        name: Some Special Name
    - environment: env2
      override:
        skip: true`), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))

	loaderContext := getFullProjectLoaderContext([]string{"alerting-profile"}, []string{"project"}, []string{"env1", "env2"})

	got, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 0, "Expected to load project without error")
	assert.Len(t, got, 1, "Expected a single loaded project")
	assert.Len(t, got[0].Configs["env1"], 1, "Expected one config for env1")
	assert.Len(t, got[0].Configs["env2"], 1, "Expected one config for env2")

	env1ConfCoordinate := got[0].Configs["env1"]["alerting-profile"][0].Coordinate
	env2ConfCoordinate := got[0].Configs["env2"]["alerting-profile"][0].Coordinate
	assert.Equal(t, env1ConfCoordinate, env2ConfCoordinate, "Expected coordinates to be the same between environments")
}

func TestLoadProjects_ContainsCoordinateWhenReturningErrorForDuplicates(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, testFs.MkdirAll("project/dashboard", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile2.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/config.yaml", []byte("configs:\n- id: DASH_OVERLAP\n  config:\n    name: Test Dash\n    template: dash.json\n  type:\n    api: dashboard\n- id: DASH_OVERLAP\n  config:\n    name: Test Dash 2\n    template: dash.json\n  type:\n    api: dashboard"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/dashboard/dash.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 2, "Expected to fail on overlapping coordinates")
	assert.ErrorContains(t, gotErrs[0], "project:alerting-profile:OVERLAP")
	assert.ErrorContains(t, gotErrs[1], "project:dashboard:DASH_OVERLAP")
}

func TestLoadProjects_ReturnsErrOnOverlappingCoordinate_InDifferentFiles(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-some-profile", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-some-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-some-profile/profile2.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-some-profile/profile.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 1, "Expected to fail on overlapping coordinates")
}

func TestLoadProjects_ReturnsErrOnOverlappingCoordinate_InSameFile(t *testing.T) {
	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.MkdirAll("project/alerting-profile", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.yaml",
		[]byte(`configs:
- id: OVERLAP
  config:
    name: Test Profile
    template: profile.json
  type:
	api: alerting-profile
- id: OVERLAP
  type:
	api: alerting-profile
  config:
    name: Some Other Profile
    template: profile.json`), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644))

	loaderContext := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(t.Context(), testFs, loaderContext, nil)

	assert.Len(t, gotErrs, 1, "Expected to fail on overlapping coordinates")
}

func Test_loadProject_returnsErrorIfProjectPathDoesNotExist(t *testing.T) {
	fs := testutils.TempFs(t)
	loaderContext := ProjectLoaderContext{}
	definition := manifest.ProjectDefinition{
		Name: "project",
		Path: "this/does/not/exist",
	}

	_, gotErrs := loadProject(t.Context(), fs, loaderContext, definition, manifest.Environments{})
	assert.Len(t, gotErrs, 1)
	assert.ErrorContains(t, gotErrs[0], "filepath `this/does/not/exist` does not exist")
}

func Test_loadProject_returnsErrorIfScopeForWebKUAhasWrongTypeOfParameter(t *testing.T) {
	testFs := testutils.TempFs(t)
	loaderContext := getFullProjectLoaderContext(
		[]string{"key-user-actions-web"},
		[]string{"project"},
		[]string{"env"})
	definition := manifest.ProjectDefinition{
		Name: "project",
		Path: "project",
	}
	require.NoError(t, testFs.MkdirAll("project/kua-web", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/kua-web/kua-web.yaml",
		[]byte(`configs:
- id: kua-web-1
  config:
    name: Loading of page /example
    template: kua-web.json
    skip: false
  type:
    api:
      name: key-user-actions-web
      scope: APPLICATION-3F2C9E73509D15B6`), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/kua-web/kua-web.json", []byte("{}"), 0644))
	_, gotErrs := loadProject(t.Context(), testFs, loaderContext, definition, manifest.Environments{
		SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
			"testEnv": {Name: "testEnv"},
		},
		AllEnvironmentNames: map[string]struct{}{
			"testenv": {},
		},
	})
	assert.Len(t, gotErrs, 1)
	assert.ErrorContains(t, gotErrs[0], "scope parameter of config of type 'key-user-actions-web' with ID 'kua-web-1' needs to be a reference parameter to another web-application config")
}

func getSimpleProjectLoaderContext(projects []string) ProjectLoaderContext {
	return getTestProjectLoaderContext([]string{"alerting-profile", "dashboard", "key-user-actions-web"}, projects)
}

func getTestProjectLoaderContext(apis []string, projects []string) ProjectLoaderContext {
	return getFullProjectLoaderContext(apis, projects, []string{"env"})
}

func getFullProjectLoaderContext(apis []string, projects []string, environments []string) ProjectLoaderContext {

	projectDefinitions := make(manifest.ProjectDefinitionByProjectID, len(projects))
	for _, p := range projects {
		projectDefinitions[p] = manifest.ProjectDefinition{
			Name: p,
			Path: p + "/",
		}
	}

	allEnvironmentNames := make(map[string]struct{}, len(environments))
	envDefinitions := make(map[string]manifest.EnvironmentDefinition, len(environments))
	for _, e := range environments {
		envDefinitions[e] = manifest.EnvironmentDefinition{
			Name: e,
			Auth: manifest.Auth{
				Token: &manifest.AuthSecret{Name: fmt.Sprintf("%s_VAR", e)},
			},
		}
		allEnvironmentNames[e] = struct{}{}
	}

	knownApis := make(map[string]struct{}, len(apis))
	for _, v := range apis {
		knownApis[v] = struct{}{}
	}

	return ProjectLoaderContext{
		KnownApis:  knownApis,
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: projectDefinitions,
			Environments: manifest.Environments{
				SelectedEnvironments: envDefinitions,
				AllEnvironmentNames:  allEnvironmentNames,
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}
}

func requireProjectsWithNames(t *testing.T, projects []Project, projectNames ...string) {
	require.Equal(t, len(projectNames), len(projects), "Unexpected number of projects")
	requiredProjectNamesMap := make(map[string]struct{}, len(projectNames))
	for _, projectName := range projectNames {
		requiredProjectNamesMap[projectName] = struct{}{}
	}

	for _, project := range projects {
		_, found := requiredProjectNamesMap[project.Id]
		require.True(t, found, "No project found with name %s", project.Id)
		delete(requiredProjectNamesMap, project.Id)
	}
}

func TestLoadProjects_Simple(t *testing.T) {
	managementZoneConfig := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneConfigWithReference := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
    parameters:
      mzId:
        type: reference
        project: a
        configType: builtin:management-zones
        configId: mz
        property: id
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)

	require.NoError(t, testFs.MkdirAll("a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("b/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/config.yaml", managementZoneConfigWithReference, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("c/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"a": {
					Name: "a",
					Path: "a/",
				},
				"b": {
					Name: "b",
					Path: "b/",
				},
				"c": {
					Name: "c",
					Path: "c/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"default": {
						Name: "default",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"default": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	t.Run("loads all projects in manifest if none are specified", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, nil)
		require.Len(t, gotErrs, 0, "Expected no errors loading all projects")
		require.Len(t, gotProjects, 3, "Expected to load 3 projects")
	})

	t.Run("loads specified projects", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"a", "c"})
		require.Len(t, gotErrs, 0, "Expected no errors loading specified projects")
		requireProjectsWithNames(t, gotProjects, "a", "c")
	})

	t.Run("returns error if specified project is not found", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"a", "d"})
		require.Len(t, gotErrs, 1, "Expected error if project is not found")
		require.Len(t, gotProjects, 0, "Expected to load no projects")
		require.Contains(t, gotErrs[0].Error(), "no project named", "Unexpected error message")
	})

	t.Run("also loads dependent projects", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"b"})
		require.Len(t, gotErrs, 0, "Expected no errors loading dependent projects")
		requireProjectsWithNames(t, gotProjects, "b", "a")
	})
}

func TestLoadProjects_Groups(t *testing.T) {
	managementZoneConfig := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)

	require.NoError(t, testFs.MkdirAll("g1/a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "g1/a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "g1/a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("g1/b/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "g1/b/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "g1/b/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("g2/a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "g2/a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "g2/a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("g2/b/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "g2/b/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "g2/b/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("c/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"g1.a": {
					Name:  "g1.a",
					Group: "g1",
					Path:  "g1/a/",
				},
				"g1.b": {
					Name:  "g1.b",
					Group: "g1",
					Path:  "g1/b/",
				},
				"g2.a": {
					Name:  "g2.a",
					Group: "g2",
					Path:  "g2/a/",
				},
				"g2.b": {
					Name:  "g2.b",
					Group: "g2",
					Path:  "g2/b/",
				},
				"c": {
					Name: "c",
					Path: "c/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"default": {
						Name: "default",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"default": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	t.Run("loads all projects in manifest if none are specified", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, nil)
		require.Len(t, gotErrs, 0, "Expected no errors loading all projects")
		require.Len(t, gotProjects, 5, "Expected to load 5 projects")
	})

	t.Run("loads specified projects", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"g1.a", "c"})
		require.Len(t, gotErrs, 0, "Expected no errors loading specified projects")
		requireProjectsWithNames(t, gotProjects, "g1.a", "c")
	})

	t.Run("loads specified groups", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"g1"})
		require.Len(t, gotErrs, 0, "Expected no errors loading specified groups")
		requireProjectsWithNames(t, gotProjects, "g1.a", "g1.b")
	})

	t.Run("loads specified groups and projects", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"g1", "c"})
		require.Len(t, gotErrs, 0, "Expected no errors loading specified groups and projects")
		requireProjectsWithNames(t, gotProjects, "g1.a", "g1.b", "c")
	})

	t.Run("returns error if specified group is not found", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"g3", "c"})
		require.Len(t, gotErrs, 1, "Expected an error if specified group is not found")
		require.Len(t, gotProjects, 0, "Expected to load no projects")
		require.Contains(t, gotErrs[0].Error(), "no project named", "Unexpected error message")
	})
}

func TestLoadProjects_WithEnvironmentOverrides(t *testing.T) {
	managementZoneConfig := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneConfigWithReference := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
    parameters:
      mzId: "ID"
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment
  environmentOverrides:
  - environment: dev
    override:
      parameters:
        mzId:
          type: reference
          project: a
          configType: builtin:management-zones
          configId: mz
          property: id
  - environment: prod
    override:
      parameters:
        mzId:
          type: reference
          project: c
          configType: builtin:management-zones
          configId: mz
          property: id`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)

	require.NoError(t, testFs.MkdirAll("a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("b/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/config.yaml", managementZoneConfigWithReference, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("c/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"a": {
					Name: "a",
					Path: "a/",
				},
				"b": {
					Name: "b",
					Path: "b/",
				},
				"c": {
					Name: "c",
					Path: "c/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"dev": {
						Name: "dev",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
					"prod": {
						Name: "prod",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"dev":  {},
					"prod": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	t.Run("loads all projects in manifest if none are specified", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, nil)
		require.Len(t, gotErrs, 0, "Expected no errors loading all projects")
		require.Len(t, gotProjects, 3, "Expected to load 3 projects")
	})

	t.Run("loads specified projects", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"a"})
		require.Len(t, gotErrs, 0, "Expected no errors loading specified projects")
		requireProjectsWithNames(t, gotProjects, "a")
	})

	t.Run("returns error if specified project is not found", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"d"})
		require.Len(t, gotErrs, 1, "Expected errors if specified project is not found")
		require.Len(t, gotProjects, 0, "Expected to load no projects")
		require.Contains(t, gotErrs[0].Error(), "no project named", "Unexpected error message")
	})

	t.Run("also loads dependent projects", func(t *testing.T) {
		gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"b"})
		require.Len(t, gotErrs, 0, "Expected no errors loading dependent projects")
		requireProjectsWithNames(t, gotProjects, "b", "a", "c")
	})
}

func TestLoadProjects_WithEnvironmentOverridesAndLimitedEnvironments(t *testing.T) {
	managementZoneConfig := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneConfigWithReference := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
    parameters:
      mzId: "ID"
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment
  environmentOverrides:
  - environment: dev
    override:
      parameters:
        mzId:
          type: reference
          project: a
          configType: builtin:management-zones
          configId: mz
          property: id
  - environment: prod
    override:
      parameters:
        mzId:
          type: reference
          project: c
          configType: builtin:management-zones
          configId: mz
          property: id`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)

	require.NoError(t, testFs.MkdirAll("a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("b/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/config.yaml", managementZoneConfigWithReference, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("c/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"a": {
					Name: "a",
					Path: "a/",
				},
				"b": {
					Name: "b",
					Path: "b/",
				},
				"c": {
					Name: "c",
					Path: "c/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"dev": {
						Name: "dev",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"dev": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"b"})
	require.Len(t, gotErrs, 0, "Expected no errors loading dependent projects ")
	requireProjectsWithNames(t, gotProjects, "b", "a")
}

func TestLoadProjects_IgnoresIrrelevantProjectWithErrors(t *testing.T) {
	managementZoneConfig := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneConfigWithError := []byte(`configurations:
- id: mz
  configurations:
    template: mz.json
      skip: false
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)

	require.NoError(t, testFs.MkdirAll("a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("b/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/config.yaml", managementZoneConfigWithError, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"a": {
					Name: "a",
					Path: "a/",
				},
				"b": {
					Name: "b",
					Path: "b/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"dev": {
						Name: "dev",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"dev": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"a"})
	require.Len(t, gotErrs, 0, "Expected no errors loading specified projects")
	requireProjectsWithNames(t, gotProjects, "a")
}

func TestLoadProjects_DeepDependencies(t *testing.T) {
	managementZoneConfig := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneConfigWithReference1 := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
    parameters:
      mzId:
        type: reference
        project: a
        configType: builtin:management-zones
        configId: mz
        property: id
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneConfigWithReference2 := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
    parameters:
      mzId:
        type: reference
        project: b
        configType: builtin:management-zones
        configId: mz
        property: id
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)

	require.NoError(t, testFs.MkdirAll("a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("b/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/config.yaml", managementZoneConfigWithReference1, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("c/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/config.yaml", managementZoneConfigWithReference2, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "c/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"a": {
					Name: "a",
					Path: "a/",
				},
				"b": {
					Name: "b",
					Path: "b/",
				},
				"c": {
					Name: "c",
					Path: "c/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"default": {
						Name: "default",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"default": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"c"})
	require.Len(t, gotErrs, 0, "Expected no errors loading dependent projects")
	requireProjectsWithNames(t, gotProjects, "c", "b", "a")
}

func TestLoadProjects_CircularDependencies(t *testing.T) {
	managementZoneConfigWithReference1 := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
    parameters:
      mzId:
        type: reference
        project: b
        configType: builtin:management-zones
        configId: mz
        property: id
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneConfigWithReference2 := []byte(`configs:
- id: mz
  config:
    template: mz.json
    skip: false
    parameters:
      mzId:
        type: reference
        project: a
        configType: builtin:management-zones
        configId: mz
        property: id
  type:
    settings:
      schema: builtin:management-zones
      schemaVersion: 1.0.9
      scope: environment`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)

	require.NoError(t, testFs.MkdirAll("a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/config.yaml", managementZoneConfigWithReference1, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	require.NoError(t, testFs.MkdirAll("b/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/config.yaml", managementZoneConfigWithReference2, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "b/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"a": {
					Name: "a",
					Path: "a/",
				},
				"b": {
					Name: "b",
					Path: "b/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"default": {
						Name: "default",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"default": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, []string{"b", "a"})
	require.Len(t, gotErrs, 0, "Expected no errors loading dependent projects")
	requireProjectsWithNames(t, gotProjects, "b", "a")
}

func TestLoadProjects_NetworkZonesContainsParameterToSetting(t *testing.T) {

	cfgYaml := `configs:
- id: nz
  config:
    name: NZ
    template: ajson.json
  type:
    api: network-zone
- id: nz2
  config:
    name: NZ2
    template: ajson.json
  type:
    api: network-zone
- id: nz-enabled
  config:
    name: NZ Enabled
    template: ajson.json
  type:
    settings:
      schema: builtin:networkzones
      schemaVersion: 1.0.2
      scope: environment
`

	testFs := testutils.TempFs(t)
	require.NoError(t, testFs.Mkdir("project", 0755))
	require.NoError(t, afero.WriteFile(testFs, "project/config.yaml", []byte(cfgYaml), 0644))
	require.NoError(t, afero.WriteFile(testFs, "project/ajson.json", []byte("{}"), 0644))

	loaderContext := getFullProjectLoaderContext(
		[]string{"network-zone", "builtin:networkzones"},
		[]string{"project"},
		[]string{"env"})

	projects, _ := LoadProjects(t.Context(), testFs, loaderContext, nil)
	networkZone1 := findConfig(t, projects[0], "env", "network-zone", 0)
	assert.Contains(t, networkZone1.Parameters, "__MONACO_NZONE_ENABLED__")

	networkZone2 := findConfig(t, projects[0], "env", "network-zone", 1)
	assert.Contains(t, networkZone2.Parameters, "__MONACO_NZONE_ENABLED__")
}

// TestLoadProjects_EnvironmentOverrideWithUndefinedEnvironmentProducesWarning tests that referencing an undefined environment in an environment override produces a warning.
func TestLoadProjects_EnvironmentOverrideWithUndefinedEnvironmentProducesWarning(t *testing.T) {
	managementZoneConfig := []byte(`configs:
- id: mz
  config:
    template: mz.json
  type:
    settings:
      schema: builtin:management-zones
      scope: environment
  environmentOverrides:
  - environment: prod
    override:
      skip: true
`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)
	logSpy := bytes.Buffer{}
	log.PrepareLogging(t.Context(), afero.NewMemMapFs(), false, &logSpy, false, false)

	require.NoError(t, testFs.MkdirAll("a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))

	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"a": {
					Name: "a",
					Path: "a/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"dev": {
						Name: "dev",
						Auth: manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"dev": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, nil)
	assert.Len(t, gotErrs, 0, "Expected no errors loading dependent projects ")
	assert.Len(t, gotProjects, 1)

	assert.Contains(t, logSpy.String(), "unknown environment")
}

// TestLoadProjects_GroupOverrideWithUndefinedGroupProducesWarning tests that referencing an undefined environment group in a group override produces a warning.
func TestLoadProjects_GroupOverrideWithUndefinedGroupProducesWarning(t *testing.T) {
	managementZoneConfig := []byte(`configs:
- id: mz
  config:
    template: mz.json
  type:
    settings:
      schema: builtin:management-zones
      scope: environment
  groupOverrides:
  - group: prod
    override:
      skip: true
`)

	managementZoneJSON := []byte(`{ "name": "", "rules": [] }`)

	testFs := testutils.TempFs(t)

	logSpy := bytes.Buffer{}
	log.PrepareLogging(t.Context(), afero.NewMemMapFs(), false, &logSpy, false, false)

	require.NoError(t, testFs.MkdirAll("a/builtinmanagement-zones", testDirectoryFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/config.yaml", managementZoneConfig, testFileFileMode))
	require.NoError(t, afero.WriteFile(testFs, "a/builtinmanagement-zones/mz.json", managementZoneJSON, testFileFileMode))
	testContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"builtin:management-zones": {}},
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"a": {
					Name: "a",
					Path: "a/",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"dev": {
						Name:  "dev",
						Group: "dev",
						Auth:  manifest.Auth{Token: &manifest.AuthSecret{Name: "ENV_VAR"}},
					},
				},
				AllEnvironmentNames: map[string]struct{}{
					"dev": {},
				},
				AllGroupNames: map[string]struct{}{
					"dev": {},
				},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	gotProjects, gotErrs := LoadProjects(t.Context(), testFs, testContext, nil)
	assert.Len(t, gotErrs, 0, "Expected no errors loading dependent projects ")
	assert.Len(t, gotProjects, 1)

	assert.Contains(t, logSpy.String(), "unknown group")
}

type propResolver func(coordinate.Coordinate, string) (any, bool)

func (p propResolver) GetResolvedProperty(coordinate coordinate.Coordinate, propertyName string) (any, bool) {
	return p(coordinate, propertyName)
}
