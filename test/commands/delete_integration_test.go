//go:build integration

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package commands

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	assert2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/assert"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestDelete(t *testing.T) {

	deleteContentTemplate := `delete:
- project: "project"
  type: "builtin:tags.auto-tagging"
  id: "%s"`
	configTemplate := "configs:\n- id: %s\n  type:\n    settings:\n      schema: builtin:tags.auto-tagging\n      scope: environment\n  config:\n    name: %s\n    template: auto-tag-setting.json\n"

	tests := []struct {
		name                  string
		manifest              string
		configTemplate        string
		deleteFile            string
		deleteContentTemplate string
		cmdFlag               string
		deployManifest        string
	}{
		{
			name:                  "Default values",
			manifest:              "manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "delete.yaml",
			deleteContentTemplate: deleteContentTemplate,
			deployManifest:        "deploy-manifest-with-oauth.yaml",
		},
		{
			name:                  "Default values - legacy delete",
			manifest:              "manifest.yaml",
			configTemplate:        "configs:\n- id: %s\n  type:\n    api: auto-tag\n  config:\n    name: %s\n    template: auto-tag.json\n",
			deleteFile:            "delete.yaml",
			deleteContentTemplate: "delete:\n  - \"auto-tag/%s\"",
			deployManifest:        "deploy-manifest-with-oauth.yaml",
		},
		{
			name:           "Default values - Automation",
			manifest:       "manifest.yaml",
			configTemplate: "configs:\n- id: %s\n  type:\n    automation:\n      resource: workflow\n  config:\n    name: %s\n    template: workflow.json\n",
			deleteFile:     "delete.yaml",
			deleteContentTemplate: `delete:
- project: "project"
  type: "workflow"
  id: "%s"`,
			deployManifest: "deploy-manifest-with-oauth.yaml",
		},
		{
			name:           "Default values - Automation w Platform token",
			manifest:       "manifest.yaml",
			configTemplate: "configs:\n- id: %s\n  type:\n    automation:\n      resource: workflow\n  config:\n    name: %s\n    template: workflow.json\n",
			deleteFile:     "delete.yaml",
			deleteContentTemplate: `delete:
- project: "project"
  type: "workflow"
  id: "%s"`,
			deployManifest: "deploy-manifest-with-platform-token.yaml",
		},
		{
			name:                  "Specific manifest",
			manifest:              "my_special_manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "delete.yaml",
			deleteContentTemplate: deleteContentTemplate,
			cmdFlag:               "--manifest=my_special_manifest.yaml",
			deployManifest:        "deploy-manifest-with-oauth.yaml",
		},
		{
			name:                  "Specific manifest (shorthand)",
			manifest:              "my_special_manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "delete.yaml",
			deleteContentTemplate: deleteContentTemplate,
			cmdFlag:               "--manifest=my_special_manifest.yaml",
			deployManifest:        "deploy-manifest-with-oauth.yaml",
		},
		{
			name:                  "Specific delete file",
			manifest:              "manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "super-special-removal-file.yaml",
			deleteContentTemplate: deleteContentTemplate,
			cmdFlag:               "--file=super-special-removal-file.yaml",
			deployManifest:        "deploy-manifest-with-oauth.yaml",
		},
		{
			name:                  "Specific manifest and delete file",
			manifest:              "my_special_manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "super-special-removal-file.yaml",
			deleteContentTemplate: deleteContentTemplate,
			cmdFlag:               "--manifest=my_special_manifest.yaml --file=super-special-removal-file.yaml",
			deployManifest:        "deploy-manifest-with-oauth.yaml",
		},
	}

	t.Setenv(featureflags.PlatformToken.EnvName(), "true")
	for _, tt := range tests {
		t.Run(tt.name, func(t1 *testing.T) {
			configFolder := "testdata/delete-test-configs/"
			deployManifestPath := configFolder + tt.deployManifest

			fs := testutils.CreateTestFileSystem()

			// create config yaml
			cfgId := fmt.Sprintf("deleteSample_%s", runner.GenerateTestSuffix(t, tt.name))
			configContent := fmt.Sprintf(tt.configTemplate, cfgId, cfgId)

			configYamlPath, err := filepath.Abs(filepath.Join(configFolder, "project", "config.yaml"))
			assert.NoError(t1, err)
			err = afero.WriteFile(fs, configYamlPath, []byte(configContent), 644)
			assert.NoError(t1, err)

			// create delete yaml
			deleteContent := fmt.Sprintf(tt.deleteContentTemplate, cfgId)
			deleteYamlPath, err := filepath.Abs(tt.deleteFile)
			assert.NoError(t1, err)
			err = afero.WriteFile(fs, deleteYamlPath, []byte(deleteContent), 644)
			assert.NoError(t1, err)

			// create manifest file
			manifestContent, err := afero.ReadFile(fs, deployManifestPath)
			assert.NoError(t1, err)
			manifestPath, err := filepath.Abs(tt.manifest)
			err = afero.WriteFile(fs, manifestPath, manifestContent, 644)
			assert.NoError(t1, err)

			// DEPLOY Config
			err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", deployManifestPath))
			assert.NoError(t1, err)
			assert2.AssertAllConfigsAvailability(t1, fs, deployManifestPath, []string{}, "", true)

			// DELETE Config
			err = monaco.Run(t, fs, fmt.Sprintf("monaco delete %s --verbose", tt.cmdFlag))
			assert.NoError(t1, err)
			assert2.AssertAllConfigsAvailability(t1, fs, deployManifestPath, []string{}, "", false)

		})
	}
}

func TestDeleteSkipsPlatformTypesWhenDeletingFromClassicEnv(t *testing.T) {

	configFolder := "testdata/delete-test-configs/"
	deployManifestPath := configFolder + "deploy-manifest-with-oauth.yaml"

	fs := testutils.CreateTestFileSystem()

	// create config yaml
	configTemplate := `
configs:
- id: %s
  type:
    automation:
      resource: workflow
  config:
    name: %s
    template: workflow.json
- id: %s
  type: bucket
  config:
    template: bucket.json
- id: %s
  type:
    settings:
      schema: builtin:tags.auto-tagging
      scope: environment
  config:
    name: %s
    template: auto-tag-setting.json`
	workflowID := fmt.Sprintf("workflowSample_%s", runner.GenerateTestSuffix(t, "skip_automations"))
	bucketID := fmt.Sprintf("bucket_%s", runner.GenerateTestSuffix(t, "")) // generate shorter name does not reach API limit
	tagID := fmt.Sprintf("tagSample_%s", runner.GenerateTestSuffix(t, "skip_automations"))
	configContent := fmt.Sprintf(configTemplate, workflowID, workflowID, bucketID, tagID, tagID)

	configYamlPath, err := filepath.Abs(filepath.Join(configFolder, "project", "config.yaml"))
	assert.NoError(t, err)
	err = afero.WriteFile(fs, configYamlPath, []byte(configContent), 644)
	assert.NoError(t, err)

	// create delete yaml
	deleteTemplate := `delete:
  - project: "project"
    type: "workflow"
    id: "%s"
  - project: "project"
    type: "bucket"
    id: "%s"
  - project: "project"
    type: "builtin:tags.auto-tagging"
    id: "%s"`

	deleteContent := fmt.Sprintf(deleteTemplate, workflowID, bucketID, tagID)
	deleteYamlPath, err := filepath.Abs("delete.yaml")
	assert.NoError(t, err)
	err = afero.WriteFile(fs, deleteYamlPath, []byte(deleteContent), 644)
	assert.NoError(t, err)

	// create manifest file without oAuth
	manifestContent := `manifestVersion: 1.0
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: environment
    url:
      type: environment
      value: URL_ENVIRONMENT_1
    auth:
      token:
        name: TOKEN_ENVIRONMENT_1`
	assert.NoError(t, err)
	manifestPath, err := filepath.Abs("manifest.yaml")
	err = afero.WriteFile(fs, manifestPath, []byte(manifestContent), 644)
	assert.NoError(t, err)

	// DEPLOY Config
	err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", deployManifestPath))
	assert.NoError(t, err)
	assert2.AssertAllConfigsAvailability(t, fs, deployManifestPath, []string{}, "", true)
	// ensure test resources are removed after test is done
	defer func() {
		monaco.Run(t, fs, "monaco delete --manifest=testdata/delete-test-configs/deploy-manifest-with-oauth.yaml --verbose")
	}()

	// DELETE Configs - with access token only Manifest
	err = monaco.Run(t, fs, "monaco delete --verbose")
	assert.NoError(t, err)

	// Assert expected deletions
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: "testdata/delete-test-configs/deploy-manifest-with-oauth.yaml", // full manifest with oAuth
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	assert.Empty(t, errs)

	envName := "environment"
	env := man.Environments.SelectedEnvironments[envName]
	clientSet := runner.CreateDynatraceClients(t, env)

	// check the setting was deleted
	assert2.AssertSetting(t, clientSet.SettingsClient, config.SettingsType{SchemaId: "builtin:tags.auto-tagging"}, envName, false, config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "builtin:tags.auto-tagging",
			ConfigId: tagID,
		},
	})

	// check the workflow still exists after deletion was skipped without error
	assert2.AssertAutomation(t, clientSet.AutClient, env, true, config.Workflow, config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "workflow",
			ConfigId: workflowID,
		},
	})

	// check the bucket still exists after deletion was skipped without error
	assert2.AssertBucket(t, clientSet.BucketClient, env, true, config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "bucket",
			ConfigId: bucketID,
		},
	})
}

func TestDeleteSubPathAPIConfigurations(t *testing.T) {
	configFolder := "testdata/delete-test-configs/"
	deployManifestPath := configFolder + "deploy-manifest-with-oauth.yaml"

	fs := testutils.CreateTestFileSystem()

	// create config yaml
	configTemplate := `
configs:
- id: app
  type: application-mobile
  config:
    name: %s
    template: application-mobile.json
- id: action
  type:
    api:
      name: key-user-actions-mobile
      scope:
        type: reference
        configType: application-mobile
        configId: app
        property: id
  config:
    name: %s
    template: key-user-action.json
`
	appName := fmt.Sprintf("app_%s", runner.GenerateTestSuffix(t, "subpath_delete"))
	actionName := fmt.Sprintf("key_ua_%s", runner.GenerateTestSuffix(t, "subpath_delete"))

	configContent := fmt.Sprintf(configTemplate, appName, actionName)

	configYamlPath, err := filepath.Abs(filepath.Join(configFolder, "project", "config.yaml"))
	assert.NoError(t, err)
	err = afero.WriteFile(fs, configYamlPath, []byte(configContent), 644)
	assert.NoError(t, err)

	// DEPLOY Config
	err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", deployManifestPath))
	require.NoError(t, err)

	// Extra sleep to ensure that the application is available - this is added to prevent HTTP 500 errors occuring later in deletion.
	time.Sleep(60 * time.Second)

	assert2.AssertAllConfigsAvailability(t, fs, deployManifestPath, []string{}, "", true)

	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: deployManifestPath,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	assert.Empty(t, errs)

	envName := "environment"
	env := man.Environments.SelectedEnvironments[envName]
	clientSet := runner.CreateDynatraceClients(t, env)
	apis := api.NewAPIs()

	// ASSERT test configs exist
	assert2.AssertAllConfigsAvailability(t, fs, deployManifestPath, []string{}, "", true)

	// get application ID
	v, err := clientSet.ConfigClient.List(t.Context(), apis["application-mobile"])
	assert.NoError(t, err)

	var appID string
	for _, app := range v {
		if app.Name == appName {
			appID = app.Id
		}
	}
	assert.NotEmpty(t, appID, "found no app with name ", appName)

	// Only DELETE key-user action config, as deleting the application would auto-remove it
	subPathOnlyDeleteTemplate := `delete:
  - type: "key-user-actions-mobile"
    scope: "%s"
    name: "%s"`

	deleteContent := fmt.Sprintf(subPathOnlyDeleteTemplate, appID, actionName)
	deleteYamlPath, err := filepath.Abs("delete.yaml")
	assert.NoError(t, err)
	err = afero.WriteFile(fs, deleteYamlPath, []byte(deleteContent), 644)
	assert.NoError(t, err)

	err = monaco.Run(t, fs, fmt.Sprintf("monaco delete --manifest %s --verbose", deployManifestPath))
	require.NoError(t, err)

	// Assert key-user-action is deleted
	assert2.AssertConfig(t, clientSet.ConfigClient, apis["key-user-actions-mobile"].ApplyParentObjectID(appID), env, false, config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "key-user-actions-mobile",
			ConfigId: "action",
		}}, actionName)

	// DELETE all
	fullDeleteTemplate := `delete:
  - type: "application-mobile"
    name: "%s"
  - type: "key-user-actions-mobile"
    scope: "%s"
    name: "%s"`

	deleteContent = fmt.Sprintf(fullDeleteTemplate, appName, appID, actionName)
	deleteYamlPath, err = filepath.Abs("delete.yaml")
	assert.NoError(t, err)
	err = afero.WriteFile(fs, deleteYamlPath, []byte(deleteContent), 644)
	assert.NoError(t, err)

	err = monaco.Run(t, fs, fmt.Sprintf("monaco delete --manifest %s --verbose", deployManifestPath))
	require.NoError(t, err)

	// Assert expected deletions
	assert2.AssertAllConfigsAvailability(t, fs, deployManifestPath, []string{}, "", false)
}

func TestDeleteWithOAuthOrTokenOnlyManifest(t *testing.T) {
	configFolder := "testdata/delete-test-configs/"
	fs := testutils.CreateTestFileSystem()

	t.Run("OAuth only should not throw error but skip delete for Classic API", func(t *testing.T) {
		// DELETE Config
		deleteFileName := configFolder + "oauth-delete.yaml"
		cmdFlag := "--manifest=" + configFolder + "oauth-only-manifest.yaml --file=" + deleteFileName
		err := monaco.Run(t, fs, fmt.Sprintf("monaco delete %s --verbose", cmdFlag))
		assert.NoError(t, err)

		logFile := log.LogFilePath()
		_, err = afero.Exists(fs, logFile)
		assert.NoError(t, err)

		// assert log for skipped deletion
		log, err := afero.ReadFile(fs, logFile)
		assert.NoError(t, err)
		assert.Contains(t, string(log), "Skipped deletion of 1 aws-credentials configuration(s) as API client was unavailable")
	})

	t.Run("Platform token only should not throw error but skip delete for Classic API", func(t *testing.T) {
		t.Setenv(featureflags.PlatformToken.EnvName(), "true")

		// DELETE Config
		deleteFileName := configFolder + "platform-token-delete.yaml"
		cmdFlag := "--manifest=" + configFolder + "platform-token-only-manifest.yaml --file=" + deleteFileName
		err := monaco.Run(t, fs, fmt.Sprintf("monaco delete %s --verbose", cmdFlag))
		assert.NoError(t, err)

		logFile := log.LogFilePath()
		_, err = afero.Exists(fs, logFile)
		assert.NoError(t, err)

		// assert log for skipped deletion
		log, err := afero.ReadFile(fs, logFile)
		assert.NoError(t, err)
		assert.Contains(t, string(log), "Skipped deletion of 1 aws-credentials configuration(s) as API client was unavailable")
	})

	t.Run("Token only should not throw error but skip delete for Automation API", func(t *testing.T) {
		// DELETE Config
		deleteFileName := configFolder + "token-delete.yaml"
		cmdFlag := "--manifest=" + configFolder + "token-only-manifest.yaml --file=" + deleteFileName
		err := monaco.Run(t, fs, fmt.Sprintf("monaco delete %s --verbose", cmdFlag))
		assert.NoError(t, err)

		logFile := log.LogFilePath()
		_, err = afero.Exists(fs, logFile)
		assert.NoError(t, err)

		// assert log for skipped deletion
		log, err := afero.ReadFile(fs, logFile)
		assert.NoError(t, err)
		assert.Contains(t, string(log), "Skipped deletion of 1 workflow configuration(s)")
	})
}
