//go:build integration_v1
// +build integration_v1

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

package v1

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/v2/runner"
	"os"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

//TestRestoreConfigs validates if the configurations can be restore from the downloaded version after being deleted
//It has 5 stages:
//Preparation: Uploads a set of configurations and return the virtual filesystem
//Execution: Download the configurations to the virtual filesystem
//Cleanup: Deletes the configurations that were uploaded during validation
//Validation: Uploads the downloaded configs and checks for status code 0 as result
//Cleanup: Deletes the configurations that were uploaded during validation

// This version runs the test against 2 simple configs (alerting profiles and management zones)
func TestRestoreConfigsSimple(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs/"
	envFile := initialConfigsFolder + "environments.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "download1"
	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, envFile, subsetOfConfigsToDownload)
}

// This version runs the test against the all_configs project, currently fails because of config dependencies
//
//	func TestRestoreConfigsFull(t *testing.T) {
//		initialConfigsFolder := "test-resources/integration-all-configs/"
//		envFile := initialConfigsFolder + "environments.yaml"
//		downloadFolder := "test-resources/download_all_configs"
//		subsetOfConfigsToDownload := "all" //value only for testing
//		suffixTest := "dl1"
//		testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, envFile, subsetOfConfigsToDownload)
//	}
func testRestoreConfigs(t *testing.T, initialConfigsFolder string, downloadFolder string, suffixTest string, envFile string, apisToDownload string) {
	t.Setenv("CONFIG_V1", "1")
	fs := util.CreateTestFileSystem()
	err := preparation_uploadConfigs(t, fs, suffixTest, initialConfigsFolder, envFile)

	assert.NilError(t, err, "Error during download preparation stage")

	err = execution_downloadConfigs(t, fs, downloadFolder, envFile, apisToDownload, suffixTest)
	assert.NilError(t, err, "Error during download execution stage")

	cleanupEnvironmentConfigs(t, fs, envFile, suffixTest)
	validation_uploadDownloadedConfigs(t, fs, downloadFolder, envFile)
	cleanupEnvironmentConfigs(t, fs, envFile, suffixTest)
}

func preparation_uploadConfigs(t *testing.T, fs afero.Fs, suffixTest string, configFolder string, envFile string) error {
	log.Info("BEGIN PREPARATION PROCESS")
	suffix := getTimestamp() + suffixTest
	transformers := []func(string) string{getTransformerFunc(suffix)}
	err := util.RewriteConfigNames(configFolder, fs, transformers)
	if err != nil {
		log.Fatal("Error rewriting configs names")
		return err
	}
	//uploads the configs

	cmd := runner.BuildCli(fs)
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--environments", envFile,
		configFolder,
	})
	err = cmd.Execute()
	assert.NilError(t, err)

	return nil
}
func execution_downloadConfigs(t *testing.T, fs afero.Fs, downloadFolder string, envFile string,
	apisToDownload string, suffixTest string) error {
	log.Info("BEGIN DOWNLOAD PROCESS")
	//Download
	//err := fs.MkdirAll(downloadFolder, 0777)
	//if err != nil {
	//	return err
	//}
	downloadFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}
	parameters := []string{}

	if apisToDownload == "all" {
		parameters = []string{
			"download",
			"--verbose",
			"--environments", envFile,
			downloadFolder,
		}
	} else {
		parameters = []string{
			"download",
			"--verbose",
			"--specific-api",
			apisToDownload,
			"--environments", envFile,
			downloadFolder,
		}
	}

	cmd := runner.BuildCli(fs)
	cmd.SetArgs(parameters)
	err = cmd.Execute()
	assert.NilError(t, err)
	return nil
}
func validation_uploadDownloadedConfigs(t *testing.T, fs afero.Fs, downloadFolder string,
	envFile string) {
	log.Info("BEGIN VALIDATION PROCESS")
	//Shows you the downloaded files list in the command line
	_ = afero.Walk(fs, downloadFolder+"/", func(path string, info os.FileInfo, err error) error {
		fpath, err := filepath.Abs(path)
		log.Info("file " + fpath)
		return nil
	})

	cmd := runner.BuildCli(fs)
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		"--environments", envFile,
		downloadFolder,
	})
	err := cmd.Execute()
	assert.NilError(t, err)
}

// Deletes all configs that end with "_suffix", where suffix == suffixTest+suffixTimestamp
func cleanupEnvironmentConfigs(t *testing.T, fs afero.Fs, envFile, suffix string) {
	log.Info("BEGIN CLEANUP PROCESS")
	environments, errs := environment.LoadEnvironmentList("", envFile, fs)
	FailOnAnyError(errs, "loading of environments failed")

	apis := api.NewApis()

	for _, environment := range environments {

		token, err := environment.GetToken()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), token)
		assert.NilError(t, err)

		for _, api := range apis {

			values, err := client.List(api)
			assert.NilError(t, err)

			for _, value := range values {
				// For the calculated-metrics-log API, the suffix is part of the ID, not name
				if strings.HasSuffix(value.Name, suffix) || strings.HasSuffix(value.Id, suffix) {
					log.Info("Deleting %s (%s)", value.Name, api.GetId())
					client.DeleteByName(api, value.Name)
				}
			}
		}
	}
}
