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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestClassicReferences(t *testing.T) {
	configFolder := "test-resources/references/"
	manifestFile := configFolder + "manifest.yaml"
	env := "classic_env"
	proj := "classic-apis"

	fs := testutils.CreateTestFileSystem()

	RunIntegrationWithCleanupOnGivenFs(t, fs, configFolder, manifestFile, env, t.Name(), func(fs afero.Fs, ctx TestContext) {

		// upsert
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", env, "--project", proj})
		err := cmd.Execute()
		assert.Nil(t, err, "create: did not expect error")

		// update just to be sure
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", env, "--project", proj})
		err = cmd.Execute()
		assert.Nil(t, err, "update: did not expect error")

		// download
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"download",
			"-v",
			"--manifest", manifestFile,
			"--environment", env,
			"--project", "proj",
			"--output-folder", "download",
			"-a", "alerting-profile,notification,management-zone",
		})
		err = cmd.Execute()
		assert.Nil(t, err, "download: did not expect error")

		// assert
		mani, errs := manifest.LoadManifest(&manifest.LoaderContext{
			Fs:           fs,
			ManifestPath: "download/manifest.yaml",
		})
		assert.Empty(t, errs, "load manifest: did not expect do get error(s)")

		projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
			KnownApis:       api.NewAPIs().GetApiNameLookup(),
			WorkingDir:      "download",
			Manifest:        mani,
			ParametersSerde: config.DefaultParameterParsers,
		})
		assert.Empty(t, errs, "load project: did not expect do get error(s)")

		projectAndEnvName := "proj_classic_env" // for manifest downloads proj + env name

		confsPerType := findConfigs(t, projects, projectAndEnvName)

		managementZone := findConfig(t, confsPerType, "management-zone", "zone_"+ctx.suffix)
		profile := findConfig(t, confsPerType, "alerting-profile", "profile_"+ctx.suffix)
		notification := findConfig(t, confsPerType, "notification", "notification_"+ctx.suffix)

		assertRefParamFromTo(t, profile, managementZone)
		assertRefParamFromTo(t, notification, profile)
	})
}

func TestSettingsReferences(t *testing.T) {
	configFolder := "test-resources/references/"
	manifestFile := configFolder + "manifest.yaml"
	env := "classic_env"
	proj := "settings"

	fs := testutils.CreateTestFileSystem()

	RunIntegrationWithCleanupOnGivenFs(t, fs, configFolder, manifestFile, env, t.Name(), func(fs afero.Fs, ctx TestContext) {

		// upsert
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", env, "--project", proj})
		err := cmd.Execute()
		assert.Nil(t, err, "create: did not expect error")

		// update just to be sure
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", env, "--project", proj})
		err = cmd.Execute()
		assert.Nil(t, err, "update: did not expect error")

		// download
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"download",
			"-v",
			"--manifest", manifestFile,
			"--environment", env,
			"--project", "proj",
			"--output-folder", "download",
			"-s", "builtin:problem.notifications,builtin:management-zones,builtin:alerting.profile",
		})
		err = cmd.Execute()
		assert.Nil(t, err, "download: did not expect error")

		// assert
		mani, errs := manifest.LoadManifest(&manifest.LoaderContext{
			Fs:           fs,
			ManifestPath: "download/manifest.yaml",
		})
		assert.Empty(t, errs, "load manifest: did not expect do get error(s)")

		projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
			KnownApis:       api.NewAPIs().GetApiNameLookup(),
			WorkingDir:      "download",
			Manifest:        mani,
			ParametersSerde: config.DefaultParameterParsers,
		})
		assert.Empty(t, errs, "load project: did not expect do get error(s)")

		projectAndEnvName := "proj_classic_env" // for manifest downloads proj + env name

		confsPerType := findConfigs(t, projects, projectAndEnvName)

		managementZone := findSetting(t, confsPerType, "builtin:management-zones", "zone_"+ctx.suffix, "name")
		profile := findSetting(t, confsPerType, "builtin:alerting.profile", "profile_"+ctx.suffix, "name")
		notification := findSetting(t, confsPerType, "builtin:problem.notifications", "notification_"+ctx.suffix, "displayName")

		assertRefParamFromTo(t, profile, managementZone)
		assertRefParamFromTo(t, notification, profile)
	})
}

func TestSettingsWithConfigMngtZone(t *testing.T) {
	configFolder := "test-resources/references/"
	manifestFile := configFolder + "manifest.yaml"
	env := "classic_env"
	proj := "settings-with-classic-mngt-zone"

	fs := testutils.CreateTestFileSystem()

	RunIntegrationWithCleanupOnGivenFs(t, fs, configFolder, manifestFile, env, "ref-with-mngt-zone", func(fs afero.Fs, ctx TestContext) {

		// upsert
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", env, "--project", proj})
		err := cmd.Execute()
		assert.Nil(t, err, "create: did not expect error")

		// update just to be sure
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", env, "--project", proj})
		err = cmd.Execute()
		assert.Nil(t, err, "update: did not expect error")

		// download
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"download",
			"-v",
			"--manifest", manifestFile,
			"--environment", env,
			"--project", "proj",
			"--output-folder", "download",
			"-a", "management-zone",
			"-s", "builtin:problem.notifications,builtin:alerting.profile",
		})
		err = cmd.Execute()
		assert.Nil(t, err, "download: did not expect error")

		// assert
		mani, errs := manifest.LoadManifest(&manifest.LoaderContext{
			Fs:           fs,
			ManifestPath: "download/manifest.yaml",
		})
		assert.Empty(t, errs, "load manifest: did not expect do get error(s)")

		projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
			KnownApis:       api.NewAPIs().GetApiNameLookup(),
			WorkingDir:      "download",
			Manifest:        mani,
			ParametersSerde: config.DefaultParameterParsers,
		})
		assert.Empty(t, errs, "load project: did not expect do get error(s)")

		projectAndEnvName := "proj_classic_env" // for manifest downloads proj + env name

		confsPerType := findConfigs(t, projects, projectAndEnvName)

		for a, confs := range confsPerType {
			for _, c := range confs {

				nameParam := c.Parameters[config.NameParameter].(*valueParam.ValueParameter).Value
				log.Info("%v/%v", a, nameParam)
			}

		}

		managementZone := findConfig(t, confsPerType, "management-zone", "zone_"+ctx.suffix)
		profile := findSetting(t, confsPerType, "builtin:alerting.profile", "profile_"+ctx.suffix, "name")
		notification := findSetting(t, confsPerType, "builtin:problem.notifications", "notification_"+ctx.suffix, "displayName")

		assertRefParamFromTo(t, profile, managementZone)
		assertRefParamFromTo(t, notification, profile)
	})
}

func TestClassicReferencesWithSettingsManagementZone(t *testing.T) {
	configFolder := "test-resources/references/"
	manifestFile := configFolder + "manifest.yaml"
	env := "classic_env"
	proj := "config-with-mngt-zone"

	fs := testutils.CreateTestFileSystem()

	RunIntegrationWithCleanupOnGivenFs(t, fs, configFolder, manifestFile, env, "ref-with-mngt-zone", func(fs afero.Fs, ctx TestContext) {

		// upsert
		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", env, "--project", proj})
		err := cmd.Execute()
		assert.Nil(t, err, "create: did not expect error")

		// update just to be sure
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "-v", manifestFile, "--environment", env, "--project", proj})
		err = cmd.Execute()
		assert.Nil(t, err, "update: did not expect error")

		// download
		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"download",
			"-v",
			"--manifest", manifestFile,
			"--environment", env,
			"--project", "proj",
			"--output-folder", "download",
			"-a", "notification,alerting-profile",
			"-s", "builtin:management-zones",
		})
		err = cmd.Execute()
		assert.Nil(t, err, "download: did not expect error")

		// assert
		mani, errs := manifest.LoadManifest(&manifest.LoaderContext{
			Fs:           fs,
			ManifestPath: "download/manifest.yaml",
		})
		assert.Empty(t, errs, "load manifest: did not expect do get error(s)")

		projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
			KnownApis:       api.NewAPIs().GetApiNameLookup(),
			WorkingDir:      "download",
			Manifest:        mani,
			ParametersSerde: config.DefaultParameterParsers,
		})
		assert.Empty(t, errs, "load project: did not expect do get error(s)")

		projectAndEnvName := "proj_classic_env" // for manifest downloads proj + env name

		confsPerType := findConfigs(t, projects, projectAndEnvName)

		for a, confs := range confsPerType {
			for _, c := range confs {

				nameParam := c.Parameters[config.NameParameter].(*valueParam.ValueParameter).Value
				log.Info("%v/%v", a, nameParam)
			}

		}

		managementZone := findSetting(t, confsPerType, "builtin:management-zones", "zone_"+ctx.suffix, "name")
		profile := findConfig(t, confsPerType, "alerting-profile", "profile_"+ctx.suffix)
		notification := findConfig(t, confsPerType, "notification", "notification_"+ctx.suffix)

		assertRefParamFromTo(t, profile, managementZone)
		assertRefParamFromTo(t, notification, profile)
	})
}

func assertRefParamFromTo(t *testing.T, from config.Config, to config.Config) {
	name := paramName(to.Coordinate.Type, to.Coordinate.ConfigId)
	param, found := from.Parameters[name]
	assert.Truef(t, found, "expected to find parameter %q", name)

	assert.Equal(t, param, refParam.NewWithCoordinate(to.Coordinate, "id"))
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

		content := c.Template.Content()
		// convert content to json
		var jsonContent map[string]interface{}
		err := json.Unmarshal([]byte(content), &jsonContent)
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

var templatePattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func paramName(typ, id string) string {
	n := fmt.Sprintf("%v__%v__id", typ, id)
	return templatePattern.ReplaceAllString(n, "")
}
