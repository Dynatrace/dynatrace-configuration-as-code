//go:build integration

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
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

// TestOnlyStringReferences explicitly tests the "only create references in string values" feature.
// The test creates an network zone with the ID "0", and when the feature flag is turned off, Monaco creates references to it often such as in coordinates for dashboard tiles.
// When turned on, such references are not created.
func TestOnlyStringReferences(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, dashboardParameters config.Parameters)
	}{
		{
			name:    "With feature flag enabled",
			envVars: map[string]string{featureflags.OnlyCreateReferencesInStringValues.EnvName(): "true"},
			validate: func(t *testing.T, dashboardParameters config.Parameters) {
				assert.Len(t, dashboardParameters, 1, "dashboard should only have one parameter")
				_, ok := dashboardParameters["name"]
				assert.True(t, ok, "dashboard should only have a name parameter")
			},
		},
		{
			name:    "With feature flag disabled",
			envVars: map[string]string{featureflags.OnlyCreateReferencesInStringValues.EnvName(): "false"},
			validate: func(t *testing.T, dashboardParameters config.Parameters) {
				assert.Len(t, dashboardParameters, 2, "dashboard should have two parameters")
				_, ok := dashboardParameters["name"]
				assert.True(t, ok, "dashboard should have a name parameter")

				_, ok = dashboardParameters["networkzone__0__id"]
				assert.True(t, ok, "dashboard should reference a network zone")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			configFolder := "test-resources/string-references/"
			manifestFile := configFolder + "manifest.yaml"

			fs := testutils.CreateTestFileSystem()

			proj := "string-references"
			env := "classic_env"

			RunIntegrationWithCleanupOnGivenFsAndEnvs(t, fs, configFolder, manifestFile, env, "string_references", tt.envVars, func(fs afero.Fs, ctx TestContext) {

				// upsert
				err := monaco.RunWithFSf(fs, "monaco deploy %s --environment=%s --project=%s --verbose", manifestFile, env, proj)
				require.NoError(t, err, "create: did not expect error")

				// download
				err = monaco.RunWithFSf(fs, "monaco download --manifest=%s --environment=%s --project=proj --output-folder=download --verbose %s", manifestFile, env, "--api=dashboard,network-zone")
				require.NoError(t, err, "download: did not expect error")

				// load downloaded project
				mani, errs := manifestloader.Load(&manifestloader.Context{
					Fs:           fs,
					ManifestPath: "download/manifest.yaml",
					Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
				})
				assert.Empty(t, errs, "unexpected error loading manifest")

				projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
					KnownApis:       api.NewAPIs().GetApiNameLookup(),
					WorkingDir:      "download",
					Manifest:        mani,
					ParametersSerde: config.DefaultParameterParsers,
				}, nil)
				assert.Empty(t, errs, "unexpected error loading project")

				// find dashboard
				projectAndEnvName := "proj_" + env // for manifest downloads proj + env name
				confsPerType := findConfigs(t, projects, projectAndEnvName)
				dashboard := findConfig(t, confsPerType, "dashboard", "Dashboard 0_"+ctx.suffix)

				require.NotEmpty(t, dashboard.Coordinate.ConfigId)

				tt.validate(t, dashboard.Parameters)
			})
		})
	}
}

func TestReferencesAreResolvedOnDownload(t *testing.T) {

	envs := []string{"classic_env", "platform_env"}

	tests := []struct {
		project      string
		downloadOpts string
		validate     func(t *testing.T, ctx TestContext, confsPerType project.ConfigsPerType)
	}{
		{
			project:      "classic-apis",
			downloadOpts: "--api=alerting-profile,notification,management-zone",
			validate: func(t *testing.T, ctx TestContext, confsPerType project.ConfigsPerType) {
				managementZone := findConfig(t, confsPerType, "management-zone", "zone-ca_"+ctx.suffix)
				profile := findConfig(t, confsPerType, "alerting-profile", "profile-ca_"+ctx.suffix)
				notification := findConfig(t, confsPerType, "notification", "notification-ca_"+ctx.suffix)

				assertRefParamFromTo(t, profile, managementZone)
				assertRefParamFromTo(t, notification, profile)
			},
		},
		{
			project:      "settings",
			downloadOpts: "--settings-schema=builtin:problem.notifications,builtin:management-zones,builtin:alerting.profile",
			validate: func(t *testing.T, ctx TestContext, confsPerType project.ConfigsPerType) {
				managementZone := findSetting(t, confsPerType, "builtin:management-zones", "zone_"+ctx.suffix, "name")
				profile := findSetting(t, confsPerType, "builtin:alerting.profile", "profile_"+ctx.suffix, "name")
				notification := findSetting(t, confsPerType, "builtin:problem.notifications", "notification_"+ctx.suffix, "displayName")

				assertRefParamFromTo(t, profile, managementZone)
				assertRefParamFromTo(t, notification, profile)
			},
		},
		{
			project:      "classic-with-settings-mngt-zone",
			downloadOpts: "--api=notification,alerting-profile --settings-schema=builtin:management-zones",
			validate: func(t *testing.T, ctx TestContext, confsPerType project.ConfigsPerType) {
				managementZone := findSetting(t, confsPerType, "builtin:management-zones", "zone-cws_"+ctx.suffix, "name")
				profile := findConfig(t, confsPerType, "alerting-profile", "profile-cws_"+ctx.suffix)
				notification := findConfig(t, confsPerType, "notification", "notification-cws_"+ctx.suffix)

				assertRefParamFromTo(t, profile, managementZone)
				assertRefParamFromTo(t, notification, profile)
			},
		},
	}

	for _, env := range envs {
		for _, tt := range tests {
			testName := env + "_" + tt.project

			t.Run(testName, func(t *testing.T) {
				configFolder := "test-resources/references/"
				manifestFile := configFolder + "manifest.yaml"
				proj := tt.project

				fs := testutils.CreateTestFileSystem()

				RunIntegrationWithCleanupOnGivenFs(t, fs, configFolder, manifestFile, env, testName, func(fs afero.Fs, ctx TestContext) {

					// upsert
					err := monaco.RunWithFSf(fs, "monaco deploy %s --environment=%s --project=%s --verbose", manifestFile, env, proj)
					require.NoError(t, err, "create: did not expect error")

					// download
					err = monaco.RunWithFSf(fs, "monaco download --manifest=%s --environment=%s --project=proj --output-folder=download --verbose %s", manifestFile, env, tt.downloadOpts)
					require.NoError(t, err, "download: did not expect error")

					// assert
					mani, errs := manifestloader.Load(&manifestloader.Context{
						Fs:           fs,
						ManifestPath: "download/manifest.yaml",
						Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
					})
					assert.Empty(t, errs, "load manifest: did not expect do get error(s)")

					projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
						KnownApis:       api.NewAPIs().GetApiNameLookup(),
						WorkingDir:      "download",
						Manifest:        mani,
						ParametersSerde: config.DefaultParameterParsers,
					}, nil)
					assert.Empty(t, errs, "load project: did not expect do get error(s)")

					projectAndEnvName := "proj_" + env // for manifest downloads proj + env name

					confsPerType := findConfigs(t, projects, projectAndEnvName)

					tt.validate(t, ctx, confsPerType)
				})
			})
		}
	}
}

func TestReferencesAreValid(t *testing.T) {
	configFolder := "test-resources/references/"
	manifestFile := configFolder + "manifest.yaml"

	err := monaco.Runf("monaco deploy %s --environment=platform_env --dry-run --verbose", manifestFile)
	assert.NoError(t, err, "expected configurations to be valid")
}

func TestReferencesFromClassicConfigsToSettingsResultInError(t *testing.T) {
	configFolder := "test-resources/references/"
	manifestFile := configFolder + "invalid-configs-manifest.yaml"

	fs := testutils.CreateTestFileSystem()
	logOutput := strings.Builder{}

	cmd := runner.BuildCmdWithLogSpy(fs, &logOutput)
	cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", "platform_env", "--dry-run"})
	err := cmd.Execute()
	assert.Error(t, err, "expected invalid configurations to result in user error")

	runLog := strings.ToLower(logOutput.String())
	assert.Contains(t, runLog, "can only reference ids of other config api types")
	assert.Contains(t, runLog, "parameter \\\"alertingprofileid\\\" references \\\"builtin:alerting.profile\\\" type")
}

func assertRefParamFromTo(t *testing.T, from config.Config, to config.Config) {
	assert.Contains(t, from.References(), to.Coordinate)
}

func findConfigs(t *testing.T, projects []project.Project, id string) project.ConfigsPerType {
	var proj *project.Project
	for i := range projects {
		if projects[i].Id == id {
			proj = &projects[i]
			break
		}
	}

	assert.NotNilf(t, proj, "Project %q not found. Projects: %v", id, projects)

	confs, found := proj.Configs[id]
	assert.Truef(t, found, "environment %q not found. Environments: %v", id, maps.Keys(confs))

	return confs
}

func findConfig(t *testing.T, confsPerType project.ConfigsPerType, api, name string) config.Config {
	confs, found := confsPerType[api]
	assert.Truef(t, found, "api %q not found, known configs: %q", api, maps.Keys(confsPerType))

	for _, c := range confs {
		// we can be quite sure that the name is always a value after a download
		nameParam, ok := c.Parameters[config.NameParameter].(*valueParam.ValueParameter)
		assert.True(t, ok, "name should be a value param")

		if nameParam.Value == name {
			return c
		}
	}

	assert.Failf(t, "failed to find config '%s/%s'", api, name)
	return config.Config{}
}

func findSetting(t *testing.T, confsPerType project.ConfigsPerType, api, name, property string) config.Config {
	confs, found := confsPerType[api]
	assert.Truef(t, found, "api %q not found, known configs: %q", api, maps.Keys(confsPerType))

	for _, c := range confs {

		content, err := c.Template.Content()
		assert.NoError(t, err)
		// convert content to json
		var jsonContent map[string]interface{}
		err = json.Unmarshal([]byte(content), &jsonContent)
		assert.Nil(t, err, "failed to unmarshal content to json")

		// get the setting name
		n := jsonContent[property].(string)
		if n == name {
			return c
		}
	}

	assert.Failf(t, "failed to find config '%s/%s' in property %q", api, name, property)
	return config.Config{}
}
