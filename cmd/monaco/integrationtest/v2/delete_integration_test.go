//go:build integration

/*
 * @license
 * Copyright 2023 Dynatrace LLC
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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
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
	}{
		{
			name:                  "Default values",
			manifest:              "manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "delete.yaml",
			deleteContentTemplate: deleteContentTemplate,
		},
		{
			name:                  "Default values - legacy delete",
			manifest:              "manifest.yaml",
			configTemplate:        "configs:\n- id: %s\n  type:\n    api: auto-tag\n  config:\n    name: %s\n    template: auto-tag.json\n",
			deleteFile:            "delete.yaml",
			deleteContentTemplate: "delete:\n  - \"auto-tag/%s\"",
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
		},
		{
			name:                  "Specific manifest",
			manifest:              "my_special_manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "delete.yaml",
			deleteContentTemplate: deleteContentTemplate,
			cmdFlag:               "--manifest=my_special_manifest.yaml",
		},
		{
			name:                  "Specific manifest (shorthand)",
			manifest:              "my_special_manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "delete.yaml",
			deleteContentTemplate: deleteContentTemplate,
			cmdFlag:               "--manifest=my_special_manifest.yaml",
		},
		{
			name:                  "Specific delete file",
			manifest:              "manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "super-special-removal-file.yaml",
			deleteContentTemplate: deleteContentTemplate,
			cmdFlag:               "--file=super-special-removal-file.yaml",
		},
		{
			name:                  "Specific manifest and delete file",
			manifest:              "my_special_manifest.yaml",
			configTemplate:        configTemplate,
			deleteFile:            "super-special-removal-file.yaml",
			deleteContentTemplate: deleteContentTemplate,
			cmdFlag:               "--manifest=my_special_manifest.yaml --file=super-special-removal-file.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t1 *testing.T) {
			configFolder := "test-resources/delete-test-configs/"
			deployManifestPath := configFolder + "deploy-manifest.yaml"

			fs := testutils.CreateTestFileSystem()

			// create config yaml
			cfgId := fmt.Sprintf("deleteSample_%s", integrationtest.GenerateTestSuffix(t, tt.name))
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
			err = monaco.RunWithFs(t, fs, fmt.Sprintf("monaco deploy %s --verbose", deployManifestPath))
			assert.NoError(t1, err)
			integrationtest.AssertAllConfigsAvailability(t1, fs, deployManifestPath, []string{}, "", true)

			// DELETE Config
			err = monaco.RunWithFs(t, fs, fmt.Sprintf("monaco delete %s --verbose", tt.cmdFlag))
			assert.NoError(t1, err)
			integrationtest.AssertAllConfigsAvailability(t1, fs, deployManifestPath, []string{}, "", false)

		})
	}
}

func TestDeleteSkipsPlatformTypesWhenDeletingFromClassicEnv(t *testing.T) {

	configFolder := "test-resources/delete-test-configs/"
	deployManifestPath := configFolder + "deploy-manifest.yaml"

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
	workflowID := fmt.Sprintf("workflowSample_%s", integrationtest.GenerateTestSuffix(t, "skip_automations"))
	bucketID := fmt.Sprintf("bucket_%s", integrationtest.GenerateTestSuffix(t, "")) // generate shorter name does not reach API limit
	tagID := fmt.Sprintf("tagSample_%s", integrationtest.GenerateTestSuffix(t, "skip_automations"))
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
	err = monaco.RunWithFs(t, fs, fmt.Sprintf("monaco deploy %s --verbose", deployManifestPath))
	assert.NoError(t, err)
	integrationtest.AssertAllConfigsAvailability(t, fs, deployManifestPath, []string{}, "", true)
	// ensure test resources are removed after test is done
	t.Cleanup(func() {
		monaco.RunWithFs(t, fs, "monaco delete --manifest=test-resources/delete-test-configs/deploy-manifest.yaml --verbose")
	})

	// DELETE Configs - with API Token only Manifest
	err = monaco.RunWithFs(t, fs, "monaco delete --verbose")
	assert.NoError(t, err)

	// Assert expected deletions
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: "test-resources/delete-test-configs/deploy-manifest.yaml", // full manifest with oAuth
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	assert.Empty(t, errs)

	envName := "environment"
	env := man.Environments[envName]
	clientSet := integrationtest.CreateDynatraceClients(t, env)

	// check the setting was deleted
	integrationtest.AssertSetting(t, clientSet.SettingsClient, config.SettingsType{SchemaId: "builtin:tags.auto-tagging"}, envName, false, config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "builtin:tags.auto-tagging",
			ConfigId: tagID,
		},
	})

	// check the workflow still exists after deletion was skipped without error
	integrationtest.AssertAutomation(t, clientSet.AutClient, env, true, config.Workflow, config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "workflow",
			ConfigId: workflowID,
		},
	})

	// check the bucket still exists after deletion was skipped without error
	integrationtest.AssertBucket(t, clientSet.BucketClient, env, true, config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "bucket",
			ConfigId: bucketID,
		},
	})
}

func TestDeleteSubPathAPIConfigurations(t *testing.T) {
	configFolder := "test-resources/delete-test-configs/"
	deployManifestPath := configFolder + "deploy-manifest.yaml"

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
	appName := fmt.Sprintf("app_%s", integrationtest.GenerateTestSuffix(t, "subpath_delete"))
	actionName := fmt.Sprintf("key_ua_%s", integrationtest.GenerateTestSuffix(t, "subpath_delete"))

	configContent := fmt.Sprintf(configTemplate, appName, actionName)

	configYamlPath, err := filepath.Abs(filepath.Join(configFolder, "project", "config.yaml"))
	assert.NoError(t, err)
	err = afero.WriteFile(fs, configYamlPath, []byte(configContent), 644)
	assert.NoError(t, err)

	// DEPLOY Config
	err = monaco.RunWithFs(t, fs, fmt.Sprintf("monaco deploy %s --verbose", deployManifestPath))
	require.NoError(t, err)
	integrationtest.AssertAllConfigsAvailability(t, fs, deployManifestPath, []string{}, "", true)

	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: deployManifestPath,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	assert.Empty(t, errs)

	envName := "environment"
	env := man.Environments[envName]
	clientSet := integrationtest.CreateDynatraceClients(t, env)
	apis := api.NewAPIs()

	// ASSERT test configs exist
	integrationtest.AssertAllConfigsAvailability(t, fs, deployManifestPath, []string{}, "", true)

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

	err = monaco.RunWithFs(t, fs, fmt.Sprintf("monaco delete --manifest %s --verbose", deployManifestPath))
	require.NoError(t, err)

	// Assert key-user-action is deleted
	integrationtest.AssertConfig(t, clientSet.ConfigClient, apis["key-user-actions-mobile"].ApplyParentObjectID(appID), env, false, config.Config{
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

	err = monaco.RunWithFs(t, fs, fmt.Sprintf("monaco delete --manifest %s --verbose", deployManifestPath))
	require.NoError(t, err)

	// Assert expected deletions
	integrationtest.AssertAllConfigsAvailability(t, fs, deployManifestPath, []string{}, "", false)
}

func TestDeleteWithOAuthOrTokenOnlyManifest(t *testing.T) {
	configFolder := "test-resources/delete-test-configs/"
	fs := testutils.CreateTestFileSystem()

	t.Run("OAuth only should not throw error but skip delete for Classic API", func(t *testing.T) {
		// DELETE Config
		deleteFileName := configFolder + "oauth-delete.yaml"
		cmdFlag := "--manifest=" + configFolder + "oauth-only-manifest.yaml --file=" + deleteFileName
		err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco delete %s --verbose", cmdFlag))
		assert.NoError(t, err)

		logFile := log.LogFilePath()
		_, err = afero.Exists(fs, logFile)
		assert.NoError(t, err)

		// assert log for skipped deletion
		log, err := afero.ReadFile(fs, logFile)
		assert.NoError(t, err)
		assert.Contains(t, string(log), "Skipped deletion of 1 Classic configuration(s) as API client was unavailable")
	})

	t.Run("Token only should not throw error but skip delete for Automation API", func(t *testing.T) {
		// DELETE Config
		deleteFileName := configFolder + "token-delete.yaml"
		cmdFlag := "--manifest=" + configFolder + "token-only-manifest.yaml --file=" + deleteFileName
		err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco delete %s --verbose", cmdFlag))
		assert.NoError(t, err)

		logFile := log.LogFilePath()
		_, err = afero.Exists(fs, logFile)
		assert.NoError(t, err)

		// assert log for skipped deletion
		log, err := afero.ReadFile(fs, logFile)
		assert.NoError(t, err)
		assert.Contains(t, string(log), "Skipped deletion of 1 Automation configuration(s)")
	})
}
