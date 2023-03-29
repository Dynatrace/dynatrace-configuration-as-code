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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"reflect"
	"testing"

	"gotest.tools/assert"
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

func TestLoadProjects_LoadsSimpleProject(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644)
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

func TestLoadProjects_LoadsSimpleProjectInFoldersNotMatchingApiName(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/not-dashboard-dir/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644)
	_ = afero.WriteFile(testFs, "project/not-dashboard-dir/board.json", []byte("{}"), 0644)

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

func TestLoadProjects_LoadsProjectInRootDir(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644)
	_ = afero.WriteFile(testFs, "project/board.json", []byte("{}"), 0644)

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

func TestLoadProjects_LoadsProjectInManyDirs(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/a/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: ../profile.json\n  type:\n    api: alerting-profile\n- id: profile2\n  config:\n    name: Test Profile\n    template: b/c/profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/a/b/c/profile.json", []byte("{}"), 0644)

	_ = afero.WriteFile(testFs, "project/a/b/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: ../../board.json\n  type:\n    api: dashboard"), 0644)
	_ = afero.WriteFile(testFs, "project/board.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context)

	errutils.PrintErrors(gotErrs)
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
	assert.Equal(t, len(a), 2, "Expected a one config to be loaded for alerting-profile")
}

func TestLoadProjects_LoadsProjectInHiddenDirDoesNotLoad(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/.a/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: ../b/profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/b/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/b/profile.json", []byte("{}"), 0644)

	_ = afero.WriteFile(testFs, "project/a/.b/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: ../../board.json\n  type:\n    api: dashboard"), 0644)
	_ = afero.WriteFile(testFs, "project/a/.b/c/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: ../../board.json\n  type:\n    api: dashboard"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context)

	errutils.PrintErrors(gotErrs)
	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 1, "Expected a single loaded project")

	c, found := got[0].Configs["env"]
	assert.Assert(t, found, "Expected configs loaded for test environment")
	assert.Equal(t, len(c), 1, "Expected a alerting-profile configs in loaded project")

	db, found := c["dashboard"]
	assert.Equal(t, found, false, "Expected no configs loaded for dashboard api")
	assert.Equal(t, len(db), 0, "Expected zero config to be loaded for dashboard")

	a, found := c["alerting-profile"]
	assert.Assert(t, found, "Expected configs loaded for dashboard api")
	assert.Equal(t, len(a), 1, "Expected a one config to be loaded for alerting-profile")
}

func TestLoadProjects_LoadsKnownAndUnknownApiNames(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/not-dashboard-dir/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: unknown-api"), 0644)
	_ = afero.WriteFile(testFs, "project/not-dashboard-dir/board.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 1, "Expected to load project with an error")
	assert.ErrorContains(t, gotErrs[0], "unknown API: unknown-api")
	assert.Equal(t, len(got), 0, "Expected no loaded projects")
}

func TestLoadProjects_LoadsProjectWithConfigAndSettingsConfigurations(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/settings.yaml", []byte(`
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
`), 0644)
	_ = afero.WriteFile(testFs, "project/my_first_setting.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/my_second_setting.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 1, "Expected a single loaded project")

	c, found := got[0].Configs["env"]
	assert.Assert(t, found, "Expected configs loaded for test environment")
	assert.Equal(t, len(c), 4, "Expected a dashboard, alerting-profile and two Settings configs in loaded project")

	db, found := c["dashboard"]
	assert.Assert(t, found, "Expected configs loaded for dashboard api")
	assert.Equal(t, len(db), 1, "Expected a one config to be loaded for dashboard")

	a, found := c["alerting-profile"]
	assert.Assert(t, found, "Expected configs loaded for dashboard api")
	assert.Equal(t, len(a), 1, "Expected a one config to be loaded for alerting-profile")

	s1, found := c["builtin:super.special.schema"]
	assert.Assert(t, found, "Expected configs loaded for setting schema 'builtin:super.special.schema'")
	assert.Equal(t, len(s1), 1, "Expected a one config to be loaded for 'builtin:super.special.schema'")

	s2, found := c["builtin:other.cool.schema"]
	assert.Assert(t, found, "Expected configs loaded for setting schema 'builtin:other.cool.schema'")
	assert.Equal(t, len(s2), 1, "Expected a one config to be loaded for 'builtin:other.cool.schema'")
}

func TestLoadProjects_LoadsProjectConfigsWithCorrectTypeInformation(t *testing.T) {

	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: board\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/settings.yaml", []byte(`
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
`), 0644)
	_ = afero.WriteFile(testFs, "project/my_first_setting.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/my_second_setting.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 1, "Expected a single loaded project")

	c, found := got[0].Configs["env"]
	assert.Assert(t, found, "Expected configs loaded for test environment")

	db, found := c["dashboard"]
	assert.Assert(t, found, "Expected configs loaded for dashboard api")
	dbType, ok := db[0].Type.(config.ClassicApiType)
	assert.Assert(t, ok)
	assert.Equal(t, dbType, config.ClassicApiType{
		Api: "dashboard",
	})

	a, found := c["alerting-profile"]
	assert.Assert(t, found, "Expected configs loaded for dashboard api")
	aType, ok := a[0].Type.(config.ClassicApiType)
	assert.Assert(t, ok)
	assert.Equal(t, aType, config.ClassicApiType{
		Api: "alerting-profile",
	})

	s1, found := c["builtin:super.special.schema"]
	assert.Assert(t, found, "Expected configs loaded for setting schema 'builtin:super.special.schema'")
	sType, ok := s1[0].Type.(config.SettingsType)
	assert.Assert(t, ok)
	assert.Equal(t, sType, config.SettingsType{
		SchemaId:      "builtin:super.special.schema",
		SchemaVersion: "1.42.14",
	})
	assert.DeepEqual(t, s1[0].Parameters[config.ScopeParameter], &value.ValueParameter{Value: "tenant"})

	s2, found := c["builtin:other.cool.schema"]
	assert.Assert(t, found, "Expected configs loaded for setting schema 'builtin:other.cool.schema'")
	s2Type, ok := s2[0].Type.(config.SettingsType)
	assert.Equal(t, s2Type, config.SettingsType{
		SchemaId:      "builtin:other.cool.schema",
		SchemaVersion: "",
	})
	assert.DeepEqual(t, s2[0].Parameters[config.ScopeParameter], &value.ValueParameter{Value: "HOST-1234567"})

}

func TestLoadProjects_AllowsOverlappingIdsInDifferentApis(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Dashboard\n    template: board.json\n  type:\n    api: dashboard"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/board.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	got, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 0, "Expected to load project without error")
	assert.Equal(t, len(got), 1, "Expected a single loaded project")
}

func TestLoadProjects_AllowsOverlappingIdsInDifferentProjects(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project2/alerting-profile/profile.yaml", []byte("configs:\n- id: profile\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
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
  type:
    api: alerting-profile
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
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile2.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/config.yaml", []byte("configs:\n- id: DASH_OVERLAP\n  config:\n    name: Test Dash\n    template: dash.json\n  type:\n    api: dashboard\n- id: DASH_OVERLAP\n  config:\n    name: Test Dash 2\n    template: dash.json\n  type:\n    api: dashboard"), 0644)
	_ = afero.WriteFile(testFs, "project/dashboard/dash.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 2, "Expected to fail on overlapping coordinates")
	assert.ErrorContains(t, gotErrs[0], "project:alerting-profile:OVERLAP")
	assert.ErrorContains(t, gotErrs[1], "project:dashboard:DASH_OVERLAP")
}

func TestLoadProjects_ReturnsErrOnOverlappingCoordinate_InDifferentFiles(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-some-profile/profile.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-some-profile/profile2.yaml", []byte("configs:\n- id: OVERLAP\n  config:\n    name: Test Profile\n    template: profile.json\n  type:\n    api: alerting-profile"), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-some-profile/profile.json", []byte("{}"), 0644)

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
  type:
	api: alerting-profile
- id: OVERLAP
  type:
	api: alerting-profile
  config:
    name: Some Other Profile
    template: profile.json`), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)

	context := getSimpleProjectLoaderContext([]string{"project"})

	_, gotErrs := LoadProjects(testFs, context)

	assert.Equal(t, len(gotErrs), 1, "Expected to fail on overlapping coordinates")
}

func Test_loadProject_returnsErrorIfProjectPathDoesNotExist(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := ProjectLoaderContext{}
	definition := manifest.ProjectDefinition{
		Name: "project",
		Path: "this/does/not/exist",
	}

	_, gotErrs := loadProject(fs, ctx, definition, []manifest.EnvironmentDefinition{})
	assert.Assert(t, len(gotErrs) == 1)
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
