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

package v2

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func Test_findDuplicatedConfigIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		input []config.Config
		want  []config.Config
	}{
		{
			"nil input produces empty output",
			nil,
			nil,
		},
		{
			"no duplicates in single config",
			[]config.Config{{Coordinate: coordinate.Coordinate{ConfigId: "id"}}},
			nil,
		},
		{
			"no duplicates if project differs",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project1", Type: "api", ConfigId: "id"}},
			},
			nil,
		},
		{
			"no duplicates if api differs",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "aws-credentials", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "azure-credentials", ConfigId: "id"}},
			},
			nil,
		},
		{
			"no duplicates in list of disparate configs",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project1", Type: "api1", ConfigId: "id1"}},
			},
			nil,
		},
		{
			"finds duplicate configs",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"}},
			},
			[]config.Config{{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}}},
		},
		{
			"finds each duplicate",
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"}},
			},
			[]config.Config{
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
				{Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findDuplicatedConfigIdentifiers(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findDuplicatedConfigIdentifiers() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadProjects_RejectsManifestsWithNoProjects(t *testing.T) {
	testFs := testutils.TempFs(t)
	context := getSimpleProjectLoaderContext([]string{})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)
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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)
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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getFullProjectLoaderContext(
		[]string{"alerting-profile", "dashboard"},
		[]string{"project"},
		[]string{"env"})

	projects, gotErrs := LoadProjects(testFs, context, nil)
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

	context := getFullProjectLoaderContext(
		[]string{"alerting-profile", "dashboard"},
		[]string{"project"},
		[]string{"env", "env2"})

	projects, gotErrs := LoadProjects(testFs, context, nil)
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

	context := getFullProjectLoaderContext(
		[]string{"alerting-profile", "dashboard"},
		[]string{"project"},
		[]string{"env", "env2"})

	projects, gotErrs := LoadProjects(testFs, context, nil)
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

	context := getFullProjectLoaderContext(
		[]string{"alerting-profile", "dashboard"},
		[]string{"project"},
		[]string{"env"})

	projects, gotErrs := LoadProjects(testFs, context, nil)
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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project", "project2"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getFullProjectLoaderContext([]string{"alerting-profile"}, []string{"project"}, []string{"env1", "env2"})

	got, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(testFs, context, nil)

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

	context := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(testFs, context, nil)

	assert.Len(t, gotErrs, 1, "Expected to fail on overlapping coordinates")
}

func Test_loadProject_returnsErrorIfProjectPathDoesNotExist(t *testing.T) {
	fs := testutils.TempFs(t)
	ctx := ProjectLoaderContext{}
	definition := manifest.ProjectDefinition{
		Name: "project",
		Path: "this/does/not/exist",
	}

	_, gotErrs := loadProject(fs, ctx, definition, []manifest.EnvironmentDefinition{})
	assert.Len(t, gotErrs, 1)
	assert.ErrorContains(t, gotErrs[0], "filepath `this/does/not/exist` does not exist")
}

func getSimpleProjectLoaderContext(projects []string) ProjectLoaderContext {
	return getTestProjectLoaderContext([]string{"alerting-profile", "dashboard"}, projects)
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

	envDefinitions := make(map[string]manifest.EnvironmentDefinition, len(environments))
	for _, e := range environments {
		envDefinitions[e] = manifest.EnvironmentDefinition{
			Name: e,
			Auth: manifest.Auth{
				Token: manifest.AuthSecret{Name: fmt.Sprintf("%s_VAR", e)},
			},
		}
	}

	knownApis := make(map[string]struct{}, len(apis))
	for _, v := range apis {
		knownApis[v] = struct{}{}
	}

	return ProjectLoaderContext{
		KnownApis:  knownApis,
		WorkingDir: ".",
		Manifest: manifest.Manifest{
			Projects:     projectDefinitions,
			Environments: envDefinitions,
		},
		ParametersSerde: config.DefaultParameterParsers,
	}
}
